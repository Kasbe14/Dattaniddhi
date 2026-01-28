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
