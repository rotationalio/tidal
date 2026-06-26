package fields_test

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	. "go.rtnl.ai/tidal/fields"
)

func TestTimestamp_Set(t *testing.T) {
	now := time.Now().In(time.FixedZone("HST", -10*60*60))
	ts := Timestamp{}
	ts.Set(now)

	norm := Time(now.UTC().Truncate(time.Millisecond))
	require.False(t, ts.IsZero())
	require.True(t, norm.Equal(ts))
}

func TestTimestamp_MarshalJSON(t *testing.T) {
	testCases := []struct {
		ts   Timestamp
		want string
		err  error
	}{
		{
			Timestamp{},
			"null",
			nil,
		},
		{
			Time(time.Date(2012, 12, 27, 17, 32, 3, 721000000, time.UTC)),
			`"2012-12-27T17:32:03.721Z"`,
			nil,
		},
	}

	for _, tc := range testCases {
		got, err := tc.ts.MarshalJSON()
		if tc.err != nil {
			require.Error(t, err)
			require.EqualError(t, err, tc.err.Error())
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.want, string(got))
		}
	}
}

func TestTimestamp_UnmarshalJSON(t *testing.T) {
	testCases := []struct {
		json string
		want Timestamp
		err  error
	}{
		{
			`""`,
			Timestamp{},
			nil,
		},
		{
			`null`,
			Timestamp{},
			nil,
		},
		{
			`"2012-12-27T17:32:03.721Z"`,
			Time(time.Date(2012, 12, 27, 17, 32, 3, 721000000, time.UTC)),
			nil,
		},
		{
			`"foo"`,
			Timestamp{},
			errors.New(`cannot parse "foo" as an ISO-8601 timestamp`),
		},
		{
			`"2012-12-27T17:32:03.721Z07:00`,
			Timestamp{},
			errors.New("unexpected end of JSON input"),
		},
	}

	for _, tc := range testCases {
		var got Timestamp
		err := got.UnmarshalJSON([]byte(tc.json))
		if tc.err != nil {
			require.Error(t, err)
			require.EqualError(t, err, tc.err.Error())
		} else {
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		}
	}
}

// Verifies Timestamp database round-trip behavior on SQLite.
func (s *FieldsSqliteTestSuite) TestTimestamp() {
	s.Run("Values", func() {
		require := s.Require()

		alpha := Time(time.Date(2025, 2, 14, 11, 21, 42, 792, time.UTC))
		bravo := Time(time.Date(2022, 7, 28, 19, 07, 39, 182, time.UTC))

		tx := s.BeginTx(nil)
		defer tx.Rollback()

		params := []sql.NamedArg{
			sql.Named("alpha", alpha),
			sql.Named("bravo", bravo),
		}

		result, err := tx.Exec("INSERT INTO timeseries (alpha, bravo) VALUES (:alpha, :bravo) RETURNING id", params...)
		require.NoError(err, "could not insert record")

		id, err := result.LastInsertId()
		require.NoError(err, "could not get last insert id")
		require.NotZero(id, "expected last insert id to be non-zero")

		// Fetch the record from the database.
		row := tx.QueryRow("SELECT alpha, bravo FROM timeseries WHERE id=:id", sql.Named("id", id))

		var alphaOut, bravoOut Timestamp
		require.NoError(row.Scan(&alphaOut, &bravoOut), "could not scan record")
		require.True(alpha.Equal(alphaOut), "expected alpha to be equal to the input")
		require.True(bravo.Equal(bravoOut), "expected bravo to be equal to the input")
	})

	s.Run("Default", func() {
		require := s.Require()

		tx := s.BeginTx(nil)
		defer tx.Rollback()

		result, err := tx.Exec("INSERT INTO timeseries (alpha) VALUES (NULL) RETURNING id")
		require.NoError(err, "could not insert record")

		id, err := result.LastInsertId()
		require.NoError(err, "could not get last insert id")
		require.NotZero(id, "expected last insert id to be non-zero")

		// Fetch the record from the database.
		row := tx.QueryRow("SELECT alpha, bravo FROM timeseries WHERE id=:id", sql.Named("id", id))

		var alphaOut, bravoOut Timestamp
		require.NoError(row.Scan(&alphaOut, &bravoOut), "could not scan record")
		require.True(alphaOut.IsZero(), "expected alpha to be zero")
		require.False(bravoOut.IsZero(), "expected bravo to be non-zero")
	})

	s.Run("Null", func() {
		require := s.Require()

		tx := s.BeginTx(nil)
		defer tx.Rollback()

		alpha := Timestamp{}
		result, err := tx.Exec("INSERT INTO timeseries (alpha) VALUES (:alpha) RETURNING id", sql.Named("alpha", alpha))
		require.NoError(err, "could not insert record")

		id, err := result.LastInsertId()
		require.NoError(err, "could not get last insert id")
		require.NotZero(id, "expected last insert id to be non-zero")

		// Fetch the record from the database.
		row := tx.QueryRow("SELECT alpha FROM timeseries WHERE id=:id", sql.Named("id", id))

		var alphaOut Timestamp
		require.NoError(row.Scan(&alphaOut), "could not scan record")
		require.True(alphaOut.IsZero(), "expected alpha to be zero")
	})
}

func (s *FieldsPostgresTestSuite) TestTimestamp() {
	s.Run("Values", func() {
		require := s.Require()

		alpha := Time(time.Date(2025, 2, 14, 11, 21, 42, 792, time.UTC))
		bravo := Time(time.Date(2022, 7, 28, 19, 07, 39, 182, time.UTC))
		charlie := Time(time.Date(2021, 12, 10, 8, 55, 27, 365, time.UTC))
		delta := Time(time.Date(2020, 6, 25, 22, 10, 59, 456, time.UTC))

		tx := s.BeginTx(nil)
		defer tx.Rollback()

		params := []sql.NamedArg{
			sql.Named("alpha", alpha),
			sql.Named("bravo", bravo),
			sql.Named("charlie", charlie),
			sql.Named("delta", delta),
		}

		var id int64
		err := tx.QueryRow("INSERT INTO timeseries (alpha, bravo) VALUES (:alpha, :bravo) RETURNING id", params...).Scan(&id)
		require.NoError(err, "could not insert record")
		require.NotZero(id, "expected last insert id to be non-zero")

		// Fetch the record from the database.
		row := tx.QueryRow("SELECT alpha, bravo FROM timeseries WHERE id=:id", sql.Named("id", id))

		var alphaOut, bravoOut Timestamp
		require.NoError(row.Scan(&alphaOut, &bravoOut), "could not scan record")
		require.True(alpha.Equal(alphaOut), "expected alpha to be equal to the input")
		require.True(bravo.Equal(bravoOut), "expected bravo to be equal to the input")
	})

	s.Run("Default", func() {
		require := s.Require()

		tx := s.BeginTx(nil)
		defer tx.Rollback()

		var id int64
		err := tx.QueryRow("INSERT INTO timeseries (alpha, charlie) VALUES (NULL, NULL) RETURNING id").Scan(&id)
		require.NoError(err, "could not insert record")

		// Fetch the record from the database.
		row := tx.QueryRow("SELECT alpha, bravo, charlie, delta FROM timeseries WHERE id=:id", sql.Named("id", id))

		var alphaOut, bravoOut, charlieOut, deltaOut Timestamp
		require.NoError(row.Scan(&alphaOut, &bravoOut, &charlieOut, &deltaOut), "could not scan record")
		require.True(alphaOut.IsZero(), "expected alpha to be zero")
		require.False(bravoOut.IsZero(), "expected bravo to be zero")
		require.True(charlieOut.IsZero(), "expected charlie to be zero")
		require.False(deltaOut.IsZero(), "expected delta to be zero")
	})

	s.Run("Null", func() {
		require := s.Require()

		tx := s.BeginTx(nil)
		defer tx.Rollback()

		var id int64
		alpha, charlie := Timestamp{}, Timestamp{}
		err := tx.QueryRow("INSERT INTO timeseries (alpha, charlie) VALUES (:alpha, :charlie) RETURNING id", sql.Named("alpha", alpha), sql.Named("charlie", charlie)).Scan(&id)
		require.NoError(err, "could not insert record")
		require.NotZero(id, "expected last insert id to be non-zero")

		// Fetch the record from the database.
		row := tx.QueryRow("SELECT alpha, charlie FROM timeseries WHERE id=:id", sql.Named("id", id))

		var alphaOut, charlieOut Timestamp
		require.NoError(row.Scan(&alphaOut, &charlieOut), "could not scan record")
		require.True(alphaOut.IsZero(), "expected alpha to be zero")
		require.True(charlieOut.IsZero(), "expected charlie to be zero")
	})
}
