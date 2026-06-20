//go:build ncruces && !mattn

package errors

import (
	"database/sql"
	"errors"

	"github.com/ncruces/go-sqlite3"
)

// Implements the sqlite3-specific error interface for converting sqlite3 errors into
// tidal errors while still preserving the original, underlying error. This method uses
// the ncruces sqlite3 driver. To use use this driver, you need to use the ncruces build
// tag, which overrides the default modernc driver.
func SQLiteError(err error) error {
	if err == nil {
		return nil
	}

	e := &Error{
		Provider: "sqlite3+ncruces",
	}

	if errors.Is(err, sql.ErrNoRows) {
		e.Err = errors.Join(ErrNotFound, err)
		return e
	}

	if sqliteErr, ok := errors.AsType[*sqlite3.Error](err); ok {
		e.Code = sqliteErr.Code()
		switch e.Code {
		case sqlite3.READONLY:
			e.Err = errors.Join(ErrReadOnly, err)
		case sqlite3.CONSTRAINT:
			switch sqliteErr.ExtendedCode() {
			case sqlite3.CONSTRAINT_UNIQUE:
				e.Err = errors.Join(ErrAlreadyExists, err)
			case sqlite3.CONSTRAINT_FOREIGNKEY:
				e.Err = errors.Join(ErrMissingReference, err)
			case sqlite3.CONSTRAINT_NOTNULL:
				e.Err = errors.Join(ErrNotNull, err)
			case sqlite3.CONSTRAINT_PRIMARYKEY:
				e.Err = errors.Join(ErrInvalidIdentifier, err)
			default:
				e.Err = errors.Join(ErrConstraint, err)
			}
		}
	}

	if e.Err == nil {
		e.Err = err
	}
	return e
}
