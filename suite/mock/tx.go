package mock

import (
	"database/sql"

	"go.rtnl.ai/tidal/conn"
)

const (
	Commit   = "Commit"
	Rollback = "Rollback"
	Query    = "Query"
	QueryRow = "QueryRow"
	Exec     = "Exec"
)

type Tx struct {
	Mock
	OnCommit   func() error
	OnRollback func() error
	OnQuery    func(query string, args ...sql.NamedArg) (*sql.Rows, error)
	OnQueryRow func(query string, args ...sql.NamedArg) *conn.Row
	OnExec     func(query string, args ...sql.NamedArg) (sql.Result, error)
}

//============================================================================
// Helper Methods
//============================================================================

func (t *Tx) ErrorOn(method string, err error) {
	switch method {
	case Commit:
		t.OnCommit = func() error { return err }
	case Rollback:
		t.OnRollback = func() error { return err }
	case Query:
		t.OnQuery = func(query string, args ...sql.NamedArg) (*sql.Rows, error) { return nil, err }
	case QueryRow:
		t.OnQueryRow = func(query string, args ...sql.NamedArg) *conn.Row { return nil }
	case Exec:
		t.OnExec = func(query string, args ...sql.NamedArg) (sql.Result, error) { return nil, err }
	}
}

func (t *Tx) Reset() {
	t.Mock.Reset()
	t.OnCommit = nil
	t.OnRollback = nil
	t.OnQuery = nil
	t.OnQueryRow = nil
	t.OnExec = nil
}

func (t *Tx) ResetCalls() {
	t.Mock.Reset()
}

//============================================================================
// Implement Tx Interface
//============================================================================

var _ conn.Tx = (*Tx)(nil)

func (t *Tx) Commit() error {
	t.increment(Commit)
	return t.OnCommit()
}

func (t *Tx) Rollback() error {
	t.increment(Rollback)
	return t.OnRollback()
}

func (t *Tx) Query(query string, args ...sql.NamedArg) (*sql.Rows, error) {
	t.increment(Query)
	return t.OnQuery(query, args...)
}

func (t *Tx) QueryRow(query string, args ...sql.NamedArg) *conn.Row {
	t.increment(QueryRow)
	return t.OnQueryRow(query, args...)
}

func (t *Tx) Exec(query string, args ...sql.NamedArg) (sql.Result, error) {
	t.increment(Exec)
	return t.OnExec(query, args...)
}
