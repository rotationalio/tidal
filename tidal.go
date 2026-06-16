// Package tidal connects to SQL databases and runs typed CRUD queries without an ORM.
//
// Open a connection with [Open], start a transaction with [DB.BeginTx], and pass the
// transaction to [CRUD] methods from [New]. Write SQL with :name placeholders; tidal
// rewrites them for Postgres ($1, $2, …) or SQLite (?).
//
// Implement [Model] on your structs and embed [BaseModel] for IDs and timestamps.
// Use [ListFilter] ([Filter] or [Clause]) to sort and paginate list queries.
//
// Subpackages (bind, conn, model, filter, store) are public for advanced use;
// most applications import tidal only.
//
// Example:
//
//	package db
//
//	import (
//		"context"
//		"database/sql"
//		"os"
//
//		"go.rtnl.ai/tidal"
//		"go.rtnl.ai/x/dsn"
//	)
//
//	func Connect(ctx context.Context) (*tidal.DB, error) {
//		uri, err := dsn.Parse(os.Getenv("DATABASE_URL"))
//		if err != nil {
//			return nil, err
//		}
//		return tidal.Open(ctx, uri)
//	}
//
//	func createUser(ctx context.Context, db *tidal.DB) error {
//		crud := tidal.New[*User]("users")
//
//		tx, err := db.BeginTx(ctx, nil)
//		if err != nil {
//			return err
//		}
//		defer tx.Rollback()
//
//		user := &User{Name: "Ada"}
//		if _, err = crud.Create(tx, user); err != nil {
//			return err
//		}
//		return tx.Commit()
//	}
package tidal

import (
	"database/sql"

	"go.rtnl.ai/tidal/bind"
	"go.rtnl.ai/tidal/conn"
	"go.rtnl.ai/tidal/filter"
	"go.rtnl.ai/tidal/filter/builder"
	"go.rtnl.ai/tidal/model"
	"go.rtnl.ai/tidal/store"
)

// Connection

type (
	DB       = conn.DB
	Tx       = conn.Tx
	Txn      = conn.Txn
	Row      = conn.Row
	Beginner = conn.Beginner
)

var (
	Open = conn.Open
	Wrap = conn.Wrap
)

// Binding

type (
	PlaceholderType = bind.PlaceholderType
	BoundQuery      = bind.BoundQuery
)

const (
	UnknownPlaceholder = bind.UnknownPlaceholder
	Positional         = bind.Positional
	Ordered            = bind.Ordered
	Named              = bind.Named
	AtP                = bind.AtP
)

var (
	Rewrite        = bind.Rewrite
	PlaceholderFor = bind.PlaceholderFor
)

// Model

type (
	Model     = model.Model
	Scanner   = model.Scanner
	Preparer  = model.Preparer
	Validator = model.Validator
	BaseModel = model.BaseModel
	Operation = model.Operation
)

const (
	Unknown  = model.Unknown
	List     = model.List
	Create   = model.Create
	Retrieve = model.Retrieve
	Update   = model.Update
	Delete   = model.Delete
)

// Make returns a new zero value of M, allocating a pointer when M is a pointer type.
func Make[M Model]() M {
	// TODO: add doc comment from model.Make[M]() above anytime it is
	// updated; Go will pass through docs for type aliases but not for
	// function redirects
	return model.Make[M]()
}

var (
	SelectOperations  = model.SelectOperations
	EditOperations    = model.EditOperations
	PrepareOperations = model.PrepareOperations
)

// Filter

type (
	ListFilter     = filter.ListFilter
	CustomFilter   = filter.CustomFilter
	Filter         = filter.Filter
	Ordering       = builder.Ordering
	OrderBy        = builder.OrderBy
	OrderDirection = builder.OrderDirection
	Limit          = builder.Limit
	Offset         = builder.Offset

	// Alias for [CustomFilter] for backwards compatibility; use [CustomFilter]
	// instead.
	Clause = CustomFilter
)

const (
	OrderASC  = builder.OrderASC
	OrderDESC = builder.OrderDESC
)

// Store

type (
	QuerySet        = store.QuerySet
	CRUD[M Model]   = store.CRUD[M]
	Cursor[M Model] = store.Cursor[M]
)

// New builds a CRUD store for table using the [Model] type M to derive SQL and parameters.
func New[M Model](table string) *CRUD[M] {
	// TODO: add doc comment from store.New[M](table) above anytime it is
	// updated; Go will pass through docs for type aliases but not for
	// function redirects
	return store.New[M](table)
}

// Rows wraps [sql.Rows] as a [Cursor] for type M.
func Rows[M Model](tx Tx, rows *sql.Rows) Cursor[M] {
	// TODO: add doc comment from store.Rows[M](tx, rows) above anytime it is
	// updated; Go will pass through docs for type aliases but not for
	// function redirects
	return store.Rows[M](tx, rows)
}

// Empty returns a [Cursor] that returns no rows and always returns the provided
// error, which may be nil.
func Empty[M Model](err error) Cursor[M] {
	// TODO: add doc comment from store.Empty[M](err) above anytime it is
	// updated; Go will pass through docs for type aliases but not for
	// function redirects
	return store.Empty[M](err)
}

// Errors

var (
	ErrMissingID              = model.ErrMissingID
	ErrNotFound               = store.ErrNotFound
	ErrUnsupportedPlaceholder = bind.ErrUnsupportedPlaceholder
	ErrConnectionOptions      = conn.ErrConnectionOptions
	ErrConnect                = conn.ErrConnect
	ErrPing                   = conn.ErrPing
)

type (
	UnsupportedProvider = conn.UnsupportedProvider
	MissingArgument     = bind.MissingArgument
)
