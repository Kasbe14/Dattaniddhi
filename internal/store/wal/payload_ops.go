package wal

import (
	"encoding/binary"
	"math"
)

// type Optype uint8

type payload interface {
	encode() []byte
	size() uint32
}

type insertPayload struct {
	externalID string
	internalID uint64
	vectorData []float32
	metaData   []byte
}

func (ip *insertPayload) encode() []byte {
	//2 -> maker; store len of external id [read this amount of next bytes for actual string data]
	//  len(ip.ExternalID) -> total number of bytes of string
	// 8 -> internalID
	// 4->marker; VectorDimension [read (4 * len(ip.VectorData)) amout of bytes for vector data]
	//(4 * len(ip.VectorData)) -> actual data
	// 4-> marker; amount of bytes in meta data
	// len(ip.Metadata)  bytes of metadata
	extIdLen := len(ip.externalID)
	vecDataLen := len(ip.vectorData)
	metaDataLen := len(ip.metaData)

	bufSize := 2 + extIdLen + 8 + 4 + (4 * vecDataLen) + 4 + metaDataLen
	buf := make([]byte, bufSize)
	offset := 0
	binary.LittleEndian.PutUint16(buf[offset:offset+2], uint16(extIdLen))
	offset += 2
	copy(buf[offset:], ip.externalID)
	offset += extIdLen
	binary.LittleEndian.PutUint64(buf[offset:offset+8], ip.internalID)
	offset += 8
	binary.LittleEndian.PutUint32(buf[offset:offset+4], uint32(vecDataLen))
	offset += 4
	//vector data
	for _, f := range ip.vectorData {
		bits := math.Float32bits(f) // convert floaat values to raw bits uint32
		binary.LittleEndian.PutUint32(buf[offset:offset+4], bits)
		offset += 4
	}

	binary.LittleEndian.PutUint32(buf[offset:offset+4], uint32(metaDataLen))
	offset += 4
	copy(buf[offset:], ip.metaData)

	return buf
}

type deletePayload struct {
	externalID string // 2-idlength
	internalID uint64
}

func (dp *deletePayload) encode() []byte {
	extIDLen := len(dp.externalID)
	bufSize := 2 + extIDLen + 8
	buf := make([]byte, bufSize)
	offset := 0
	binary.LittleEndian.PutUint16(buf[offset:offset+2], uint16(extIDLen))
	offset += 2
	copy(buf[offset:], dp.externalID)
	offset += extIDLen
	binary.LittleEndian.PutUint64(buf[offset:], dp.internalID)

	return buf
}

func (ip *insertPayload) size() uint32 {
	return uint32(2 + len(ip.externalID) + 8 + 4 + (4 * len(ip.vectorData)) + 4 + len(ip.metaData))
}
func (dp *deletePayload) size() uint32 {
	return uint32(2 + len(dp.externalID) + 8)
}

var _ payload = (*insertPayload)(nil)
var _ payload = (*deletePayload)(nil)
