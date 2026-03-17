package wal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWAL_LifecycleAndRecovery(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Initialize New WAL
	wal, err := NewWAL(tempDir, SyncAlways)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	if wal.segID != 1 {
		t.Errorf("Expected starting segID 1, got %d", wal.segID)
	}

	// 2. Append some records
	lsn1, err := wal.AppendInsert("doc-1", 100, []float32{1.1, 2.2}, []byte("meta1"))
	if err != nil {
		t.Fatalf("AppendInsert failed: %v", err)
	}
	lsn2, _ := wal.AppendInsert("doc-2", 101, []float32{3.3, 4.4}, []byte("meta2"))

	if lsn1 != 1 || lsn2 != 2 {
		t.Errorf("Expected LSNs 1 and 2, got %d and %d", lsn1, lsn2)
	}

	// Close the file to simulate server shutdown
	wal.activeSegment.file.Close()

	// 3. RECOVERY TEST: Open a new WAL instance on the same directory
	walRecovered, err := NewWAL(tempDir, SyncAlways)
	if err != nil {
		t.Fatalf("Failed to recover WAL: %v", err)
	}
	defer walRecovered.activeSegment.file.Close()

	// getLatestLSN should have scanned the file and found LSN 2
	if walRecovered.lsn != 2 {
		t.Errorf("Recovery failed. Expected LSN 2, got %d", walRecovered.lsn)
	}
	if walRecovered.segID != 1 {
		t.Errorf("Recovery failed. Expected active segID 1, got %d", walRecovered.segID)
	}
}

func TestWAL_Rotation(t *testing.T) {
	tempDir := t.TempDir()
	wal, err := NewWAL(tempDir, SyncAlways)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}

	// FIX: Add a dynamic defer to close whichever file is active at the end of the test.
	// We use an anonymous function so it evaluates wal.activeSegment at the END of the test,
	// not at the beginning (since rotation will change the active segment).
	defer func() {
		if wal != nil && wal.activeSegment != nil && wal.activeSegment.file != nil {
			wal.activeSegment.file.Close()
		}
	}()

	// TRICK: We artificially inflate the active segment size to be just
	// a few bytes shy of the 64MB limit (maxSegmentFileSize).
	// This forces the next AppendInsert to trigger rotateSegment().
	wal.activeSegment.currentSize = maxSegmentFileSize - 10

	// Act: Append a record that will push it over the 64MB edge
	lsn, err := wal.AppendInsert("doc-trigger", 999, []float32{0.0}, []byte{})
	if err != nil {
		t.Fatalf("Append failed during rotation: %v", err)
	}

	if lsn != 1 {
		t.Errorf("Expected LSN 1, got %d", lsn)
	}

	// Assert: Rotation should have occurred
	if wal.segID != 2 {
		t.Errorf("Expected segment ID to rotate to 2, got %d", wal.segID)
	}

	// Assert: Check that the old file was actually closed and set to read-only
	oldPath := filepath.Join(tempDir, "0000000001.waldrky")
	info, err := os.Stat(oldPath)
	if err != nil {
		t.Fatalf("Failed to stat old segment: %v", err)
	}

	// 0444 permission check (Read-only)
	// Note: Windows handles permissions differently than Linux, but 0444 usually
	// translates to the Windows "Read-only" attribute correctly in Go.
	if info.Mode().Perm() != 0444 {
		t.Errorf("Expected old segment permissions to be 0444, got %v", info.Mode().Perm())
	}
}
