//go:build !mattn && !ncruces

package errors_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	. "go.rtnl.ai/tidal/errors"
	"go.rtnl.ai/tidal/suite"
	"go.rtnl.ai/tidal/suite/fixtures"
)

func TestSQLite(t *testing.T) {
	s := &ErrorsTests{}
	s.DatabaseError = SQLiteError
	s.Provider = &suite.SQLiteProvider{}
	s.Teardown = suite.TeardownNone
	s.Migrations = fixtures.File("errors/sqlite_schema.sql")

	// NOTE: modernc needs the PRAGMA to be specified on every connection.
	// To ensure this happens, you need to pass the pragma on the connection string.

	suite.Run(t, s)
}

func TestSQLiteError(t *testing.T) {
	err := errors.New("testing error")
	dberr := SQLiteError(err)

	require.EqualError(t, dberr, "sqlite3+modernc: testing error")

	e, ok := errors.AsType[*Error](dberr)
	require.True(t, ok)
	require.Equal(t, "sqlite3+modernc", e.Provider)
	require.ErrorIs(t, e.Err, err)
}
