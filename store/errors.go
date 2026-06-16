package store

import "errors"

var (
	// ErrNotFound is returned when an update or delete targets a row that does not exist.
	ErrNotFound = errors.New("not found")
)
