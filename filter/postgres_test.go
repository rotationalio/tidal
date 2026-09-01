package filter_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal/filter"
	"go.rtnl.ai/tidal/suite"
)

// Exercises ANY and ALL array parameters through the configured PostgreSQL
// database/sql driver and verifies the rows returned by each comparison.
type filterPostgresSuite struct {
	suite.PostgresSuite
}

// Exercises ANY and ALL array parameters through the configured PostgreSQL
// database/sql driver and verifies the rows returned by each comparison.
func TestPostgresFilter(t *testing.T) {
	s := &filterPostgresSuite{}
	suite.Run(t, s)
}

func (s *filterPostgresSuite) TestAnyAndAllArrayParameters() {
	// Set up a temporary table with scalar BIGINT columns. The scalar columns
	// let PostgreSQL compare each row against the array parameter supplied to
	// ANY or ALL.
	_, err := s.SQLDB().ExecContext(s.Context(), `
		CREATE TEMPORARY TABLE filter_quantified_values (
			id BIGINT PRIMARY KEY,
			score BIGINT NOT NULL
		)`)
	require.NoError(s.T(), err)

	// Insert predictable IDs and scores so the expected results for equality
	// and greater-than quantified comparisons are unambiguous.
	_, err = s.SQLDB().ExecContext(s.Context(), `
		INSERT INTO filter_quantified_values (id, score)
		VALUES (1, 10), (2, 20), (3, 30)`)
	require.NoError(s.T(), err)

	// Use a real Tidal transaction for both queries. Passing the filter's
	// named arguments through tx.Query exercises placeholder rewriting and
	// pgx's conversion of []int64 into a PostgreSQL array.
	tx := s.BeginTx(nil)
	defer tx.Rollback()

	// Match IDs equal to any element in [1, 3]. The assertion below expects
	// PostgreSQL to return only the first and third rows.
	anyFilter := filter.Where("id", filter.Any(filter.Eq), []int64{1, 3})
	anyRows, err := tx.Query(
		"SELECT id FROM filter_quantified_values "+anyFilter.Clause()+" ORDER BY id",
		anyFilter.Params()...,
	)
	require.NoError(s.T(), err)
	defer anyRows.Close()

	var anyIDs []int64
	for anyRows.Next() {
		var id int64
		require.NoError(s.T(), anyRows.Scan(&id))
		anyIDs = append(anyIDs, id)
	}
	require.NoError(s.T(), anyRows.Err())
	require.Equal(s.T(), []int64{1, 3}, anyIDs)

	// Match scores greater than every element in [1, 2]. Since all inserted
	// scores satisfy that comparison, the assertion expects all three IDs.
	allFilter := filter.Where("score", filter.All(filter.Gt), []int64{1, 2})
	allRows, err := tx.Query(
		"SELECT id FROM filter_quantified_values "+allFilter.Clause()+" ORDER BY id",
		allFilter.Params()...,
	)
	require.NoError(s.T(), err)
	defer allRows.Close()

	var allIDs []int64
	for allRows.Next() {
		var id int64
		require.NoError(s.T(), allRows.Scan(&id))
		allIDs = append(allIDs, id)
	}
	require.NoError(s.T(), allRows.Err())
	require.Equal(s.T(), []int64{1, 2, 3}, allIDs)
}
