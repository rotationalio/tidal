package errors

import (
	"errors"
	"fmt"
)

var (
	// Database Errors
	ErrNotFound         = errors.New("record not found")
	ErrNotNull          = errors.New("cannot set a required field to null")
	ErrReadOnly         = errors.New("cannot perform operation in read-only mode")
	ErrAlreadyExists    = errors.New("record already exists (unique constraint violation)")
	ErrConstraint       = errors.New("database constraint violated")
	ErrMissingReference = errors.New("missing required reference to another record")
	ErrDeleteRestricted = errors.New("cannot delete record due to foreign key constraint")

	// Connection Errors
	ErrConnectionOptions = errors.New("could not get connection options")
	ErrConnect           = errors.New("could not connect to database")
	ErrPing              = errors.New("could not ping database")

	// Returned when [bind.Rewrite] is called with an unknown placeholder type.
	ErrUnsupportedPlaceholder = errors.New("unsupported placeholder type")

	// Model Errors
	ErrMissingID          = errors.New("missing ID")
	ErrInvalidIdentifier  = errors.New("invalid identifier for model or reference")
	ErrMissingAssociation = errors.New("associated records not cached on the model")
)

//============================================================================
// String Error Types
//============================================================================

// Returned when [conn.Open] is called with a DSN provider that tidal does not support.
type UnsupportedProvider string

func (e UnsupportedProvider) Error() string {
	return fmt.Sprintf("unsupported database provider: %q", string(e))
}

// MissingArgument is returned when a :name placeholder has no matching [sql.NamedArg].
type MissingArgument string

func (e MissingArgument) Error() string {
	return fmt.Sprintf("missing argument %q for query", string(e))
}

//============================================================================
// Wrapped Errors
//============================================================================

// Error wraps a database error with provider-specific information.
type Error struct {
	Provider string
	Err      error
	Code     any
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Provider, e.Err.Error())
}

func (e *Error) Unwrap() error {
	return e.Err
}

//============================================================================
// Package Parity
//============================================================================

var Join = errors.Join
var Is = errors.Is
