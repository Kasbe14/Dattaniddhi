package collection

import (
	"sync"
	"testing"
	"time"

	"github.com/Kasbe14/Dattaniddhi/internal/store/wal"
	"github.com/Kasbe14/Dattaniddhi/internal/types"
	"github.com/Kasbe14/Dattaniddhi/internal/vector"
)

// -----------------------------------------------------------------------------
// Test 1: Constructor Contracts (Validation)
// -----------------------------------------------------------------------------
func TestCreateCollection_Validation(t *testing.T) {
	rootDir := t.TempDir()

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
				DataType:  types.Text,
				ModelName: "test",
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
			col, err := CreateCollection(tt.cfg, rootDir, wal.SyncAlways)

			if err != tt.expectedErr {
				t.Errorf("Expected error '%v', got '%v'", tt.expectedErr, err)
			}
			if err == nil && col == nil {
				t.Error("Expected valid collection instance, got nil")
			}
			if col != nil {
				col.Close()
			}
		})
	}
}

// -----------------------------------------------------------------------------
// Test 2: Insert (Happy Path)
// -----------------------------------------------------------------------------
func TestCollection_InsertAndGet(t *testing.T) {
	rootDir := t.TempDir()
	cfg := CollectionConfig{
		Name:      "payload-test",
		Dimension: 2,
		Metric:    types.Cosine,
		IndexType: types.LinearIndex,
		DataType:  types.Text,
		ModelName: "test",
	}
	col, _ := CreateCollection(cfg, rootDir, wal.SyncAlways)
	defer col.Close()

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
	rootDir := t.TempDir()
	cfg := CollectionConfig{
		Name:      "rollback-test",
		Dimension: 2,
		Metric:    types.Cosine,
		IndexType: types.LinearIndex,
		DataType:  types.Text,
		ModelName: "test",
	}
	col, _ := CreateCollection(cfg, rootDir, wal.SyncAlways)
	defer col.Close()

	// SABOTAGE: Inject a vector directly into the underlying index with ID 1.
	vec, _ := vector.NewVector([]float32{9.9, 9.9}, 2)
	col.index.Add(1, vec)

	// Act: Try to insert a new document through the collection
	_, err := col.Insert([]float32{1.0, 1.0}, "this should fail")

	// Assert: It should fail and return the collision error
	if err == nil {
		t.Fatal("Expected Insert to fail due to ID collision, but it succeeded")
	}

	// Assert: Ensure NO memory was updated.
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
	rootDir := t.TempDir()
	col, _ := CreateCollection(CollectionConfig{Name: "del-test", Dimension: 2, Metric: types.Cosine, IndexType: types.LinearIndex, DataType: types.Text, ModelName: "t"}, rootDir, wal.SyncAlways)
	defer col.Close()

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
// Test 5: Delete (Sad Path - Split Brain Healing)
// -----------------------------------------------------------------------------
func TestCollection_Delete_SplitBrainSelfHealing(t *testing.T) {
	rootDir := t.TempDir()
	c, _ := CreateCollection(CollectionConfig{Name: "split-brain", Dimension: 2, Metric: types.Cosine, IndexType: types.LinearIndex, DataType: types.Text, ModelName: "t"}, rootDir, wal.SyncAlways)
	defer c.Close()

	// 1. Sabotage: Induce a split-brain state (maps updated, index empty)
	c.mu.Lock()
	c.extToInt["ghost-doc"] = 99
	c.intToExt[99] = "ghost-doc"
	c.payload["ghost-doc"] = []byte(`{}`)
	c.mu.Unlock()

	// 2. Act
	err := c.Delete("ghost-doc")

	// 3. Assert
	if err != nil {
		t.Fatalf("Expected nil error when healing split-brain, got: %v", err)
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	if _, exists := c.extToInt["ghost-doc"]; exists {
		t.Error("Split-brain was not healed: extToInt still contains ghost-doc")
	}
}

// -----------------------------------------------------------------------------
// Test 6: Search Logic
// -----------------------------------------------------------------------------
// -----------------------------------------------------------------------------
// Core API: Search (Rigorous Sorting & Capacity Limits)
// -----------------------------------------------------------------------------
func TestCollection_Search_Rigorous(t *testing.T) {
	rootDir := t.TempDir()
	col, _ := CreateCollection(CollectionConfig{
		Name:      "search-rigorous",
		Dimension: 2,
		Metric:    types.Cosine,
		IndexType: types.LinearIndex,
		DataType:  types.Text,
		ModelName: "test",
	}, rootDir, wal.SyncAlways)
	defer col.Close()

	// 1. Insert 4 distinct vectors so we have strictly predictable mathematical scores.
	// We will query with [1.0, 0.0] (Pointing straight right on the X-axis)
	idExact, _ := col.Insert([]float32{1.0, 0.0}, "Exact Match") // Cosine: 1.0
	idHalf, _ := col.Insert([]float32{1.0, 1.0}, "45 Degrees")   // Cosine: ~0.707
	idOrtho, _ := col.Insert([]float32{0.0, 1.0}, "Orthogonal")  // Cosine: 0.0
	idOpp, _ := col.Insert([]float32{-1.0, 0.0}, "Opposite")     // Cosine: -1.0

	// Sub-Test A: Requesting fewer results than the total index size
	t.Run("K less than index size", func(t *testing.T) {
		results, err := col.Search([]float32{1.0, 0.0}, 2)

		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("Expected exactly 2 results, got %d", len(results))
		}

		// Assert strict ordering by score (Highest score must be index 0)
		if results[0].VecID != idExact {
			t.Errorf("Rank 1 expected %s (Exact Match), got %s", idExact, results[0].VecID)
		}
		if results[1].VecID != idHalf {
			t.Errorf("Rank 2 expected %s (45 Degrees), got %s", idHalf, results[1].VecID)
		}
	})

	// Sub-Test B: Requesting more results than the total index size
	t.Run("K greater than index size", func(t *testing.T) {
		results, err := col.Search([]float32{1.0, 0.0}, 10)

		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		// The index only has 4 vectors, so it should gracefully cap at 4
		if len(results) != 4 {
			t.Fatalf("Expected exactly 4 results (total index capacity), got %d", len(results))
		}

		// Assert strict descending order for the entire dataset
		expectedOrder := []string{idExact, idHalf, idOrtho, idOpp}

		for i, expectedID := range expectedOrder {
			if results[i].VecID != expectedID {
				t.Errorf("Rank %d expected %s, got %s", i+1, expectedID, results[i].VecID)
			}

			// Verify the actual scores in the result array are strictly decreasing
			if i > 0 && results[i].Score >= results[i-1].Score {
				t.Errorf("Sorting failure: Score at rank %d (%f) is not strictly less than rank %d (%f)",
					i+1, results[i].Score, i, results[i-1].Score)
			}
		}
	})
}

// -----------------------------------------------------------------------------
// Test 7: Concurrency & Stress
// -----------------------------------------------------------------------------
func TestCollection_Concurrency(t *testing.T) {
	rootDir := t.TempDir()
	col, _ := CreateCollection(CollectionConfig{Name: "concur", Dimension: 2, Metric: types.Cosine, IndexType: types.LinearIndex, DataType: types.Text, ModelName: "t"}, rootDir, wal.SyncAlways)
	defer col.Close()

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

func TestCollection_Lifecycle_3Insert1Delete(t *testing.T) {
	rootDir := t.TempDir()
	collName := "test_sequence"

	cfg := CollectionConfig{
		Name:      collName,
		Dimension: 2,
		Metric:    types.Cosine,
		IndexType: types.LinearIndex,
		DataType:  types.Text,
		ModelName: "t",
	}

	// PHASE 1: Boot, Insert 3, Delete 1, Close
	c1, err := CreateCollection(cfg, rootDir, wal.SyncAlways)
	if err != nil {
		t.Fatalf("Phase 1 Boot failed: %v", err)
	}

	id1, _ := c1.Insert([]float32{1.1, 1.1}, map[string]string{"name": "alpha"})
	id2, _ := c1.Insert([]float32{2.2, 2.2}, map[string]string{"name": "beta"})
	id3, _ := c1.Insert([]float32{3.3, 3.3}, map[string]string{"name": "gamma"})

	c1.Delete(id2)
	c1.Close()

	// PHASE 2: Reboot & Verify automatic recovery via OpenCollection
	c2, err := OpenCollection(rootDir, collName, wal.SyncAlways)
	if err != nil {
		t.Fatalf("Phase 2 Boot failed: %v", err)
	}
	defer c2.Close()

	if c2.idCounter != 3 {
		t.Errorf("Expected idCounter to be 3, got %d", c2.idCounter)
	}

	if _, exists := c2.extToInt[id2]; exists {
		t.Error("doc-2 should be permanently deleted, but it exists in extToInt")
	}

	if _, exists := c2.extToInt[id1]; !exists {
		t.Error("doc-1 mapping failed to recover")
	}
	if _, exists := c2.extToInt[id3]; !exists {
		t.Error("doc-3 mapping failed to recover")
	}
}

// -----------------------------------------------------------------------------
// High Volume Benchmark
// -----------------------------------------------------------------------------
func TestCollection_HighVolumeRecovery(t *testing.T) {
	rootDir := t.TempDir()
	collName := "test_scale"

	cfg := CollectionConfig{
		Name:      collName,
		Dimension: 128,
		Metric:    types.Cosine,
		IndexType: types.LinearIndex,
		DataType:  types.Text,
		ModelName: "test-model",
	}

	c1, err := CreateCollection(cfg, rootDir, wal.SyncOS)
	if err != nil {
		t.Fatalf("Boot failed: %v", err)
	}

	const numVectors = 1000000
	dummyMeta := map[string]string{"status": "active"}
	dummyVec := make([]float32, 128)
	dummyVec[0] = 2.0 // Valid magnitude

	t.Logf("Writing %d vectors to disk via Insert API...", numVectors)
	for i := 1; i <= numVectors; i++ {
		c1.Insert(dummyVec, dummyMeta)
	}
	c1.Close()

	t.Log("Simulating crash and timing disk read speed...")
	start := time.Now()

	c2, err := OpenCollection(rootDir, collName, wal.SyncAlways)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Recovery failed: %v", err)
	}
	defer c2.Close()

	t.Logf("Successfully recovered %d vectors into RAM in %v", numVectors, duration)

	if c2.idCounter != numVectors {
		t.Errorf("Expected idCounter to be %d, got %d", numVectors, c2.idCounter)
	}
}
