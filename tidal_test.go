package tidal_test

import (
	"database/sql"
	"embed"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal/migrations"
	"go.rtnl.ai/tidal/suite"
	"go.rtnl.ai/ulid"
)

type TidalTestSuite struct {
	suite.DatabaseSuite
}

//============================================================================
// Postgres Tests
//============================================================================

//go:embed testdata/postgres
var postgresFS embed.FS

func TestPostgres(t *testing.T) {
	var err error
	s := &TidalTestSuite{}
	s.Provider = &suite.PostgresProvider{}
	s.Migrations, err = migrations.Load(postgresFS)
	require.NoError(t, err, "could not load postgres migrations")

	_, err = s.ResolveDSN("")
	require.NoError(t, err, "could not resolve postgres DSN")

	suite.Run(t, s)
}

//============================================================================
// SQLite Tests
//============================================================================

//go:embed testdata/sqlite
var sqliteFS embed.FS

func TestSQLite3(t *testing.T) {
	var err error
	s := &TidalTestSuite{}
	s.Provider = &suite.SQLiteProvider{}
	s.Migrations, err = migrations.Load(sqliteFS)
	require.NoError(t, err, "could not load sqlite migrations")

	_, err = s.ResolveDSN("")
	require.NoError(t, err, "could not resolve sqlite DSN")

	suite.Run(t, s)
}

//============================================================================
// CRUD Conformance Tests Testing (User Model)
//============================================================================

func (s *TidalTestSuite) TestUserCRUDConformance() {
	suite.ConformsCRUD(&s.DatabaseSuite, suite.CRUDConformance[*User]{
		Table:  "users",
		Create: newConformanceUser,
		Update: func(u *User) {
			u.Name = "Updated Conformance User"
		},
	})
}

func newConformanceUser() *User {
	return &User{
		Name:     "Conformance User",
		DOB:      sql.NullTime{Valid: true, Time: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)},
		Email:    fmt.Sprintf("conformance-%s@example.com", ulid.MakeSecure().String()),
		Password: "test-password",
		Verified: true,
		LastSeen: sql.NullTime{Valid: true, Time: time.Now().Add(-1 * time.Hour)},
	}
}
