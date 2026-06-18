package conn_test

import (
	"go.rtnl.ai/tidal/suite"
	"go.rtnl.ai/x/dsn"
)

type connSuite struct {
	suite.DatabaseSuite
}

func (s *connSuite) requirePostgres() {
	if s.DSN().Provider != dsn.Postgres {
		s.T().Skip("postgres required")
	}
}

func (s *connSuite) requireSQLite3() {
	if s.DSN().Provider != dsn.SQLite3 {
		s.T().Skip("sqlite3 required")
	}
}
