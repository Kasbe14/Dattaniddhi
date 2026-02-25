package wal

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
)

type segmentHeader struct {
	version   uint8
	segmentID uint64
}

type segment struct {
	segID       uint64
	file        *os.File
	currentSize uint32
}

func newSegmentHeader(v uint8, segID uint64) *segmentHeader {
	return &segmentHeader{
		version:   v,
		segmentID: segID,
	}
}

func encodeSegmentHeader(segHeader segmentHeader) []byte {
	// 21 bytes buffer
	buf := make([]byte, segmentHeaderByteSize)

	// Assing magic bytes to the buf go -> []bytes("string")
	copy(buf[0:7], magicBytes)
	//struct data
	buf[7] = segHeader.version
	binary.LittleEndian.PutUint64(buf[8:16], segHeader.segmentID)

	return buf
}

func createSegment(dirPath string, id uint64) (*segment, error) {
	//create zero padded id based file name
	fileName := fmt.Sprintf("%010d.waldrky", id)
	//   ful path to store to
	fullPath := filepath.Join(dirPath, fileName)
	// Calling kernel for file
	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &segment{
		segID:       id,
		file:        file,
		currentSize: 0,
	}, nil
}
