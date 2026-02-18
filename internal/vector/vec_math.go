package vector

import (
	"errors"
	"fmt"
	"math"
)

const epsilon = 1e-6

// Gives only length of the vector in vector space
func Magnitude(values []float32) float64 {
	sum := 0.0
	for _, v := range values {
		val := float64(v)
		sum += val * val
	}
	return math.Sqrt(sum)
}

// dot validates lengths
func DotProduct(vec1, vec2 []float32) (float64, error) {
	var dotProduct float64
	if len(vec1) != len(vec2) {
		return 0, errors.New("unequal vector lengths")
	}
	for i := range vec1 {
		dotProduct += float64(vec1[i]) * float64(vec2[i])
	}
	return dotProduct, nil
}

// measurement of direction
func Cosine(vec1, vec2 []float32) (float64, error) {
	if len(vec1) != len(vec2) {
		return 0, errors.New("unequal vector lengths")
	}
	//ALL VECTORS PASSED HERE MUST BE NORMALISED
	// Vectors are normalized at constructions and normalize validates zero-magnitude vector
	//magA := Magnitude(vec1)
	// //magB := Magnitude(vec2)
	// if magA < epsilon || magB < epsilon {
	// 	return 0, errors.New("zero magnitude vector")
	// }
	// cosine similarity of normalized vectors (magnitude=1) = dotproduct
	cosine, err := DotProduct(vec1, vec2) /*/ (magA * magB)*/
	if err != nil {
		return 0, fmt.Errorf("eror from dot, %w", err)
	}
	return cosine, nil
}

func Euclidean(vec1, vec2 []float32) (float64, error) {
	if len(vec1) != len(vec2) {
		return 0.0, errors.New("unequal vector lengths")
	}
	sum := 0.0
	for i := range vec1 {
		sum += (float64(vec1[i]) - float64(vec2[i])) * (float64(vec1[i]) - float64(vec2[i]))
	}
	return -(sum), nil
}
func (v *Vector) Similarity(other *Vector) (float64, error) {
	if other == nil || v == nil {
		return 0.0, errors.New("nil vectors")
	}
	if v.dimensions != other.dimensions {
		return 0.0, errors.New("dimension mismatch")
	}
	return Cosine(v.values, other.values)
}
