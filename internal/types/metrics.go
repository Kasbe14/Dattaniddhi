package types

type SimilarityMetric int

const (
	Cosine SimilarityMetric = iota + 1
	Dot
	Euclidean
)

//Todo implement stringer interface on similaritymetric
