package index

import (
	v "VectorDatabase/internal/vector"
)

type IndexConfig struct {
	DataType  v.DataType
	Metric    v.SimilarityMetric
	Dimension int
}
