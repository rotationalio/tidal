package fields_test

import (
	"testing"

	"go.rtnl.ai/tidal/fixtures"
	"go.rtnl.ai/tidal/suite"
)

//============================================================================
// SQLite3 Tests
//============================================================================

type FieldsSqliteTestSuite struct {
	suite.SQLiteSuite
}

func TestSQLiteFields(t *testing.T) {
	s := &FieldsSqliteTestSuite{}
	s.Migrations = fixtures.Fixture("testdata/sqlite_schema.sql")

	// Run the tests
	suite.Run(t, s)
}

func (s *FieldsSqliteTestSuite) AfterTest(suiteName, testName string) {
	s.ResetDB()
}

//============================================================================
// PostgreSQL Tests
//============================================================================

type FieldsPostgresTestSuite struct {
	suite.PostgresSuite
}

func TestPostgresFields(t *testing.T) {
	s := &FieldsPostgresTestSuite{}
	s.Migrations = fixtures.Fixture("testdata/postgres_schema.sql")

	// Run the tests
	suite.Run(t, s)
}

func (s *FieldsPostgresTestSuite) AfterTest(suiteName, testName string) {
	s.ResetDB()
}
