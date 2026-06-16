package store

import (
	"database/sql"

	"go.rtnl.ai/tidal/conn"
	"go.rtnl.ai/tidal/model"
)

// Cursor is a generic interface that allows for iteration and instantiation over a
// collection of models usually from the rows of a database query.
type Cursor[M model.Model] interface {
	// Advances the cursor to the next row; returns false when there are no more
	// rows.
	Next() bool

	// Returns the current row as a model instance. It is only valid after
	// [Cursor.Next] has been called and returned true.
	Model() (M, error)

	// Reads every remaining row into a slice.
	List() ([]M, error)

	// Releases the cursor and ends its associated transaction. Use
	// [Cursor.CloseRows] when you want to reuse the transaction for subsequent
	// operations.
	Close() error

	// Releases the result set without ending the transaction. Use this when
	// the same transaction will run more queries after the cursor is closed.
	CloseRows() error

	// Returns the first error encountered during iteration.
	Err() error
}

//============================================================================
// SQL Rows Cursor
//============================================================================

// Rows wraps [sql.Rows] as a [Cursor] for type M.
func Rows[M model.Model](tx conn.Tx, rows *sql.Rows) Cursor[M] {
	return &rowsCursor[M]{
		tx:   tx,
		rows: rows,
	}
}

type rowsCursor[M model.Model] struct {
	tx   conn.Tx
	rows *sql.Rows
}

func (c *rowsCursor[M]) Next() bool {
	return c.rows.Next()
}

func (c *rowsCursor[M]) Model() (M, error) {
	m := model.Make[M]()
	if err := m.Scan(model.List, c.rows); err != nil {
		return m, err
	}
	return m, nil
}

func (c *rowsCursor[M]) List() (models []M, err error) {
	models = make([]M, 0)
	for c.Next() {
		var model M
		if model, err = c.Model(); err != nil {
			return nil, err
		}
		models = append(models, model)
	}
	return models, c.Err()
}

func (c *rowsCursor[M]) Close() error {
	c.tx.Rollback()
	return c.rows.Close()
}

func (c *rowsCursor[M]) CloseRows() error {
	return c.rows.Close()
}

func (c *rowsCursor[M]) Err() error {
	return c.rows.Err()
}

//============================================================================
// Empty Cursor
//============================================================================

// Empty returns a [Cursor] that returns no rows and always returns the provided
// error, which may be nil.
func Empty[M model.Model](err error) Cursor[M] {
	return &emptyCursor[M]{
		err: err,
	}
}

type emptyCursor[M model.Model] struct {
	err error
}

func (c *emptyCursor[M]) Next() bool {
	return false
}

func (c *emptyCursor[M]) Model() (model M, _ error) {
	return model, c.err
}

func (c *emptyCursor[M]) List() ([]M, error) {
	return nil, c.err
}

func (c *emptyCursor[M]) Close() error {
	return c.err
}

func (c *emptyCursor[M]) CloseRows() error {
	return c.err
}

func (c *emptyCursor[M]) Err() error {
	return c.err
}
