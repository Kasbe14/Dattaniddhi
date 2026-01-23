package vector

import "errors"

type Vector struct {
	id         string
	values     []float32
	dimensions int
}

// consturctor for immutable vector
func NewVector(id string, vecValues []float32) (*Vector, error) {
	if len(vecValues) == 0 {
		return nil, errors.New("a vector must have atleast one dimensions")
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
		id:         id,
		values:     normalVec,
		dimensions: len(normalVec),
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

func (v *Vector) Similarity(other *Vector) (float64, error) {
	if other == nil || v == nil {
		return 0.0, errors.New("nil vectors")
	}
	if v.dimensions != other.dimensions {
		return 0.0, errors.New("dimension mismatch")
	}
	return CosineSimilarity(v.values, other.values)
}
