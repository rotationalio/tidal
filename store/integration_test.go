package store_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal/migrations"
	"go.rtnl.ai/tidal/suite"
)

type StoreTestSuite struct {
	suite.DatabaseSuite
}

//============================================================================
// Postgres Tests
//============================================================================

func TestPostgres(t *testing.T) {
	var err error
	s := &StoreTestSuite{}
	s.Provider = &suite.PostgresProvider{}
	s.Migrations, err = migrations.Load(suite.PostgresTestdata)
	require.NoError(t, err, "could not load postgres migrations")

	_, err = s.ResolveDSN("")
	require.NoError(t, err, "could not resolve postgres DSN")

	suite.Run(t, s)
}

//============================================================================
// SQLite Tests
//============================================================================

func TestSQLite3(t *testing.T) {
	var err error
	s := &StoreTestSuite{}
	s.Provider = &suite.SQLiteProvider{}
	s.Migrations, err = migrations.Load(suite.SQLiteTestdata)
	require.NoError(t, err, "could not load sqlite migrations")

	_, err = s.ResolveDSN("")
	require.NoError(t, err, "could not resolve sqlite DSN")

	suite.Run(t, s)
}
