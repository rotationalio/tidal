package conn

import (
	"errors"
	"fmt"
)

var (
	// Connection errors

	ErrConnectionOptions = errors.New("could not get connection options")
	ErrConnect           = errors.New("could not connect to database")
	ErrPing              = errors.New("could not ping database")
	ErrReadOnly          = errors.New("cannot perform operation in read-only mode")
)

// UnsupportedProvider is returned when [Open] is called with a DSN provider that tidal
// does not support.
type UnsupportedProvider string

func (e UnsupportedProvider) Error() string {
	return fmt.Sprintf("unsupported database provider: %q", string(e))
}
