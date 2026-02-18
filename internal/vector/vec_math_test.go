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
	cos, err := Cosine(norm, norm)
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
		result, err := DotProduct(tt.vec1, tt.vec2)
		if err != nil {
			t.Errorf("%v", err)
		}
		if math.Abs(result-tt.expected) > 1e-10 {
			t.Errorf("DotProduct(%v, %v): got %v, want %v", tt.vec1, tt.vec2, result, tt.expected)
		}
	}
}
func TestEuclidean(t *testing.T) {
	tests := []struct {
		name        string
		vec1        []float32
		vec2        []float32
		expected    float64
		expectError bool
	}{
		{
			name:        "Identity (Same Vectors)",
			vec1:        []float32{1.0, 2.0, 3.0},
			vec2:        []float32{1.0, 2.0, 3.0},
			expected:    0.0, // Distance is 0, return -0
			expectError: false,
		},
		{
			name: "Simple Distance",
			vec1: []float32{1.0, 1.0},
			vec2: []float32{4.0, 5.0},
			// Math: (1-4)^2 + (1-5)^2 = (-3)^2 + (-4)^2 = 9 + 16 = 25
			// Function returns -(sum), so expected is -25.0
			expected:    -25.0,
			expectError: false,
		},
		{
			name: "Negative Values in Vector",
			vec1: []float32{-2.0, 0.0},
			vec2: []float32{2.0, 0.0},
			// Math: (-2 - 2)^2 + (0 - 0)^2 = (-4)^2 = 16
			// Return -16.0
			expected:    -16.0,
			expectError: false,
		},
		{
			name: "Floating Point Precision",
			vec1: []float32{0.1, 0.2},
			vec2: []float32{0.3, 0.4},
			// Math: (0.1-0.3)^2 + (0.2-0.4)^2 = (-0.2)^2 + (-0.2)^2
			// = 0.04 + 0.04 = 0.08
			// Return -0.08
			expected:    -0.08,
			expectError: false,
		},
		{
			name:        "Contract Violation: Length Mismatch",
			vec1:        []float32{1.0, 2.0},
			vec2:        []float32{1.0, 2.0, 3.0},
			expected:    0.0,
			expectError: true,
		},
		{
			name:        "Edge Case: Empty Vectors",
			vec1:        []float32{},
			vec2:        []float32{},
			expected:    0.0, // Loop doesn't run, sum is 0
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Euclidean(tt.vec1, tt.vec2)

			// 1. Check Error State
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, got nil", tt.name)
				}
				return // Stop checking result if we expected an error
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.name, err)
				}
			}

			// 2. Check Result (using epsilon for float comparison)
			// We use a small epsilon (0.000001) because float math is rarely exact
			if math.Abs(result-tt.expected) > 1e-6 {
				t.Errorf("Result mismatch. Got %f, want %f", result, tt.expected)
			}
		})
	}
}

// ==============================moving similarity test here from vector_test ================
// ============================as vector is pure data now will be modified later =================

// func TestSimilarityIdentity(t *testing.T) {
// 	vPtr, err := NewVector("xyz420", []float32{69, 69, 420, 67, 6, 7, 6.9}, "text", "cosine")
// 	if err != nil {
// 		t.Fatal("vector construction failed")
// 	}
// 	simi, err := vPtr.Similarity(vPtr)
// 	if err != nil {
// 		t.Fatal(err)
// 	} else if math.Abs(simi-1.0) > epsilon {
// 		t.Fatal("similarity not close to 1", simi)
// 	}
// }

// // TODO : floating point comparison logic study and implement
// func TestSimilarityOrthogonal(t *testing.T) {
// 	vec1, err := NewVector("bc69mc", []float32{1, 0}, "text", "cosine")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	vec2, err := NewVector("mc69bc", []float32{0, 1}, "text", "cosine")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	simi, err := vec1.Similarity(vec2)
// 	if err != nil {
// 		t.Fatal(err)
// 	} else if math.Abs(simi) > epsilon {
// 		t.Fatal("failed orthogonal similarity, expected 0 got ", simi)
// 	}
// }

// func TestSimilarityDimensionMis(t *testing.T) {
// 	vec1, err := NewVector("bc69mc", []float32{1, 0}, "text", "cosine")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	vec2, err := NewVector("mc69bc", []float32{0, 1, 3}, "text", "cosine")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	_, err = vec1.Similarity(vec2)
// 	if err == nil {
// 		t.Fatal("expected dimension mismatch error")
// 	}
// }
