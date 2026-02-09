package ingest

import (
	"VectorDatabase/internal/index"
	"sync"
)

// to store different indexes
type indexRegistry struct {
	mu       sync.RWMutex
	registry map[index.IndexConfig]index.VectorIndex
	factory  index.IndexFactory
}

func NewIndexRegistry(f index.IndexFactory) *indexRegistry {
	return &indexRegistry{
		registry: make(map[index.IndexConfig]index.VectorIndex),
		factory:  f,
	}
}

func (ir *indexRegistry) GetOrCreateIndex(cfg index.IndexConfig) index.VectorIndex {
	//write lock
	ir.mu.Lock()
	defer ir.mu.Unlock()
	//Return if index Already exists with config locked
	if idx, ok := ir.registry[cfg]; ok {
		return idx
	}
	//If index doesn't exit create and return new instance of a empty index, unlocked config and add to lookup register
	idx := ir.factory.CreateIndex(cfg)
	ir.registry[cfg] = idx
	return idx
}
