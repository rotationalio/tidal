package tidal_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal/fields"
	"go.rtnl.ai/tidal/migrations"
	"go.rtnl.ai/tidal/suite"
	"go.rtnl.ai/tidal/suite/fixtures"
)

type TidalTestSuite struct {
	suite.DatabaseSuite
}

//============================================================================
// Postgres Tests
//============================================================================

func TestPostgres(t *testing.T) {
	var err error
	s := &TidalTestSuite{}
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
	s := &TidalTestSuite{}
	s.Provider = &suite.SQLiteProvider{}
	s.Migrations, err = migrations.Load(suite.SQLiteTestdata)
	require.NoError(t, err, "could not load sqlite migrations")

	_, err = s.ResolveDSN("")
	require.NoError(t, err, "could not resolve sqlite DSN")

	suite.Run(t, s)
}

//============================================================================
// CRUD Conformance Tests Testing (User Model)
//============================================================================

func (s *TidalTestSuite) TestUserCRUDConformance() {
	suite.ConformsCRUD(&s.DatabaseSuite, suite.CRUDConformance[*fixtures.User]{
		Table:  "users",
		Create: fixtures.NewConformanceUser,
		Update: func(u *fixtures.User) {
			// Mutate every persisted update field.
			u.Name = "Updated Conformance User"
			u.DOB = fields.Time(time.Date(1988, 5, 21, 0, 0, 0, 0, time.UTC))
			u.Email = "updated-conformance@example.com"
		},
	})
}
