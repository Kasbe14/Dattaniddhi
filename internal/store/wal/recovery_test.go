package wal

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// Helper to quickly create empty files for scanning tests
func touchFile(t *testing.T, dir, name string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte{}, 0644)
	if err != nil {
		t.Fatalf("failed to create dummy file %s: %v", name, err)
	}
}

// -----------------------------------------------------------------------------
// Test 1: Segment Discovery (Happy Path & Architectural Proof)
// Requirement: Find .waldrky files, sort them, and DO NOT open them (prevent FD leaks).
// -----------------------------------------------------------------------------
func TestWAL_getAllSegment_ArchitectureAndSorting(t *testing.T) {
	tempDir := t.TempDir()
	w := &WAL{dir: tempDir}

	touchFile(t, tempDir, "0000000010.waldrky")
	touchFile(t, tempDir, "0000000002.waldrky")
	touchFile(t, tempDir, "0000000001.waldrky")
	touchFile(t, tempDir, "ignore_me.txt") // System must ignore this

	segments, err := w.getAllSegment()
	if err != nil {
		t.Fatalf("getAllSegment failed: %v", err)
	}

	// Assert 1: Filtering and Sorting
	if len(segments) != 3 {
		t.Fatalf("Expected 3 segments, got %d", len(segments))
	}
	if segments[0].segID != 1 || segments[1].segID != 2 || segments[2].segID != 10 {
		t.Errorf("Segments sorted incorrectly. Got IDs: %d, %d, %d",
			segments[0].segID, segments[1].segID, segments[2].segID)
	}

	// Assert 2: The Architectural Proof (No FD Leaks)
	for _, s := range segments {
		if s.file != nil {
			t.Errorf("FATAL: getAllSegment opened file %d! This causes FD leaks.", s.segID)
		}
		if s.path == "" {
			t.Errorf("FATAL: path was not set for segment %d", s.segID)
		}
	}
}

// -----------------------------------------------------------------------------
// Test 2: Invalid Segment Names (Sad Path)
// Requirement: System must abort if a .waldrky file cannot be parsed into an ID.
// -----------------------------------------------------------------------------
func TestWAL_getAllSegment_InvalidName(t *testing.T) {
	tempDir := t.TempDir()
	w := &WAL{dir: tempDir}

	touchFile(t, tempDir, "corrupted_name.waldrky")

	_, err := w.getAllSegment()
	if err == nil {
		t.Fatal("Expected error when parsing non-numeric .waldrky file, got nil")
	}
}

// -----------------------------------------------------------------------------
// Test 3: Empty Directory (Edge Case)
// Requirement: Booting a fresh database should return 0 segments cleanly.
// -----------------------------------------------------------------------------
func TestWAL_Recover_EmptyDirectory(t *testing.T) {
	w := &WAL{dir: t.TempDir()}

	records, err := w.Recover()
	if err != nil {
		t.Fatalf("Expected nil error for empty dir, got %v", err)
	}
	if len(records) != 0 {
		t.Errorf("Expected 0 records, got %d", len(records))
	}
}

// -----------------------------------------------------------------------------
// Test 4: Full Recovery Integration (Happy Path)
// Requirement: Physically write to disk, close it, and successfully recover it.
// -----------------------------------------------------------------------------
func TestWAL_Recover_Integration(t *testing.T) {
	tempDir := t.TempDir()
	w, err := NewWAL(tempDir, SyncAlways)
	if err != nil {
		t.Fatalf("Failed to init WAL: %v", err)
	}

	// Write real data to the disk
	w.AppendInsert("doc-1", 1, []float32{1.5, 2.5}, []byte(`{"test":"alpha"}`))
	w.AppendDelete("doc-1", 1)
	w.Close() // Flush and release all OS locks

	// Re-open fresh instance
	recoveryWal := &WAL{dir: tempDir}
	records, err := recoveryWal.Recover()

	if err != nil {
		t.Fatalf("Recover failed: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("Expected 2 records recovered from disk, got %d", len(records))
	}
	if records[0].OpType != OpInsert || records[0].ExtID != "doc-1" {
		t.Errorf("First record mismatch")
	}
	if records[1].OpType != OpDelete || records[1].ExtID != "doc-1" {
		t.Errorf("Second record mismatch")
	}
}

// -----------------------------------------------------------------------------
// Test 5: Corrupted File Contents (Crash Prevention)
// Requirement: If scanSegmentFile encounters broken bytes, Recover must abort.
// -----------------------------------------------------------------------------
func TestWAL_Recover_CorruptedFile(t *testing.T) {
	tempDir := t.TempDir()
	w := &WAL{dir: tempDir}

	// Write absolute garbage to a valid WAL file name
	err := os.WriteFile(filepath.Join(tempDir, "0000000001.waldrky"), []byte("this is not binary wal data"), 0644)
	if err != nil {
		t.Fatalf("Failed to write garbage file: %v", err)
	}

	_, err = w.Recover()
	if err == nil {
		t.Fatal("System Requirement Violation: Recover succeeded on physically corrupted file")
	}
}

// -----------------------------------------------------------------------------
// ADVANCED EDGE CASE 1: Torn Writes (Truncation)
// Requirement: If the OS crashes mid-write, the WAL must safely recover all
// completely written records and elegantly discard the torn bytes without panicking.
// -----------------------------------------------------------------------------
func TestWAL_Recover_TornWrites(t *testing.T) {
	tempDir := t.TempDir()
	w, _ := NewWAL(tempDir, SyncAlways)

	// Write two valid records
	w.AppendInsert("doc-1", 1, []float32{1.1, 1.1}, []byte(`{}`))
	w.AppendInsert("doc-2", 2, []float32{2.2, 2.2}, []byte(`{}`))
	w.Close()

	// Get the file size of the active segment
	entries, _ := os.ReadDir(tempDir)
	var activeFile string
	for _, e := range entries {
		if !e.IsDir() {
			activeFile = filepath.Join(tempDir, e.Name())
		}
	}
	info, _ := os.Stat(activeFile)
	originalSize := info.Size()

	// Sub-Test A: Truncate the CRC32 (last 2 bytes)
	os.Truncate(activeFile, originalSize-2)
	recoveryWal1 := &WAL{dir: tempDir}
	records1, err1 := recoveryWal1.Recover()
	if err1 != nil {
		t.Fatalf("Failed to handle CRC torn write safely: %v", err1)
	}
	if len(records1) != 1 {
		t.Errorf("Expected exactly 1 surviving record from CRC truncation, got %d", len(records1))
	}

	// Sub-Test B: Truncate deep into the payload
	os.Truncate(activeFile, originalSize-20)
	recoveryWal2 := &WAL{dir: tempDir}
	records2, err2 := recoveryWal2.Recover()
	if err2 != nil {
		t.Fatalf("Failed to handle payload torn write safely: %v", err2)
	}
	if len(records2) != 1 {
		t.Errorf("Expected exactly 1 surviving record from Payload truncation, got %d", len(records2))
	}
}

// -----------------------------------------------------------------------------
// ADVANCED EDGE CASE 2 & 3: Multi-Segment Replay & Monotonic LSN
// Requirement: Records spanning multiple segments must be read in strict
// ascending SegmentID order, producing strictly ascending LSNs.
// -----------------------------------------------------------------------------
func TestWAL_Recover_MultiSegmentAndLSN(t *testing.T) {
	tempDir := t.TempDir()
	w, _ := NewWAL(tempDir, SyncAlways)

	// Write to Segment 1
	w.AppendInsert("doc-1", 1, []float32{1.1}, []byte{})

	// Force a segment rotation manually (simulating crossing the 64MB boundary)
	w.rotateSegment()

	// Write to Segment 2
	w.AppendInsert("doc-2", 2, []float32{2.2}, []byte{})

	// Force another rotation
	w.rotateSegment()

	// Write to Segment 3
	w.AppendInsert("doc-3", 3, []float32{3.3}, []byte{})
	w.Close()

	recoveryWal := &WAL{dir: tempDir}
	records, err := recoveryWal.Recover()
	if err != nil {
		t.Fatalf("Multi-segment recovery failed: %v", err)
	}

	if len(records) != 3 {
		t.Fatalf("Expected 3 records across 3 segments, got %d", len(records))
	}

	// Assertion: Strict Monotonic LSN
	var prevLSN uint64 = 0
	for i, rec := range records {
		if rec.LSN <= prevLSN {
			t.Errorf("Monotonic LSN violation at index %d: current LSN %d is not greater than previous LSN %d", i, rec.LSN, prevLSN)
		}
		prevLSN = rec.LSN
	}
}

// -----------------------------------------------------------------------------
// ADVANCED EDGE CASE 6: Goroutine Leak Prevention
// Requirement: Closing a WAL initialized with SyncEverySec MUST terminate
// the background ticker, releasing the Goroutine.
// -----------------------------------------------------------------------------
func TestWAL_GoroutineLeak_Shutdown(t *testing.T) {
	// Note: You must import "runtime" and "time" at the top of the file for this!
	initialGoroutines := runtime.NumGoroutine()

	w, _ := NewWAL(t.TempDir(), SyncEverySec)

	// Goroutine should have spawned
	if runtime.NumGoroutine() <= initialGoroutines {
		t.Fatal("Expected backgroundSync goroutine to start, but count did not increase")
	}

	// Close the WAL, which should trigger the kill switch
	w.Close()

	// Give the scheduler a tiny window to terminate the background routine
	time.Sleep(100 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()

	// Assert no leaks
	if finalGoroutines > initialGoroutines {
		t.Errorf("Goroutine leak detected! Initial: %d, Final: %d. backgroundSync never died.", initialGoroutines, finalGoroutines)
	}
}
