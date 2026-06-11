package tidal

import (
	"errors"
	"fmt"
)

var (
	// Constraint errors

	ErrMissingID = errors.New("missing ID")
	ErrNotFound  = errors.New("not found")

	// Connection errors

	ErrUnsupportedPlaceholder = errors.New("unsupported placeholder type")
	ErrConnectionOptions      = errors.New("could not get connection options")
	ErrConnect                = errors.New("could not connect to database")
	ErrPing                   = errors.New("could not ping database")
)

// UnsupportedProvider is returned when [Open] is called with a DSN provider that tidal
// does not support.
type UnsupportedProvider string

func (e UnsupportedProvider) Error() string {
	return fmt.Sprintf("unsupported database provider: %q", string(e))
}

// MissingArgument is returned when a :name placeholder has no matching [sql.NamedArg].
type MissingArgument string

func (e MissingArgument) Error() string {
	return fmt.Sprintf("missing argument %q for query", string(e))
}
