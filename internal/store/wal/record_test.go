package wal

import (
	"encoding/binary"
	"hash/crc32"
	"testing"
)

// Mock payload for testing record wrappers
type mockPayload struct {
	data []byte
}

func (m *mockPayload) encode() []byte { return m.data }
func (m *mockPayload) size() uint32   { return uint32(len(m.data)) }

func TestRecordHeader_Encode(t *testing.T) {
	rh := newRecordHeader(1, 42, OpInsert)
	// We need to set recordLength manually for this test since newRecordWrapper usually sets it
	rh.recordLength = 100

	encoded := encodeRecordHeader(*rh)

	if len(encoded) != recordHeaderByteSize {
		t.Fatalf("Expected header size %d, got %d", recordHeaderByteSize, len(encoded))
	}

	// Verify Magic Bytes
	if string(encoded[0:7]) != magicBytes {
		t.Errorf("Expected magic bytes %s, got %s", magicBytes, string(encoded[0:7]))
	}

	// Verify Version
	if encoded[7] != 1 {
		t.Errorf("Expected version 1, got %d", encoded[7])
	}

	// Verify LSN
	lsn := binary.LittleEndian.Uint64(encoded[16:24])
	if lsn != 42 {
		t.Errorf("Expected LSN 42, got %d", lsn)
	}

	// Verify OpType
	if encoded[24] != OpInsert {
		t.Errorf("Expected OpType %d, got %d", OpInsert, encoded[24])
	}
}

func TestRecordWrapper_Encode(t *testing.T) {
	rh := newRecordHeader(1, 42, OpInsert)
	mp := &mockPayload{data: []byte("test payload")}

	rw := newRecordWrapper(*rh, mp)

	// Ensure length was calculated correctly (32 header + payload size + 4 CRC)
	expectedRecordLen := uint32(32 + len(mp.data) + 4)
	if rw.recHeader.recordLength != expectedRecordLen {
		t.Errorf("Expected record length %d, got %d", expectedRecordLen, rw.recHeader.recordLength)
	}

	encoded := rw.encode()
	if uint32(len(encoded)) != expectedRecordLen {
		t.Errorf("Expected encoded length %d, got %d", expectedRecordLen, len(encoded))
	}

	// Verify CRC32
	// The CRC is calculated over everything EXCEPT the last 4 bytes
	dataForCRC := encoded[:len(encoded)-4]
	expectedCRC := crc32.ChecksumIEEE(dataForCRC)

	actualCRC := binary.LittleEndian.Uint32(encoded[len(encoded)-4:])
	if actualCRC != expectedCRC {
		t.Errorf("CRC mismatch. Expected %d, got %d", expectedCRC, actualCRC)
	}
}
