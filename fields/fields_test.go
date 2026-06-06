package fields_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
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
	if _, err := s.ResolveDSN(""); err != nil {
		t.Skipf("skipping fields sqlite tests because of DSN resolution error: %v", err)
	}

	// Run the tests
	suite.Run(t, s)
}

func (s *FieldsSqliteTestSuite) CreateDB() {
	// Create a new SQLite3 database with the test data schema.
	s.SQLiteSuite.CreateDB("")

	// Execute the test data schema.
	require := s.Require()
	schema := readSQL(require, "sqlite_schema.sql")

	_, err := s.DB.Exec(schema)
	require.NoError(err, "could not execute schema")
}

func (s *FieldsSqliteTestSuite) ResetDB() {
	if s.DB != nil {
		require := s.Require()
		require.NoError(s.DB.Close(), "could not close database")
		require.NoError(os.Remove(s.DSN().Path), "could not delete database file")
		s.CreateDB()
	} else {
		s.T().Log("cannot reset the fields sqlite database because s.DB is nil")
	}
}

func (s *FieldsSqliteTestSuite) SetupSuite() {
	s.CreateDB()
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
	if _, err := s.ResolveDSN(""); err != nil {
		t.Skipf("skipping fields postgres tests because of DSN resolution error: %v", err)
	}

	// Run the tests
	suite.Run(t, s)
}

func (s *FieldsPostgresTestSuite) CreateDB() {
	s.T().Log("creating the fields postgres database with postgres_schema.sql")

	// Create a new SQLite3 database with the test data schema.
	s.PostgresSuite.CreateDB("")

	// Execute the test data schema.
	require := s.Require()
	schema := readSQL(require, "postgres_schema.sql")

	_, err := s.DB.Exec(schema)
	require.NoError(err, "could not execute schema")
}

func (s *FieldsPostgresTestSuite) SetupSuite() {
	s.T().Log("setting up the fields postgres test suite")
	s.CreateDB()
}

func (s *FieldsPostgresTestSuite) AfterTest(suiteName, testName string) {
	s.T().Logf("resetting the fields postgres test suite after test %s.%s", suiteName, testName)
	s.ResetDB()
}

//============================================================================
// Helper Functions
//============================================================================

func readSQL(r *require.Assertions, filename string) string {
	path := filepath.Join("testdata", filename)
	f, err := os.Open(path)
	r.NoError(err, "could not open file %s", path)
	defer f.Close()

	content, err := io.ReadAll(f)
	r.NoError(err, "could not read file %s", path)
	return string(content)
}
