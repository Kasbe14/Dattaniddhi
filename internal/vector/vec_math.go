package vector

import (
	"errors"
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

// assume vec1 and v2 have equal length; caller must ensure
func DotProduct(vec1, vec2 []float32) float64 {
	var dotProduct float64
	for i := range vec1 {
		dotProduct += float64(vec1[i]) * float64(vec2[i])
	}
	return dotProduct
}

// measurement of direction
func CosineSimilarity(vec1, vec2 []float32) (float64, error) {
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
	cosine := DotProduct(vec1, vec2) /*/ (magA * magB)*/
	return cosine, nil
}
