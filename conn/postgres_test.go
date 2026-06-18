package conn_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal/suite"
)

func TestPostgres(t *testing.T) {
	s := &connSuite{}
	s.Provider = &suite.PostgresProvider{}
	s.Teardown = suite.TeardownNone

	_, err := s.ResolveDSN("")
	require.NoError(t, err, "could not resolve postgres DSN")

	suite.Run(t, s)
}

// TestTimestamptzReturnsUTC tests that Postgres timestamptz values are returned as UTC.
func (s *connSuite) TestTimestamptzReturnsUTC() {
	s.requirePostgres()
	require := s.Require()
	ctx := s.Context()

	hst := time.FixedZone("HST", -10*60*60)

	cases := []struct {
		name     string
		param    time.Time
		expected time.Time
	}{
		{
			name:     "UTC",
			param:    time.Date(2025, 2, 14, 11, 21, 42, 0, time.UTC),
			expected: time.Date(2025, 2, 14, 11, 21, 42, 0, time.UTC),
		},
		{
			name:     "NonUTCInput",
			param:    time.Date(2025, 2, 14, 1, 21, 42, 0, hst),
			expected: time.Date(2025, 2, 14, 11, 21, 42, 0, time.UTC),
		},
		{
			name:     "FixtureLiteral",
			param:    time.Date(2025, 3, 4, 19, 9, 6, 0, time.UTC),
			expected: time.Date(2025, 3, 4, 19, 9, 6, 0, time.UTC),
		},
		{
			name:     "MicrosecondPrecision",
			param:    time.Date(2024, 6, 1, 12, 0, 0, 123456000, time.UTC),
			expected: time.Date(2024, 6, 1, 12, 0, 0, 123456000, time.UTC),
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			var got time.Time
			err := s.DB.QueryRowContext(ctx, "SELECT $1::timestamptz", tc.param).Scan(&got)
			require.NoError(err)
			require.Equal(time.UTC, got.Location())
			require.True(got.Round(time.Microsecond).Equal(tc.expected.Round(time.Microsecond)))
		})
	}

	s.Run("SQLLiteral", func() {
		var got time.Time
		err := s.DB.QueryRowContext(ctx, "SELECT '2025-02-14T11:21:42+00:00'::timestamptz").Scan(&got)
		require.NoError(err)
		require.Equal(time.UTC, got.Location())
		require.Equal(
			time.Date(2025, 2, 14, 11, 21, 42, 0, time.UTC),
			got,
		)
	})
}
