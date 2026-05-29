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
		return nil, fmt.Errorf("failed to get file info of segment %d: %w", segment.segID, err)
	}
	segmentFileSize := segmentFileInfo.Size()
	segmentHeaderBuf := make([]byte, segmentHeaderByteSize)
	//moving curosr to the start if file was already open or cursor was move
	_, err = segment.file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("failed to seek in the segment file: %w", err)
	}
	if _, err := io.ReadFull(segment.file, segmentHeaderBuf); err != nil {
		return nil, fmt.Errorf("failed to read segment header: %w", err)
	}
	_, err = decodeSegmentHeader(segmentHeaderBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to decode segment header in segment %d: %w", segment.segID, err)
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
			return nil, fmt.Errorf("failed to seek in segment file: %w", err)
		}
		recordHeaderBytes := make([]byte, recordHeaderByteSize)
		if _, err := io.ReadFull(segment.file, recordHeaderBytes); err != nil {
			return nil, fmt.Errorf("failed to read record header: %w", err)
		}
		decodedRecordHeader, err := decodeRecordHeader(recordHeaderBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to decode record header in segment %d: %w", segment.segID, err)
		}
		recordLenght := decodedRecordHeader.recordLength
		singleRecord.LSN = decodedRecordHeader.lsn
		singleRecord.OpType = decodedRecordHeader.opType
		// Payload and checksum Scanning and decoding
		if offset+int64(recordLenght) > segmentFileSize {
			// torn write
			break
		}
		//check minimum record length is 36 record header 32b + crc4b
		if recordLenght < recordHeaderByteSize+4 {
			return nil, fmt.Errorf("invalid record length %d", recordLenght)
		}

		recordBytes := make([]byte, recordLenght-recordHeaderByteSize)
		if _, err := io.ReadFull(segment.file, recordBytes); err != nil {
			return nil, fmt.Errorf("failed to read record bytes(payload and crc32): %w", err)
		}
		//Extracting payload bytes
		payloadBytes := recordBytes[:len(recordBytes)-4]
		//Verify the checksum
		storedCrc32 := binary.LittleEndian.Uint32(recordBytes[len(recordBytes)-4:])

		crcData := make([]byte, 0, len(recordHeaderBytes)+len(payloadBytes))
		crcData = append(crcData, recordHeaderBytes...)
		crcData = append(crcData, payloadBytes...)
		computedCrc32 := crc32.ChecksumIEEE(crcData)

		if storedCrc32 != computedCrc32 {
			return nil, fmt.Errorf("corupt data: invalid checksum")
		}
		//decoding payload bytes
		//var decodedPayloadBytes any
		switch decodedRecordHeader.opType {
		case OpInsert:
			decodedPayloadBytes, err := decodeInsertPayload(payloadBytes)
			if err != nil {
				return nil, fmt.Errorf("failed to decode insert payload in segment %d: %w", segment.segID, err)
			}
			singleRecord.ExtID = decodedPayloadBytes.externalID
			singleRecord.IntID = decodedPayloadBytes.internalID
			singleRecord.Vector = decodedPayloadBytes.vectorData
			singleRecord.MetaData = decodedPayloadBytes.metaData

		case OpDelete:
			decodedPayloadBytes, err := decodeDeletePayload(payloadBytes)
			if err != nil {
				return nil, fmt.Errorf("failed to decode delete payload in segment %d: %w", segment.segID, err)
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
