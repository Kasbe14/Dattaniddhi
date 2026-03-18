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
	//max segment file size 64mb
	maxSegmentFileSize uint64 = 64 * 1024 * 1024
)

// Orchestrator for the segment files and api for collections, Own by the database layer (not implemented yet)
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

func (wal *WAL) AppendInsert(extID string, intID uint64, vecData []float32, metaData []byte) (uint64, error) {
	// before lock (concurrent) creating and encooding the payload
	//creating payload
	pl := &insertPayload{
		externalID: extID,
		internalID: intID,
		vectorData: vecData,
		metaData:   metaData,
	}
	//encode payload
	payloadBytes := pl.encode()
	// payloadSize := pl.size()
	//locking for go routines writes
	wal.mu.Lock()
	defer wal.mu.Unlock()
	//update the lsn
	wal.lsn++
	currentLSN := wal.lsn
	//create and enocde the recordWrapper to write to segment
	rh := newRecordHeader(walVersion, currentLSN, OpInsert)
	rw := newRecordWrapper(*rh, payloadBytes)
	recordWrapperBytes := rw.encode()

	//roll over policy if segment file >= 64mb
	segmentFileSize := uint64(len(recordWrapperBytes)) + wal.activeSegment.currentSize
	if segmentFileSize > maxSegmentFileSize {
		//New segment file with new SegId and SegmentHeader written
		err := wal.rotateSegment()
		if err != nil {
			return 0, err
		}
	}
	//append to the segment file (new segment file if rotated or same)
	wal.activeSegment.append(recordWrapperBytes)
	// write bytes to disk according to sync policy
	switch wal.syncPolicy {
	case SyncAlways:
		err := wal.activeSegment.file.Sync()
		if err != nil {
			return 0, err
		}
	case SyncOS:
		// do nothing os handles the sync
	case SyncEverySec:
		//do nothing TODO: separate background functon to handle this sync
	}
	return currentLSN, nil
}

func (wal *WAL) AppendDelete(extID string, intID uint64) (uint64, error) {
	//create and enocode payload before lock (concurrent)
	pl := &deletePayload{
		externalID: extID,
		internalID: intID,
	}
	payloadBytes := pl.encode()
	//lock to write sequntially
	wal.mu.Lock()
	defer wal.mu.Unlock()
	//update lsn and append payload
	wal.lsn++
	currentLSN := wal.lsn
	rh := newRecordHeader(walVersion, wal.lsn, OpDelete)
	rw := newRecordWrapper(*rh, payloadBytes)
	recordWrapperBytes := rw.encode()

	//roll over policy if segment file >= 64mb
	segmentFileSize := uint64(len(recordWrapperBytes)) + wal.activeSegment.currentSize
	if segmentFileSize > maxSegmentFileSize {
		err := wal.rotateSegment()
		if err != nil {
			return 0, err
		}
	}
	wal.activeSegment.append(recordWrapperBytes)
	// write bytes to disk
	switch wal.syncPolicy {
	case SyncAlways:
		err := wal.activeSegment.file.Sync()
		if err != nil {
			return 0, err
		}
	case SyncOS:
		//do nothing OS handles
	case SyncEverySec:
		//Todo backgorund go rouine to handle sync every sec

	}
	return currentLSN, nil
}

// Create  new active segment file with new segID and writes SegmentHeader
func (wal *WAL) rotateSegment() error {
	// change current semgnent file to readOnly
	wal.activeSegment.file.Chmod(0444)
	// Close current segment file
	err := wal.activeSegment.file.Close()
	if err != nil {
		return err
	}
	wal.segID++
	//create new segment
	newSegment, err := createSegment(wal.dir, wal.segID)
	if err != nil {
		return err
	}
	//write segment header
	segHeader := newSegmentHeader(walVersion, wal.segID)
	segHeaderBytes := encodeSegmentHeader(*segHeader)
	//append the header bytes and incrrement the current size to bytes written
	_, err = newSegment.append(segHeaderBytes)
	if err != nil {
		return err
	}
	wal.activeSegment = newSegment
	return nil
}

// This implements the standard io.Closer interface.
// go garbage collector doesn't clean OS resources so implement close for structures that hold OS resources
func (wal *WAL) Close() error {
	wal.mu.Lock()
	defer wal.mu.Unlock()

	if wal.activeSegment != nil && wal.activeSegment.file != nil {
		// Sync to disk before closing to ensure no data is lost
		wal.activeSegment.file.Sync()
		return wal.activeSegment.file.Close()
	}
	return nil
}

// TODO : update operation uisng delete and insert 2 operations for now later crate and update payload
