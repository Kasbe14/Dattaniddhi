package types

type SimilarityMetric int

const (
	Cosine SimilarityMetric = iota
	Dot
	Euclidean
)

//Todo implement stringer interface on similaritymetric
