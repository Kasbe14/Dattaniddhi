package index

import (
	"VectorDatabase/internal/vector"
	"testing"
)

func TestLinearIndex_AddAndGet(t *testing.T) {
	idx := NewLinearIndex()
	v, err := vector.NewVector(
		"mcV1bc",
		[]float32{1, 2, 3},
	)
	if err != nil {
		t.Fatalf("vector creation failed: %v", err)
	}
	if err := idx.Add(v); err != nil {
		t.Fatalf("failed to add vector %v", err)
	}
	got, ok := idx.Get("mcV1bc")
	if !ok {
		t.Fatalf("didn't got vector after add")
	}
	if got.ID() != v.ID() {
		t.Fatalf("expexted id %s, got %s", v.ID(), got.ID())
	}
	if got.Dimensions() != v.Dimensions() {
		t.Fatalf("dimensions mismatch")
	}

}
func TestLinearIndex_Delete(t *testing.T) {
	idx := NewLinearIndex()
	v, err := vector.NewVector(
		"delV1Test",
		[]float32{1, 2, 3},
	)
	if err != nil {
		t.Fatalf("vector creation failed: %v", err)
	}
	if err := idx.Add(v); err != nil {
		t.Fatalf("failed to add vector %v", err)
	}
	//Case deleting non-existing id
	err = idx.Delete("nonExistingId1")
	if err == nil {
		t.Fatal("deleted a non-existing id")
	}
	//deletes and get return nothing
	err = idx.Delete("delV1Test")
	if err != nil {
		t.Fatalf("expected delete to succeed, got %v", err)
	}
	_, ok := idx.Get("delV1Test")
	if ok {
		t.Fatal("expected vector to be deleted")
	}
}
func TestLinearIndex_Search(t *testing.T) {
	emptyIdx := NewLinearIndex()
	searchIdx := NewLinearIndex()
	query, _ := vector.NewVector("q", []float32{1, 2, 3})
	v1, _ := vector.NewVector("v1", []float32{1, 2, 3})
	v2, _ := vector.NewVector("v2", []float32{3, 2, 1})
	v3, _ := vector.NewVector("v3", []float32{1, 1, 2})
	if err := searchIdx.Add(v1); err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if err := searchIdx.Add(v2); err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if err := searchIdx.Add(v3); err != nil {
		t.Fatalf("add failed: %v", err)
	}
	// edge case empty index test
	result1, err := emptyIdx.Search(query, 2)
	if err != nil {
		t.Fatalf("search failed on empty index %v", err)
	}
	if len(result1) != 0 {
		t.Fatalf("expected zero results, got %d", len(result1))
	}
	// valid search results and order test
	result2, err := searchIdx.Search(query, 2)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(result2) != 2 {
		t.Fatalf("expected 2 results, got %d", len(result2))
	}
	if result2[0].Score() < result2[1].Score() {
		t.Fatalf("results not sorted in descending similarity score")
	}
	// nil query test
	_, err = searchIdx.Search(nil, 5)
	if err == nil {
		t.Fatal("expected error for nil query input")
	}
	// invalid amount of results test k<=0
	_, err = searchIdx.Search(query, 0)
	if err == nil {
		t.Fatal("expected error for input k = 0")
	}
	_, err = searchIdx.Search(query, -13)
	if err == nil {
		t.Fatal("expect error for input k < 0")
	}
	// k > index size test
	result3, _ := searchIdx.Search(query, 4)
	if len(result3) < searchIdx.Size() {
		t.Fatalf("expected result length equal to index size, got %d", len(result3))
	}
}
