package collection

import (
	"sync"
	"testing"

	"github.com/Kasbe14/Dattaniddhi/internal/store/wal"
	"github.com/Kasbe14/Dattaniddhi/internal/types"
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
		// ... (other validation tests remain the same)
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
// Test 2: Insert, Payload Serialization, and Get
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

	vecVals := []float32{1.0, 0.0}

	// Testing with a map since it marshals safely to JSON
	payloadData := map[string]string{"content": "hello world"}

	// Act: Insert (This now writes to WAL!)
	id, err := col.Insert(vecVals, payloadData)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if id == "" {
		t.Fatal("Returned empty ID")
	}

	// Act: Get Payload
	retrieved, found := col.Get(id)

	if !found {
		t.Error("Payload not found via Get()")
	}

	dataMap, ok := retrieved.(map[string]string)
	if !ok {
		t.Fatal("Payload corrupted: Type mismatch")
	}
	if dataMap["content"] != "hello world" {
		t.Errorf("Payload mismatch: got %v", dataMap)
	}
}

// -----------------------------------------------------------------------------
// Test 3: Search Logic & Internal Corruption Check
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

	query := []float32{1.1, 1.1}
	results, err := col.Search(query, 1)

	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatal("Expected 1 result")
	}

	if results[0].VecID != idA {
		t.Errorf("Search returned wrong ID. Expected %s, got %s", idA, results[0].VecID)
	}
}

// -----------------------------------------------------------------------------
// Test 4: Delete & Cleanup (WAL AppendDelete)
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

	// Assert: Invariants (Everything must be gone)
	if len(col.extToInt) != 0 {
		t.Error("extToInt map not cleared")
	}
	if len(col.payload) != 0 {
		t.Error("payload map not cleared")
	}
}

// -----------------------------------------------------------------------------
// Test 5: Concurrency Safety with Disk I/O
// -----------------------------------------------------------------------------
func TestCollection_Concurrency(t *testing.T) {
	testWal := setupTestWAL(t)
	defer testWal.Close()
	col, _ := NewCollection(CollectionConfig{Name: "concur", Dimension: 2, Metric: types.Cosine, IndexType: types.LinearIndex}, testWal)

	var wg sync.WaitGroup
	count := 50 // Keep slightly lower to avoid blasting OS temp files too hard during tests

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
