package collection

import (
	"encoding/json"
	"fmt"

	"github.com/Kasbe14/Dattaniddhi/internal/store/wal"
	"github.com/Kasbe14/Dattaniddhi/internal/vector"
)

// reconstruts existing collection from wal
func (c *Collection) LoadState() error {
	// Lock the collection completely while we rebuild its memory
	c.mu.Lock()
	defer c.mu.Unlock()

	c.idCounter = 0
	records, err := c.wal.Recover()
	if err != nil {
		return fmt.Errorf("wal failed to recover records: %w", err)
	}
	for _, record := range records {
		switch record.OpType {
		case wal.OpInsert:
			vector, err := vector.NewVector(record.Vector, c.config.Dimension)
			if err != nil {
				return fmt.Errorf("failed to create vector while loading collection %s: %w", c.config.Name, err)
			}

			added, err := c.index.Add(int(record.IntID), vector)
			if err != nil {
				return fmt.Errorf("recovery failed: %w", err)
			}
			if !added {
				return fmt.Errorf("recovery failed: internal id collision")
			}

			c.extToInt[record.ExtID] = int(record.IntID)
			c.intToExt[int(record.IntID)] = record.ExtID

			if int(record.IntID) > c.idCounter {
				c.idCounter = int(record.IntID)
			}

			var payloadData any
			// Only attempt to unmarshal if there is actual JSON data
			if len(record.MetaData) > 0 {
				err = json.Unmarshal(record.MetaData, &payloadData)
				if err != nil {
					return fmt.Errorf("recovery failed: corrupt payload data on ID %s", record.ExtID)
				}
			}
			c.payload[record.ExtID] = payloadData

		case wal.OpDelete:
			internalID := record.IntID
			extID := record.ExtID

			if int(record.IntID) > c.idCounter {
				c.idCounter = int(record.IntID)
			}

			err := c.index.Delete(int(internalID))
			if err != nil {
				return fmt.Errorf("FATAL: Index failed to delete %s, WAL out of sync: %v", extID, err)
			}

			delete(c.extToInt, extID)
			delete(c.intToExt, int(internalID))
			delete(c.payload, extID)
		}
	}
	return nil
}
