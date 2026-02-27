package wal

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestSegmentHeader_Encode(t *testing.T) {
	sh := newSegmentHeader(walVersion, 99)
	encoded := encodeSegmentHeader(*sh)

	if len(encoded) != segmentHeaderByteSize {
		t.Fatalf("Expected size %d, got %d", segmentHeaderByteSize, len(encoded))
	}

	if string(encoded[0:7]) != magicBytes {
		t.Errorf("Expected magic bytes %s, got %s", magicBytes, string(encoded[0:7]))
	}

	segID := binary.LittleEndian.Uint64(encoded[8:16])
	if segID != 99 {
		t.Errorf("Expected segment ID 99, got %d", segID)
	}
}

func TestCreateAndOpenSegment(t *testing.T) {
	// Use Go's built-in temp directory for safe file testing
	tempDir := t.TempDir()

	t.Run("Create Segment", func(t *testing.T) {
		seg, err := createSegment(tempDir, 1)
		if err != nil {
			t.Fatalf("Failed to create segment: %v", err)
		}
		defer seg.file.Close()

		if seg.segID != 1 {
			t.Errorf("Expected segment ID 1, got %d", seg.segID)
		}

		// Verify file exists on disk with correct padding
		expectedPath := filepath.Join(tempDir, "0000000001.waldrky")
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Errorf("File %s was not created on disk", expectedPath)
		}
	})

	t.Run("Open Existing Segment", func(t *testing.T) {
		// First, write some dummy data to the file created above
		expectedPath := filepath.Join(tempDir, "0000000001.waldrky")
		os.WriteFile(expectedPath, []byte("dummy data"), 0644)

		seg, err := openExistingSegment(tempDir, 1)
		if err != nil {
			t.Fatalf("Failed to open segment: %v", err)
		}
		defer seg.file.Close()

		if seg.currentSize != 10 { // "dummy data" is 10 bytes
			t.Errorf("Expected currentSize 10, got %d", seg.currentSize)
		}
	})
}
