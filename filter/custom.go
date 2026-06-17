package filter

import "database/sql"

// Deprecated: use [CustomFilter] instead.
type Clause = CustomFilter

// Implements the [ListFilter] interface; the user must manually construct the
// SQL clause and parameter arguments to be returned.
//
// Example ([Filter] cannot do GROUP BY):
//
//	f := &filter.CustomFilter{
//		SQL:  "WHERE status = :status GROUP BY name",
//		Args: []sql.NamedArg{sql.Named("status", "active")},
//	}
//	cursor, err := crud.List(tx, f)
type CustomFilter struct {
	SQL  string
	Args []sql.NamedArg
}

// Returns the SQL clause string unchanged.
func (c *CustomFilter) Clause() string {
	return c.SQL
}

// Returns the parameters unchanged.
func (c *CustomFilter) Params() []sql.NamedArg {
	return c.Args
}
