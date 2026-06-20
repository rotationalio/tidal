//go:build ncruces && !mattn

package errors

import (
	"database/sql"
	"errors"
	"strings"

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

	switch {
	case errors.Is(err, sql.ErrNoRows):
		e.Err = errors.Join(ErrNotFound, err)
	case errors.Is(err, sqlite3.READONLY):
		e.Err = errors.Join(ErrReadOnly, err)
	case errors.Is(err, sqlite3.CONSTRAINT_UNIQUE):
		e.Err = errors.Join(ErrAlreadyExists, err)

	// NOTE: this switch appears not to work with the current version of ncruces; even
	// though it is implemented in the ncruces gorm extender; see:
	// https://github.com/ncruces/go-sqlite3/blob/3fdcb0a066c12a1c67612dc1be2245eb5494fa5c/gormlite/error_translator.go#L19
	case errors.Is(err, sqlite3.CONSTRAINT_FOREIGNKEY):
		e.Err = errors.Join(ErrMissingReference, err)
	case errors.Is(err, sqlite3.CONSTRAINT_NOTNULL):
		e.Err = errors.Join(ErrNotNull, err)
	case errors.Is(err, sqlite3.CONSTRAINT_PRIMARYKEY):
		e.Err = errors.Join(ErrInvalidIdentifier, err)
	case errors.Is(err, sqlite3.CONSTRAINT):
		if strings.Contains(err.Error(), "FOREIGN KEY") {
			e.Err = errors.Join(ErrDeleteRestricted, err)
		} else {
			e.Err = errors.Join(ErrConstraint, err)
		}
	}

	if sqliteErr, ok := errors.AsType[*sqlite3.Error](err); ok {
		e.Code = sqliteErr.Code()
	} else if sqliteErr, ok := errors.AsType[sqlite3.ExtendedErrorCode](err); ok {
		e.Code = sqliteErr
	} else if sqliteErr, ok := errors.AsType[sqlite3.ErrorCode](err); ok {
		e.Code = sqliteErr
	}

	if e.Err == nil {
		e.Err = err
	}
	return e
}
