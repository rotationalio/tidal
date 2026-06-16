package store

import (
	"database/sql"
	"fmt"
	"strings"

	"go.rtnl.ai/tidal/conn"
	"go.rtnl.ai/tidal/filter"
	"go.rtnl.ai/tidal/model"
)

// A QuerySet is the precomputed SQL query strings for a given model. These are
// computed at initialization to avoid repeated string templating for every query.
type QuerySet struct {
	List     string // Must be ready to attach filter clauses using concatenation.
	Create   string // Will not be edited by the CRUD model.
	Retrieve string // Must not contain a WHERE clause.
	Update   string // Will not be edited by the CRUD model.
	Delete   string // Must not contain a WHERE clause.
}

// CRUD runs create, read, update, delete, and list operations for a [model.Model] type.
// Build one with [New] and call its methods inside a [conn.Tx].
type CRUD[M model.Model] struct {
	Queries QuerySet
	fields  map[model.Operation][]string
	params  map[model.Operation][]string
}

// New builds a CRUD store for table using the [model.Model] type M to derive SQL and parameters.
func New[M model.Model](table string) *CRUD[M] {
	c := &CRUD[M]{
		fields: make(map[model.Operation][]string),
		params: make(map[model.Operation][]string),
	}

	// Precompute the parameters for the all operations that have parameters.
	for _, op := range []model.Operation{model.List, model.Create, model.Retrieve, model.Update, model.Delete} {
		m := model.Make[M]()
		ps := m.Params(op)
		if len(ps) == 0 {
			continue
		}
		names := make([]string, len(ps))
		for i, p := range ps {
			names[i] = p.Name
		}
		c.params[op] = names
	}

	c.Queries = QuerySet{
		List:     c.ListQuery(table),
		Create:   c.CreateQuery(table),
		Retrieve: c.RetrieveQuery(table),
		Update:   c.UpdateQuery(table, "id"),
		Delete:   c.DeleteQuery(table),
	}
	return c
}

// List returns a [Cursor] over rows matching filter. Pass nil for no filter clause.
func (c *CRUD[M]) List(tx conn.Tx, filter filter.ListFilter) (_ Cursor[M], err error) {
	var params []sql.NamedArg
	query := c.Queries.List

	if filter != nil {
		if clause := filter.Clause(); clause != "" {
			query += " " + clause
		}
		params = filter.Params()
	}

	var rows *sql.Rows
	if rows, err = tx.Query(query, params...); err != nil {
		return nil, err
	}
	return Rows[M](tx, rows), nil
}

// Create inserts m. [Preparer] and [Validator] hooks run when implemented.
func (c *CRUD[M]) Create(tx conn.Tx, m M) (result sql.Result, err error) {
	if prepare, ok := any(m).(model.Preparer); ok {
		prepare.Prepare(model.Create)
	}
	if validator, ok := any(m).(model.Validator); ok {
		if err = validator.Validate(model.Create); err != nil {
			return nil, err
		}
	}
	return tx.Exec(c.Queries.Create, m.Params(model.Create)...)
}

// Retrieve loads the row where the id column equals id.
func (c *CRUD[M]) Retrieve(tx conn.Tx, id sql.NamedArg) (m M, err error) {
	m = model.Make[M]()
	query := c.Queries.Retrieve + id.Name + " = :" + id.Name
	if err = m.Scan(model.Retrieve, tx.QueryRow(query, id)); err != nil {
		return m, err
	}
	return m, nil
}

// Update saves m. Returns [ErrNotFound] when no row matches m's id.
func (c *CRUD[M]) Update(tx conn.Tx, m M) (err error) {
	if prepare, ok := any(m).(model.Preparer); ok {
		prepare.Prepare(model.Update)
	}
	if validator, ok := any(m).(model.Validator); ok {
		if err = validator.Validate(model.Update); err != nil {
			return err
		}
	}
	var result sql.Result
	if result, err = tx.Exec(c.Queries.Update, m.Params(model.Update)...); err != nil {
		return err
	}
	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes the row where the id column equals id.
func (c *CRUD[M]) Delete(tx conn.Tx, id sql.NamedArg) (result sql.Result, err error) {
	query := c.Queries.Delete + id.Name + " = :" + id.Name
	return tx.Exec(query, id)
}

// Fields returns the column names for op, cached after the first call.
func (c *CRUD[M]) Fields(op model.Operation) (fields []string) {
	fields, ok := c.fields[op]
	if !ok {
		m := model.Make[M]()
		fields = m.Fields(op)
		c.fields[op] = fields
	}
	return fields
}

// Params returns column names and :name placeholders for op.
func (c *CRUD[M]) Params(op model.Operation) (fields []string, placeholders []string) {
	names, ok := c.params[op]
	if !ok {
		m := model.Make[M]()
		ps := m.Params(op)
		names = make([]string, len(ps))
		for i, p := range ps {
			names[i] = p.Name
		}
		c.params[op] = names
	}
	placeholders = make([]string, len(names))
	for i, name := range names {
		placeholders[i] = ":" + name
	}
	return names, placeholders
}

// ListQuery builds the SELECT used by [CRUD.List].
func (c *CRUD[M]) ListQuery(table string) string {
	return fmt.Sprintf("SELECT %s FROM %s", strings.Join(c.Fields(model.List), ", "), table)
}

// CreateQuery builds the INSERT used by [CRUD.Create].
func (c *CRUD[M]) CreateQuery(table string) string {
	fields, placeholders := c.Params(model.Create)
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(fields, ", "), strings.Join(placeholders, ", "))
}

// RetrieveQuery builds the SELECT prefix used by [CRUD.Retrieve] (caller appends the id predicate).
func (c *CRUD[M]) RetrieveQuery(table string) string {
	return fmt.Sprintf("SELECT %s FROM %s WHERE ", strings.Join(c.Fields(model.Retrieve), ", "), table)
}

// UpdateQuery builds the UPDATE used by [CRUD.Update].
func (c *CRUD[M]) UpdateQuery(table string, fieldID string) string {
	fields, placeholders := c.Params(model.Update)
	setters := make([]string, 0, len(fields))
	for i, field := range fields {
		if field == fieldID {
			continue
		}
		setters = append(setters, fmt.Sprintf("%s=%s", field, placeholders[i]))
	}
	return fmt.Sprintf("UPDATE %s SET %s WHERE %s=:%s", table, strings.Join(setters, ", "), fieldID, fieldID)
}

// DeleteQuery builds the DELETE prefix used by [CRUD.Delete] (caller appends the id predicate).
func (c *CRUD[M]) DeleteQuery(table string) string {
	return fmt.Sprintf("DELETE FROM %s WHERE ", table)
}
