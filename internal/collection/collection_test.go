package collection

import (
	"sync"
	"testing"

	"github.com/Kasbe14/Dattaniddhi/internal/store/wal"
	"github.com/Kasbe14/Dattaniddhi/internal/types"
	"github.com/Kasbe14/Dattaniddhi/internal/vector"
)

// setupTestWAL is a helper function to create a fresh WAL in a temporary directory
func setupTestWAL(t *testing.T) *wal.WAL {
	t.Helper()
	tempDir := t.TempDir()
	w, err := wal.NewWAL(tempDir, wal.SyncAlways)
	if err != nil {
		t.Fatalf("Failed to setup test WAL: %v", err)
	}
	return w
}

// -----------------------------------------------------------------------------
// Test 1: Constructor Contracts (Validation)
// -----------------------------------------------------------------------------
func TestNewCollection_Validation(t *testing.T) {
	testWal := setupTestWAL(t)
	defer testWal.Close()

	tests := []struct {
		name        string
		cfg         CollectionConfig
		expectedErr error
	}{
		{
			name: "Success: Valid Config",
			cfg: CollectionConfig{
				Name:      "valid-col",
				Dimension: 128,
				Metric:    types.Cosine,
				IndexType: types.LinearIndex,
			},
			expectedErr: nil,
		},
		{
			name: "Failure: Invalid Dimension",
			cfg: CollectionConfig{
				Name:      "test",
				Dimension: 0,
				Metric:    types.Cosine,
				IndexType: types.LinearIndex,
			},
			expectedErr: ErrInvalidDimension,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col, err := NewCollection(tt.cfg, testWal)

			if err != tt.expectedErr {
				t.Errorf("Expected error '%v', got '%v'", tt.expectedErr, err)
			}
			if err == nil && col == nil {
				t.Error("Expected valid collection instance, got nil")
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Test 2: Insert (Happy Path)
// -----------------------------------------------------------------------------
func TestCollection_InsertAndGet(t *testing.T) {
	testWal := setupTestWAL(t)
	defer testWal.Close()

	cfg := CollectionConfig{
		Name:      "payload-test",
		Dimension: 2,
		Metric:    types.Cosine,
		IndexType: types.LinearIndex,
	}
	col, _ := NewCollection(cfg, testWal)

	payloadData := map[string]string{"content": "hello world"}

	// Act
	id, err := col.Insert([]float32{1.0, 0.0}, payloadData)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// Assert
	retrieved, found := col.Get(id)
	if !found {
		t.Error("Payload not found via Get()")
	}

	dataMap, _ := retrieved.(map[string]string)
	if dataMap["content"] != "hello world" {
		t.Errorf("Payload mismatch: got %v", dataMap)
	}
}

// -----------------------------------------------------------------------------
// Test 3: Insert (Sad Path - Rollback on Index Failure)
// -----------------------------------------------------------------------------
func TestCollection_Insert_Rollback(t *testing.T) {
	testWal := setupTestWAL(t)
	defer testWal.Close()

	cfg := CollectionConfig{
		Name:      "rollback-test",
		Dimension: 2,
		Metric:    types.Cosine,
		IndexType: types.LinearIndex,
	}
	col, _ := NewCollection(cfg, testWal)

	// SABOTAGE: We manually inject a vector directly into the underlying index with ID 1.
	// This simulates a scenario where the DB is corrupted, so when the Collection tries
	// to insert a new vector with internal ID 1, the index will reject it.
	vec, _ := vector.NewVector([]float32{9.9, 9.9}, 2)
	col.index.Add(1, vec)

	// Act: Try to insert a new document through the collection
	_, err := col.Insert([]float32{1.0, 1.0}, "this should fail")

	// Assert: It should fail and return the collision error
	if err == nil {
		t.Fatal("Expected Insert to fail due to ID collision, but it succeeded")
	}

	// Assert: The State Leak check. Ensure NO memory was updated.
	if col.idCounter != 0 {
		t.Errorf("State Leak! idCounter incremented to %d despite failure", col.idCounter)
	}
	if len(col.extToInt) != 0 {
		t.Errorf("State Leak! extToInt map has %d entries despite failure", len(col.extToInt))
	}
}

// -----------------------------------------------------------------------------
// Test 4: Delete (Happy Path)
// -----------------------------------------------------------------------------
func TestCollection_Delete(t *testing.T) {
	testWal := setupTestWAL(t)
	defer testWal.Close()

	col, _ := NewCollection(CollectionConfig{Name: "del-test", Dimension: 2, Metric: types.Cosine, IndexType: types.LinearIndex}, testWal)
	id, _ := col.Insert([]float32{1.0, 1.0}, "data")

	// Act
	err := col.Delete(id)
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	// Assert
	if len(col.extToInt) != 0 || len(col.payload) != 0 {
		t.Error("Memory maps not cleared after delete")
	}
}

// -----------------------------------------------------------------------------
// Test 5: Delete (Sad Path - Panic on Index Failure)
// -----------------------------------------------------------------------------
func TestCollection_Delete_Panic(t *testing.T) {
	testWal := setupTestWAL(t)
	defer testWal.Close()

	col, _ := NewCollection(CollectionConfig{Name: "panic-test", Dimension: 2, Metric: types.Cosine, IndexType: types.LinearIndex}, testWal)

	// 1. Insert a valid record
	id, _ := col.Insert([]float32{1.0, 1.0}, "data")

	// 2. SABOTAGE: Delete it directly from the underlying index to create a split-brain state
	// We know the internal ID is 1 because it's the first insert.
	col.index.Delete(1)

	// 3. Setup a defer to catch the expected panic
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("Expected code to PANIC due to split-brain state, but it didn't")
		}
	}()

	// 4. Act: Try to delete via collection. This writes to WAL (success),
	// but fails to delete from index (because we already deleted it). It MUST panic.
	col.Delete(id)
}

// -----------------------------------------------------------------------------
// Test 6: Search Logic
// -----------------------------------------------------------------------------
func TestCollection_Search(t *testing.T) {
	testWal := setupTestWAL(t)
	defer testWal.Close()

	col, _ := NewCollection(CollectionConfig{
		Name:      "search-test",
		Dimension: 2,
		Metric:    types.Euclidean,
		IndexType: types.LinearIndex,
	}, testWal)

	idA, _ := col.Insert([]float32{1.0, 1.0}, "Point A")
	col.Insert([]float32{50.0, 50.0}, "Point B")

	results, err := col.Search([]float32{1.1, 1.1}, 1)

	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if results[0].VecID != idA {
		t.Errorf("Search returned wrong ID. Expected %s, got %s", idA, results[0].VecID)
	}
}

// -----------------------------------------------------------------------------
// Test 7: Concurrency
// -----------------------------------------------------------------------------
func TestCollection_Concurrency(t *testing.T) {
	testWal := setupTestWAL(t)
	defer testWal.Close()

	col, _ := NewCollection(CollectionConfig{Name: "concur", Dimension: 2, Metric: types.Cosine, IndexType: types.LinearIndex}, testWal)

	var wg sync.WaitGroup
	count := 50

	wg.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()
			col.Insert([]float32{1.0, 0.0}, map[string]string{"key": "concurrent"})
		}()
	}
	wg.Wait()

	if len(col.payload) != count {
		t.Errorf("Expected %d items, got %d", count, len(col.payload))
	}
}
