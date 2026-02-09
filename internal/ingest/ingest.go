package ingest

import (
	v "VectorDatabase/internal/vector"
	"context"
)

type InsertResult struct {
	ExternalId   string
	AlreadyExist bool
}
type Inserter interface {
	Insert(ctx context.Context, inputData any) (InsertResult, error)
	InsertPreEmbed(
		ctx context.Context,
		vec []float32,
		inputDataType v.DataType,
		simMetric v.SimilarityMetric,
		model string) (
		InsertResult, error)
}
