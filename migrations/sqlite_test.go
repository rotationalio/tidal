package migrations_test

import (
	"context"
	"embed"
	"testing"

	"go.rtnl.ai/tidal/suite"

	. "go.rtnl.ai/tidal/migrations"
)

//go:embed testdata/sqlite
var sqliteFS embed.FS

type SQLiteTestSuite struct {
	suite.SQLiteSuite
}

func TestSQLite(t *testing.T) {
	// Run the tests
	suite.Run(t, &SQLiteTestSuite{})
}

func (s *SQLiteTestSuite) TestMigrations() {
	require := s.Require()
	migrations, err := Load(sqliteFS)
	require.NoError(err)
	require.NotNil(migrations)

	err = migrations.ApplySQLite(context.Background(), s.DB, "v1.0.0")
	require.NoError(err)

	last, err := LastApplied(context.Background(), s.DB)
	require.NoError(err)
	require.NotNil(last)
	require.Equal(3, last.ID)
	require.Equal("Post Meta", last.Name)
	require.Equal("v1.0.0", last.Version)
	require.NotNil(last.Applied)
}

func (s *SQLiteTestSuite) TestApplyOrdered() {
	require := s.Require()
	migrations, err := Load(sqliteFS)
	require.NoError(err)
	require.NotNil(migrations)

	alpha := migrations[0:1]
	bravo := migrations[1:2]
	charlie := migrations[2:]

	require.Len(alpha, 1, "expected 1 migration")
	err = alpha.ApplySQLite(context.Background(), s.DB, "v1.0.0")
	require.NoError(err)

	last, err := LastApplied(context.Background(), s.DB)
	require.NoError(err)
	require.Equal(1, last.ID)
	require.Equal("v1.0.0", last.Version)

	require.Len(bravo, 1, "expected 1 migration")
	err = bravo.ApplySQLite(context.Background(), s.DB, "v1.1.0")
	require.NoError(err)

	last, err = LastApplied(context.Background(), s.DB)
	require.NoError(err)
	require.Equal(2, last.ID)
	require.Equal("v1.1.0", last.Version)

	require.Len(charlie, 1, "expected 1 migration")
	err = charlie.ApplySQLite(context.Background(), s.DB, "v1.2.0")
	require.NoError(err)

	last, err = LastApplied(context.Background(), s.DB)
	require.NoError(err)
	require.Equal(3, last.ID)
	require.Equal("v1.2.0", last.Version)
}
