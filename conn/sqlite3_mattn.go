//go:build mattn && !ncruces

package conn

import (
	"context"
	"database/sql"

	"go.rtnl.ai/tidal/errors"
	"go.rtnl.ai/x/dsn"

	_ "github.com/mattn/go-sqlite3"
)

const (
	driverName   = "mattn"
	sqliteDriver = "sqlite3"
)

func openSQLite3(_ context.Context, uri *dsn.DSN) (*sql.DB, error) {
	if uri.Driver != "" {
		if uri.Driver != driverName {
			return nil, errors.Join(errors.ErrConnect, errors.UnsupportedProvider(uri.Driver))
		}
	} else {
		uri.Driver = driverName
	}

	db, err := sql.Open(sqliteDriver, uri.Path)
	if err != nil {
		return nil, errors.Join(errors.ErrConnect, err)
	}
	return db, nil
}
