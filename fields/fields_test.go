package fields_test

import (
	"testing"

	"go.rtnl.ai/tidal/suite"
	"go.rtnl.ai/tidal/suite/fixtures"
)

//============================================================================
// SQLite3 Tests
//============================================================================

type FieldsSqliteTestSuite struct {
	suite.SQLiteSuite
}

func TestSQLiteFields(t *testing.T) {
	s := &FieldsSqliteTestSuite{}
	s.Migrations = fixtures.File("fields/sqlite_schema.sql")

	// Run the tests
	suite.Run(t, s)
}

//============================================================================
// PostgreSQL Tests
//============================================================================

type FieldsPostgresTestSuite struct {
	suite.PostgresSuite
}

func TestPostgresFields(t *testing.T) {
	s := &FieldsPostgresTestSuite{}
	s.Migrations = fixtures.File("fields/postgres_schema.sql")

	// Run the tests
	suite.Run(t, s)
}

