package builder_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal/filter/builder"
)

func TestWhere_Prefix(t *testing.T) {
	where := func(field string, op builder.WhereOp, value any) *builder.Where {
		return (&builder.Where{}).Where(field, op, value)
	}

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

	for i, tc := range testCases {
		tc.orig.Prefix(tc.alias)
		actualClause, actualParams := tc.orig.Render()
		require.Equal(t, tc.expectedClause, actualClause, "test case %d failed", i)
		require.Equal(t, tc.expectedParams, actualParams, "test case %d failed", i)
	}
}
