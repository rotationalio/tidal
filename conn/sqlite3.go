package conn

import (
	"context"
	"database/sql"
	"errors"

	"go.rtnl.ai/x/dsn"
)

func openSQLite3(_ context.Context, uri *dsn.DSN) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", uri.Path)
	if err != nil {
		return nil, errors.Join(ErrConnect, err)
	}
	return db, nil
}
