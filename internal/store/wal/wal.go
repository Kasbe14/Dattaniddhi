package wal

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
)

type SyncPolicy int

const (
	SyncEverySec SyncPolicy = iota + 1
	SyncAlways
	SyncOS
)

const (
	// Magic number
	magicBytes = "SANGITA"
	// Header size bytes
	segmentHeaderByteSize = 16
	recordHeaderByteSize  = 32
	// Operation Types
	OpInsert uint8 = 1
	OpDelete uint8 = 2
	OpUpdate uint8 = 3
	//Version
	walVersion uint8 = 1
)

// Orchestrator for the segment files and api for collections
type WAL struct {
	mu            sync.Mutex
	dir           string
	activeSegment *segment
	lsn           uint64
	segID         uint64
	syncPolicy    SyncPolicy
}

func NewWAL(dir string, sp SyncPolicy) (*WAL, error) {
	if sp < 1 || sp > 3 {
		return nil, fmt.Errorf("invalid sync policy %d", sp)
	}
	//directroy scanning for latest segment
	segID, found, err := getLatestSegmentID(dir)
	if err != nil {
		return nil, err
	}
	var activeSeg *segment
	var lsn uint64
	switch found {
	case true:
		//open latest segment file
		activeSeg, err = openExistingSegment(dir, segID)
		if err != nil {
			return nil, err
		}
		// : helper function getLatestLSN() to scan the segment for record and get lates lsn
		lsn, err = getLatestLSN(*activeSeg)
		if err != nil {
			return nil, err
		}
	case false:
		// no segment file exists create new one
		segID = 1
		// lsn increments after append
		lsn = 0
		activeSeg, err = createSegment(dir, segID)
		if err != nil {
			return nil, err
		}
		// pointer to new segmentHeader
		segHead := newSegmentHeader(walVersion, segID)

		segHeadBytes := encodeSegmentHeader(*segHead)
		//writing header to the segment file -> exactly one header per segment file
		bytesWritten, err := activeSeg.file.Write(segHeadBytes)
		if err != nil {
			return nil, err
		}
		if bytesWritten != segmentHeaderByteSize {
			return nil, fmt.Errorf("Expected %d bytes segment header, got %d bytes written", segmentHeaderByteSize, bytesWritten)
		}
		//update the current size of the segment file
		activeSeg.currentSize += uint64(bytesWritten)
	}
	return &WAL{
		dir:           dir,
		activeSegment: activeSeg,
		lsn:           lsn,
		segID:         segID,
		syncPolicy:    sp,
	}, nil
}

// helper function to scan segment file and return latest (highest) segId
func getLatestSegmentID(dir string) (uint64, bool, error) {
	var segId uint64
	var found bool
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, false, err
	}
	for _, entry := range entries {
		fileName := entry.Name()
		if !entry.IsDir() && strings.HasSuffix(fileName, ".waldrky") {
			numStr := strings.TrimSuffix(fileName, ".waldrky")
			id, err := strconv.ParseUint(numStr, 10, 64)
			if err != nil {
				return 0, false, err
			}
			if !found || id > segId {
				segId = id
			}
			found = true
		}
	}
	return segId, found, nil
}

// gets the latest log sequence number from the existing segment file
func getLatestLSN(activeSeg segment) (uint64, error) {
	//open temporary second read only file descriptor and close it on function exit
	file, err := os.Open(activeSeg.file.Name())
	if err != nil {
		return 0, err
	}
	defer file.Close()
	offset := segmentHeaderByteSize
	// Jump the segment header
	_, err = file.Seek(int64(offset), io.SeekStart)
	if err != nil {
		return 0, err
	}
	var latestLSN uint64
	// buff to laod the recrod header
	buffer := make([]byte, int(recordHeaderByteSize))
	for {
		//load record header into ther buffer
		n, err := io.ReadFull(file, buffer)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				// Torn write or EOF reached
				break
			}
			return 0, err
		}
		if n != int(recordHeaderByteSize) {
			break
		}
		recordLength := binary.LittleEndian.Uint32(buffer[8:12])
		if recordLength < 36 {
			// torn write or corruption :
			// A valid record is atleast 36 bytes (recheader + crc)
			break
		}
		latestLSN = binary.LittleEndian.Uint64(buffer[16:24])
		// payloadSize + 4 crc bytes offset
		// recordlenth = total size or record wrapper (32B recheader + Variable (payload) + 4B CRC)
		newOffset := recordLength - 32
		//jump to next record header byte
		_, err = file.Seek(int64(newOffset), io.SeekCurrent)
		if err != nil {
			return 0, err
		}
	}
	return latestLSN, nil
}

func (wal *WAL) Append() error {

	return nil
}
