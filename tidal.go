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
	"go.rtnl.ai/tidal/conn"
	"go.rtnl.ai/tidal/errors"
	"go.rtnl.ai/tidal/filter"
	"go.rtnl.ai/tidal/filter/builder"
	"go.rtnl.ai/tidal/model"
	"go.rtnl.ai/tidal/store"
)

// Connection

type (
	DB = conn.DB
	Tx = conn.Tx
)

var (
	Open = conn.Open
	Wrap = conn.Wrap
)

// Model

type (
	BaseModel = model.BaseModel
	Model     = model.Model
	Scanner   = model.Scanner
	Preparer  = model.Preparer
	Validator = model.Validator
	Operation = model.Operation
)

// Make returns a new zero value of M, allocating a pointer when M is a pointer type.
func Make[M Model]() M {
	// TODO: add doc comment from model.Make[M]() above anytime it is
	// updated; Go will pass through docs for type aliases but not for
	// function redirects
	return model.Make[M]()
}

// CRUD Operations

const (
	List     = model.List
	Create   = model.Create
	Retrieve = model.Retrieve
	Update   = model.Update
	Delete   = model.Delete
)

// Conformance Test Scan Operations

var (
	SelectOperations  = model.SelectOperations
	EditOperations    = model.EditOperations
	PrepareOperations = model.PrepareOperations
)

// Filter

type (
	Filter       = filter.Filter
	CustomFilter = filter.CustomFilter
	ListFilter   = filter.ListFilter
	WhereGroup   = filter.WhereGroup
	WhereOp      = builder.WhereOp

	// Deprecated: use [CustomFilter] instead.
	Clause = CustomFilter
)

var (
	NewFilter = filter.New
	Where     = filter.Where
	OrderBy   = filter.OrderBy
	Limit     = filter.Limit
	Offset    = filter.Offset
)

const (
	// Where Operations

	Eq        = builder.Eq
	Ne        = builder.Ne
	Gt        = builder.Gt
	Lt        = builder.Lt
	Gte       = builder.Gte
	Lte       = builder.Lte
	Like      = builder.Like
	IsNull    = builder.IsNull
	IsNotNull = builder.IsNotNull
	In        = builder.In

	// Ordering

	OrderASC  = builder.OrderASC
	OrderDESC = builder.OrderDESC
)

// Store

type (
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

// Errors

var (
	ErrMissingID              = errors.ErrMissingID
	ErrNotFound               = errors.ErrNotFound
	ErrUnsupportedPlaceholder = errors.ErrUnsupportedPlaceholder
	ErrConnectionOptions      = errors.ErrConnectionOptions
	ErrConnect                = errors.ErrConnect
	ErrPing                   = errors.ErrPing
)

type (
	UnsupportedProvider = errors.UnsupportedProvider
	MissingArgument     = errors.MissingArgument
)
