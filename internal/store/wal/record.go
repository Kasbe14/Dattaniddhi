package wal

import (
	"encoding/binary"
	"hash/crc32"
)

type recordHeader struct {
	//  Offsets magicbytes 0-6
	version      uint8  // 7
	recordLength uint32 // 8-11
	// padding skiped 12 - 15
	lsn    uint64 // 16-23
	opType uint8  // 24
	// padding skiped 25-31 Total := 32bytes
}

func newRecordHeader(v uint8, lsn uint64, op uint8) *recordHeader {
	return &recordHeader{
		version: v,
		lsn:     lsn,
		opType:  op,
	}
}

func encodeRecordHeader(recHeader recordHeader) []byte {
	buf := make([]byte, recordHeaderByteSize)
	// Assign the magic bytes
	copy(buf[0:7], magicBytes)
	buf[7] = recHeader.version
	binary.LittleEndian.PutUint32(buf[8:12], recHeader.recordLength)
	binary.LittleEndian.PutUint64(buf[16:24], recHeader.lsn)
	buf[24] = recHeader.opType

	return buf
}

type recordWrapper struct {
	recHeader     recordHeader
	payload       payload
	checksumCRC32 uint32
}

func newRecordWrapper(rh recordHeader, pl payload) *recordWrapper {
	// Assing the record length with variable payload size
	rh.recordLength = uint32(32 + pl.size() + 4)
	return &recordWrapper{
		recHeader: rh,
		payload:   pl,
	}
}

func (rw *recordWrapper) encode() []byte {
	plSize := rw.payload.size()
	bufSize := 32 + int(plSize) + 4
	buf := make([]byte, bufSize)
	rhBytes := encodeRecordHeader(rw.recHeader)
	plBytes := rw.payload.encode()
	offset := 0
	copy(buf[offset:offset+32], rhBytes)
	offset += 32
	copy(buf[offset:offset+int(plSize)], plBytes)
	// Create a check sum
	offset += int(plSize)
	//checksum  only for bytes till payload size last 4 bytes empty
	rw.checksumCRC32 = crc32.ChecksumIEEE(buf[:offset])
	// encoding the last four bytes
	binary.LittleEndian.PutUint32(buf[offset:], rw.checksumCRC32)
	return buf
}
