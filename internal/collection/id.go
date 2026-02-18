package ingest

type IDGenerator interface {
	NewID() string
}

type UUIDv7Generator struct{}

func (uid UUIDv7Generator) NewID() string {
	return "id"
}
