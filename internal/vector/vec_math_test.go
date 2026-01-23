package vector

import (
	"math"
	"testing"
)

func TestCosineSimilaritySameVector(t *testing.T) {
	vec := []float32{1, 2, 3}
	cos, err := CosineSimilarity(vec, vec)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(cos-1.0) > 1e-9 {
		t.Fatalf("expected -1, got %v", cos)
	}
}
