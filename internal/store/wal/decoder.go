package wal

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func decodeRecordHeader(rhBytes []byte) (*recordHeader, error) {
	//check record header byte length 32bytes
	if len(rhBytes) != int(recordHeaderByteSize) {
		return nil, fmt.Errorf("invalid record header size")
	}
	//get and verify magic bytes form the buffer
	if !bytes.Equal(rhBytes[0:7], []byte(magicBytes)) {
		return nil, fmt.Errorf("corrupted record: invalid magic byte")
	}
	//construct the record header
	rh := &recordHeader{
		version:      rhBytes[7],
		recordLength: binary.LittleEndian.Uint32(rhBytes[8:12]),
		//padding 12-15 skiped
		lsn:    binary.LittleEndian.Uint64(rhBytes[16:24]),
		opType: rhBytes[24],
		//pading skipped 24-32
	}
	//version check
	if rh.version != walVersion {
		return nil, fmt.Errorf("unsupported wal version : %d", rh.version)
	}
	//optype check
	if !isValidOptype(rh.opType) {
		return nil, fmt.Errorf("invalid operation type")
	}
	//sanity check record lenght
	// record length min-> 36 max-> segment file size 64mb
	if rh.recordLength < minRecordLength {
		return nil, fmt.Errorf("corrupted record : length %d, too small", rh.recordLength)
	}
	if uint64(rh.recordLength) > maxSegmentFileSize {
		return nil, fmt.Errorf("corrupted record : length %d, exceeds max segment file size", rh.recordLength)
	}

	return rh, nil
}

func isValidOptype(op uint8) bool {
	switch op {
	case OpInsert, OpDelete, OpUpdate:
		//ok
		return true
	default:
		return false
	}
}
