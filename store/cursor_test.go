package store_test

import (
	"database/sql"
	"fmt"
	"strings"

	"go.rtnl.ai/tidal"
	"go.rtnl.ai/tidal/store"
	"go.rtnl.ai/tidal/suite/fixtures"
)

func (s *StoreTestSuite) TestCursor() {
	require := s.Require()
	tx := s.BeginTx(&sql.TxOptions{
		ReadOnly: true,
	})
	defer tx.Rollback()

	user := &fixtures.User{}
	fields := strings.Join(user.Fields(tidal.List), ", ")

	rows, err := tx.Query(fmt.Sprintf("SELECT %s FROM users", fields))
	require.NoError(err)

	cursor := store.Rows[*fixtures.User](tx, rows)
	models, err := cursor.List()
	require.NoError(err)
	require.Len(models, 25)

	require.NoError(cursor.Close())
}
