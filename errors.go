package tidal

import "errors"

var (
	ErrMissingID = errors.New("missing ID")
	ErrNotFound  = errors.New("not found")
)
