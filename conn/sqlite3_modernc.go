//go:build !mattn && !ncruces

package conn

import (
	"context"
	"database/sql"
	"time"

	"go.rtnl.ai/tidal/errors"
	"go.rtnl.ai/x/dsn"

	"modernc.org/sqlite"
)

const (
	driverName   = "modernc"
	sqliteDriver = "sqlite"
)

const moderncPragmas = `
	PRAGMA journal_mode = WAL;
	PRAGMA synchronous = NORMAL;
	PRAGMA temp_store = MEMORY;
	PRAGMA mmap_size = 30000000000; -- 30GB
	PRAGMA busy_timeout = 5000;
	PRAGMA automatic_index = true;
	PRAGMA foreign_keys = ON;
	PRAGMA analysis_limit = 1000;
	PRAGMA trusted_schema = OFF;
`

// Open a database connection using the modernc sqlite3 driver (default).
//
// Note there is some duplicate code here between the different sqlite3 drivers.
// This is necessary because of the driver import on line 12 that ensures the right
// database driver is used to open the connection.
func openSQLite3(_ context.Context, uri *dsn.DSN) (*sql.DB, error) {
	if uri.Driver != "" {
		if uri.Driver != driverName {
			return nil, errors.Join(errors.ErrConnect, errors.UnsupportedProvider(uri.Driver))
		}
	} else {
		uri.Driver = driverName
	}

	// Modern C requires PRAMGAs to be run on every connection. This hook ensures
	// that the the PRAGMAs are respected.
	// TODO: ensure DSN options are passed through to the driver.
	// TODO: optionally we could use a DSN connection string rather than uri.Path
	sqlite.RegisterConnectionHook(func(conn sqlite.ExecQuerierContext, _ string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_, err := conn.ExecContext(ctx, moderncPragmas, nil)
		return err
	})

	uri.Driver = driverName
	db, err := sql.Open(sqliteDriver, uri.Path)
	if err != nil {
		return nil, errors.Join(errors.ErrConnect, err)
	}
	return db, nil
}
