//go:build mattn && !ncruces

package errors

import (
	"database/sql"
	"errors"

	"github.com/mattn/go-sqlite3"
)

// Implements the sqlite3-specific error interface for converting sqlite3 errors into
// tidal errors while still preserving the original, underlying error. This method uses
// the mattn sqlite3 driver. To use use this driver, you need to use the mattn build
// tag, which overrides the default modernc driver.
func SQLiteError(err error) error {
	if err == nil {
		return nil
	}

	e := &Error{
		Provider: "sqlite3+mattn",
	}

	if errors.Is(err, sql.ErrNoRows) {
		e.Err = errors.Join(ErrNotFound, err)
		return e
	}

	if sqliteErr, ok := errors.AsType[*sqlite3.Error](err); ok {
		switch sqliteErr.Code {
		case sqlite3.ErrReadonly:
			e.Err = errors.Join(ErrReadOnly, err)
			e.Code = sqliteErr.Code
		case sqlite3.ErrConstraint:
			e.Code = sqliteErr.ExtendedCode
			switch sqliteErr.ExtendedCode {
			case sqlite3.ErrConstraintUnique:
				e.Err = errors.Join(ErrAlreadyExists, err)
			case sqlite3.ErrConstraintForeignKey:
				e.Err = errors.Join(ErrMissingReference, err)
			case sqlite3.ErrConstraintNotNull:
				e.Err = errors.Join(ErrNotNull, err)
			case sqlite3.ErrConstraintPrimaryKey:
				e.Err = errors.Join(ErrInvalidIdentifier, err)
			default:
				e.Err = errors.Join(ErrConstraint, err)
			}
		default:
			e.Code = sqliteErr.Code
			e.Err = err
		}
	}

	if e.Err == nil {
		e.Err = err
	}
	return e
}
