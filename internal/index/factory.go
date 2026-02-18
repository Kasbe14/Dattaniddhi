package index

import (
	"VectorDatabase/internal/types"
)

type IndexFactory interface {
	CreateIndex(cfg IndexConfig) (VectorIndex, error)
}

// empty struct to implement IndexFactory and bind Registery struct and interface
type DefaultIndexFactory struct {
}

func (d DefaultIndexFactory) CreateIndex(cfg IndexConfig) (VectorIndex, error) {
	switch cfg.IndexType() {
	case types.LinearIndex:
		return NewLinearIndex(cfg)
	// case IndexHNSW :
	// 	return NewHNSWIndex()
	// case IndexIVF :
	// 	return NewIVFIndex()
	default:
		panic("unsupported index type")
	}
}
