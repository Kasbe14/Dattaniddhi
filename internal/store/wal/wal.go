package wal

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
)
