package builder_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal/filter/builder"
)

func TestOrderBy_Prefix(t *testing.T) {
	testCases := []struct {
		orig   builder.OrderBy
		alias  string
		fields []string
		want   string
	}{
		{
			orig:  builder.OrderBy{Column: "title"},
			alias: "t",
			want:  "t.title",
		},
		{
			orig:  builder.OrderBy{Column: "t.title"},
			alias: "t",
			want:  "t.title",
		},
		{
			orig:  builder.OrderBy{Column: "t.title"},
			alias: "",
			want:  "title",
		},
		{
			orig:  builder.OrderBy{Column: "t.title"},
			alias: "m",
			want:  "m.title",
		},
		{
			orig:   builder.OrderBy{Column: "t.title"},
			alias:  "m",
			fields: []string{"id", "title", "slug"},
			want:   "m.title",
		},
		{
			orig:   builder.OrderBy{Column: "title"},
			alias:  "m",
			fields: []string{"id", "title", "slug"},
			want:   "m.title",
		},
		{
			orig:   builder.OrderBy{Column: "t.title"},
			alias:  "m",
			fields: []string{"id", "slug", "name"},
			want:   "t.title",
		},
		{
			orig:   builder.OrderBy{Column: "title"},
			alias:  "m",
			fields: []string{"id", "slug", "name"},
			want:   "title",
		},
		{
			orig:   builder.OrderBy{Column: "t.title"},
			alias:  "",
			fields: []string{"id", "slug", "title"},
			want:   "title",
		},
	}

	for i, tc := range testCases {
		tc.orig.Prefix(tc.alias, tc.fields...)
		require.Equal(t, tc.want, tc.orig.Column, "test case %d failed", i)
	}
}
