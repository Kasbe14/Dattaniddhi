package index

import (
	v "github.com/Kasbe14/Dattaniddhi/internal/vector"
)

type VectorIndex interface {
	Add(id int, v *v.Vector) (bool, error)
	// Delete removes an internal ID from the index.
	//CRITICAL CONTRACT FOR WAL RECOVERY (IDEMPOTENCY):
	// This method MUST be naturally idempotent regarding missing or already deleted IDs.
	// - If the ID does not exist in the index, it must do nothing and return a NIL error.
	// - Why: During crash recovery, the WAL might replay a delete operation for an item
	//   that was already cleared from RAM before the crash.
	// IMPLEMENTATION NOTES:
	// - LinearIndex: Uses Go's native map delete(), which is naturally a safe no-op if the
	//   key is missing, meaning it will always return nil.
	// - Future Complex Indexes (e.g., HNSW): If an ID is missing, swallow the condition
	//   and return nil. A non-nil error must ONLY be returned if the underlying structure
	//   is physically corrupted (e.g., a broken graph pointer or memory panic).
	Delete(id int) error
	Get(id int) (*v.Vector, bool)
	Search(query *v.Vector, k int) ([]SearchResult, error)
	Size() int
}
