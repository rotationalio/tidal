package model

import "errors"

var (
	// ErrMissingID is returned when a model is updated without an ID.
	ErrMissingID = errors.New("missing ID")
)
