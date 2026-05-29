package main_test

import (
	"path/filepath"
	"testing"

	"github.com/Kasbe14/Dattaniddhi/internal/collection"
	"github.com/Kasbe14/Dattaniddhi/internal/store/wal"
	"github.com/Kasbe14/Dattaniddhi/internal/types"
)

func TestDriver_EndToEndLifecycle(t *testing.T) {
	rootDir := filepath.Join(t.TempDir(), "data")
	collName := "production_users"

	cfg := collection.CollectionConfig{
		Name:      collName,
		Dimension: 4,
		Metric:    types.Cosine,
		IndexType: types.LinearIndex,
		DataType:  types.Text,
		ModelName: "model-test",
	}

	// =========================================================================
	// CYCLE 1: System Boot (Create) & Traffic
	// =========================================================================

	coll1, err := collection.CreateCollection(cfg, rootDir, wal.SyncAlways)
	if err != nil {
		t.Fatalf("Initial Boot: Collection failed to create: %v", err)
	}

	extID1, _ := coll1.Insert([]float32{1.0, 0.5, 0.2, 0.1}, map[string]string{"name": "user1"})
	extID2, _ := coll1.Insert([]float32{0.9, 0.8, 0.7, 0.6}, map[string]string{"name": "user2"})

	_ = coll1.Delete(extID1)

	if err := coll1.Close(); err != nil {
		t.Fatalf("Shutdown: Collection failed to close cleanly: %v", err)
	}

	// =========================================================================
	// CYCLE 2: System Reboot (Open) & Hydration Verification
	// =========================================================================

	coll2, err := collection.OpenCollection(rootDir, collName, wal.SyncAlways)
	if err != nil {
		t.Fatalf("Reboot: Collection failed to open/hydrate: %v", err)
	}
	t.Cleanup(func() { coll2.Close() })

	if coll2.GetIdCounter() != 2 {
		t.Errorf("Hydration failure: Expected idCounter to be 2, got %d", coll2.GetIdCounter())
	}

	if _, exists := coll2.Get(extID1); exists {
		t.Errorf("Hydration failure: Deleted vector %s resurrected from disk", extID1)
	}

	recoveredPayload, exists := coll2.Get(extID2)
	if !exists {
		t.Fatalf("Hydration failure: Surviving vector %s was lost", extID2)
	}

	payloadMap, ok := recoveredPayload.(map[string]interface{})
	if !ok || payloadMap["name"] != "user2" {
		t.Errorf("Hydration failure: Payload data corrupted or missing. Got: %v", recoveredPayload)
	}
}
