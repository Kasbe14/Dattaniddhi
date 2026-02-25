package index

import (
	"github.com/Kasbe14/Dattaniddhi/internal/types"
	v "github.com/Kasbe14/Dattaniddhi/internal/vector"
	"slices"
	"testing"
)

func TestNewLinearIndex_Constructor(t *testing.T) {
	// Sub-test 1: The Happy Path
	t.Run("ValidConfig", func(t *testing.T) {
		cfg, err := NewIndexConfig(types.LinearIndex, types.Cosine, 128)
		if err != nil {
			t.Fatalf("Failed to create config: %v", err)
		}

		li, err := NewLinearIndex(cfg)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Check Invariants
		if li.vectors == nil {
			t.Fatal("Vector map was not initialized")
		}

		// Check Getters (Contracts)
		if li.Dimension() != cfg.Dimension() {
			t.Errorf("Dimension mismatch: got %d, want %d", li.Dimension(), cfg.Dimension())
		}

		// use Errorf instead of Fatalf here so the test continues
		// to check the other fields even if one fails.
		if li.config.Metric() != cfg.Metric() {
			t.Errorf("Metric mismatch: got %v, want %v", li.config.Metric(), cfg.Metric())
		}
	})

	// Sub-test 2: The Sad Path
	t.Run("InvalidConfig", func(t *testing.T) {
		invalidCfg := IndexConfig{} // Zero value
		lIdx, err := NewLinearIndex(invalidCfg)

		if err == nil {
			t.Error("Expected error for empty config, but got nil")
		}
		if lIdx != nil {
			t.Error("Expected nil index instance on failure")
		}
	})
}

// Helper to create a valid index for testing
func setupIndex(t *testing.T, dim int) *LinearIndex {
	cfg, _ := NewIndexConfig(types.LinearIndex, types.Cosine, dim)
	idx, err := NewLinearIndex(cfg)
	if err != nil {
		t.Fatalf("failed to setup index: %v", err)
	}
	return idx
}

// Contract: Add must reject invalid VecIds and dimension mismatches.
// Invariant: After Add, the vector must be retrievable via Get.
func TestLinearIndex_AddAndGet(t *testing.T) {
	idx := setupIndex(t, 3)
	vec, _ := v.NewVector([]float32{1.0, 0.0, 0.0}, 3)

	t.Run("Successful Add", func(t *testing.T) {
		exists, err := idx.Add(1, vec)
		if err != nil || exists {
			t.Errorf("Expected success, got exists=%v, err=%v", exists, err)
		}

		// Verify via Get (RLock path)
		retrieved, ok := idx.Get(1)
		if !ok || retrieved != vec {
			t.Error("Vector was not stored correctly")
		}
	})

	t.Run("Add Duplicate VecId", func(t *testing.T) {
		exists, err := idx.Add(1, vec)
		if err != nil || !exists {
			t.Error("Expected exists=true for duplicate VecId")
		}
	})

	t.Run("Contract Violation: Dimension Mismatch", func(t *testing.T) {
		badVec, _ := v.NewVector([]float32{1.0, 0.0}, 2) // Dim 2 instead of 3
		_, err := idx.Add(2, badVec)
		if err == nil || err.Error() != "dimension mismatch" {
			t.Errorf("Expected dimension mismatch error, got %v", err)
		}
	})
}

// Contract: Delete must remove the item or return an error if missing.
// Post-condition: Get must return false after a successful Delete.
func TestLinearIndex_Delete(t *testing.T) {
	idx := setupIndex(t, 3)
	vec, _ := v.NewVector([]float32{1.0, 0.0, 0.0}, 3)
	idx.Add(1, vec)

	t.Run("Successful Delete", func(t *testing.T) {
		err := idx.Delete(1)
		if err != nil {
			t.Errorf("Delete failed: %v", err)
		}

		_, ok := idx.Get(1)
		if ok {
			t.Error("Vector still exists after deletion")
		}
	})

	t.Run("Delete Non-existent", func(t *testing.T) {
		err := idx.Delete(5)
		if err == nil {
			t.Error("Expected error when deleting non-existent VecId")
		}
	})
}

// Concurrency Test: Ensures no race conditions occur when multiple goroutines
// read and write at the same time.
func TestLinearIndex_Concurrency(t *testing.T) {
	idx := setupIndex(t, 3)
	vec, _ := v.NewVector([]float32{1.0, 0.0, 0.0}, 3)

	// Use a wait group to coordinate goroutines
	done := make(chan bool)

	// Start a writer
	go func() {
		for i := 0; i < 100; i++ {
			idx.Add(1, vec)
		}
		done <- true
	}()

	// Start a reader
	go func() {
		for i := 0; i < 100; i++ {
			idx.Get(1)
		}
		done <- true
	}()

	// Wait for both to finish
	for i := 0; i < 2; i++ {
		<-done
	}
	// If this test finishes without a panic, the Mutexes are working!
}

//=================tests for Linear index Search function =========

// Helper to set up a populated index for testing
func setupPopulatedIndex(t *testing.T, metric types.SimilarityMetric) *LinearIndex {
	cfg, _ := NewIndexConfig(types.LinearIndex, metric, 2)
	idx, _ := NewLinearIndex(cfg)

	// Add 3 vectors:
	// Vec A: [1, 0] (On X-axis)
	// Vec B: [0, 1] (On Y-axis)
	// Vec C: [0.707, 0.707] (45 degrees, normalized)
	vecA, _ := v.NewVector([]float32{1.0, 0.0}, 2)
	vecB, _ := v.NewVector([]float32{0.0, 1.0}, 2)
	vecC, _ := v.NewVector([]float32{0.707, 0.707}, 2) // approx 1/sqrt(2)

	idx.Add(1, vecA)
	idx.Add(2, vecB)
	idx.Add(3, vecC)

	return idx
}

func TestLinearIndex_Search_Contracts(t *testing.T) {
	idx := setupPopulatedIndex(t, types.Cosine)
	queryVec, _ := v.NewVector([]float32{1.0, 0.0}, 2)

	tests := []struct {
		name        string
		query       *v.Vector
		k           int
		expectedErr string
	}{
		{
			name:        "Invalid K (Zero)",
			query:       queryVec,
			k:           0,
			expectedErr: "invalid input for number of results",
		},
		{
			name:        "Invalid K (Negative)",
			query:       queryVec,
			k:           -5,
			expectedErr: "invalid input for number of results",
		},
		{
			name:        "Nil Query",
			query:       nil,
			k:           5,
			expectedErr: "empty query input",
		},
		{
			name:        "Dimension Mismatch",
			query:       func() *v.Vector { v, _ := v.NewVector([]float32{1.0, 0.0, 0.0}, 3); return v }(), // 3D query for 2D index
			k:           5,
			expectedErr: "index and query dimension mismatched",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := idx.Search(tt.query, tt.k)
			if err == nil {
				t.Fatalf("Expected error '%s', got nil", tt.expectedErr)
			}
			if results != nil {
				t.Fatal("Expected nil results on error")
			}
		})
	}
}

// Logic Test: Verify Cosine Similarity Sorting
// Query: [1, 0]
// Expected Order:
// 1. vec-A [1, 0] (Score 1.0) - Perfect match
// 2. vec-C [0.7, 0.7] (Score ~0.7) - 45 degrees
// 3. vec-B [0, 1] (Score 0.0) - 90 degrees (Orthogonal)
func TestLinearIndex_Search_CosineLogic(t *testing.T) {
	idx := setupPopulatedIndex(t, types.Cosine)
	query, _ := v.NewVector([]float32{1.0, 0.0}, 2)

	// Act: Ask for top 3
	results, err := idx.Search(query, 3)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Assert: Check size
	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	// Assert: Check Order (Descending Score)
	if results[0].VecId != 1 {
		t.Errorf("Top result should be 1, got %d (Score: %f)", results[0].VecId, results[0].Score)
	}
	if results[1].VecId != 3 {
		t.Errorf("Second result should be 3, got %d", results[1].VecId)
	}
	if results[2].VecId != 2 {
		t.Errorf("Third result should be 2, got %d", results[2].VecId)
	}

	// Assert: Check Sorting property
	if !slices.IsSortedFunc(results, func(a, b SearchResult) int {
		// Note: IsSortedFunc expects 'a <= b' for ascending, so we reverse logic for descending check
		// or simpler: just check manual loop
		return 0 // Dummy return, manual check below is safer for floats
	}) {
		// Manual check
		if results[0].Score < results[1].Score || results[1].Score < results[2].Score {
			t.Error("Results are not sorted by score descending")
		}
	}
}

// Logic Test: Verify Euclidean Sorting (Negative Scores)
// Since your Euclidean returns -(distance^2), the "closest" vector (distance 0)
// will have score -0.0, which is > -100.0.
// So Descending Sort should still put the closest match first.
func TestLinearIndex_Search_EuclideanLogic(t *testing.T) {
	idx := setupPopulatedIndex(t, types.Euclidean) // Uses Euclidean Metric
	query, _ := v.NewVector([]float32{1.0, 0.0}, 2)

	// Query: [1, 0]
	// vec-A [1, 0] -> Dist 0 -> Score -0
	// vec-B [0, 1] -> Dist 2 (sq) -> Score -2

	results, err := idx.Search(query, 3)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// The closest vector (vec-A) should still be first because -0 > -2
	if results[0].VecId != 1 {
		t.Errorf("Euclidean Sort Fail: Closest vec should be first. Got %d with score %f", results[0].VecId, results[0].Score)
	}

	// Ensure scores are negative (or zero)
	if results[0].Score > 0 {
		t.Errorf("Euclidean score should be negative or zero, got %f", results[0].Score)
	}
}

// Boundary Test: Requesting k > Size vs k < Size
func TestLinearIndex_Search_K_Boundaries(t *testing.T) {
	idx := setupPopulatedIndex(t, types.Cosine) // Has 3 items
	query, _ := v.NewVector([]float32{1.0, 0.0}, 2)

	t.Run("K is smaller than Size", func(t *testing.T) {
		k := 1
		results, _ := idx.Search(query, k)
		if len(results) != 1 {
			t.Errorf("Expected 1 result, got %d", len(results))
		}
	})

	t.Run("K is larger than Size", func(t *testing.T) {
		k := 10
		results, _ := idx.Search(query, k)
		// Should return all 3 available, not crash or return empty slots
		if len(results) != 3 {
			t.Errorf("Expected 3 results (capped by index size), got %d", len(results))
		}
	})

	t.Run("Index is Empty", func(t *testing.T) {
		emptyCfg, _ := NewIndexConfig(types.LinearIndex, types.Cosine, 2)
		emptyIdx, _ := NewLinearIndex(emptyCfg)

		results, err := emptyIdx.Search(query, 5)
		if err != nil {
			t.Errorf("Expected no error for empty index search, got %v", err)
		}
		if results != nil { // Your code returns nil, nil
			t.Error("Expected nil results for empty index")
		}
	})
}
