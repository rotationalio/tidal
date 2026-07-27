package store

import (
	"database/sql"
	"fmt"
	"strings"

	"go.rtnl.ai/tidal/conn"
	"go.rtnl.ai/tidal/errors"
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
	options Options
}

// New builds a CRUD store for table using the [model.Model] type M to derive SQL and parameters.
func New[M model.Model](table string, opts ...Option) *CRUD[M] {
	c := &CRUD[M]{
		options: makeOptions(opts...),
	}

	c.Queries = QuerySet{
		List:     c.ListQuery(table),
		Create:   c.CreateQuery(table),
		Retrieve: c.RetrieveQuery(table),
		Update:   c.UpdateQuery(table),
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
	query := c.Queries.Retrieve + id.Name + "=:" + id.Name
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

	// Construct the update query from the identifier if implemented, otherwise use the default identifier field.
	query := c.Queries.Update
	params := m.Params(model.Update)
	if identifier, ok := any(m).(model.Identifier); ok {
		// Get the identifier from the model and ensure it is in the parameters list.
		var identifiers []sql.NamedArg
		if identifiers = identifier.Identifiers(); len(identifiers) == 0 {
			return errors.ErrNoIdentifiers
		}

		// Check that all identifiers are in the parameters list and add them if not.
		for _, identifier := range identifiers {
			found := false
			for _, param := range params {
				if param.Name == identifier.Name {
					found = true
					break
				}
			}
			if !found {
				params = append(params, identifier)
			}
		}

		// Construct the WHERE clause from the identifiers.
		// For performance, only use the string builder for composite identifiers.
		if len(identifiers) == 1 {
			query += identifiers[0].Name + "=:" + identifiers[0].Name
		} else {
			sb := strings.Builder{}
			for i, identifier := range identifiers {
				if i > 0 {
					sb.WriteString(" AND ")
				}
				sb.WriteString(identifier.Name + "=:" + identifier.Name)
			}
			query += sb.String()
		}
	} else {
		// Use the default identifier field.
		query += c.options.IDField + "=:" + c.options.IDField

		// Check that the default identifier field is in the parameters list.
		found := false
		for _, param := range params {
			if param.Name == c.options.IDField {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("default identifier field %q not found in update parameters", c.options.IDField)
		}
	}

	// Add a limit clause to ensure only one row is updated if the option is enabled.
	if c.options.UpdateLimit {
		query += " LIMIT 1"
	}

	var result sql.Result
	if result, err = tx.Exec(query, params...); err != nil {
		return err
	}

	if nRows, _ := result.RowsAffected(); nRows == 0 {
		return errors.ErrNotFound
	}

	return nil
}

// Delete removes the row where the id column equals id.
func (c *CRUD[M]) Delete(tx conn.Tx, id sql.NamedArg) (result sql.Result, err error) {
	query := c.Queries.Delete + id.Name + "=:" + id.Name
	return tx.Exec(query, id)
}

// Fields returns the column names for op, cached after the first call.
func (c *CRUD[M]) Fields(op model.Operation) (fields []string) {
	m := model.Make[M]()
	return m.Fields(op)
}

// Params returns column names and :name placeholders for op.
func (c *CRUD[M]) Params(op model.Operation) (fields []string, placeholders []string) {
	m := model.Make[M]()
	params := m.Params(op)
	if len(params) == 0 {
		return nil, nil
	}

	fields = make([]string, len(params))
	placeholders = make([]string, len(params))

	for i, param := range params {
		fields[i] = param.Name
		placeholders[i] = ":" + param.Name
	}

	return fields, placeholders
}

// ListQuery builds the SELECT used by [CRUD.List].
func (c *CRUD[M]) ListQuery(table string) string {
	return fmt.Sprintf("SELECT %s FROM %s", FieldList("", c.Fields(model.List)), table)
}

// CreateQuery builds the INSERT used by [CRUD.Create].
func (c *CRUD[M]) CreateQuery(table string) string {
	fields, placeholders := c.Params(model.Create)
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, FieldList("", fields), strings.Join(placeholders, ", "))
}

// RetrieveQuery builds the SELECT prefix used by [CRUD.Retrieve] (caller appends the id predicate).
func (c *CRUD[M]) RetrieveQuery(table string) string {
	return fmt.Sprintf("SELECT %s FROM %s WHERE ", FieldList("", c.Fields(model.Retrieve)), table)
}

// UpdateQuery builds the UPDATE used by [CRUD.Update].
func (c *CRUD[M]) UpdateQuery(table string) string {
	fields, placeholders := c.Params(model.Update)
	setters := make([]string, 0, len(fields))
	for i, field := range fields {
		if field == c.options.IDField {
			continue
		}
		setters = append(setters, fmt.Sprintf("%s=%s", field, placeholders[i]))
	}
	return fmt.Sprintf("UPDATE %s SET %s WHERE ", table, strings.Join(setters, ", "))
}

// DeleteQuery builds the DELETE prefix used by [CRUD.Delete] (caller appends the id predicate).
func (c *CRUD[M]) DeleteQuery(table string) string {
	return fmt.Sprintf("DELETE FROM %s WHERE ", table)
}
