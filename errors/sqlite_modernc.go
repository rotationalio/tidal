//go:build !mattn && !ncruces

package errors

import (
	"database/sql"
	"errors"
	"strings"

	modernc "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// Implements the sqlite3-specific error interface for converting sqlite3 errors into
// tidal errors while still preserving the original, underlying error. This method uses
// the default modernc sqlite3 driver. To use either the mattn or ncruces drivers,
// ensure you use the appropriate build tag.
func SQLiteError(err error) error {
	if err == nil {
		return nil
	}

	e := &Error{
		Provider: "sqlite3+modernc",
	}

	if Is(err, sql.ErrNoRows) {
		e.Err = errors.Join(ErrNotFound, err)
		return e
	}

	if sqliteErr, ok := err.(*modernc.Error); ok {
		e.Code = sqliteErr.Code()
		switch e.Code {
		case sqlite3.SQLITE_READONLY:
			e.Err = errors.Join(ErrReadOnly, err)
		case sqlite3.SQLITE_CONSTRAINT_UNIQUE:
			e.Err = errors.Join(ErrAlreadyExists, err)
		case sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY:
			e.Err = errors.Join(ErrMissingReference, err)
		case sqlite3.SQLITE_CONSTRAINT_NOTNULL:
			e.Err = errors.Join(ErrNotNull, err)
		case sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY:
			e.Err = errors.Join(ErrInvalidIdentifier, err)
		case sqlite3.SQLITE_CONSTRAINT_CHECK:
			e.Err = errors.Join(ErrConstraint, err)
		case sqlite3.SQLITE_CONSTRAINT, sqlite3.SQLITE_CONSTRAINT_TRIGGER:
			if strings.Contains(err.Error(), "FOREIGN KEY") {
				e.Err = errors.Join(ErrDeleteRestricted, err)
			} else {
				e.Err = errors.Join(ErrConstraint, err)
			}
		}
	}

	// If the error is still not set, just wrap it directly.
	if e.Err == nil {
		e.Err = err
	}
	return e
}
