//go:build mattn && !ncruces

package errors_test

import (
	"testing"

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

	suite.Run(t, s)
}
