package tidal

import "database/sql"

// Cursor is a generic interface that allows for iteration and instantiation over a
// collection of models usually from the rows of a database query.
type Cursor[M Model] interface {
	Next() bool
	Model() (M, error)
	List() ([]M, error)
	Close() error
	Err() error
}

//============================================================================
// SQL Rows Cursor
//============================================================================

func Rows[M Model](tx Tx, rows *sql.Rows) Cursor[M] {
	return &rowsCursor[M]{
		tx:   tx,
		rows: rows,
	}
}

type rowsCursor[M Model] struct {
	tx   Tx
	rows *sql.Rows
}

func (c *rowsCursor[M]) Next() bool {
	return c.rows.Next()
}

func (c *rowsCursor[M]) Model() (M, error) {
	model := Make[M]()
	if err := model.Scan(List, c.rows); err != nil {
		return model, err
	}
	return model, nil
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

func (c *rowsCursor[M]) Err() error {
	return c.rows.Err()
}

//============================================================================
// Empty Cursor
//============================================================================

func Empty[M Model](err error) Cursor[M] {
	return &emptyCursor[M]{
		err: err,
	}
}

type emptyCursor[M Model] struct {
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

func (c *emptyCursor[M]) Err() error {
	return c.err
}
