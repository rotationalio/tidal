package conn_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal/conn"
	"go.rtnl.ai/tidal/errors"
	"go.rtnl.ai/x/dsn"

	_ "github.com/mattn/go-sqlite3"
)

//============================================================================
// Tx Bind Errors
//============================================================================

// Unlike [conn.Row] (deferred to Scan), Exec and Query return bind errors immediately
// when the placeholder type is unsupported.
func TestTxnBindErrors(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	tidalDB := conn.Wrap(db, &dsn.DSN{Provider: "mysql"})
	tx, err := tidalDB.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback() })

	t.Run("Exec", func(t *testing.T) {
		_, err := tx.Exec("SELECT :x", sql.Named("x", 1))
		require.ErrorIs(t, err, errors.ErrUnsupportedPlaceholder)
	})

	t.Run("Query", func(t *testing.T) {
		_, err := tx.Query("SELECT :x", sql.Named("x", 1))
		require.ErrorIs(t, err, errors.ErrUnsupportedPlaceholder)
	})
}
