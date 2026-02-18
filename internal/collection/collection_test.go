package collection

import (
	"sync"
	"testing"

	"VectorDatabase/internal/types"
)

// -----------------------------------------------------------------------------
// Test 1: Constructor Contracts (Validation)
// -----------------------------------------------------------------------------
func TestNewCollection_Validation(t *testing.T) {
	tests := []struct {
		name        string
		cfg         CollectionConfig // Must be exported!
		expectedErr error
	}{
		{
			name: "Success: Valid Linear Config",
			cfg: CollectionConfig{
				Name:      "valid-col",
				Dimension: 128,
				Metric:    types.Cosine,
				IndexType: types.LinearIndex,
			},
			expectedErr: nil,
		},
		{
			name: "Failure: Invalid Name",
			cfg: CollectionConfig{
				Name:      "",
				Dimension: 128,
				Metric:    types.Cosine,
				IndexType: types.LinearIndex,
			},
			expectedErr: ErrInvalidCollectionName,
		},
		{
			name: "Failure: Invalid Dimension (0)",
			cfg: CollectionConfig{
				Name:      "test",
				Dimension: 0,
				Metric:    types.Cosine,
				IndexType: types.LinearIndex,
			},
			expectedErr: ErrInvalidDimension,
		},
		{
			name: "Failure: Invalid Metric",
			cfg: CollectionConfig{
				Name:      "test",
				Dimension: 128,
				Metric:    types.SimilarityMetric(99),
				IndexType: types.LinearIndex,
			},
			expectedErr: ErrInvalidMetric,
		},
		{
			name: "Failure: Invalid Index Type",
			cfg: CollectionConfig{
				Name:      "test",
				Dimension: 128,
				Metric:    types.Cosine,
				IndexType: types.IndexType(99),
			},
			expectedErr: ErrInvalidIndexType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col, err := NewCollection(tt.cfg)

			if err != tt.expectedErr {
				t.Errorf("Expected error '%v', got '%v'", tt.expectedErr, err)
			}
			if err == nil && col == nil {
				t.Error("Expected valid collection instance, got nil")
			}
			// Invariant Check: If successful, maps should be initialized
			if err == nil {
				if col.extToInt == nil || col.intToExt == nil || col.payload == nil {
					t.Error("Invariant Violation: Collection maps were not initialized")
				}
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Test 2: Insert, Payload, and Get (Happy Path)
// -----------------------------------------------------------------------------
func TestCollection_InsertAndGet(t *testing.T) {
	// Setup
	cfg := CollectionConfig{
		Name:      "payload-test",
		Dimension: 2,
		Metric:    types.Cosine,
		IndexType: types.LinearIndex,
	}
	col, _ := NewCollection(cfg)

	// Input Data
	vecVals := []float32{1.0, 0.0}
	payloadData := map[string]string{"content": "hello world"}

	// Act: Insert
	id, err := col.Insert(vecVals, payloadData)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}
	if id == "" {
		t.Fatal("Returned empty ID")
	}

	// Act: Get Payload
	retrieved, found := col.Get(id)

	// Assert
	if !found {
		t.Error("Payload not found via Get()")
	}

	// Type Assertion to check content
	dataMap, ok := retrieved.(map[string]string)
	if !ok {
		t.Fatal("Payload corrupted: Type mismatch")
	}
	if dataMap["content"] != "hello world" {
		t.Errorf("Payload mismatch: got %v", dataMap)
	}
}

// -----------------------------------------------------------------------------
// Test 3: Search Logic & ID Mapping
// -----------------------------------------------------------------------------
func TestCollection_Search(t *testing.T) {
	col, _ := NewCollection(CollectionConfig{
		Name:      "search-test",
		Dimension: 2,
		Metric:    types.Euclidean, // Using Euclidean to test math integration
		IndexType: types.LinearIndex,
	})

	// Add vector A at [0, 0]
	idA, _ := col.Insert([]float32{0.1, 0.1}, "Point A")
	if idA == "" {
		t.Fatalf("id did not get assigned")
	}
	// Add vector B at [10, 10]
	col.Insert([]float32{10.0, 10.0}, "Point B")

	// Search for [0.1, 0.1] (Should be closest to Point A)
	query := []float32{0.1, 0.1}
	results, err := col.Search(query, 1) // Top 1

	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatal("Expected 1 result")
	}

	// Invariant: External ID returned must match the one generated during Insert
	if results[0].VecID != idA {
		t.Errorf("Search returned wrong ID. Expected %s, got %s", idA, results[0].VecID)
	}
}

// -----------------------------------------------------------------------------
// Test 4: Delete & Cleanup (Invariant Check)
// -----------------------------------------------------------------------------
func TestCollection_Delete(t *testing.T) {
	col, _ := NewCollection(CollectionConfig{Name: "del-test", Dimension: 2, Metric: types.Cosine, IndexType: types.LinearIndex})

	id, _ := col.Insert([]float32{1.0, 1.0}, "data")

	// Pre-check
	if len(col.extToInt) != 1 || len(col.payload) != 1 {
		t.Fatal("Setup failed: maps not populated")
	}

	// Act
	err := col.Delete(id)
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	// Assert: Invariants (Everything must be gone)
	if len(col.extToInt) != 0 {
		t.Error("extToInt map not cleared")
	}
	if len(col.intToExt) != 0 {
		t.Error("intToExt map not cleared")
	}
	if len(col.payload) != 0 {
		t.Error("payload map not cleared")
	}

	// Double check Get
	_, found := col.Get(id)
	if found {
		t.Error("Payload still retrievable after delete")
	}
}

// -----------------------------------------------------------------------------
// Test 5: Rollback on Failure (Edge Case)
// -----------------------------------------------------------------------------
func TestCollection_InsertRollback(t *testing.T) {
	// Note: Since we are using the REAL index factory, it is hard to force
	// the LinearIndex to fail on Add() unless we simulate a collision or OOM.
	// However, we can test the Dimension Mismatch contract here.

	col, _ := NewCollection(CollectionConfig{Name: "rb-test", Dimension: 2, Metric: types.Cosine, IndexType: types.LinearIndex})

	// Try to insert invalid dimension
	_, err := col.Insert([]float32{1.0, 2.0, 3.0}, "bad data") // 3D vector into 2D col

	if err != ErrInvalidDimension {
		t.Errorf("Expected ErrInvalidDimension, got %v", err)
	}

	// Invariant: Maps must remain empty
	if len(col.extToInt) != 0 {
		t.Error("Rollback invariant failed: Maps were modified during invalid insert")
	}
}

// -----------------------------------------------------------------------------
// Test 6: Thread Safety (Concurrency)
// -----------------------------------------------------------------------------
func TestCollection_Concurrency(t *testing.T) {
	col, _ := NewCollection(CollectionConfig{Name: "concur", Dimension: 2, Metric: types.Cosine, IndexType: types.LinearIndex})

	var wg sync.WaitGroup
	count := 100

	// Concurrent Inserts
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()
			col.Insert([]float32{1.0, 0.0}, "data")
		}()
	}
	wg.Wait()

	// Concurrent Reads
	// We need an ID to read. Since IDs are random UUIDs, let's insert one known ID main thread
	// or just check count.

	if len(col.payload) != count {
		t.Errorf("Expected %d items, got %d", count, len(col.payload))
	}
}
