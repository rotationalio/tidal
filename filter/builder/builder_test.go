package builder_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal/filter/builder"
)

func TestPrefix(t *testing.T) {
	testCases := []struct {
		name     string
		field    string
		alias    string
		fields   []string
		expected string
	}{
		{
			name:     "Simple",
			field:    "title",
			alias:    "m",
			fields:   nil,
			expected: "m.title",
		},
		{
			name:     "SimpleFields",
			field:    "title",
			alias:    "m",
			fields:   []string{"id", "title", "slug"},
			expected: "m.title",
		},
		{
			name:     "SkipNotInFields",
			field:    "title",
			alias:    "m",
			fields:   []string{"id", "slug", "name"},
			expected: "title",
		},
		{
			name:     "RemovePrefix",
			field:    "m.title",
			alias:    "",
			fields:   nil,
			expected: "title",
		},
		{
			name:     "RemovePrefixFields",
			field:    "m.title",
			alias:    "",
			fields:   []string{"id", "title", "slug"},
			expected: "title",
		},
		{
			name:     "RemovePrefixNotInFields",
			field:    "m.title",
			alias:    "",
			fields:   []string{"id", "slug", "name"},
			expected: "m.title",
		},
		{
			name:     "ReplacePrefix",
			field:    "m.title",
			alias:    "b",
			fields:   nil,
			expected: "b.title",
		},
		{
			name:     "ReplacePrefixNotInFields",
			field:    "m.title",
			alias:    "b",
			fields:   []string{"id", "slug", "name"},
			expected: "m.title",
		},
		{
			name:     "ReplacePrefixNotInFields",
			field:    "m.title",
			alias:    "b",
			fields:   []string{"id", "slug", "name"},
			expected: "m.title",
		},
		{
			name:     "ReplacePrefixFields",
			field:    "m.title",
			alias:    "b",
			fields:   []string{"id", "title", "slug"},
			expected: "b.title",
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := builder.Prefix(tc.field, tc.alias, tc.fields...)
			require.Equal(t, tc.expected, result, "test case %d failed", i)
		})
	}

}
