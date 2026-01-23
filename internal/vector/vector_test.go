package vector

import "testing"

func TestSimilarityIdentity(t *testing.T) {
	vPtr, err := NewVector("xyz420", []float32{69, 69, 420, 67, 6, 7, 6.9})
	if err != nil {
		t.Fatal("vector construction failed")
	}
	simi, err := vPtr.Similarity(vPtr)
	if err != nil {
		t.Fatal(err)
	} else if simi != 1 {
		t.Fatal("failed identity similarity")
	}
}

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
	} else if simi != 0 {
		t.Fatal("failed orthogonal similarity")
	}
}

func TestDimensionMis(t *testing.T) {
	vec1, err := NewVector("bc69mc", []float32{1, 0})
	if err != nil {
		t.Fatal(err)
	}
	vec2, err := NewVector("mc69bc", []float32{0, 1, 3})
	if err != nil {
		t.Fatal(err)
	}
	simi, err := vec1.Similarity(vec2)
	if err != nil {
		t.Fatal(err)
	} else {
		t.Fatal("pass mismatch with similarity", simi)
	}
}
