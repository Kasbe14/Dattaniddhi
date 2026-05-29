package integration_test

import (
	"testing"

	"github.com/Kasbe14/Dattaniddhi/internal/collection"
	"github.com/Kasbe14/Dattaniddhi/internal/store/wal"
	"github.com/Kasbe14/Dattaniddhi/internal/types"
)

func TestCrashRecovery_InsertReplay(t *testing.T) {
	rootDir := t.TempDir()

	cfg := collection.CollectionConfig{
		Name:      "recovery_test",
		Dimension: 3,
		Metric:    types.Cosine,
		IndexType: types.LinearIndex,
		DataType:  types.Text,
		Version:   1,
		ModelName: "test-model",
	}

	// Create collection
	coll, err := collection.CreateCollection(cfg, rootDir, wal.SyncAlways)
	if err != nil {
		t.Fatalf("failed to create collection: %v", err)
	}

	// Insert records
	id1, err := coll.Insert([]float32{1, 2, 3}, "payload-1")
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	id2, err := coll.Insert([]float32{4, 5, 6}, "payload-2")
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	// Delete one record
	err = coll.Delete(id1)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	// Simulate graceful shutdown before reboot
	err = coll.Close()
	if err != nil {
		t.Fatalf("close failed: %v", err)
	}

	// Re-open collection (simulates reboot/recovery)
	recoveredColl, err := collection.OpenCollection(rootDir, cfg.Name, wal.SyncAlways)
	if err != nil {
		t.Fatalf("failed to reopen collection: %v", err)
	}

	// Deleted record should NOT exist
	if _, ok := recoveredColl.ExtToInt()[id1]; ok {
		t.Fatalf("deleted record recovered unexpectedly")
	}

	// Existing record SHOULD exist
	if _, ok := recoveredColl.ExtToInt()[id2]; !ok {
		t.Fatalf("existing record missing after recovery")
	}

	// ID counter should persist correctly
	if recoveredColl.IDCounter() != 2 {
		t.Fatalf("expected idCounter=2 got=%d", recoveredColl.IDCounter())
	}

	// Internal map invariants
	if len(recoveredColl.ExtToInt()) != len(recoveredColl.IntToExt()) {
		t.Fatalf("mapping invariant broken after recovery")
	}

	err = recoveredColl.Close()
	if err != nil {
		t.Fatalf("close failed: %v", err)
	}
}
