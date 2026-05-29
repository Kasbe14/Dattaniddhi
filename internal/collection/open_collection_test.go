package collection

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Kasbe14/Dattaniddhi/internal/store/wal"
	"github.com/Kasbe14/Dattaniddhi/internal/types"
)

func TestOpenCollection_HappyPath(t *testing.T) {
	rootDir := t.TempDir()

	cfg := CollectionConfig{
		Name:      "test_collection",
		Dimension: 3,
		Metric:    types.Cosine,
		IndexType: types.LinearIndex,
		DataType:  types.Text,
		Version:   collectionConfigVersion,
		ModelName: "test-model",
	}

	created, err := CreateCollection(cfg, rootDir, wal.SyncAlways)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	_, err = created.Insert([]float32{1, 2, 3}, map[string]string{"a": "b"})
	if err != nil {
		t.Fatalf("insert failed: %v", err)
	}

	if err := created.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	opened, err := OpenCollection(rootDir, cfg.Name, wal.SyncAlways)
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}

	if opened.idCounter != 1 {
		t.Fatalf("expected idCounter 1 got %d", opened.idCounter)
	}
	if err := opened.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
}

func TestOpenCollection_MissingConfig(t *testing.T) {
	rootDir := t.TempDir()

	_, err := OpenCollection(rootDir, "missing", wal.SyncAlways)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenCollection_CorruptedConfig(t *testing.T) {
	rootDir := t.TempDir()

	collDir := filepath.Join(rootDir, "bad")
	if err := os.MkdirAll(collDir, 0755); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(collDir, "config.json")
	if err := os.WriteFile(configPath, []byte("{broken-json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := OpenCollection(rootDir, "bad", wal.SyncAlways)
	if err == nil {
		t.Fatal("expected corruption error")
	}
}

func TestOpenCollection_RepeatedOpenClose(t *testing.T) {
	rootDir := t.TempDir()

	cfg := CollectionConfig{
		Name:      "repeat",
		Dimension: 3,
		Metric:    types.Cosine,
		IndexType: types.LinearIndex,
		DataType:  types.Text,
		Version:   collectionConfigVersion,
		ModelName: "test-model",
	}

	coll, err := CreateCollection(cfg, rootDir, wal.SyncAlways)
	if err != nil {
		t.Fatal(err)
	}

	if err := coll.Close(); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		opened, err := OpenCollection(rootDir, cfg.Name, wal.SyncAlways)
		if err != nil {
			t.Fatalf("iteration %d failed: %v", i, err)
		}

		if err := opened.Close(); err != nil {
			t.Fatalf("close failed: %v", err)
		}
	}
}
