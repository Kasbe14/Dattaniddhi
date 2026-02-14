package index

import (
	"VectorDatabase/internal/types"
	"errors"
)

type IndexConfig struct {
	indexType types.IndexType
	metric    types.SimilarityMetric
	dimension int
}

// IndexConfig constructor with invariants checks
func NewIndexConfig(
	indexType types.IndexType,
	metric types.SimilarityMetric,
	dimension int,
) (IndexConfig, error) {
	if dimension <= 0 {
		return IndexConfig{}, errors.New("invalid dimension")
	}
	switch indexType {
	case types.LinearIndex, types.HNSWIndex, types.IVFIndex, types.PQIndex:
		//ok valid input
	default:
		return IndexConfig{}, errors.New("invalid index type")
	}
	switch metric {
	case types.Cosine, types.Dot, types.Euclidean:
		//ok valid metrics
	default:
		return IndexConfig{}, errors.New("invalid metric type")
	}
	return IndexConfig{
		indexType: indexType,
		metric:    metric,
		dimension: dimension,
	}, nil

}

//getters for IndexConfig

func (c IndexConfig) IndexType() types.IndexType     { return c.indexType }
func (c IndexConfig) Metric() types.SimilarityMetric { return c.metric }
func (c IndexConfig) Dimension() int                 { return c.dimension }

// validate config
func (c IndexConfig) Validate() error {
	if c.indexType == 0 {
		return errors.New("index type is required")
	}
	if c.metric == 0 {
		return errors.New("similarity metric is required")
	}
	if c.dimension <= 0 {
		return errors.New("dimenison must be a positive integer")
	}
	return nil
}
