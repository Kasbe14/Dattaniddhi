package index

import (
	"VectorDatabase/internal/types"
)

type IndexConfig struct {
	IndexType IndexType
	ModelType types.ModelType
	DataType  types.DataType
	Metric    types.SimilarityMetric
	Dimension int
}
