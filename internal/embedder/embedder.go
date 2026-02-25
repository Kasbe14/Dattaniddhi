package embedder

import (
	"github.com/Kasbe14/Dattaniddhi/internal/vector"
	"context"
)

// ID uniquely identifies a vector produced by an Embedder
// Ownership : Embedder
type VectorID string

// modality represents type of raw input data
type DataType string

const (
	DataTypeText  DataType = "text"
	DataTypeImage DataType = "image"
	// future
	// DataTypeAudio DataType = "audio"
	// DataTypeVideo DataType = "video"
)

// SimilarityMetrics define how vector are compared
// Note declare early for clarity , the usage will come later
type SimilarityMetrics string

const (
	MetricCosine    SimilarityMetrics = "cosine"
	MetricDot       SimilarityMetrics = "dot"
	MetricEuclidean SimilarityMetrics = "euclidean"
)

// Embedder defines the contract  for converting  raw data into vectors
type Embedder interface {
	//converts raw data into vectors
	//guaratees -> Stable dimension and Unique ID for vector
	Embed(ctx context.Context, input any) (*vector.Vector, error)

	// Dimension returns fixed vector dimension produced by this Embedder
	Dimension() int

	//DataType returns supported input data type (modality : text, image, audio, video)
	DataType() DataType

	//Metric returns the similarity metric this metric was trained for..(use same metric for similarity score)
	Metric() SimilarityMetrics

	//Name returns a stable identifeir for thsi Embedder (eg - "openai-text-embedding-3-large")
	Name() string
}
