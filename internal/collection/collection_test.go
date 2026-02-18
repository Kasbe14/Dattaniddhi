package collection

import (
	//"errors"
	"testing"

	"VectorDatabase/internal/index"
	"VectorDatabase/internal/vector"
)

// --- Mocks ---

// MockVectorIndex simulates the underlying index behavior
type MockVectorIndex struct {
	AddFunc    func(id int, vec *vector.Vector) (bool, error)
	SearchFunc func(query *vector.Vector, k int) ([]index.SearchResult, error)
	DeleteFunc func(id int) error
}

func (m *MockVectorIndex) Size() int {
	// Stub implementation of size
	return 0
}
func (m *MockVectorIndex) Get(id int) (*vector.Vector, bool) {
	// Stub implementation: Return nil and false (not found)
	// Since Collection doesn't use this yet, it won't affect the tests.
	return nil, false
}

func (m *MockVectorIndex) Add(id int, vec *vector.Vector) (bool, error) {
	if m.AddFunc != nil {
		return m.AddFunc(id, vec)
	}
	return false, nil
}

func (m *MockVectorIndex) Search(query *vector.Vector, k int) ([]index.SearchResult, error) {
	if m.SearchFunc != nil {
		return m.SearchFunc(query, k)
	}
	return nil, nil
}

func (m *MockVectorIndex) Delete(id int) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	return nil
}

// MockSearchResult helps us return fake results
// type MockSearchResult struct {
// 	id    int
// 	score float64
// }

// func (m MockSearchResult) ID() int        { return m.id }
// func (m MockSearchResult) Score() float64 { return m.score }

// --- Unit Tests ---

func TestCollection_Insert(t *testing.T) {
	mockIdx := &MockVectorIndex{}
	// Setup with basic valid config (Dimension 2)
	col := &Collection{
		config:    CollectionConfig{Dimension: 2, Name: "test-col"},
		index:     mockIdx,
		idCounter: 0,
		extToInt:  make(map[string]int),
		intToExt:  make(map[int]string),
	}

	t.Run("Success: Insert valid vector", func(t *testing.T) {
		mockIdx.AddFunc = func(id int, vec *vector.Vector) (bool, error) {
			return false, nil
		}

		vecVals := []float32{1.0, 2.0}
		extID, err := col.Insert(vecVals, nil)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if extID == "" {
			t.Error("Expected valid external UUID, got empty string")
		}
		if len(col.extToInt) != 1 {
			t.Error("extToInt map was not updated")
		}
	})

	t.Run("Rollback: Internal Collision", func(t *testing.T) {
		// Mock behavior: Index says "True" (Collision occurred)
		mockIdx.AddFunc = func(id int, vec *vector.Vector) (bool, error) {
			return true, nil
		}

		// Reset maps
		col.extToInt = make(map[string]int)
		col.intToExt = make(map[int]string)

		vecVals := []float32{1.0, 2.0}
		_, err := col.Insert(vecVals, nil)

		// CHECK 1: Ensure we got your specific error
		if err != ErrInternalIDCollision {
			t.Errorf("Expected ErrInternalIDCollision, got %v", err)
		}

		// CHECK 2: Ensure maps were rolled back (empty)
		if len(col.extToInt) != 0 {
			t.Error("Rollback failed! Maps not cleared after collision.")
		}
	})
}

func TestCollection_Delete(t *testing.T) {
	mockIdx := &MockVectorIndex{}
	col := &Collection{
		config:   CollectionConfig{Dimension: 2},
		index:    mockIdx,
		extToInt: make(map[string]int),
		intToExt: make(map[int]string),
	}

	col.extToInt["exist-id"] = 5
	col.intToExt[5] = "exist-id"

	t.Run("Success: Delete", func(t *testing.T) {
		err := col.Delete("exist-id")
		if err != nil {
			t.Errorf("Delete failed: %v", err)
		}
		if len(col.extToInt) != 0 {
			t.Error("Map not cleared after delete")
		}
	})

	t.Run("Failure: Delete Non-Existent", func(t *testing.T) {
		err := col.Delete("ghost-id")

		// CHECK: Ensure specific error
		if err != ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})
}
func TestCollection_Search(t *testing.T) {
	mockIdx := &MockVectorIndex{}

	col := &Collection{
		config:   CollectionConfig{Dimension: 2, Name: "search-col"},
		index:    mockIdx,
		extToInt: make(map[string]int),
		intToExt: make(map[int]string),
	}

	// Pre-populate mapping (Internal ID 10 -> External "abc-123")
	col.intToExt[10] = "abc-123"

	t.Run("Success: Search returns mapped results", func(t *testing.T) {
		// Mock behavior: Return the REAL struct directly
		mockIdx.SearchFunc = func(query *vector.Vector, k int) ([]index.SearchResult, error) {

			// FIX: Initialize the struct directly.
			// Ensure field names 'ID' and 'Score' match your index package exactly.
			return []index.SearchResult{
				{
					VecId: 10,   // Internal ID
					Score: 0.99, // Similarity Score
				},
			}, nil
		}

		query := []float32{1.0, 0.0}
		results, err := col.Search(query, 1)

		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}

		// Verify the Collection converted Internal ID 10 -> "abc-123"
		if results[0].VecID != "abc-123" {
			t.Errorf("Expected external ID 'abc-123', got '%s'", results[0].VecID)
		}
	})

	// ... (rest of the tests)
}
