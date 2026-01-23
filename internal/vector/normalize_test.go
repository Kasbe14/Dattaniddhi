package vector

import (
	"math"
	"testing"
)

func TestNormalizeMagnitude(t *testing.T) {
	vec := []float32{53, 56, 57, 2, 1, 0}
	norm, err := Normalize(vec)
	if err != nil {
		t.Fatal(err)
	}
	if len(norm) != len(vec) {
		t.Fatal("normalized vector length mismatch")
	}
	mag := Magnitude(norm)
	if math.Abs(mag-1.0) > epsilon {
		t.Fatalf("expected magnitude ~1, got %f", mag)
	}
}
func TestNormalizeZeroVector(t *testing.T) {
	_, err := Normalize([]float32{0, 0, 0})
	if err == nil {
		t.Fatal("expected error for zero vector")
	}
}
