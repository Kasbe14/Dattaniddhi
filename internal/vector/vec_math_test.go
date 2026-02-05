package vector

import (
	"math"
	"testing"
)

func TestCosineSimilaritySameVector(t *testing.T) {
	vec := []float32{1, 2, 3}
	//normalizing here because all vectors are normalized at consturction
	norm, err := Normalize(vec)
	if err != nil {
		t.Fatalf("vector Normalized function failed %v", err)
	}
	cos, err := CosineSimilarity(norm, norm)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(cos-1.0) > epsilon {
		t.Fatalf("expected -1, got %v", cos)
	}
}
func TestMagnitude(t *testing.T) {
	tests := []struct {
		input    []float32
		expected float64
	}{
		{[]float32{0, 0, 0}, 0.0},
		{[]float32{1, 2, 3}, math.Sqrt(14)},
		{[]float32{-1, -2, -3}, math.Sqrt(14)},
		{[]float32{1}, 1.0},
	}

	for _, tt := range tests {
		result := Magnitude(tt.input)
		if math.Abs(result-tt.expected) > 1e-10 {
			t.Errorf("Magnitude(%v): got %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestDotProduct(t *testing.T) {
	tests := []struct {
		vec1, vec2 []float32
		expected   float64
	}{
		{[]float32{1, 2, 3}, []float32{1, 2, 3}, 14.0},
		{[]float32{1, 0, 0}, []float32{0, 1, 0}, 0.0}, // orthogonal
		{[]float32{-1, -2}, []float32{2, 1}, -4.0},
	}

	for _, tt := range tests {
		result := DotProduct(tt.vec1, tt.vec2)
		if math.Abs(result-tt.expected) > 1e-10 {
			t.Errorf("DotProduct(%v, %v): got %v, want %v", tt.vec1, tt.vec2, result, tt.expected)
		}
	}
}
