package conn

import (
	"database/sql"

	"go.rtnl.ai/tidal/bind"
	"go.rtnl.ai/tidal/errors"
	"go.rtnl.ai/x/dsn"
)

// Tx is a database transaction that accepts canonical :name SQL and named arguments.
type Tx interface {
	Commit() error
	Rollback() error

	Query(query string, args ...sql.NamedArg) (*sql.Rows, error)
	QueryRow(query string, args ...sql.NamedArg) *Row
	Exec(query string, args ...sql.NamedArg) (sql.Result, error)
}

// Wraps a [sql.Tx] and returns a Tx that rewrites placeholders for the given
// database connection.
func newTxn(tx *sql.Tx, uri *dsn.DSN) Tx {
	return &Txn{
		Tx:          tx,
		dsn:         uri.Clone(),
		placeholder: bind.PlaceholderFor(uri.Provider),
		mapErr:      errorMapperFor(uri.Provider),
	}
}

// Wraps [sql.Tx] and binds queries through [Rewrite] before execution.
type Txn struct {
	*sql.Tx
	dsn         *dsn.DSN
	placeholder bind.PlaceholderType
	mapErr      func(error) error
}

// Returns the DSN provider (for example [dsn.Postgres] or [dsn.SQLite3]).
func (t *Txn) Provider() string {
	return t.dsn.Provider
}

// Returns a copy of the connection DSN.
func (t *Txn) DSN() *dsn.DSN {
	return t.dsn.Clone()
}

// Returns the placeholder type for the configured database connection.
func (t *Txn) Placeholder() bind.PlaceholderType {
	return t.placeholder
}

// Exec runs a query after rewriting placeholders for the configured driver.
func (t *Txn) Exec(query string, args ...sql.NamedArg) (sql.Result, error) {
	p, err := bind.Rewrite(query, args, t.placeholder)
	if err != nil {
		return nil, t.mapErr(err)
	}
	result, err := t.Tx.Exec(p.SQL(), p.Args()...)
	return result, t.mapErr(err)
}

// Query runs a query after rewriting placeholders for the configured driver.
func (t *Txn) Query(query string, args ...sql.NamedArg) (*sql.Rows, error) {
	p, err := bind.Rewrite(query, args, t.placeholder)
	if err != nil {
		return nil, t.mapErr(err)
	}
	rows, err := t.Tx.Query(p.SQL(), p.Args()...)
	return rows, t.mapErr(err)
}

// QueryRow runs a query after rewriting placeholders for the configured driver.
func (t *Txn) QueryRow(query string, args ...sql.NamedArg) *Row {
	p, err := bind.Rewrite(query, args, t.placeholder)
	if err != nil {
		return &Row{err: err, mapErr: t.mapErr}
	}
	return &Row{row: t.Tx.QueryRow(p.SQL(), p.Args()...), mapErr: t.mapErr}
}

// errorMapperFor returns the database error mapper for provider. If the
// provider is not supported, returns the identity function (no-op).
func errorMapperFor(provider string) func(error) error {
	switch provider {
	case dsn.Postgres:
		return errors.PostgresError
	case dsn.SQLite3:
		return errors.SQLiteError
	default:
		return func(err error) error { return err }
	}
}
