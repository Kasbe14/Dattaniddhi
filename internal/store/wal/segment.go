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
	currentSize uint64
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

// Create New Segment file
func createSegment(dirPath string, id uint64) (*segment, error) {
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, err
	}
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("WAL directory doesn't exists %s", dirPath)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path exists but is not a directory %s", dirPath)
	}
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

// to open existing segment file with highest segID
func openExistingSegment(dirPath string, id uint64) (*segment, error) {
	info, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("WAL directory doesn't exists %s", dirPath)
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path exists but is not a directory %s", dirPath)
	}
	//create zero padded id based file name
	fileName := fmt.Sprintf("%010d.waldrky", id)
	//   ful path to store to
	fullPath := filepath.Join(dirPath, fileName)
	// existing file open only for read and writing
	file, err := os.OpenFile(fullPath, os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	return &segment{
		segID:       id,
		file:        file,
		currentSize: uint64(fileInfo.Size()),
	}, nil
}
