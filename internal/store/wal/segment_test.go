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

func TestSegment_CreateOpenAndAppend(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("Create and Append", func(t *testing.T) {
		seg, err := createSegment(tempDir, 1)
		if err != nil {
			t.Fatalf("Failed to create segment: %v", err)
		}

		// Test the new append method
		data := []byte("hello WAL")
		n, err := seg.append(data)
		if err != nil || n != len(data) {
			t.Errorf("Append failed. Written: %d, Err: %v", n, err)
		}

		if seg.currentSize != uint64(len(data)) {
			t.Errorf("Expected currentSize %d, got %d", len(data), seg.currentSize)
		}
		seg.file.Close()

		// FIX: Actually use the path and os.Stat to verify the file exists!
		expectedPath := filepath.Join(tempDir, "0000000001.waldrky")
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Errorf("File %s was not created on disk", expectedPath)
		}
	})

	t.Run("Open Existing Segment", func(t *testing.T) {
		seg, err := openExistingSegment(tempDir, 1)
		if err != nil {
			t.Fatalf("Failed to open segment: %v", err)
		}
		defer seg.file.Close()

		if seg.currentSize != 9 { // "hello WAL" is 9 bytes
			t.Errorf("Expected currentSize 9, got %d", seg.currentSize)
		}
	})
}
