package vector

import (
	"math"
	"testing"
)

func TestSimilarityIdentity(t *testing.T) {
	vPtr, err := NewVector("xyz420", []float32{69, 69, 420, 67, 6, 7, 6.9})
	if err != nil {
		t.Fatal("vector construction failed")
	}
	simi, err := vPtr.Similarity(vPtr)
	if err != nil {
		t.Fatal(err)
	} else if math.Abs(simi-1.0) > epsilon {
		t.Fatal("similarity not close to 1", simi)
	}
}

// TODO : floating point comparison logic study and implement
func TestSimilarityOrthogonal(t *testing.T) {
	vec1, err := NewVector("bc69mc", []float32{1, 0})
	if err != nil {
		t.Fatal(err)
	}
	vec2, err := NewVector("mc69bc", []float32{0, 1})
	if err != nil {
		t.Fatal(err)
	}
	simi, err := vec1.Similarity(vec2)
	if err != nil {
		t.Fatal(err)
	} else if math.Abs(simi) > epsilon {
		t.Fatal("failed orthogonal similarity, expected 0 got ", simi)
	}
}

func TestSimilarityDimensionMis(t *testing.T) {
	vec1, err := NewVector("bc69mc", []float32{1, 0})
	if err != nil {
		t.Fatal(err)
	}
	vec2, err := NewVector("mc69bc", []float32{0, 1, 3})
	if err != nil {
		t.Fatal(err)
	}
	_, err = vec1.Similarity(vec2)
	if err == nil {
		t.Fatal("expected dimension mismatch error")
	}
}
