package tidal

import (
	"context"
	"database/sql"
	"errors"

	"go.rtnl.ai/x/dsn"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// DB is a database connection that knows its provider and returns Tx transactions
// with automatic :name placeholder rewriting. Use [Open] to connect to a
// database, or [Wrap] to wrap an existing [sql.DB].
type DB struct {
	*sql.DB
	provider string
}

// Open connects to the database described by uri.
func Open(ctx context.Context, uri *dsn.DSN) (*DB, error) {
	sqlDB, err := open(ctx, uri)
	if err != nil {
		return nil, err
	}
	return Wrap(sqlDB, uri), nil
}

// Wraps an existing [sql.DB] with the provider from uri.
func Wrap(sqlDB *sql.DB, uri *dsn.DSN) *DB {
	return &DB{
		DB:       sqlDB,
		provider: uri.Provider,
	}
}

// Provider returns the DSN provider (for example dsn.Postgres or dsn.SQLite3).
func (db *DB) Provider() string {
	return db.provider
}

// BeginTx starts a transaction with automatic placeholder binding for this DB's provider.
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	tx, err := db.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return newTxn(tx, db.provider), nil
}

// Opens a new database connection to the database described by uri.
func open(ctx context.Context, uri *dsn.DSN) (db *sql.DB, err error) {
	switch uri.Provider {
	case dsn.SQLite3:
		if db, err = sql.Open("sqlite3", uri.Path); err != nil {
			return nil, errors.Join(ErrConnect, err)
		}
	case dsn.Postgres:
		connStr, pgopts, err := dsn.PGConnectionOptions(uri, nil)
		if err != nil {
			return nil, errors.Join(ErrConnectionOptions, err)
		}
		if db, err = sql.Open("postgres", connStr); err != nil {
			return nil, errors.Join(ErrConnect, err)
		}
		db.SetMaxIdleConns(pgopts.MaxIdleConns)
		db.SetMaxOpenConns(pgopts.MaxOpenConns)
		db.SetConnMaxLifetime(pgopts.ConnMaxLifetime)
		db.SetConnMaxIdleTime(pgopts.ConnMaxIdleTime)
	default:
		return nil, &UnsupportedProviderError{Provider: uri.Provider}
	}

	if err = db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, errors.Join(ErrPing, err)
	}
	return db, nil
}
