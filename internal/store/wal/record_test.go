package wal

import (
	"encoding/binary"
	"hash/crc32"
	"testing"
)

func TestRecordHeader_Encode(t *testing.T) {
	rh := newRecordHeader(1, 42, OpInsert)
	rh.recordLength = 100 // Manually set for this test

	encoded := encodeRecordHeader(*rh)

	if len(encoded) != recordHeaderByteSize {
		t.Fatalf("Expected header size %d, got %d", recordHeaderByteSize, len(encoded))
	}

	if string(encoded[0:7]) != magicBytes {
		t.Errorf("Expected magic bytes %s, got %s", magicBytes, string(encoded[0:7]))
	}
	if encoded[7] != 1 {
		t.Errorf("Expected version 1, got %d", encoded[7])
	}
	lsn := binary.LittleEndian.Uint64(encoded[16:24])
	if lsn != 42 {
		t.Errorf("Expected LSN 42, got %d", lsn)
	}
	if encoded[24] != OpInsert {
		t.Errorf("Expected OpType %d, got %d", OpInsert, encoded[24])
	}
}

func TestRecordWrapper_Encode(t *testing.T) {
	rh := newRecordHeader(1, 42, OpInsert)
	dummyPayload := []byte("test payload data")

	// Updated to pass []byte directly
	rw := newRecordWrapper(*rh, dummyPayload)

	expectedRecordLen := uint32(32 + len(dummyPayload) + 4)
	if rw.recHeader.recordLength != expectedRecordLen {
		t.Errorf("Expected record length %d, got %d", expectedRecordLen, rw.recHeader.recordLength)
	}

	encoded := rw.encode()
	if uint32(len(encoded)) != expectedRecordLen {
		t.Errorf("Expected encoded length %d, got %d", expectedRecordLen, len(encoded))
	}

	dataForCRC := encoded[:len(encoded)-4]
	expectedCRC := crc32.ChecksumIEEE(dataForCRC)

	actualCRC := binary.LittleEndian.Uint32(encoded[len(encoded)-4:])
	if actualCRC != expectedCRC {
		t.Errorf("CRC mismatch. Expected %d, got %d", expectedCRC, actualCRC)
	}
}
