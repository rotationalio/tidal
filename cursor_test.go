package tidal_test

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal"
)

func CursorTest(tx *sql.Tx, require *require.Assertions) func() {
	return func() {
		user := &User{}
		fields := strings.Join(user.Fields(tidal.List), ", ")

		rows, err := tx.Query(fmt.Sprintf("SELECT %s FROM users", fields))
		require.NoError(err)

		cursor := tidal.Rows[*User](tidal.Wrap(tx), rows)
		models, err := cursor.List()
		require.NoError(err)
		require.Len(models, 25)

		require.NoError(cursor.Close())
	}
}

func (s *PostgresTestSuite) TestCursor() {
	tx, cancel := s.BeginTx(&sql.TxOptions{
		ReadOnly: true,
	})
	defer cancel()
	defer tx.Rollback()

	s.Run("test cursor", CursorTest(tx, s.Require()))
}

func (s *SQLiteTestSuite) TestCursor() {
	tx, cancel := s.BeginTx(&sql.TxOptions{
		ReadOnly: true,
	})
	defer cancel()
	defer tx.Rollback()

	s.Run("test cursor", CursorTest(tx, s.Require()))
}
