package builder_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal/filter/builder"
)

// Check that table aliases are added to matching fields without changing the
// operators or parameter values.
func TestWhere_Prefix(t *testing.T) {
	// Make simple WHERE expressions for each alias case and compare the SQL and
	// parameters after applying the alias.
	where := func(field string, op builder.WhereOp, value any) *builder.Where {
		return (&builder.Where{}).Where(field, op, value)
	}

	// Cover adding, keeping, clearing, and replacing aliases, including an OR
	// expression.
	testCases := []struct {
		orig           *builder.Where
		alias          string
		expectedClause string
		expectedParams []sql.NamedArg
	}{
		{
			orig:           where("id", builder.Eq, 42),
			alias:          "t",
			expectedClause: "WHERE t.id = :w1",
			expectedParams: []sql.NamedArg{
				{Name: "w1", Value: 42},
			},
		},
		{
			orig:           where("t.id", builder.Eq, 42),
			alias:          "t",
			expectedClause: "WHERE t.id = :w1",
			expectedParams: []sql.NamedArg{
				{Name: "w1", Value: 42},
			},
		},
		{
			orig:           where("t.id", builder.Eq, 42),
			alias:          "",
			expectedClause: "WHERE id = :w1",
			expectedParams: []sql.NamedArg{
				{Name: "w1", Value: 42},
			},
		},
		{
			orig:           where("t.id", builder.Eq, 42),
			alias:          "m",
			expectedClause: "WHERE m.id = :w1",
			expectedParams: []sql.NamedArg{
				{Name: "w1", Value: 42},
			},
		},
		{
			orig:           where("id", builder.Eq, 42).Or("slug", builder.Eq, "typhoon-actual"),
			alias:          "b",
			expectedClause: "WHERE b.id = :w1 OR b.slug = :w2",
			expectedParams: []sql.NamedArg{
				{Name: "w1", Value: 42},
				{Name: "w2", Value: "typhoon-actual"},
			},
		},
	}

	// Apply each alias and check that only the field names change and the
	// parameters stay in the same order.
	for i, tc := range testCases {
		tc.orig.Prefix(tc.alias)
		actualClause, actualParams := tc.orig.Render()
		require.Equal(t, tc.expectedClause, actualClause, "test case %d failed", i)
		require.Equal(t, tc.expectedParams, actualParams, "test case %d failed", i)
	}
}

// Check conditions, lists, subqueries, ANY/ALL comparisons, and their
// parameters.
func TestWhere_Operators(t *testing.T) {
	t.Run("WhereAppends", func(t *testing.T) {
		// Add two conditions and check that the second is joined with AND instead
		// of replacing the first.
		where := (&builder.Where{}).
			Where("status", builder.Eq, "active").
			Where("role", builder.Eq, "admin")

		clause, params := where.Render()
		require.Equal(t, "WHERE status = :w1 AND role = :w2", clause)
		require.Equal(t, []sql.NamedArg{
			sql.Named("w1", "active"),
			sql.Named("w2", "admin"),
		}, params)
	})

	t.Run("NotIn", func(t *testing.T) {
		// Make a NOT IN list and check that each value gets a named placeholder
		// in the same order.
		where := (&builder.Where{}).Where("id", builder.NotIn, []int{1, 2, 3})

		clause, params := where.Render()
		require.Equal(t, "WHERE id NOT IN (:w1, :w2, :w3)", clause)
		require.Equal(t, []sql.NamedArg{
			sql.Named("w1", 1),
			sql.Named("w2", 2),
			sql.Named("w3", 3),
		}, params)
	})

	t.Run("EmptySetIdentities", func(t *testing.T) {
		// Check that empty IN and NOT IN lists become false and true conditions
		// instead of disappearing.
		in := (&builder.Where{}).Where("id", builder.In, []int{})
		notIn := (&builder.Where{}).Where("id", builder.NotIn, []int{})

		inClause, inParams := in.Render()
		notInClause, notInParams := notIn.Render()
		require.Equal(t, "WHERE 1=0", inClause)
		require.Empty(t, inParams)
		require.Equal(t, "WHERE 1=1", notInClause)
		require.Empty(t, notInParams)
	})

	t.Run("SubselectAndParameter", func(t *testing.T) {
		// Use a trusted subquery with a named parameter and check that the query
		// is kept as SQL instead of being bound as a string.
		where := (&builder.Where{}).
			Where("id", builder.NotIn, builder.Subselect("SELECT id FROM deleted WHERE owner=:owner")).
			Param("owner", "ada")

		clause, params := where.Render()
		require.Equal(t, "WHERE id NOT IN (SELECT id FROM deleted WHERE owner=:owner)", clause)
		require.Equal(t, []sql.NamedArg{sql.Named("owner", "ada")}, params)
	})

	t.Run("AnySubselect", func(t *testing.T) {
		// Check that an ANY comparison keeps its operator and trusted subquery.
		where := (&builder.Where{}).
			Where("id", builder.Any(builder.Eq), builder.Subselect("SELECT id FROM candidates"))

		clause, params := where.Render()
		require.Equal(t, "WHERE id = ANY (SELECT id FROM candidates)", clause)
		require.Empty(t, params)
	})

	t.Run("WithSubselectString", func(t *testing.T) {
		// Check that a WITH query string is treated as a subquery instead of a
		// string parameter.
		where := (&builder.Where{}).
			Where("id", builder.Any(builder.Eq), "WITH candidates AS (SELECT id FROM source) SELECT id FROM candidates")

		clause, params := where.Render()
		require.Equal(t, "WHERE id = ANY (WITH candidates AS (SELECT id FROM source) SELECT id FROM candidates)", clause)
		require.Empty(t, params)
	})

	t.Run("AllArray", func(t *testing.T) {
		// Check that an ALL comparison keeps the whole slice as one array
		// parameter.
		where := (&builder.Where{}).Where("id", builder.All(builder.Gt), []int64{1, 2})

		clause, params := where.Render()
		require.Equal(t, "WHERE id > ALL (:w1)", clause)
		require.Equal(t, []sql.NamedArg{sql.Named("w1", []int64{1, 2})}, params)
	})

	t.Run("QuantifiedEmptySetIdentities", func(t *testing.T) {
		// Check the empty-set rules: ANY is false and ALL is true. The condition
		// must remain in the query instead of disappearing.
		any := (&builder.Where{}).Where("id", builder.Any(builder.Eq), []int{})
		all := (&builder.Where{}).Where("id", builder.All(builder.Eq), []int{})

		anyClause, anyParams := any.Render()
		allClause, allParams := all.Render()
		require.Equal(t, "WHERE 1=0", anyClause)
		require.Empty(t, anyParams)
		require.Equal(t, "WHERE 1=1", allClause)
		require.Empty(t, allParams)
	})

	t.Run("QuantifiedComparisonOperators", func(t *testing.T) {
		// Check that Any and All keep the chosen comparison operator and bind
		// each slice as one parameter.
		any := (&builder.Where{}).Where("name", builder.Any(builder.Ne), []string{"ada", "grace"})
		all := (&builder.Where{}).Where("score", builder.All(builder.Lte), []int64{10, 20})

		anyClause, anyParams := any.Render()
		allClause, allParams := all.Render()
		require.Equal(t, "WHERE name != ANY (:w1)", anyClause)
		require.Equal(t, []sql.NamedArg{sql.Named("w1", []string{"ada", "grace"})}, anyParams)
		require.Equal(t, "WHERE score <= ALL (:w1)", allClause)
		require.Equal(t, []sql.NamedArg{sql.Named("w1", []int64{10, 20})}, allParams)
	})
}
