package index

import (
	"github.com/Kasbe14/Dattaniddhi/internal/types"
	"github.com/Kasbe14/Dattaniddhi/internal/vector"
	v "github.com/Kasbe14/Dattaniddhi/internal/vector"
	"cmp"
	"errors"
	"fmt"
	"slices"
	"sync"
)

// Initial index state is empty, no dimension assigned, no lock
// after first add index state each index gets its own fixed dimension,
// IndexConfig is now Imutable and index is schema driven not data driven i.e first IndexConfig structure is defined
type LinearIndex struct {
	mu      sync.RWMutex
	vectors map[int]*v.Vector
	config  IndexConfig
}

// Index must know its invariants at birth, IndexConfig enforces invariants
func NewLinearIndex(cfg IndexConfig) (*LinearIndex, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed to initialize linear index: %w", err)
	}
	return &LinearIndex{
		mu:      sync.RWMutex{},
		vectors: make(map[int]*v.Vector),
		config:  cfg,
	}, nil
}
func (li *LinearIndex) Dimension() int {
	li.mu.RLock()
	defer li.mu.RUnlock()
	return li.config.Dimension()
}

// Returns true if vector already exist else error
func (li *LinearIndex) Add(id int, vec *v.Vector) (bool, error) {
	li.mu.Lock()
	defer li.mu.Unlock()
	if id < 0 {
		return false, errors.New("invalid ID")
	}
	if vec == nil {
		return false, errors.New("empty vector")
	}
	if li.config.Dimension() != vec.Dimensions() {
		return false, errors.New("dimension mismatch")
	}
	_, ok := li.vectors[id]
	if ok {
		return true, nil
	}
	li.vectors[id] = vec
	return false, nil
}
func (li *LinearIndex) Delete(id int) error {
	li.mu.Lock()
	defer li.mu.Unlock()
	_, ok := li.vectors[id]
	if !ok {
		return errors.New("vector doesn't exist in index")
	}
	delete(li.vectors, id)
	return nil

}
func (li *LinearIndex) Get(id int) (*v.Vector, bool) {
	li.mu.RLock()
	defer li.mu.RUnlock()
	vec, ok := li.vectors[id]
	return vec, ok
}
func (li *LinearIndex) Search(query *v.Vector, k int) ([]SearchResult, error) {
	li.mu.RLock()
	defer li.mu.RUnlock()
	if li.Size() == 0 {
		return nil, nil
	}
	if query == nil {
		return nil, errors.New("empty query input")
	}
	if li.Dimension() != query.Dimensions() {
		return nil, errors.New("index and query dimension mismatched")
	}
	// if li.config.DataType != query.DataType() {
	// 	return nil, errors.New("index and vector data type mismatch")
	// }
	// if li.config.Metric != query.Metric() {
	// 	return nil, errors.New("index and query similarity metric mismatch")
	// }
	if k <= 0 {
		return nil, errors.New("invalid input for number of results")
	}
	// for k >= index size might need li.Size() memory capacity
	result := make([]SearchResult, 0, li.Size())
	//sort descending similarity score
	qVal := query.Values()
	switch li.config.Metric() {
	case types.Cosine:
		for key, val := range li.vectors {
			simScore, err := vector.Cosine(val.Values(), qVal)
			if err != nil {
				return nil, err
			}
			result = append(result, SearchResult{
				VecId: key,
				Score: simScore,
			})
		}
		// For current design vector values are nomalized during creation so cosine = dot, (might change later)
	case types.Dot:
		for key, val := range li.vectors {
			simScore, err := vector.DotProduct(val.Values(), qVal)
			if err != nil {
				return nil, err
			}
			result = append(result, SearchResult{
				VecId: key,
				Score: simScore,
			})
		}
	case types.Euclidean:
		for key, val := range li.vectors {
			simScore, err := vector.Euclidean(val.Values(), qVal)
			if err != nil {
				return nil, err
			}
			result = append(result, SearchResult{
				VecId: key,
				Score: simScore,
			})
		}
	}
	slices.SortFunc(result, func(a, b SearchResult) int {
		return cmp.Compare(b.Score, a.Score)
	})
	if k > li.Size() {
		return result, nil
	}
	return result[:k], nil
}
func (li *LinearIndex) Size() int {
	li.mu.RLock()
	defer li.mu.RUnlock()
	return len(li.vectors)
}

var _ VectorIndex = (*LinearIndex)(nil)
