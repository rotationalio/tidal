package tidal

import (
	"database/sql"
	"fmt"
	"strings"
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

// TODO: the CRUD struct needs to know the PlaceholderType to use for the query.
type CRUD[M Model] struct {
	Queries QuerySet
	params  map[Operation]Params
	fields  map[Operation][]string
}

func New[M Model](table string) *CRUD[M] {
	c := &CRUD[M]{}
	c.Queries = QuerySet{
		List:     c.ListQuery(table),
		Create:   c.CreateQuery(table),
		Retrieve: c.RetrieveQuery(table),
		Update:   c.UpdateQuery(table, "id"),
		Delete:   c.DeleteQuery(table),
	}
	return c
}

func (c *CRUD[M]) List(tx Tx, filter ListFilter) (_ Cursor[M], err error) {
	// Add the filtering constraints to the query (if any).
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

func (c *CRUD[M]) Create(tx Tx, m M) (result sql.Result, err error) {
	if prepare, ok := any(m).(Preparer); ok {
		prepare.Prepare(Create)
	}

	if validator, ok := any(m).(Validator); ok {
		if err = validator.Validate(Create); err != nil {
			return nil, err
		}
	}

	if result, err = tx.Exec(c.Queries.Create, m.Params(Create)...); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *CRUD[M]) Retrieve(tx Tx, id sql.NamedArg) (m M, err error) {
	m = Make[M]()
	query := c.Queries.Retrieve + id.Name + " = :" + id.Name
	if err = m.Scan(Retrieve, tx.QueryRow(query, id)); err != nil {
		return m, err
	}
	return m, nil
}

func (c *CRUD[M]) Update(tx Tx, m M) (err error) {
	if prepare, ok := any(m).(Preparer); ok {
		prepare.Prepare(Update)
	}

	if validator, ok := any(m).(Validator); ok {
		if err = validator.Validate(Update); err != nil {
			return err
		}
	}

	var result sql.Result
	if result, err = tx.Exec(c.Queries.Update, m.Params(Update)...); err != nil {
		return err
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return ErrNotFound
	}
	return nil
}

func (c *CRUD[M]) Delete(tx Tx, id sql.NamedArg) (result sql.Result, err error) {
	query := c.Queries.Delete + id.Name + " = :" + id.Name
	if result, err = tx.Exec(query, id); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *CRUD[M]) Fields(op Operation) (fields []string) {
	var ok bool
	if fields, ok = c.fields[op]; !ok {
		var m M
		m = Make[M]()
		fields = m.Fields(op)
		c.fields[op] = fields
	}
	return fields
}

// TODO: cache so that we don't have to create a new model and params for every call.
func (c *CRUD[M]) Params(op Operation) (fields []string, placeholders []string) {
	m := Make[M]()
	params := m.Params(op)

	fields = make([]string, len(params))
	placeholders = make([]string, len(params))

	for i, param := range params {
		fields[i] = param.Name
		placeholders[i] = ":" + param.Name
	}
	return fields, placeholders
}

func (c *CRUD[M]) ListQuery(table string) string {
	return fmt.Sprintf("SELECT %s FROM %s", strings.Join(c.Fields(List), ", "), table)
}

func (c *CRUD[M]) CreateQuery(table string) string {
	fields, placeholders := c.Params(Create)
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(fields, ", "), strings.Join(placeholders, ", "))
}

func (c *CRUD[M]) RetrieveQuery(table string) string {
	return fmt.Sprintf("SELECT %s FROM %s WHERE ", strings.Join(c.Fields(Retrieve), ", "), table)
}

func (c *CRUD[M]) UpdateQuery(table string, fieldID string) string {
	fields, placeholders := c.Params(Update)
	setters := make([]string, 0, len(fields))

	for i, field := range fields {
		if field == fieldID {
			continue
		}
		setters = append(setters, fmt.Sprintf("%s=%s", field, placeholders[i]))
	}

	return fmt.Sprintf("UPDATE %s SET %s WHERE %s=:%s", table, strings.Join(setters, ", "), fieldID, fieldID)
}

func (c *CRUD[M]) DeleteQuery(table string) string {
	return fmt.Sprintf("DELETE FROM %s WHERE ", table)
}
