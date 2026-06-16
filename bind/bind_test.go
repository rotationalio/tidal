package bind

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

// Tests that [Rewrite] rewrites :name placeholders for the given placeholder type.
func TestRewrite(t *testing.T) {
	// Prepare test data for all tests

	const complex = "named_with_underscore_and_123_nums"
	query := "INSERT INTO items (sku, qty, note) VALUES (:" + complex + ", :qty, :alpha_bravo) WHERE tenant_id = :tenant_id"
	args := []sql.NamedArg{
		sql.Named(complex, "SKU-42"),
		sql.Named("qty", 7),
		sql.Named("alpha_bravo", "notes"),
		sql.Named("tenant_id", 99),
	}
	wantValues := []any{"SKU-42", 7, "notes", 99}

	// Happy Path

	t.Run("Positional", func(t *testing.T) {
		b, err := Rewrite(query, args, Positional)
		require.NoError(t, err)
		require.Equal(t,
			"INSERT INTO items (sku, qty, note) VALUES (?, ?, ?) WHERE tenant_id = ?",
			b.SQL(),
		)
		require.Equal(t, wantValues, b.Args())
	})

	t.Run("Ordered", func(t *testing.T) {
		b, err := Rewrite(query, args, Ordered)
		require.NoError(t, err)
		require.Equal(t,
			"INSERT INTO items (sku, qty, note) VALUES ($1, $2, $3) WHERE tenant_id = $4",
			b.SQL(),
		)
		require.Equal(t, wantValues, b.Args())
	})

	t.Run("Named", func(t *testing.T) {
		b, err := Rewrite(query, args, Named)
		require.NoError(t, err)
		require.Equal(t, query, b.SQL())
		require.Equal(t, []any{
			sql.Named(complex, "SKU-42"),
			sql.Named("qty", 7),
			sql.Named("alpha_bravo", "notes"),
			sql.Named("tenant_id", 99),
		}, b.Args())
	})

	t.Run("AtP", func(t *testing.T) {
		b, err := Rewrite(query, args, AtP)
		require.NoError(t, err)
		require.Equal(t,
			"INSERT INTO items (sku, qty, note) VALUES (@p1, @p2, @p3) WHERE tenant_id = @p4",
			b.SQL(),
		)
		require.Equal(t, wantValues, b.Args())
	})

	// Failures / Errors

	t.Run("UnknownPlaceholder", func(t *testing.T) {
		_, err := Rewrite(query, args, UnknownPlaceholder)
		require.ErrorIs(t, err, ErrUnsupportedPlaceholder, "unknown placeholder type should return an error")
	})

	t.Run("MissingArgument", func(t *testing.T) {
		_, err := Rewrite(
			"SELECT * FROM items WHERE sku = :sku",
			[]sql.NamedArg{},
			Ordered,
		)
		var missing MissingArgument
		require.ErrorAs(t, err, &missing)
		require.Equal(t, MissingArgument("sku"), missing)
	})

	// Placeholder Rewriter Edge Cases

	// Bound arg values follow left-to-right query appearance, not the NamedArg slice order.
	t.Run("ArgOrderFollowsQueryText", func(t *testing.T) {
		b, err := Rewrite(
			"SELECT * FROM t WHERE a = :b AND b = :a",
			[]sql.NamedArg{sql.Named("a", 1), sql.Named("b", 2)},
			Ordered,
		)
		require.NoError(t, err)
		require.Equal(t, "SELECT * FROM t WHERE a = $1 AND b = $2", b.SQL())
		require.Equal(t, []any{2, 1}, b.Args())
	})

	// Repeated :name tokens reuse the same numbered placeholder and bind once.
	t.Run("RepeatedPlaceholderOrdered", func(t *testing.T) {
		b, err := Rewrite(
			"SELECT * FROM t WHERE id = :id OR parent_id = :id",
			[]sql.NamedArg{sql.Named("id", 42)},
			Ordered,
		)
		require.NoError(t, err)
		require.Equal(t, "SELECT * FROM t WHERE id = $1 OR parent_id = $1", b.SQL())
		require.Equal(t, []any{42}, b.Args())
	})

	// Anonymous ? placeholders still need one arg per occurrence — no index reuse.
	t.Run("RepeatedPlaceholderPositional", func(t *testing.T) {
		b, err := Rewrite(
			"SELECT * FROM t WHERE id = :id OR parent_id = :id",
			[]sql.NamedArg{sql.Named("id", 42)},
			Positional,
		)
		require.NoError(t, err)
		require.Equal(t, "SELECT * FROM t WHERE id = ? OR parent_id = ?", b.SQL())
		require.Equal(t, []any{42, 42}, b.Args())
	})

	// Postgres ::type casts must pass through without being treated as :name placeholders.
	t.Run("PostgresCastWithNamedArg", func(t *testing.T) {
		b, err := Rewrite(
			"SELECT * FROM t WHERE id::text = :id",
			[]sql.NamedArg{sql.Named("id", "abc")},
			Ordered,
		)
		require.NoError(t, err)
		require.Equal(t, "SELECT * FROM t WHERE id::text = $1", b.SQL())
		require.Equal(t, []any{"abc"}, b.Args())
	})
}
