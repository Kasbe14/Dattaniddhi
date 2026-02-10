package index

import (
	v "VectorDatabase/internal/vector"
	"cmp"
	"errors"
	"slices"
	"sync"
)

// Initial index state is empty, no dimension assigned, no lock
// after first add index state each index gets its own fixed dimension, lock ensures dimension doesn't change
type LinearIndex struct {
	mu           sync.RWMutex
	vectors      map[string]*v.Vector
	config       IndexConfig
	configLocked bool
}

// initial empty index doesn't have any config i.e no identity
func NewLinearIndex() *LinearIndex {
	return &LinearIndex{
		mu:           sync.RWMutex{},
		vectors:      make(map[string]*v.Vector),
		configLocked: false,
	}
}
func (li *LinearIndex) Dimension() int {
	li.mu.RLock()
	defer li.mu.RUnlock()
	return li.config.Dimension
}

func (li *LinearIndex) Add(vec *v.Vector) error {
	li.mu.Lock()
	defer li.mu.Unlock()
	key := vec.ID()
	// TODO IN VECTORS API FOR DataType and SimilarityMetric
	vecDataType := vec.DataType()
	vecSimMetric := vec.Metric()
	vecDim := vec.Dimensions()
	if key == "" {
		return errors.New("vector id empty")
	}
	_, ok := li.vectors[key]
	if ok {
		return errors.New("the index key already exists")
	}
	if !li.configLocked {
		li.config.Dimension = vecDim
		li.config.DataType = vecDataType
		li.config.Metric = vecSimMetric
		li.configLocked = true
	} else {
		if li.config.Dimension != vecDim {
			return errors.New("index and vector dimension mismatch")
		}
		if li.config.DataType != vecDataType {
			return errors.New("data type mismatch")
		}
		if li.config.Metric != vecSimMetric {
			return errors.New("similarity mismatch")
		}
	}
	li.vectors[key] = vec
	return nil
}
func (li *LinearIndex) Delete(id string) error {
	li.mu.Lock()
	defer li.mu.Unlock()
	_, ok := li.vectors[id]
	if !ok {
		return errors.New("vector doesn't exist in index")
	} else {
		delete(li.vectors, id)
		return nil
	}
}
func (li *LinearIndex) Get(id string) (*v.Vector, bool) {
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
	if li.config.DataType != query.DataType() {
		return nil, errors.New("index and vector data type mismatch")
	}
	if li.config.Metric != query.Metric() {
		return nil, errors.New("index and query similarity metric mismatch")
	}
	if k <= 0 {
		return nil, errors.New("invalid input for number of results")
	}
	// for k >= index size might need li.Size() memory capacity
	result := make([]SearchResult, 0, li.Size())
	for key, val := range li.vectors {
		simScore, err := query.Similarity(val)
		if err != nil {
			return nil, err
		}
		result = append(result, SearchResult{
			vecId: key,
			score: simScore,
		})
	}
	//sort descending similarity score
	slices.SortFunc(result, func(a, b SearchResult) int {
		return cmp.Compare(b.score, a.score)
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
