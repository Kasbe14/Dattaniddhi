package stress_test

import (
	"fmt"
	"testing"

	"github.com/Kasbe14/Dattaniddhi/internal/collection"
	"github.com/Kasbe14/Dattaniddhi/internal/store/wal"
	"github.com/Kasbe14/Dattaniddhi/internal/types"
)

func BenchmarkReplayLargeWAL(b *testing.B) {
	rootDir := b.TempDir()

	cfg := collection.CollectionConfig{
		Name:      "bench",
		Dimension: 3,
		Metric:    types.Cosine,
		IndexType: types.LinearIndex,
		DataType:  types.Text,
		Version:   1,
	}

	coll, err := collection.CreateCollection(cfg, rootDir, wal.SyncAlways)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < 10000; i++ {
		_, err := coll.Insert([]float32{1, 2, 3}, fmt.Sprintf("payload-%d", i))
		if err != nil {
			b.Fatal(err)
		}
	}

	if err := coll.Close(); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		opened, err := collection.OpenCollection(rootDir, cfg.Name, wal.SyncAlways)
		if err != nil {
			b.Fatal(err)
		}

		opened.Close()
	}
}
