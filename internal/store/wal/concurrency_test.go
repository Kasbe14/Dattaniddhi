package wal

import (
	"sync"
	"testing"
)

func TestWAL_ConcurrentAppendInsert(t *testing.T) {
	dir := t.TempDir()

	w, err := NewWAL(dir, SyncAlways)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			_, err := w.AppendInsert(
				"id",
				uint64(i),
				[]float32{1, 2, 3},
				[]byte("meta"),
			)

			if err != nil {
				t.Errorf("append failed: %v", err)
			}
		}(i)
	}

	wg.Wait()
}

func TestWAL_ConcurrentAppendDelete(t *testing.T) {
	dir := t.TempDir()

	w, err := NewWAL(dir, SyncAlways)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()

			_, err := w.AppendDelete("id", uint64(i))
			if err != nil {
				t.Errorf("delete append failed: %v", err)
			}
		}(i)
	}

	wg.Wait()
}

func TestWAL_DoubleClose(t *testing.T) {
	dir := t.TempDir()

	w, err := NewWAL(dir, SyncAlways)
	if err != nil {
		t.Fatal(err)
	}

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("double close should be safe: %v", err)
	}
}
