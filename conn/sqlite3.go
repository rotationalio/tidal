package conn

import (
	"context"
	"database/sql"

	"go.rtnl.ai/tidal/errors"
	"go.rtnl.ai/x/dsn"
)

func openSQLite3(_ context.Context, uri *dsn.DSN) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", uri.Path)
	if err != nil {
		return nil, errors.Join(errors.ErrConnect, err)
	}
	return db, nil
}
