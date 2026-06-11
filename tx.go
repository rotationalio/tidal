package tidal

import (
	"database/sql"
)

// Tx is a database transaction that accepts canonical :name SQL and named arguments.
type Tx interface {
	Commit() error
	Rollback() error

	Query(query string, args ...sql.NamedArg) (*sql.Rows, error)
	QueryRow(query string, args ...sql.NamedArg) *Row
	Exec(query string, args ...sql.NamedArg) (sql.Result, error)
}

// Wraps a sql.Tx and returns a Tx that rewrites placeholders for the given DSN
// provider (for example [dsn.Postgres] or [dsn.SQLite3]).
func newTxn(tx *sql.Tx, provider string) Tx {
	return &Txn{
		Tx:          tx,
		placeholder: PlaceholderFor(provider),
	}
}

// Txn wraps sql.Tx and binds queries through QueryParams before execution.
type Txn struct {
	*sql.Tx
	placeholder PlaceholderType
}

// Exec runs a query after rewriting placeholders for the configured driver.
func (t *Txn) Exec(query string, args ...sql.NamedArg) (sql.Result, error) {
	p, err := QueryParams(query, args, t.placeholder)
	if err != nil {
		return nil, err
	}
	return t.Tx.Exec(p.SQL(), p.Args()...)
}

// Query runs a query after rewriting placeholders for the configured driver.
func (t *Txn) Query(query string, args ...sql.NamedArg) (*sql.Rows, error) {
	p, err := QueryParams(query, args, t.placeholder)
	if err != nil {
		return nil, err
	}
	return t.Tx.Query(p.SQL(), p.Args()...)
}

// QueryRow runs a query after rewriting placeholders for the configured driver.
func (t *Txn) QueryRow(query string, args ...sql.NamedArg) *Row {
	p, err := QueryParams(query, args, t.placeholder)
	if err != nil {
		return &Row{err: err}
	}
	return &Row{row: t.Tx.QueryRow(p.SQL(), p.Args()...)}
}
