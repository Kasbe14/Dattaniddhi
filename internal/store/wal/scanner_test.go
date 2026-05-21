package wal

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// setupMockSegment is a helper to quickly generate a segment file with specific bytes
func setupMockSegment(t *testing.T, data []byte) *segment {
	t.Helper()
	tempDir := t.TempDir()
	fullPath := filepath.Join(tempDir, "0000000001.waldrky")

	err := os.WriteFile(fullPath, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write mock segment: %v", err)
	}

	file, err := os.OpenFile(fullPath, os.O_RDONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open mock segment: %v", err)
	}

	return &segment{
		segID:       1,
		file:        file,
		currentSize: uint64(len(data)),
	}
}

func TestScanSegmentFile(t *testing.T) {
	// 1. Prepare Valid Data blocks
	segHeaderBytes := encodeSegmentHeader(*newSegmentHeader(walVersion, 1))

	mockInsertPayload := &insertPayload{
		externalID: "doc-1",
		internalID: 100,
		vectorData: []float32{1.0, 2.0},
		metaData:   []byte(`{"test":"meta"}`),
	}
	insertRecHeader := newRecordHeader(walVersion, 1, OpInsert)
	insertWrapper := newRecordWrapper(*insertRecHeader, mockInsertPayload.encode())
	insertBytes := insertWrapper.encode()

	mockDeletePayload := &deletePayload{
		externalID: "doc-1",
		internalID: 100,
	}
	deleteRecHeader := newRecordHeader(walVersion, 2, OpDelete)
	deleteWrapper := newRecordWrapper(*deleteRecHeader, mockDeletePayload.encode())
	deleteBytes := deleteWrapper.encode()

	// Combine into a valid file layout
	var validFileBytes []byte
	validFileBytes = append(validFileBytes, segHeaderBytes...)
	validFileBytes = append(validFileBytes, insertBytes...)
	validFileBytes = append(validFileBytes, deleteBytes...)

	t.Run("Success: Happy Path", func(t *testing.T) {
		seg := setupMockSegment(t, validFileBytes)
		defer seg.file.Close()

		records, err := scanSegmentFile(seg)
		if err != nil {
			t.Fatalf("Scanner failed: %v", err)
		}

		if len(records) != 2 {
			t.Fatalf("Expected 2 records, got %d", len(records))
		}

		// Verify Record 1 (Insert)
		if records[0].recordHeader.lsn != 1 || records[0].recordHeader.opType != OpInsert {
			t.Errorf("First record header mismatch")
		}
		// Go type assertion to check if payload was decoded into correct struct
		if _, ok := records[0].payload.(*insertPayload); !ok {
			t.Errorf("Expected payload to be *insertPayload")
		}

		// Verify Record 2 (Delete)
		if records[1].recordHeader.lsn != 2 || records[1].recordHeader.opType != OpDelete {
			t.Errorf("Second record header mismatch")
		}
		if _, ok := records[1].payload.(*deletePayload); !ok {
			t.Errorf("Expected payload to be *deletePayload")
		}
	})

	t.Run("Success: Empty Segment (Only Header)", func(t *testing.T) {
		seg := setupMockSegment(t, segHeaderBytes)
		defer seg.file.Close()

		records, err := scanSegmentFile(seg)
		if err != nil {
			t.Fatalf("Scanner failed on empty segment: %v", err)
		}
		if len(records) != 0 {
			t.Errorf("Expected 0 records, got %d", len(records))
		}
	})

	t.Run("Recovery: Torn Write (Header Cut Off)", func(t *testing.T) {
		// Cut the file halfway through the Insert Record's header
		tornHeaderBytes := validFileBytes[:segmentHeaderByteSize+10]
		seg := setupMockSegment(t, tornHeaderBytes)
		defer seg.file.Close()

		records, err := scanSegmentFile(seg)
		if err != nil {
			t.Fatalf("Expected graceful recovery, got error: %v", err)
		}
		if len(records) != 0 {
			t.Errorf("Expected 0 valid records recovered, got %d", len(records))
		}
	})

	t.Run("Recovery: Torn Write (Payload Cut Off)", func(t *testing.T) {
		// Cut the file halfway through the Insert Record's payload
		tornPayloadBytes := validFileBytes[:segmentHeaderByteSize+recordHeaderByteSize+5]
		seg := setupMockSegment(t, tornPayloadBytes)
		defer seg.file.Close()

		records, err := scanSegmentFile(seg)
		if err != nil {
			t.Fatalf("Expected graceful recovery, got error: %v", err)
		}
		if len(records) != 0 {
			t.Errorf("Expected 0 valid records recovered, got %d", len(records))
		}
	})

	t.Run("Failure: Checksum Mismatch", func(t *testing.T) {
		corruptedBytes := bytes.Clone(validFileBytes)
		// Flip a byte deep inside the insert payload
		corruptedBytes[segmentHeaderByteSize+recordHeaderByteSize+5] = 0xFF

		seg := setupMockSegment(t, corruptedBytes)
		defer seg.file.Close()

		_, err := scanSegmentFile(seg)
		if err == nil || err.Error() != "corupt data : invalid checksum" {
			t.Fatalf("Expected invalid checksum error, got: %v", err)
		}
	})

	t.Run("Failure: Corrupted Segment Header", func(t *testing.T) {
		corruptedHeader := bytes.Clone(validFileBytes)
		corruptedHeader[0] = 'X' // Break magic byte

		seg := setupMockSegment(t, corruptedHeader)
		defer seg.file.Close()

		_, err := scanSegmentFile(seg)
		if err == nil {
			t.Fatalf("Expected segment header error, got nil")
		}
	})
}
