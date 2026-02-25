package collection

import (
	"github.com/Kasbe14/Dattaniddhi/internal/index"
	"github.com/Kasbe14/Dattaniddhi/internal/types"
	"github.com/Kasbe14/Dattaniddhi/internal/vector"
	"errors"
	"sync"

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
}

type Result struct {
	VecID string
	Score float64
}

func NewCollection(cfg CollectionConfig) (*Collection, error) {
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
	}, nil
}

func (c *Collection) Insert(vecVals []float32, payload any) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(vecVals) != c.config.Dimension {
		return "", ErrInvalidDimension
	}
	//Constructing the vector
	vector, err := vector.NewVector(vecVals, c.config.Dimension)
	if err != nil {
		return "", err
	}
	//generate external id
	externalID := uuid.NewString()
	//internal id and total vector add in index ever
	c.idCounter += 1
	internalID := c.idCounter
	//storing and mapping ids
	c.extToInt[externalID] = internalID
	c.intToExt[internalID] = externalID

	//Add vector to index
	added, err := c.index.Add(internalID, vector)
	//rollback if id exist internal courruption
	if err != nil {
		delete(c.extToInt, externalID)
		delete(c.intToExt, internalID)
		return "", err
	}
	if added {
		delete(c.extToInt, externalID)
		delete(c.intToExt, internalID)
		return "", ErrInternalIDCollision
	}
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
	err := c.index.Delete(internalID)
	if err != nil {
		return err
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
