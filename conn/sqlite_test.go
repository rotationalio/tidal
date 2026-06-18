package conn_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal/suite"
)

func TestSQLite3(t *testing.T) {
	s := &connSuite{}
	s.Provider = &suite.SQLiteProvider{}
	s.Teardown = suite.TeardownNone

	_, err := s.ResolveDSN("")
	require.NoError(t, err, "could not resolve sqlite DSN")

	suite.Run(t, s)
}
