package bind

import (
	"errors"
	"fmt"
)

var (
	// ErrUnsupportedPlaceholder is returned when [Rewrite] is called with an unknown placeholder type.
	ErrUnsupportedPlaceholder = errors.New("unsupported placeholder type")
)

// MissingArgument is returned when a :name placeholder has no matching [sql.NamedArg].
type MissingArgument string

func (e MissingArgument) Error() string {
	return fmt.Sprintf("missing argument %q for query", string(e))
}
