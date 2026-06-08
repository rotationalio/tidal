package tidal_test

import (
	"context"
	"embed"
	"os"
	"testing"

	"go.rtnl.ai/tidal/migrations"
	"go.rtnl.ai/tidal/suite"
)

func TestPostgres(t *testing.T) {
	s := &PostgresTestSuite{}
	if _, err := s.ResolveDSN(""); err != nil {
		t.Skipf("skipping postgres tests because of DSN resolution error: %v", err)
	}

	// Run the tests
	suite.Run(t, s)
}

func TestSQLite(t *testing.T) {
	s := &SQLiteTestSuite{}
	if _, err := s.ResolveDSN(""); err != nil {
		t.Skipf("skipping sqlite tests because of DSN resolution error: %v", err)
	}

	// Run the tests
	suite.Run(t, s)
}

//go:embed testdata/postgres
var postgresFS embed.FS

type PostgresTestSuite struct {
	suite.PostgresSuite
}

func (s *PostgresTestSuite) CreateDB() {
	s.PostgresSuite.CreateDB("")

	require := s.Require()
	migrations, err := migrations.Load(postgresFS)
	require.NoError(err)

	err = migrations.ApplyPostgres(context.Background(), s.DB, "test")
	require.NoError(err)
}

func (s *PostgresTestSuite) SetupSuite() {
	s.CreateDB()
}

//go:embed testdata/sqlite
var sqliteFS embed.FS

type SQLiteTestSuite struct {
	suite.SQLiteSuite
}

func (s *SQLiteTestSuite) CreateDB() {
	s.SQLiteSuite.CreateDB("")

	require := s.Require()
	migrations, err := migrations.Load(sqliteFS)
	require.NoError(err)

	err = migrations.ApplySQLite(context.Background(), s.DB, "test")
	require.NoError(err)
}

func (s *SQLiteTestSuite) ResetDB() {
	if s.DB != nil {
		require := s.Require()
		require.NoError(s.DB.Close(), "could not close database")
		require.NoError(os.Remove(s.DSN().Path), "could not delete database file")
		s.CreateDB()
	} else {
		s.T().Log("cannot reset the sqlite database because s.DB is nil")
	}
}

func (s *SQLiteTestSuite) SetupSuite() {
	s.CreateDB()
}

func (s *SQLiteTestSuite) AfterTest(suiteName, testName string) {
	s.ResetDB()
}
