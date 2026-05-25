package collection

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/Kasbe14/Dattaniddhi/internal/index"
	"github.com/Kasbe14/Dattaniddhi/internal/store/wal"
	"github.com/Kasbe14/Dattaniddhi/internal/types"
	"github.com/Kasbe14/Dattaniddhi/internal/vector"
	"github.com/google/uuid"
)

// collection owns index and its lifecyle ensures collection level invariants
type Collection struct {
	mu        sync.RWMutex
	config    CollectionConfig
	index     index.VectorIndex
	idCounter int
	//id mappings exteranl user usage internal internal processing
	extToInt map[string]int
	intToExt map[int]string
	//  Playload in-memory
	payload map[string]any
	wal     *wal.WAL
}

type Result struct {
	VecID string
	Score float64
}

func NewCollection(cfg CollectionConfig, wal *wal.WAL) (*Collection, error) {
	if cfg.Dimension <= 0 {
		return nil, ErrInvalidDimension
	}
	if cfg.Name == "" {
		return nil, ErrInvalidCollectionName
	}
	switch cfg.Metric {
	case types.Cosine, types.Dot, types.Euclidean:
	//ok valid input
	default:
		return nil, ErrInvalidMetric
	}
	switch cfg.IndexType {
	case types.HNSWIndex, types.LinearIndex, types.IVFIndex, types.PQIndex:
	//ok valid input
	default:
		return nil, ErrInvalidIndexType
	}
	indexConfig, err := index.NewIndexConfig(cfg.IndexType, cfg.Metric, cfg.Dimension)
	if err != nil {
		return nil, err
	}
	//pointer value satisfying IndexFactory interface
	//indexFactory := &index.DefaultIndexFactory{}
	var indexFactory index.DefaultIndexFactory
	//return new index instance of type cfg.IndexType
	idx, err := indexFactory.CreateIndex(indexConfig)
	if err != nil {
		return nil, err
	}

	return &Collection{
		config:    cfg,
		index:     idx,
		idCounter: 0,
		extToInt:  make(map[string]int),
		intToExt:  make(map[int]string),
		payload:   make(map[string]any),
		wal:       wal,
	}, nil
}

func (c *Collection) Insert(vecVals []float32, payload any) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. Validation phase (Fails fast, no state changed)
	if len(vecVals) != c.config.Dimension {
		return "", ErrInvalidDimension
	}
	vector, err := vector.NewVector(vecVals, c.config.Dimension)
	if err != nil {
		return "", err
	}
	metaData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	// 2. Prepare data (Still no state permanently changed)
	externalID := uuid.NewString()
	// Calculate the NEXT internal ID, but don't commit it to c.idCounter yet!
	internalID := c.idCounter + 1

	// 3. The Point of No Return: Write to WAL
	_, err = c.wal.AppendInsert(externalID, uint64(internalID), vecVals, metaData)
	if err != nil {
		// If disk fails, we just return. No memory was mutated, so nothing to clean up!
		return "", err
	}

	// 4. Memory Mutation (The WAL succeeded, now we update everything)
	added, err := c.index.Add(internalID, vector)
	if err != nil || !added {
		// CRITICAL EDGE CASE: If the WAL succeeded but the memory index fails,
		// the DB is now in an inconsistent state.
		// You must append a compensation record (Delete) to the WAL to cancel it out!
		_, walErr := c.wal.AppendDelete(externalID, uint64(internalID))
		if walErr != nil {
			// If appending the delete fails, the disk is truly in a bad state.
			// Standard databases would trigger a panic or safe shutdown here.
			return "", fmt.Errorf("index failed: %v, critical wal rollback failed: %v", err, walErr)
		}

		if err != nil {
			return "", err
		}
		return "", ErrInternalIDCollision
	}

	// 5. Finalize State
	// The index succeeded, so we finally commit the ID counter and maps
	c.idCounter = internalID
	c.extToInt[externalID] = internalID
	c.intToExt[internalID] = externalID
	c.payload[externalID] = payload

	return externalID, nil
}
func (c *Collection) Search(queryVals []float32, k int) ([]Result, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.config.Dimension != len(queryVals) {
		return []Result{}, ErrInvalidDimension
	}
	queryVector, err := vector.NewVector(queryVals, c.config.Dimension)
	if err != nil {
		return []Result{}, err
	}
	idxResult, err := c.index.Search(queryVector, k)
	if err != nil {
		return []Result{}, err
	}
	colResult := make([]Result, len(idxResult))

	for i, val := range idxResult {
		if _, ok := c.intToExt[val.VecId]; !ok {
			return []Result{}, errors.New("id doesn't exist internal corruption")
		}
		colResult[i] = Result{c.intToExt[val.VecId], val.Score}
	}
	return colResult, nil
}
func (c *Collection) Delete(id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	internalID, ok := c.extToInt[id]
	if !ok {
		return ErrNotFound
	}
	//payload := c.payload // used for soft deletes tombstones(easeir rollback)
	// write/append operation to the Wal segment
	_, err := c.wal.AppendDelete(id, uint64(internalID))
	if err != nil {
		return err
	}
	err = c.index.Delete(internalID)
	if err != nil {
		// The disk and memory are now permanently out of sync.
		// Solution: Crash the database immediately to prevent data corruption.
		panic(fmt.Sprintf("FATAL: Index failed to delete %s, WAL out of sync: %v", id, err))
	}
	delete(c.extToInt, id)
	delete(c.intToExt, internalID)
	delete(c.payload, id)
	return nil
}
func (c *Collection) Get(id string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	payload, ok := c.payload[id]
	return payload, ok
}
