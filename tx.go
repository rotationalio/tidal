package tidal

import (
	"database/sql"
)

type Tx interface {
	Commit() error
	Rollback() error

	Query(query string, args ...sql.NamedArg) (*sql.Rows, error)
	QueryRow(query string, args ...sql.NamedArg) *sql.Row
	Exec(query string, args ...sql.NamedArg) (sql.Result, error)
}

func Wrap(tx *sql.Tx) Tx {
	return &Txn{
		Tx: tx,
	}
}

// Txn is a wrapper around a sql.Tx that implements the Tx interface by requiring
// sql.NamedArgs rather than any values being passed as variadic arguments.
type Txn struct {
	*sql.Tx
}

func (t *Txn) Exec(query string, args ...sql.NamedArg) (sql.Result, error) {
	return t.Tx.Exec(query, QueryArgs(args)...)
}

func (t *Txn) Query(query string, args ...sql.NamedArg) (*sql.Rows, error) {
	return t.Tx.Query(query, QueryArgs(args)...)
}

func (t *Txn) QueryRow(query string, args ...sql.NamedArg) *sql.Row {
	return t.Tx.QueryRow(query, QueryArgs(args)...)
}

func QueryArgs(args []sql.NamedArg) []any {
	out := make([]any, len(args))
	for i, arg := range args {
		out[i] = arg.Value
	}
	return out
}
