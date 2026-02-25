package wal

import (
	"fmt"
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
		// TODO : helper function getLatestLSN() to get scan the segment for record and get lates lsn
		lsn, err = getLatestLSN(*activeSeg)
		// if err != nil {
		// 	return nil, err
		// }
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

func getLatestLSN(seg segment) (uint64, error) {

	return 0, nil
}

func (wal *WAL) Append() error {

	return nil
}
