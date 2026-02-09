package index

type IndexFactory interface {
	CreateIndex(cfg IndexConfig) VectorIndex
}

// empty struct to implement IndexFactory and bind Registery struct and interface
type DefaultIndexFactory struct {
}

func (d *DefaultIndexFactory) CreateIndex(cfg IndexConfig) VectorIndex {
	switch cfg.IndexType {
	case IndexLinear:
		return NewLinearIndex()
	// case IndexHNSW :
	// 	return NewHNSWIndex()
	// case IndexIVF :
	// 	return NewIVFIndex()
	default:
		panic("unsupported index type")
	}
}
