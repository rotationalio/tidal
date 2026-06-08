package migrations_test

import (
	"context"
	"embed"
	"testing"

	"go.rtnl.ai/tidal/suite"

	. "go.rtnl.ai/tidal/migrations"
)

type PostgresTestSuite struct {
	suite.PostgresSuite
}

//go:embed testdata/postgres
var postgresFS embed.FS

func TestPostgres(t *testing.T) {
	s := &PostgresTestSuite{}
	if _, err := s.ResolveDSN(""); err != nil {
		t.Skipf("skipping fields postgres tests because of DSN resolution error: %v", err)
	}

	// Run the tests
	suite.Run(t, s)
}

const dropTableQuery = `
DO $$
DECLARE
	l_stmt TEXT;
BEGIN
	SELECT 'DROP TABLE IF EXISTS ' || string_agg(format('%I.%I', schemaname, tablename), ', ') || ' CASCADE'
	INTO l_stmt
	FROM pg_tables
	WHERE schemaname = 'public';

	IF l_stmt IS NOT NULL THEN
		EXECUTE l_stmt;
	END IF;
END $$;
`

func (s *PostgresTestSuite) ResetDB() {
	if s.DB != nil {
		require := s.Require()
		_, err := s.DB.Exec(dropTableQuery)
		require.NoError(err, "could not drop tables")
	} else {
		s.T().Log("cannot reset the postgres database because s.DB is nil")
	}
}

func (s *PostgresTestSuite) AfterTest(suiteName, testName string) {
	s.T().Logf("resetting the postgres test suite after test %s.%s", suiteName, testName)
	s.ResetDB()
}

func (s *PostgresTestSuite) TestMigrations() {
	require := s.Require()
	migrations, err := Load(postgresFS)
	require.NoError(err)
	require.NotNil(migrations)

	err = migrations.ApplyPostgres(context.Background(), s.DB, "v1.0.0")
	require.NoError(err)

	last, err := LastApplied(context.Background(), s.DB)
	require.NoError(err)
	require.NotNil(last)
	require.Equal(3, last.ID)
	require.Equal("Post Meta", last.Name)
	require.Equal("v1.0.0", last.Version)
	require.NotNil(last.Applied)
}

func (s *PostgresTestSuite) TestApplyOrdered() {
	require := s.Require()
	migrations, err := Load(postgresFS)
	require.NoError(err)
	require.NotNil(migrations)

	alpha := migrations[0:1]
	bravo := migrations[1:2]
	charlie := migrations[2:]

	require.Len(alpha, 1, "expected 1 migration")
	err = alpha.ApplyPostgres(context.Background(), s.DB, "v1.0.0")
	require.NoError(err)

	last, err := LastApplied(context.Background(), s.DB)
	require.NoError(err)
	require.Equal(1, last.ID)
	require.Equal("v1.0.0", last.Version)

	require.Len(bravo, 1, "expected 1 migration")
	err = bravo.ApplyPostgres(context.Background(), s.DB, "v1.1.0")
	require.NoError(err)

	last, err = LastApplied(context.Background(), s.DB)
	require.NoError(err)
	require.Equal(2, last.ID)
	require.Equal("v1.1.0", last.Version)

	require.Len(charlie, 1, "expected 1 migration")
	err = charlie.ApplyPostgres(context.Background(), s.DB, "v1.2.0")
	require.NoError(err)

	last, err = LastApplied(context.Background(), s.DB)
	require.NoError(err)
	require.Equal(3, last.ID)
	require.Equal("v1.2.0", last.Version)
}
