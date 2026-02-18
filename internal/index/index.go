package index

import (
	v "VectorDatabase/internal/vector"
)

type VectorIndex interface {
	Add(id int, v *v.Vector) (bool, error)
	Delete(id int) error
	Get(id int) (*v.Vector, bool)
	Search(query *v.Vector, k int) ([]SearchResult, error)
	Size() int
}
