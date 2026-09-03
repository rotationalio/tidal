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
	// Open opens a database connection for the provided DSN.
	Open = conn.Open

	// Wrap wraps an existing database connection.
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
	WhereOp      = filter.WhereOp
	Subquery     = filter.Subquery
)

var (
	// NewFilter creates an empty filter.
	NewFilter = filter.New

	// Where creates a filter with an initial WHERE condition.
	Where = filter.Where

	// Subselect builds a trusted SQL subquery for use with In, NotIn, Any, or All.
	Subselect = filter.Subselect

	// OrderBy creates a filter with an ORDER BY clause.
	OrderBy = filter.OrderBy

	// Limit creates a filter with a LIMIT clause.
	Limit = filter.Limit

	// Offset creates a filter with an OFFSET clause.
	Offset = filter.Offset
)

// Where Operations

const (
	Eq                = builder.Eq
	Ne                = builder.Ne
	Gt                = builder.Gt
	Lt                = builder.Lt
	Gte               = builder.Gte
	Lte               = builder.Lte
	Like              = builder.Like
	ILike             = builder.ILike
	In                = builder.In
	NotIn             = builder.NotIn
	Is                = builder.Is
	IsNot             = builder.IsNot
	IsDistinctFrom    = builder.IsDistinctFrom
	IsNotDistinctFrom = builder.IsNotDistinctFrom

	// Literals (used with [Is] and [IsNot])

	Null    = builder.Null
	True    = builder.True
	False   = builder.False
	Unknown = builder.Unknown

	// Ordering

	OrderASC  = builder.OrderASC
	OrderDESC = builder.OrderDESC
)

var (
	// Any builds an ANY comparison from a comparison operator. Unsupported
	// operators are rendered as provided and may fail when the database executes
	// the query.
	Any = filter.Any

	// All builds an ALL comparison from a comparison operator. Unsupported
	// operators are rendered as provided and may fail when the database executes
	// the query.
	All = filter.All
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
	ErrNotFound               = errors.ErrNotFound
	ErrNotNull                = errors.ErrNotNull
	ErrReadOnly               = errors.ErrReadOnly
	ErrAlreadyExists          = errors.ErrAlreadyExists
	ErrConstraint             = errors.ErrConstraint
	ErrMissingReference       = errors.ErrMissingReference
	ErrDeleteRestricted       = errors.ErrDeleteRestricted
	ErrConnectionOptions      = errors.ErrConnectionOptions
	ErrConnect                = errors.ErrConnect
	ErrPing                   = errors.ErrPing
	ErrUnsupportedPlaceholder = errors.ErrUnsupportedPlaceholder
	ErrMissingID              = errors.ErrMissingID
	ErrInvalidIdentifier      = errors.ErrInvalidIdentifier
	ErrMissingAssociation     = errors.ErrMissingAssociation
	ErrNoIdentifiers          = errors.ErrNoIdentifiers
)

type (
	UnsupportedProvider = errors.UnsupportedProvider
	MissingArgument     = errors.MissingArgument
	Error               = errors.Error
)
