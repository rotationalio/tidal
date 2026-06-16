package filter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

//============================================================================
// Filter Clause Builder
//============================================================================

// Verifies [Filter.Clause] produces correct ANSI SQL for ordering, pagination, and
// fluent-builder overwrite/clear semantics. No database required.
func TestFilterClause(t *testing.T) {
	t.Run("OrderByAscDesc", func(t *testing.T) {
		// Bare column is ASC; -prefix is DESC.
		f := (&Filter{}).OrderBy("name", "-created")
		require.Equal(t, "ORDER BY name ASC, created DESC", f.Clause())
	})

	t.Run("LimitOffset", func(t *testing.T) {
		f := (&Filter{}).OrderBy("id").Limit(10).Offset(5)
		require.Equal(t, "ORDER BY id ASC LIMIT 10 OFFSET 5", f.Clause())
	})

	t.Run("LimitOffsetWithoutOrder", func(t *testing.T) {
		f := (&Filter{}).Limit(10).Offset(5)
		require.Equal(t, "LIMIT 10 OFFSET 5", f.Clause())
	})

	t.Run("ClearLimitOffset", func(t *testing.T) {
		// n=-1 clears a previously set limit or offset.
		f := (&Filter{}).Limit(10).Offset(5).Limit(-1).Offset(-1)
		require.Empty(t, f.Clause())
	})

	t.Run("OrderByOverwrite", func(t *testing.T) {
		// Each OrderBy call replaces the previous ordering entirely.
		f := (&Filter{}).OrderBy("name").OrderBy("-email")
		require.Equal(t, "ORDER BY email DESC", f.Clause())
	})
}
