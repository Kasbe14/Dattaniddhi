package wal

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

func decodeSegmentHeader(shBytes []byte) (*segmentHeader, error) {
	if len(shBytes) != segmentHeaderByteSize {
		return nil, fmt.Errorf("corrupt segment header: incomplete segment header")
	}
	//magic byte check
	if !bytes.Equal(shBytes[0:7], []byte(magicBytes)) {
		return nil, fmt.Errorf("corrupted segment: invalid magic byte")
	}
	//wal version chek
	version := shBytes[7]
	if version != walVersion {
		return nil, fmt.Errorf("corrupted segment: invalid wal version")
	}
	segID := binary.LittleEndian.Uint64(shBytes[8:])

	return &segmentHeader{
		version:   version,
		segmentID: segID,
	}, nil
}

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
		return nil, fmt.Errorf("unsupported wal version: %d", rh.version)
	}
	//optype check
	if !isValidOptype(rh.opType) {
		return nil, fmt.Errorf("invalid operation type")
	}
	//sanity check record lenght
	// record length min-> 36 max-> segment file size 64mb
	if rh.recordLength < minRecordLength {
		return nil, fmt.Errorf("corrupted record: length %d, too small", rh.recordLength)
	}
	if uint64(rh.recordLength) > maxSegmentFileSize {
		return nil, fmt.Errorf("corrupted record: length %d, exceeds max segment file size", rh.recordLength)
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

func decodeInsertPayload(plBytes []byte) (*insertPayload, error) {

	//read size of the external id string
	offset := 0
	if len(plBytes) < 2 {
		return nil, fmt.Errorf("corrupted payload: incomplete external id length")
	}
	extIDLen := binary.LittleEndian.Uint16(plBytes)
	offset += 2

	//read the external id string
	if offset+int(extIDLen) > len(plBytes) {
		return nil, fmt.Errorf("corrupted payload : incomplete external id")
	}
	extID := string(bytes.Clone(plBytes[offset : offset+int(extIDLen)]))
	offset += int(extIDLen)
	//read internal id
	if offset+8 > len(plBytes) {
		return nil, fmt.Errorf("corrupted payload : incomplete internal id")
	}
	intID := binary.LittleEndian.Uint64(plBytes[offset:])
	offset += 8
	//read vector size
	if offset+4 > len(plBytes) {
		return nil, fmt.Errorf("corrupted payload: incomplete vector size")
	}
	vectorDimension := binary.LittleEndian.Uint32(plBytes[offset:])
	offset += 4
	// create and read vector values
	vectorData := make([]float32, vectorDimension)
	var i uint32
	for i = 0; i < (vectorDimension); i++ {
		//converting uint32 bit to float32
		if offset+4 > len(plBytes) {
			return nil, fmt.Errorf("corrupted payload : incomplete vector values")
		}
		vectorData[i] = math.Float32frombits(binary.LittleEndian.Uint32(plBytes[offset:]))
		offset += 4
	}
	if offset+4 > len(plBytes) {
		return nil, fmt.Errorf("corrupted payload: incorrect meta data size")
	}
	metaDataSize := binary.LittleEndian.Uint32(plBytes[offset:])
	offset += 4
	// TODO : meta data is stored as bytes so check how to represent it ?
	metaDataBytes := make([]byte, metaDataSize)
	if offset+int(metaDataSize) > len(plBytes) {
		return nil, fmt.Errorf("corrupted payload: incomplete metadata")
	}
	copy(metaDataBytes, plBytes[offset:offset+int(metaDataSize)])
	offset += int(metaDataSize)
	return &insertPayload{
		externalID: extID,
		internalID: intID,
		vectorData: vectorData,
		metaData:   metaDataBytes,
	}, nil
}

func decodeDeletePayload(plBytes []byte) (*deletePayload, error) {
	offset := 0
	//read external id length
	if len(plBytes) < 2 {
		return nil, fmt.Errorf("corrupted payload: incomplete external id length")
	}
	extIDLen := binary.LittleEndian.Uint16(plBytes)
	offset += 2
	//read external id string
	if offset+int(extIDLen) > len(plBytes) {
		return nil, fmt.Errorf("corrupted payload: incomplete external id")
	}
	extID := string(bytes.Clone(plBytes[offset : offset+int(extIDLen)]))
	offset += int(extIDLen)

	//read internal id
	if offset+8 > len(plBytes) {
		return nil, fmt.Errorf("corrupted payload: incomplete internal id")
	}
	intId := binary.LittleEndian.Uint64(plBytes[offset:])

	return &deletePayload{
		externalID: extID,
		internalID: intId,
	}, nil
}

func decodeCheckSumCrc32(chBytes []byte) (uint32, error) {
	if len(chBytes) != 4 {
		return 0, fmt.Errorf("corrupt checksum: incomplete checksum")
	}
	checksum := binary.LittleEndian.Uint32(chBytes)
	return checksum, nil
}

//TODO : update Operation and format
//func decodeUpdatePayload(plByte []byte) (*updatePayload, error)
