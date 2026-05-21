package wal

import (
	"bytes"
	"reflect"
	"testing"
)

// -----------------------------------------------------------------------------
// Test: Record Header Decoder
// -----------------------------------------------------------------------------
func TestDecodeRecordHeader(t *testing.T) {
	// 1. Happy Path: Create a valid header and encode it
	validHeader := newRecordHeader(walVersion, 1024, OpInsert)
	validHeader.recordLength = 100 // Set a valid length (>= 36, <= 64MB)
	validBytes := encodeRecordHeader(*validHeader)

	t.Run("Success: Valid Header", func(t *testing.T) {
		decoded, err := decodeRecordHeader(validBytes)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if decoded.lsn != 1024 {
			t.Errorf("Expected LSN 1024, got %d", decoded.lsn)
		}
		if decoded.opType != OpInsert {
			t.Errorf("Expected OpType %d, got %d", OpInsert, decoded.opType)
		}
	})

	t.Run("Failure: Invalid Size", func(t *testing.T) {
		_, err := decodeRecordHeader(validBytes[:20]) // Cut short
		if err == nil {
			t.Error("Expected error for invalid byte size, got nil")
		}
	})

	t.Run("Failure: Corrupted Magic Bytes", func(t *testing.T) {
		badMagic := bytes.Clone(validBytes)
		badMagic[0] = 'X' // Change 'S' in "SANGITA" to 'X'
		_, err := decodeRecordHeader(badMagic)
		if err == nil || err.Error() != "corrupted record: invalid magic byte" {
			t.Errorf("Expected magic byte error, got: %v", err)
		}
	})

	t.Run("Failure: Invalid Version", func(t *testing.T) {
		badVersion := bytes.Clone(validBytes)
		badVersion[7] = 99 // Set unsupported version
		_, err := decodeRecordHeader(badVersion)
		if err == nil {
			t.Error("Expected error for invalid version, got nil")
		}
	})

	t.Run("Failure: Length Too Small", func(t *testing.T) {
		smallLen := bytes.Clone(validBytes)
		smallLen[8] = 10 // Set length to 10 (less than minimum 36)
		smallLen[9], smallLen[10], smallLen[11] = 0, 0, 0
		_, err := decodeRecordHeader(smallLen)
		if err == nil {
			t.Error("Expected error for length too small, got nil")
		}
	})
}

// -----------------------------------------------------------------------------
// Test: Insert Payload Decoder
// -----------------------------------------------------------------------------
func TestInsertPayloadDecoder(t *testing.T) {
	original := &insertPayload{
		externalID: "doc-alpha",
		internalID: 999,
		vectorData: []float32{1.5, -2.5, 3.1415},
		metaData:   []byte(`{"color":"red"}`),
	}
	encodedBytes := original.encode()

	t.Run("Success: Valid Insert Payload", func(t *testing.T) {
		decoded, err := decodeInsertPayload(encodedBytes)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if decoded.externalID != original.externalID {
			t.Errorf("Expected extID %s, got %s", original.externalID, decoded.externalID)
		}
		if decoded.internalID != original.internalID {
			t.Errorf("Expected intID %d, got %d", original.internalID, decoded.internalID)
		}
		// reflect.DeepEqual is perfect for comparing slices (vectors and metadata)
		if !reflect.DeepEqual(decoded.vectorData, original.vectorData) {
			t.Errorf("Vector data mismatch.\nExpected: %v\nGot:      %v", original.vectorData, decoded.vectorData)
		}
		if !reflect.DeepEqual(decoded.metaData, original.metaData) {
			t.Errorf("Metadata mismatch.\nExpected: %s\nGot:      %s", original.metaData, decoded.metaData)
		}
	})

	t.Run("Failure: Truncated Bytes (Bounds Checking)", func(t *testing.T) {
		// Test various truncation lengths to trigger different bounds checks
		truncations := []int{
			1,  // Fails extID length
			5,  // Fails reading extID string
			15, // Fails reading internal ID
			30, // Fails reading vector values
		}

		for _, truncateLen := range truncations {
			_, err := decodeInsertPayload(encodedBytes[:truncateLen])
			if err == nil {
				t.Errorf("Expected error when payload is truncated to %d bytes", truncateLen)
			}
		}
	})
}

// -----------------------------------------------------------------------------
// Test: Delete Payload Decoder
// -----------------------------------------------------------------------------
func TestDeletePayloadDecoder(t *testing.T) {
	original := &deletePayload{
		externalID: "doc-omega",
		internalID: 404,
	}
	encodedBytes := original.encode()

	t.Run("Success: Valid Delete Payload", func(t *testing.T) {
		decoded, err := decodeDeletePayload(encodedBytes)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if decoded.externalID != original.externalID {
			t.Errorf("Expected extID %s, got %s", original.externalID, decoded.externalID)
		}
		if decoded.internalID != original.internalID {
			t.Errorf("Expected intID %d, got %d", original.internalID, decoded.internalID)
		}
	})

	t.Run("Failure: Truncated Delete Bytes", func(t *testing.T) {
		// Try parsing a slice that is missing its last byte
		_, err := decodeDeletePayload(encodedBytes[:len(encodedBytes)-1])
		if err == nil {
			t.Error("Expected error for truncated delete payload, got nil")
		}
	})
}

// -----------------------------------------------------------------------------
// Test: Segment Header Decoder
// -----------------------------------------------------------------------------
func TestDecodeSegmentHeader(t *testing.T) {
	validHeader := newSegmentHeader(walVersion, 42)
	validBytes := encodeSegmentHeader(*validHeader)

	t.Run("Success: Valid Segment Header", func(t *testing.T) {
		decoded, err := decodeSegmentHeader(validBytes)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if decoded.version != walVersion {
			t.Errorf("Expected version %d, got %d", walVersion, decoded.version)
		}
		if decoded.segmentID != 42 {
			t.Errorf("Expected segmentID 42, got %d", decoded.segmentID)
		}
	})

	t.Run("Failure: Invalid Length", func(t *testing.T) {
		_, err := decodeSegmentHeader(validBytes[:10]) // Cut short
		if err == nil {
			t.Error("Expected error for invalid byte size, got nil")
		}
	})

	t.Run("Failure: Corrupted Magic Bytes", func(t *testing.T) {
		badMagic := bytes.Clone(validBytes)
		badMagic[0] = 'X'
		_, err := decodeSegmentHeader(badMagic)
		if err == nil || err.Error() != "corrupted segment: invalid magic byte" {
			t.Errorf("Expected magic byte error, got: %v", err)
		}
	})

	t.Run("Failure: Invalid Version", func(t *testing.T) {
		badVersion := bytes.Clone(validBytes)
		badVersion[7] = 99
		_, err := decodeSegmentHeader(badVersion)
		if err == nil {
			t.Error("Expected error for invalid version, got nil")
		}
	})
}
