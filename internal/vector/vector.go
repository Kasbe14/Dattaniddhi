package vector

import "errors"

type DataType string
type SimilarityMetric string

//vector is pure data object
type Vector struct {
	values     []float32
	dimensions int
}

// consturctor for immutable vector
func NewVector(vecValues []float32, dim int) (*Vector, error) {
	vecDim := len(vecValues)
	if vecDim == 0 {
		return nil, errors.New("a vector must have atleast one dimension")
	}
	if vecDim != dim {
		return nil, errors.New("number of vector values not equal to given dimension")
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
	vec := &Vector{
		values:     normalVec,
		dimensions: dim,
	}
	return vec, nil
}

//vector api
func (v *Vector) Dimensions() int {
	return v.dimensions
}
func (v *Vector) Values() []float32 {
	vecVals := make([]float32, len(v.values))
	copy(vecVals, v.values)
	return vecVals
}
