package collection

import (
	"sync"
	"testing"

	"github.com/Kasbe14/Dattaniddhi/internal/store/wal"
	"github.com/Kasbe14/Dattaniddhi/internal/types"
)

func TestCloseDuringConcurrentInsert(t *testing.T) {

	rootDir := t.TempDir()

	cfg := CollectionConfig{
		Name:      "close-race",
		Dimension: 3,
		Metric:    types.Cosine,
		IndexType: types.LinearIndex,
		DataType:  types.Text,
		Version:   1,
		ModelName: "test-model",
	}

	coll, err := CreateCollection(cfg, rootDir, wal.SyncEverySec)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {

		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			_, _ = coll.Insert([]float32{1, 2, 3}, i)

		}(i)
	}

	_ = coll.Close()

	wg.Wait()
}
