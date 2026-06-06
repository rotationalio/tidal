package suite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.rtnl.ai/x/dsn"
)

const (
	DATABASE_URL          = "DATABASE_URL"
	TEST_DATABASE_URL     = "TEST_DATABASE_URL"
	TIDAL_DATABASE_URL    = "TIDAL_DATABASE_URL"
	SQLITE_DATABASE_URL   = "SQLITE_DATABASE_URL"
	POSTGRES_DATABASE_URL = "POSTGRES_DATABASE_URL"
)

var (
	ErrInvalidProvider  = errors.New("invalid database provider for this suite")
	ErrSqliteRequired   = errors.New("valid sqlite3 database url is required for this suite")
	ErrPostgresRequired = errors.New("valid postgres database url is required for this suite")
	ErrNoDatabaseURL    = errors.New("could not resolve or load database URL from the environment")
)

func Run(t *testing.T, s suite.TestingSuite) {
	suite.Run(t, s)
}

// Lookups the database url from the environment. It first tries all the vars in the
// order they are specified. If none are found, it will return the value of the
// DATABASE_URL environment variable. Uses the same semantics as os.LookupEnv.
func DatabaseURL(vars ...string) string {
	for _, v := range vars {
		if value, ok := os.LookupEnv(v); ok && value != "" {
			return value
		}
	}
	return os.Getenv(DATABASE_URL)
}

type DatabaseSuite struct {
	suite.Suite
	*sql.DB

	dsn *dsn.DSN
}

func (s *DatabaseSuite) TearDownSuite() {
	if s.DB != nil {
		require := s.Require()
		require.NoError(s.Close(), "could not close connection to database")

		s.DB = nil
		s.dsn = nil
	} else {
		s.T().Log("cannot tear down the database test suite because s.DB is nil")
	}
}

func (s *DatabaseSuite) BeginTx(opts *sql.TxOptions) (*sql.Tx, context.CancelFunc) {
	if s.DB == nil {
		s.T().Log("cannot begin transaction because s.DB is nil")
		s.T().FailNow()
	}

	ctx, cancel := s.Context()
	tx, err := s.DB.BeginTx(ctx, opts)
	s.Require().NoError(err, "could not begin transaction")
	return tx, cancel
}

func (s *DatabaseSuite) ReadOnly() bool {
	return s.dsn.Options.ReadOnly()
}

func (s *DatabaseSuite) Context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func (s *DatabaseSuite) DSN() dsn.DSN {
	return *s.dsn
}
