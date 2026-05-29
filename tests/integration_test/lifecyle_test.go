package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/Kasbe14/Dattaniddhi/internal/collection"
	"github.com/Kasbe14/Dattaniddhi/internal/store/wal"
	"github.com/Kasbe14/Dattaniddhi/internal/types"
)

func TestSystemLifecycle_EndToEnd(t *testing.T) {
	rootDir := filepath.Join(t.TempDir(), "data")
	collName := "integration_test_collection"

	cfg := collection.CollectionConfig{
		Name:      collName,
		Dimension: 3,
		Metric:    types.Cosine,
		IndexType: types.LinearIndex,
		DataType:  types.Text,
		ModelName: "test-model",
	}

	// =========================================================================
	// PHASE 1: System Boot & Data Ingestion
	// =========================================================================

	c1, err := collection.CreateCollection(cfg, rootDir, wal.SyncAlways)
	if err != nil {
		t.Fatalf("Phase 1: Collection boot failed: %v", err)
	}

	payloadA := map[string]string{"name": "alpha", "status": "active"}
	payloadB := map[string]string{"name": "beta", "status": "inactive"}

	idA, err := c1.Insert([]float32{1.0, 0.0, 0.0}, payloadA)
	if err != nil {
		t.Fatalf("Failed to insert vector A: %v", err)
	}

	idB, err := c1.Insert([]float32{0.0, 1.0, 0.0}, payloadB)
	if err != nil {
		t.Fatalf("Failed to insert vector B: %v", err)
	}

	if err := c1.Delete(idA); err != nil {
		t.Fatalf("Failed to delete vector A: %v", err)
	}

	if err := c1.Close(); err != nil {
		t.Fatalf("Phase 1: Collection failed to close cleanly: %v", err)
	}

	// =========================================================================
	// PHASE 2: System Crash Recovery (Booting via OpenCollection)
	// =========================================================================

	c2, err := collection.OpenCollection(rootDir, collName, wal.SyncAlways)
	if err != nil {
		t.Fatalf("Phase 2: Collection recovery boot failed: %v", err)
	}
	defer c2.Close()

	// =========================================================================
	// PHASE 3: Black-Box Assertions
	// =========================================================================

	if _, exists := c2.Get(idA); exists {
		t.Errorf("Hydration failure: Deleted vector %s was resurrected from disk", idA)
	}

	recoveredPayload, exists := c2.Get(idB)
	if !exists {
		t.Fatalf("Hydration failure: Surviving vector %s was lost", idB)
	}

	payloadMap, ok := recoveredPayload.(map[string]any)
	if !ok {
		t.Fatalf("Hydration failure: Payload data corrupted. Got: %T", recoveredPayload)
	}

	if payloadMap["name"] != "beta" || payloadMap["status"] != "inactive" {
		t.Errorf("Hydration failure: Payload values corrupted. Got: %v", payloadMap)
	}

	_, err = c2.Insert([]float32{0.0, 0.0, 1.0}, map[string]string{"name": "gamma"})
	if err != nil {
		t.Errorf("Hydration failure: Could not insert new data after recovery. Counter likely corrupted. Error: %v", err)
	}
}
