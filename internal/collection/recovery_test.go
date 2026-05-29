package collection

import (
	"encoding/json"
	"testing"

	"github.com/Kasbe14/Dattaniddhi/internal/store/wal"
	"github.com/Kasbe14/Dattaniddhi/internal/types"
	"github.com/Kasbe14/Dattaniddhi/internal/vector"
)

// Helper to quickly spin up a test collection
func setupTestCollection(t *testing.T) (*Collection, string) {
	t.Helper()
	rootDir := t.TempDir()

	c, err := CreateCollection(CollectionConfig{
		Name:      "integration-recovery",
		Dimension: 2,
		Metric:    types.Cosine,
		IndexType: types.LinearIndex,
		DataType:  types.Text,
		ModelName: "model",
	}, rootDir, wal.SyncAlways)

	if err != nil {
		t.Fatalf("Failed to init Collection: %v", err)
	}
	return c, rootDir
}

// -----------------------------------------------------------------------------
// Test 1: Full System Recovery (Happy Path)
// -----------------------------------------------------------------------------
func TestCollection_LoadState_HappyPath(t *testing.T) {
	c, _ := setupTestCollection(t)
	defer c.Close()

	metaA, _ := json.Marshal(map[string]string{"name": "alpha"})
	metaB, _ := json.Marshal(map[string]string{"name": "beta"})

	// Inject directly into the WAL bypassing the RAM API
	c.wal.AppendInsert("doc-A", 1, []float32{1.0, 1.0}, metaA)
	c.wal.AppendInsert("doc-B", 2, []float32{2.0, 2.0}, metaB)
	c.wal.AppendDelete("doc-A", 1)

	// Force state reload
	err := c.LoadState()
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if c.idCounter != 2 {
		t.Errorf("Expected idCounter to be 2, got %d", c.idCounter)
	}
	if _, exists := c.extToInt["doc-A"]; exists {
		t.Error("doc-A should have been deleted")
	}
	if c.extToInt["doc-B"] != 2 {
		t.Errorf("Expected doc-B to map to internal ID 2, got %d", c.extToInt["doc-B"])
	}
}

// -----------------------------------------------------------------------------
// Test 2: Idempotent Delete
// -----------------------------------------------------------------------------
func TestCollection_LoadState_IdempotentDelete(t *testing.T) {
	c, _ := setupTestCollection(t)
	defer c.Close()

	c.wal.AppendDelete("ghost-doc", 99)
	err := c.LoadState()

	if err != nil {
		t.Fatalf("Expected LoadState to succeed due to index idempotency, got error: %v", err)
	}
	if c.idCounter != 99 {
		t.Errorf("Counter failed to track ghost ID. Expected 99, got %d", c.idCounter)
	}
}

// -----------------------------------------------------------------------------
// Test 3: Dimension Mismatch & Corrupt Metadata
// -----------------------------------------------------------------------------
func TestCollection_LoadState_InvalidDimension(t *testing.T) {
	c, _ := setupTestCollection(t)
	defer c.Close()

	c.wal.AppendInsert("bad-dim", 1, []float32{1.0, 2.0, 3.0}, []byte(`{}`))
	err := c.LoadState()
	if err == nil {
		t.Fatal("System Requirement Violation: LoadState succeeded despite dimension mismatch")
	}
}

func TestCollection_LoadState_CorruptJSON(t *testing.T) {
	c, _ := setupTestCollection(t)
	defer c.Close()

	c.wal.AppendInsert("corrupt-doc", 1, []float32{1.0, 1.0}, []byte(`{bad-json`))
	err := c.LoadState()
	if err == nil {
		t.Fatal("System Requirement Violation: LoadState succeeded despite corrupt JSON")
	}
}

// -----------------------------------------------------------------------------
// Test 4: Internal ID Collision
// -----------------------------------------------------------------------------
func TestCollection_LoadState_IdCollision(t *testing.T) {
	c, _ := setupTestCollection(t)
	defer c.Close()

	vec, _ := vector.NewVector([]float32{9.9, 9.9}, 2)
	c.index.Add(5, vec)
	c.wal.AppendInsert("doc-5", 5, []float32{1.0, 1.0}, []byte(`{}`))

	err := c.LoadState()
	if err == nil {
		t.Fatal("System Requirement Violation: LoadState ignored an internal ID collision")
	}
}

// -----------------------------------------------------------------------------
// Test 5: Startup Config Mismatch (Testing OpenCollection constraints)
// -----------------------------------------------------------------------------
func TestCollection_LoadState_ConfigMismatchDimension(t *testing.T) {
	rootDir := t.TempDir()

	// 1. Boot and insert 256-dimension vector
	cfg := CollectionConfig{Name: "mismatch", Dimension: 256, Metric: types.Cosine, IndexType: types.LinearIndex, DataType: types.Text, ModelName: "m"}
	c1, _ := CreateCollection(cfg, rootDir, wal.SyncAlways)

	dummy256 := make([]float32, 256)
	dummy256[0] = 1.0
	c1.Insert(dummy256, nil)
	c1.Close()

	// 2. Maliciously overwrite the config.json on disk to claim it's 128 dimension
	badCfg := cfg
	badCfg.Dimension = 128
	saveConfig(badCfg, rootDir)

	// 3. Opening must fail during WAL replay because dimensions conflict
	_, err := OpenCollection(rootDir, "mismatch", wal.SyncAlways)
	if err == nil {
		t.Fatal("System Requirement Violation: Recovery succeeded despite disk Dimension mismatch.")
	}
}

// -----------------------------------------------------------------------------
// Test 6: Recovery After Compensation Records
// -----------------------------------------------------------------------------
func TestCollection_LoadState_CompensationReplay(t *testing.T) {
	c, _ := setupTestCollection(t)
	defer c.Close()

	c.wal.AppendInsert("doc-compensation", 1, []float32{1.0, 0.0}, []byte(`{}`))
	c.wal.AppendDelete("doc-compensation", 1)

	err := c.LoadState()
	if err != nil {
		t.Fatalf("Recovery failed during compensation replay: %v", err)
	}

	if c.GetIdCounter() != 1 {
		t.Errorf("Counter tracking failed. Expected 1, got %d", c.GetIdCounter())
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	if _, exists := c.extToInt["doc-compensation"]; exists {
		t.Error("Compensation failed: Vector remains in extToInt map")
	}
}
