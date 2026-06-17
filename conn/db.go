package conn

import (
	"context"
	"database/sql"
	"errors"

	"go.rtnl.ai/x/dsn"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

// Beginner is implemented by [DB]. Optional tidal packages such as migrations depend
// on this interface instead of the root tidal facade.
type Beginner interface {
	SQLDB() *sql.DB
	Provider() string

	BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)
	BeginReadTx(ctx context.Context) (Tx, error)
	WithTx(ctx context.Context, opts *sql.TxOptions, fn func(Tx) error) error
	WithReadTx(ctx context.Context, fn func(Tx) error) error
}

var _ Beginner = (*DB)(nil)

// SQLDB returns the underlying [sql.DB] connection pool.
func (db *DB) SQLDB() *sql.DB {
	return db.DB
}

// DB is a database connection that knows its provider and returns Tx transactions
// with automatic :name placeholder rewriting. Use [Open] to connect to a
// database, or [Wrap] to wrap an existing [sql.DB].
type DB struct {
	*sql.DB
	dsn *dsn.DSN
}

// Open connects to the database described by uri.
//
// NOTE: after successfully opening a connection this method pings the database
// to check liveness and connectivity. The database must be ready before calling
// Open.
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
		DB:  sqlDB,
		dsn: uri.Clone(),
	}
}

// Provider returns the DSN provider (for example dsn.Postgres or dsn.SQLite3).
func (db *DB) Provider() string {
	return db.dsn.Provider
}

// DSN returns a copy of the connection DSN.
func (db *DB) DSN() *dsn.DSN {
	return db.dsn.Clone()
}

// BeginTx starts a transaction with automatic placeholder binding for this DB's provider.
func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	if opts == nil {
		opts = &sql.TxOptions{
			ReadOnly: db.dsn.Options.ReadOnly(),
		}
	}

	if db.dsn.Options.ReadOnly() && !opts.ReadOnly {
		return nil, ErrReadOnly
	}

	tx, err := db.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return newTxn(tx, db.dsn), nil
}

// BeginReadTx starts a read-only transaction with automatic placeholder binding for this DB's provider.
func (db *DB) BeginReadTx(ctx context.Context) (Tx, error) {
	return db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
}

// WithTx runs a function with a transaction for this DB's provider.
func (db *DB) WithTx(ctx context.Context, opts *sql.TxOptions, fn func(Tx) error) (err error) {
	var tx Tx
	if tx, err = db.BeginTx(ctx, opts); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = fn(tx); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// WithReadTx runs a function with a read-only transaction for this DB's provider.
func (db *DB) WithReadTx(ctx context.Context, fn func(Tx) error) (err error) {
	return db.WithTx(ctx, &sql.TxOptions{ReadOnly: true}, fn)
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
		return nil, UnsupportedProvider(uri.Provider)
	}

	if err = db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, errors.Join(ErrPing, err)
	}
	return db, nil
}
