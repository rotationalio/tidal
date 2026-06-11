package tidal

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

// Tests that [QueryParams] rewrites :name placeholders for the given placeholder type.
func TestQueryParams(t *testing.T) {
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
		b, err := QueryParams(query, args, Positional)
		require.NoError(t, err)
		require.Equal(t,
			"INSERT INTO items (sku, qty, note) VALUES (?, ?, ?) WHERE tenant_id = ?",
			b.SQL(),
		)
		require.Equal(t, wantValues, b.Args())
	})

	t.Run("Ordered", func(t *testing.T) {
		b, err := QueryParams(query, args, Ordered)
		require.NoError(t, err)
		require.Equal(t,
			"INSERT INTO items (sku, qty, note) VALUES ($1, $2, $3) WHERE tenant_id = $4",
			b.SQL(),
		)
		require.Equal(t, wantValues, b.Args())
	})

	t.Run("Named", func(t *testing.T) {
		b, err := QueryParams(query, args, Named)
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
		b, err := QueryParams(query, args, AtP)
		require.NoError(t, err)
		require.Equal(t,
			"INSERT INTO items (sku, qty, note) VALUES (@p1, @p2, @p3) WHERE tenant_id = @p4",
			b.SQL(),
		)
		require.Equal(t, wantValues, b.Args())
	})

	// Failures / Errors

	t.Run("UnknownPlaceholder", func(t *testing.T) {
		_, err := QueryParams(query, args, UnknownPlaceholder)
		require.ErrorIs(t, err, ErrUnsupportedPlaceholder, "unknown placeholder type should return an error")
	})

	t.Run("MissingArgument", func(t *testing.T) {
		_, err := QueryParams(
			"SELECT * FROM items WHERE sku = :sku",
			[]sql.NamedArg{},
			Ordered,
		)
		var missing *MissingArgumentError
		require.ErrorAs(t, err, &missing)
		require.Equal(t, "sku", missing.Name)
	})
}
