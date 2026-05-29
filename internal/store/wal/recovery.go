package wal

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// returns []*segment{} for segments in directory dir in ascending order of segID
func (w *WAL) getAllSegment() ([]*segment, error) {
	var result []*segment
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read the WAL directory %s: %w", w.dir, err)
	}
	for _, entry := range entries {
		var id uint64
		fileName := entry.Name()
		filePath := filepath.Join(w.dir, fileName)
		if !entry.IsDir() && strings.HasSuffix(fileName, ".waldrky") {
			numStr := strings.TrimSuffix(fileName, ".waldrky")
			id, err = strconv.ParseUint(numStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse the segment id from file name %s: %w", fileName, err)
			}
			fileInfo, err := entry.Info()
			if err != nil {
				return nil, fmt.Errorf("failed to get file info for file %s: %w", fileName, err)
			}
			//segmentFile, err := os.OpenFile(filePath, os.O_RDONLY, 0444)
			segment := &segment{segID: id, path: filePath, currentSize: uint64(fileInfo.Size())}
			result = append(result, segment)
		}
	}
	slices.SortFunc(result, func(a, b *segment) int {
		return cmp.Compare(a.segID, b.segID)
	})
	return result, nil
}

func (w *WAL) Recover() ([]DecodedRecords, error) {
	segments, err := w.getAllSegment()
	if err != nil {
		return nil, fmt.Errorf("failed to get segments from WAL directory: %w", err)
	}
	var records []DecodedRecords
	for _, segment := range segments {
		file, err := os.OpenFile(segment.path, os.O_RDONLY, 0444)
		if err != nil {
			return nil, fmt.Errorf("failed to open the segment file %d: %w", segment.segID, err)
		}
		segment.file = file
		segmentRecord, err := scanSegmentFile(segment)
		segment.file.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to scan the segment file %d: %w", segment.segID, err)
		}
		records = append(records, segmentRecord...)
	}
	return records, nil
}
