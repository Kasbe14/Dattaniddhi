package collection

import (
	"testing"

	"github.com/Kasbe14/Dattaniddhi/internal/store/wal"
	"github.com/Kasbe14/Dattaniddhi/internal/types"
)

func TestReplay_Idempotent(t *testing.T) {
	rootDir := t.TempDir()

	cfg := CollectionConfig{
		Name:      "idem",
		Dimension: 3,
		Metric:    types.Cosine,
		IndexType: types.LinearIndex,
		DataType:  types.Text,
		Version:   collectionConfigVersion,
	}

	coll, err := CreateCollection(cfg, rootDir, wal.SyncAlways)
	if err != nil {
		t.Fatal(err)
	}

	_, err = coll.Insert([]float32{1, 2, 3}, "payload")
	if err != nil {
		t.Fatal(err)
	}

	if err := coll.Close(); err != nil {
		t.Fatal(err)
	}

	opened1, err := OpenCollection(rootDir, cfg.Name, wal.SyncAlways)
	if err != nil {
		t.Fatal(err)
	}

	if opened1.idCounter != 1 {
		t.Fatalf("expected 1 got %d", opened1.idCounter)
	}

	if err := opened1.Close(); err != nil {
		t.Fatal(err)
	}

	opened2, err := OpenCollection(rootDir, cfg.Name, wal.SyncAlways)
	if err != nil {
		t.Fatal(err)
	}

	if opened2.idCounter != 1 {
		t.Fatalf("expected stable replay got %d", opened2.idCounter)
	}
	if err := opened2.Close(); err != nil {
		t.Fatal(err)
	}
}
