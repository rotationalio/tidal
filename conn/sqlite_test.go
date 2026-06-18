package conn_test

import (
	"testing"
	"time"

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

// TestDatetimeReturnsUTC tests that SQLite datetime values are returned as UTC.
func (s *connSuite) TestDatetimeReturnsUTC() {
	s.requireSQLite3()
	require := s.Require()
	ctx := s.Context()

	_, err := s.DB.ExecContext(ctx, "CREATE TEMP TABLE conn_ts_test (ts DATETIME NOT NULL)")
	require.NoError(err)

	want := time.Date(2025, 2, 14, 11, 21, 42, 0, time.UTC)
	_, err = s.DB.ExecContext(ctx, "INSERT INTO conn_ts_test (ts) VALUES (?)", want)
	require.NoError(err)

	var got time.Time
	err = s.DB.QueryRowContext(ctx, "SELECT ts FROM conn_ts_test").Scan(&got)
	require.NoError(err)
	require.Equal(time.UTC, got.Location())
	// SQLite datetime storage is second precision in practice; compare at that granularity.
	require.True(want.Truncate(time.Second).Equal(got.Truncate(time.Second)))
}
