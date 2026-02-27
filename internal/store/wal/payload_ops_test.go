package wal

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestInsertPayload_SizeAndEncode(t *testing.T) {
	ip := &insertPayload{
		externalID: "doc-1",
		internalID: 100,
		vectorData: []float32{1.5, -2.0, 3.14},
		metaData:   []byte(`{"key":"value"}`),
	}

	// 1. Test Size Calculation
	expectedSize := uint32(2 + len("doc-1") + 8 + 4 + (4 * 3) + 4 + len(`{"key":"value"}`))
	if ip.size() != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, ip.size())
	}

	// 2. Test Encoding
	encoded := ip.encode()
	if uint32(len(encoded)) != expectedSize {
		t.Errorf("Encoded byte length %d does not match calculated size %d", len(encoded), expectedSize)
	}

	// 3. Verify Specific Decoded Values (Sanity Check)
	offset := 0
	extLen := binary.LittleEndian.Uint16(encoded[offset : offset+2])
	if extLen != 5 {
		t.Errorf("Expected external ID length 5, got %d", extLen)
	}
	offset += 2 + int(extLen)

	intID := binary.LittleEndian.Uint64(encoded[offset : offset+8])
	if intID != 100 {
		t.Errorf("Expected internal ID 100, got %d", intID)
	}
	offset += 8

	vecLen := binary.LittleEndian.Uint32(encoded[offset : offset+4])
	if vecLen != 3 {
		t.Errorf("Expected vector length 3, got %d", vecLen)
	}
	offset += 4

	// Spot check the first float32
	firstFloatBits := binary.LittleEndian.Uint32(encoded[offset : offset+4])
	firstFloat := math.Float32frombits(firstFloatBits)
	if firstFloat != 1.5 {
		t.Errorf("Expected first vector value 1.5, got %f", firstFloat)
	}
}

func TestDeletePayload_SizeAndEncode(t *testing.T) {
	dp := &deletePayload{
		externalID: "doc-1",
		internalID: 100,
	}

	expectedSize := uint32(2 + len("doc-1") + 8)
	if dp.size() != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, dp.size())
	}

	encoded := dp.encode()
	if uint32(len(encoded)) != expectedSize {
		t.Errorf("Encoded byte length %d does not match calculated size %d", len(encoded), expectedSize)
	}

	extLen := binary.LittleEndian.Uint16(encoded[0:2])
	if extLen != 5 {
		t.Errorf("Expected external ID length 5, got %d", extLen)
	}
}
