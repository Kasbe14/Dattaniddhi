package wal

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
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
	// -------------------------------------------------------------------------
	// 1. Prepare Valid Mock Data Blocks
	// -------------------------------------------------------------------------
	segHeaderBytes := encodeSegmentHeader(*newSegmentHeader(walVersion, 1))

	// Mock Insert Record
	mockInsert := &insertPayload{
		externalID: "doc-alpha",
		internalID: 100,
		vectorData: []float32{1.5, -2.5},
		metaData:   []byte(`{"color":"red"}`),
	}
	insertRecHeader := newRecordHeader(walVersion, 1, OpInsert)
	insertWrapper := newRecordWrapper(*insertRecHeader, mockInsert.encode())
	insertBytes := insertWrapper.encode()

	// Mock Delete Record
	mockDelete := &deletePayload{
		externalID: "doc-beta",
		internalID: 200,
	}
	deleteRecHeader := newRecordHeader(walVersion, 2, OpDelete)
	deleteWrapper := newRecordWrapper(*deleteRecHeader, mockDelete.encode())
	deleteBytes := deleteWrapper.encode()

	// Combine into a valid file layout (Header -> Insert -> Delete)
	var validFileBytes []byte
	validFileBytes = append(validFileBytes, segHeaderBytes...)
	validFileBytes = append(validFileBytes, insertBytes...)
	validFileBytes = append(validFileBytes, deleteBytes...)

	// -------------------------------------------------------------------------
	// 2. Test Cases
	// -------------------------------------------------------------------------

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
		rec1 := records[0]
		if rec1.LSN != 1 || rec1.OpType != OpInsert {
			t.Errorf("First record header mismatch")
		}
		if rec1.ExtID != "doc-alpha" || rec1.IntID != 100 {
			t.Errorf("Insert IDs mismatch: got Ext:%s Int:%d", rec1.ExtID, rec1.IntID)
		}
		if !reflect.DeepEqual(rec1.Vector, []float32{1.5, -2.5}) {
			t.Errorf("Insert Vector mismatch: got %v", rec1.Vector)
		}
		if !reflect.DeepEqual(rec1.MetaData, []byte(`{"color":"red"}`)) {
			t.Errorf("Insert Metadata mismatch: got %s", rec1.MetaData)
		}

		// Verify Record 2 (Delete)
		rec2 := records[1]
		if rec2.LSN != 2 || rec2.OpType != OpDelete {
			t.Errorf("Second record header mismatch")
		}
		if rec2.ExtID != "doc-beta" || rec2.IntID != 200 {
			t.Errorf("Delete IDs mismatch: got Ext:%s Int:%d", rec2.ExtID, rec2.IntID)
		}
		if rec2.Vector != nil || rec2.MetaData != nil {
			t.Errorf("Delete record should not have vector or metadata populated")
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
		// (16 byte segment header + 10 bytes of the 32 byte record header)
		tornHeaderBytes := validFileBytes[:segmentHeaderByteSize+10]
		seg := setupMockSegment(t, tornHeaderBytes)
		defer seg.file.Close()

		records, err := scanSegmentFile(seg)
		if err != nil {
			t.Fatalf("Expected graceful recovery, got error: %v", err)
		}
		// It should silently ignore the torn write and return 0 valid records
		if len(records) != 0 {
			t.Errorf("Expected 0 valid records recovered, got %d", len(records))
		}
	})

	t.Run("Recovery: Torn Write (Payload Cut Off)", func(t *testing.T) {
		// Cut the file halfway through the Insert Record's payload
		// (16 byte seg header + 32 byte rec header + 5 bytes of payload)
		tornPayloadBytes := validFileBytes[:segmentHeaderByteSize+recordHeaderByteSize+5]
		seg := setupMockSegment(t, tornPayloadBytes)
		defer seg.file.Close()

		records, err := scanSegmentFile(seg)
		if err != nil {
			t.Fatalf("Expected graceful recovery, got error: %v", err)
		}
		// It should silently ignore the torn write and return 0 valid records
		if len(records) != 0 {
			t.Errorf("Expected 0 valid records recovered, got %d", len(records))
		}
	})

	t.Run("Failure: Checksum Mismatch", func(t *testing.T) {
		corruptedBytes := bytes.Clone(validFileBytes)
		// Flip a byte deep inside the insert payload (e.g., corrupt the vector data)
		corruptIndex := segmentHeaderByteSize + recordHeaderByteSize + 5
		corruptedBytes[corruptIndex] = 0xFF

		seg := setupMockSegment(t, corruptedBytes)
		defer seg.file.Close()

		_, err := scanSegmentFile(seg)
		if err == nil || err.Error() != "corupt data: invalid checksum" {
			t.Fatalf("Expected invalid checksum error, got: %v", err)
		}
	})

	t.Run("Failure: Corrupted Segment Header", func(t *testing.T) {
		corruptedHeader := bytes.Clone(validFileBytes)
		corruptedHeader[0] = 'X' // Break magic byte "SANGITA"

		seg := setupMockSegment(t, corruptedHeader)
		defer seg.file.Close()

		_, err := scanSegmentFile(seg)
		if err == nil {
			t.Fatalf("Expected segment header error, got nil")
		}
	})
}
