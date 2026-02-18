package collection

import "errors"

var (
	ErrInvalidDimension      = errors.New("invalid vector dimension")
	ErrInvalidMetric         = errors.New("invalid similarity metric")
	ErrDuplicateID           = errors.New("duplicate id")
	ErrNotFound              = errors.New("not found")
	ErrInvalidIndexType      = errors.New("invalid index type")
	ErrInvalidDataType       = errors.New("invalid data type")
	ErrInvalidModelName      = errors.New("invalid model name type")
	ErrInternalIDCollision   = errors.New("internal id collision")
	ErrInvalidCollectionName = errors.New("invalid collection name")
)
