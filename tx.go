package tidal

import (
	"database/sql"

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
		placeholder: PlaceholderFor(uri.Provider),
	}
}

// Wraps [sql.Tx] and binds queries through [Rewrite] before execution.
type Txn struct {
	*sql.Tx
	dsn         *dsn.DSN
	placeholder PlaceholderType
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
func (t *Txn) Placeholder() PlaceholderType {
	return t.placeholder
}

// Exec runs a query after rewriting placeholders for the configured driver.
func (t *Txn) Exec(query string, args ...sql.NamedArg) (sql.Result, error) {
	p, err := Rewrite(query, args, t.placeholder)
	if err != nil {
		return nil, err
	}
	return t.Tx.Exec(p.SQL(), p.Args()...)
}

// Query runs a query after rewriting placeholders for the configured driver.
func (t *Txn) Query(query string, args ...sql.NamedArg) (*sql.Rows, error) {
	p, err := Rewrite(query, args, t.placeholder)
	if err != nil {
		return nil, err
	}
	return t.Tx.Query(p.SQL(), p.Args()...)
}

// QueryRow runs a query after rewriting placeholders for the configured driver.
func (t *Txn) QueryRow(query string, args ...sql.NamedArg) *Row {
	p, err := Rewrite(query, args, t.placeholder)
	if err != nil {
		return &Row{err: err}
	}
	return &Row{row: t.Tx.QueryRow(p.SQL(), p.Args()...)}
}
