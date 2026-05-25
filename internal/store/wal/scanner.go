package wal

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
)

// data required for recovery
type DecodedRecords struct {
	LSN      uint64
	OpType   uint8
	ExtID    string
	IntID    uint64
	Vector   []float32 // Only populated for Inserts
	MetaData []byte    // Only populated for Inserts
}

// scans all the segment files validates and returns the records written to the segment file
func scanSegmentFile(segment *segment) ([]DecodedRecords, error) {
	var records []DecodedRecords
	segmentFileInfo, err := segment.file.Stat()
	if err != nil {
		return nil, err
	}
	segmentFileSize := segmentFileInfo.Size()
	segmentHeaderBuf := make([]byte, segmentHeaderByteSize)
	if _, err := io.ReadFull(segment.file, segmentHeaderBuf); err != nil {
		return nil, err
	}
	_, err = decodeSegmentHeader(segmentHeaderBuf)
	if err != nil {
		return nil, err
	}
	//skipping the segment header and offset start at 16 byte
	var offset int64
	offset = int64(segmentHeaderByteSize)
	//Scanner loop
	for offset < (segmentFileSize) {
		var singleRecord DecodedRecords
		// RecordHeader Scanning and decoding
		if offset+32 > (segmentFileSize) {
			//torn write
			break
		}
		offset, err = segment.file.Seek(offset, io.SeekStart)
		if err != nil {
			return nil, err
		}
		recordHeaderBytes := make([]byte, recordHeaderByteSize)
		if _, err := io.ReadFull(segment.file, recordHeaderBytes); err != nil {
			return nil, err
		}
		decodedRecordHeader, err := decodeRecordHeader(recordHeaderBytes)
		if err != nil {
			return nil, err
		}
		recordLenght := decodedRecordHeader.recordLength
		singleRecord.LSN = decodedRecordHeader.lsn
		singleRecord.OpType = decodedRecordHeader.opType
		// Payload and checksum Scanning and decoding
		if offset+int64(recordLenght) > segmentFileSize {
			// torn write
			break
		}

		if err != nil {
			return nil, err
		}
		recordBytes := make([]byte, recordLenght-recordHeaderByteSize)
		if _, err := io.ReadFull(segment.file, recordBytes); err != nil {
			return nil, err
		}
		//Extracting payload bytes
		payloadBytes := recordBytes[:len(recordBytes)-4]
		//Verify the checksum
		storedCrc32 := binary.LittleEndian.Uint32(recordBytes[len(recordBytes)-4:])
		computedCrc32 := crc32.ChecksumIEEE(append(recordHeaderBytes, payloadBytes...))
		if storedCrc32 != computedCrc32 {
			return nil, fmt.Errorf("corupt data : invalid checksum")
		}
		//decoding payload bytes
		//var decodedPayloadBytes any
		switch decodedRecordHeader.opType {
		case OpInsert:
			decodedPayloadBytes, err := decodeInsertPayload(payloadBytes)
			if err != nil {
				return nil, err
			}
			singleRecord.ExtID = decodedPayloadBytes.externalID
			singleRecord.IntID = decodedPayloadBytes.internalID
			singleRecord.Vector = decodedPayloadBytes.vectorData
			singleRecord.MetaData = decodedPayloadBytes.metaData

		case OpDelete:
			decodedPayloadBytes, err := decodeDeletePayload(payloadBytes)
			if err != nil {
				return nil, err
			}
			singleRecord.ExtID = decodedPayloadBytes.externalID
			singleRecord.IntID = decodedPayloadBytes.internalID
			//TODO : update operation ,format, payload , deocode
			/*case OpDelete:
			decodedPayloadBytes, err = decodeUpdatePayload(payloadBytes)*/
		default:
			return nil, fmt.Errorf("invalid Operation type")
		}
		records = append(records, singleRecord)
		offset += int64(recordLenght)
	}
	return records, nil
}
