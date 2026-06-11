package tidal_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/x/dsn"

	_ "github.com/mattn/go-sqlite3"
)

// QueryRow bind failures return a Row; the error shows up on Scan and Err.
func TestRowBindError(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	tidalDB := tidal.Wrap(db, &dsn.DSN{Provider: "mysql"})
	tx, err := tidalDB.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })

	row := tx.QueryRow("SELECT :x", sql.Named("x", 1))
	require.ErrorIs(t, row.Scan(new(int)), tidal.ErrUnsupportedPlaceholder)
	require.ErrorIs(t, row.Err(), tidal.ErrUnsupportedPlaceholder)
}
