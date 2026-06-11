package tidal_test

import (
	"database/sql"
	"fmt"
	"strings"

	"go.rtnl.ai/tidal"
)

func (s *TidalTestSuite) TestCursor() {
	require := s.Require()
	tx := s.BeginTx(&sql.TxOptions{
		ReadOnly: true,
	})
	defer tx.Rollback()

	user := &User{}
	fields := strings.Join(user.Fields(tidal.List), ", ")

	rows, err := tx.Query(fmt.Sprintf("SELECT %s FROM users", fields))
	require.NoError(err)

	cursor := tidal.Rows[*User](tx, rows)
	models, err := cursor.List()
	require.NoError(err)
	require.Len(models, 25)

	require.NoError(cursor.Close())
}
