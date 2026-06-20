package conn_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal/conn"
	"go.rtnl.ai/tidal/errors"
	"go.rtnl.ai/x/dsn"
)

// QueryRow bind failures return a Row; the error shows up on Scan and Err.
func TestRowBindError(t *testing.T) {
	db, err := conn.OpenSQLite3(context.Background(), &dsn.DSN{Provider: "sqlite3", Path: ":memory:"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	tidalDB := conn.Wrap(db, &dsn.DSN{Provider: "mysql"})
	tx, err := tidalDB.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })

	row := tx.QueryRow("SELECT :x", sql.Named("x", 1))
	require.ErrorIs(t, row.Scan(new(int)), errors.ErrUnsupportedPlaceholder)
	require.ErrorIs(t, row.Err(), errors.ErrUnsupportedPlaceholder)
}
