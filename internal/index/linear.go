package index

import (
	v "VectorDatabase/internal/vector"
	"cmp"
	"errors"
	"slices"
)

type LinearIndex struct {
	vectors map[string]*v.Vector
}

func NewLinearIndex() *LinearIndex {
	return &LinearIndex{
		vectors: make(map[string]*v.Vector),
	}
}

func (li *LinearIndex) Add(vec *v.Vector) error {
	key := vec.ID()
	if key == "" {
		return errors.New("vector id empty")
	}
	_, ok := li.vectors[key]
	if ok {
		return errors.New("the index key already exists")
	}
	li.vectors[vec.ID()] = vec

	return nil
}
func (li *LinearIndex) Delete(id string) error {
	_, ok := li.vectors[id]
	if !ok {
		return errors.New("vector doesn't exist in index")
	} else {
		delete(li.vectors, id)
		return nil
	}
}
func (li *LinearIndex) Get(id string) (*v.Vector, bool) {
	vec, ok := li.vectors[id]
	return vec, ok
}
func (li *LinearIndex) Search(query *v.Vector, k int) ([]SearchResult, error) {
	if query == nil {
		return nil, errors.New("empty query input")
	}
	if k <= 0 {
		return nil, errors.New("invalid input for number of results")
	}
	if li.Size() == 0 {
		return nil, nil
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
	slices.SortFunc(result, func(a, b SearchResult) int {
		return cmp.Compare(b.score, a.score)
	})
	if k > li.Size() {
		return result, nil
	}
	return result[:k], nil
}
func (li *LinearIndex) Size() int {
	return len(li.vectors)
}

var _ VectorIndex = (*LinearIndex)(nil)
