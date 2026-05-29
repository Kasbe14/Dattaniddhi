package wal

import "testing"

func FuzzDecodeRecordHeader(f *testing.F) {
	f.Add([]byte("random-data"))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = decodeRecordHeader(data)
	})
}

func FuzzDecodeInsertPayload(f *testing.F) {
	f.Add([]byte("payload"))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = decodeInsertPayload(data)
	})
}

func FuzzDecodeDeletePayload(f *testing.F) {
	f.Add([]byte("payload"))

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = decodeDeletePayload(data)
	})
}
