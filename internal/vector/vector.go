package vector

import "errors"

type DataType string
type SimilarityMetric string

type Vector struct {
	id               string
	values           []float32
	dimensions       int
	dataType         DataType
	similarityMetric SimilarityMetric
}

// consturctor for immutable vector
func NewVector(id string, vecValues []float32, dataType DataType, simMetric SimilarityMetric) (*Vector, error) {
	if len(vecValues) == 0 {
		return nil, errors.New("a vector must have atleast one dimension")
	}
	if dataType == "" {
		return nil, errors.New("data type empty")
	}
	if simMetric == "" {
		return nil, errors.New("similarity metric empty")
	}
	//validate vector
	if err := validateValues(vecValues); err != nil {
		return nil, err
	}
	//normalize vector
	normalVec, err := Normalize(vecValues)
	if err != nil {
		return nil, err
	}
	//  copying for imutability
	// copied := make([]float32, len(vecValues))
	// copy(copied, vecValues)
	vec := &Vector{
		id:               id,
		values:           normalVec,
		dimensions:       len(normalVec),
		dataType:         dataType,
		similarityMetric: simMetric,
	}
	return vec, nil
}

//vector api
func (v *Vector) ID() string {
	return v.id
}

func (v *Vector) Dimensions() int {
	return v.dimensions
}

func (v *Vector) Values() []float32 {
	vecVals := make([]float32, len(v.values))
	copy(vecVals, v.values)
	return vecVals
}
func (v *Vector) DataType() DataType {
	return v.dataType
}
func (v *Vector) Metric() SimilarityMetric {
	return v.similarityMetric
}

func (v *Vector) Similarity(other *Vector) (float64, error) {
	if other == nil || v == nil {
		return 0.0, errors.New("nil vectors")
	}
	if v.dimensions != other.dimensions {
		return 0.0, errors.New("dimension mismatch")
	}
	return CosineSimilarity(v.values, other.values)
}
