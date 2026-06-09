package tidal_test

import (
	"embed"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal/migrations"
	"go.rtnl.ai/tidal/suite"
)

type TidalTestSuite struct {
	suite.DatabaseSuite
}

//go:embed testdata/postgres
var postgresFS embed.FS

func TestTidalPostgres(t *testing.T) {
	var err error
	s := &TidalTestSuite{}
	s.Provider = &suite.PostgresProvider{}
	s.Migrations, err = migrations.Load(postgresFS)
	require.NoError(t, err, "could not load postgres migrations")

	_, err = s.ResolveDSN("")
	require.NoError(t, err, "could not resolve postgres DSN")

	// Run the tests
	suite.Run(t, s)
}

//go:embed testdata/sqlite
var sqliteFS embed.FS

func TestSQLite(t *testing.T) {
	var err error
	s := &TidalTestSuite{}
	s.Provider = &suite.SQLiteProvider{}
	s.Migrations, err = migrations.Load(sqliteFS)
	require.NoError(t, err, "could not load sqlite migrations")

	_, err = s.ResolveDSN("")
	require.NoError(t, err, "could not resolve sqlite DSN")

	// Run the tests
	suite.Run(t, s)
}
