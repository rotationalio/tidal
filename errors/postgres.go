package errors

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// Implements the postgres-specific error interface for converting postgres errors into
// tidal errors while still preserving the original, underlying error.
func PostgresError(err error) error {
	if err == nil {
		return nil
	}

	e := &Error{
		Provider: "postgres",
	}

	if errors.Is(err, sql.ErrNoRows) {
		e.Err = errors.Join(ErrNotFound, err)
		return e
	}

	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		e.Code = pgErr.Code
		switch pgErr.Code {
		case "23505":
			e.Err = errors.Join(ErrAlreadyExists, err)
		case "23503":
			e.Err = errors.Join(ErrMissingReference, err)
		case "23502":
			e.Err = errors.Join(ErrNotNull, err)
		case "23000", "23514":
			return errors.Join(ErrConstraint, err)
		case "23001":
			e.Err = errors.Join(ErrDeleteRestricted, err)
		case "25006":
			e.Err = errors.Join(ErrReadOnly, err)
		}
	}

	// If the error is still not set, just wrap it directly.
	if e.Err == nil {
		e.Err = err
	}
	return e
}
