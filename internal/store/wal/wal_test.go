package wal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetLatestSegmentID(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Test Empty Directory
	_, found, err := getLatestSegmentID(tempDir)
	if err != nil {
		t.Fatalf("Unexpected error on empty dir: %v", err)
	}
	if found {
		t.Error("Should not find a segment in an empty directory")
	}

	// 2. Create fake segment files
	files := []string{"0000000001.waldrky", "0000000005.waldrky", "0000000003.waldrky", "ignored.txt"}
	for _, f := range files {
		os.WriteFile(filepath.Join(tempDir, f), []byte(""), 0644)
	}

	// 3. Test Populated Directory
	segID, found, err := getLatestSegmentID(tempDir)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !found {
		t.Fatal("Expected to find segments")
	}
	if segID != 5 {
		t.Errorf("Expected highest segment ID to be 5, got %d", segID)
	}
}

func TestNewWAL(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("Initialize New WAL (Empty Dir)", func(t *testing.T) {
		wal, err := NewWAL(tempDir, SyncAlways)
		if err != nil {
			t.Fatalf("Failed to create WAL: %v", err)
		}
		defer wal.activeSegment.file.Close()

		if wal.segID != 1 {
			t.Errorf("Expected starting segID 1, got %d", wal.segID)
		}
		if wal.lsn != 0 {
			t.Errorf("Expected starting LSN 0, got %d", wal.lsn)
		}

		// Ensure segment header was written
		if wal.activeSegment.currentSize != segmentHeaderByteSize {
			t.Errorf("Expected segment size %d, got %d", segmentHeaderByteSize, wal.activeSegment.currentSize)
		}
	})

	t.Run("Initialize Existing WAL", func(t *testing.T) {
		// We are reusing the tempDir, so segment 1 already exists from the previous test.
		// NewWAL should discover it and open it.
		wal, err := NewWAL(tempDir, SyncAlways)
		if err != nil {
			t.Fatalf("Failed to open existing WAL: %v", err)
		}
		defer wal.activeSegment.file.Close()

		if wal.segID != 1 {
			t.Errorf("Expected to open segID 1, got %d", wal.segID)
		}
	})

	t.Run("Invalid Sync Policy", func(t *testing.T) {
		_, err := NewWAL(tempDir, SyncPolicy(99))
		if err == nil {
			t.Error("Expected error for invalid sync policy, got nil")
		}
	})
}
