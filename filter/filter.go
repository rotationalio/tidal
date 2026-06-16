// Package filter builds list-query clauses for filtering, sorting, and pagination.
//
// Use [Filter] for WHERE, ORDER BY, LIMIT, and OFFSET. Use [CustomFilter] when you
// need hand-written SQL (for example GROUP BY).
//
// Building a composable [Filter] (can use tidal aliases):
//
//	f := (&tidal.Filter{}).
//		Where("status", tidal.Eq, "active").
//		AndGroup(func(g *tidal.Where) {
//			g.Where("role", tidal.Eq, "admin").
//				Or("role", tidal.Eq, "editor").
//				Or("role", tidal.Eq, "viewer")
//		}).
//		OrderBy("-created").
//		Limit(20)
//	cursor, err := crud.List(tx, f)
//
// [CustomFilter] combined with [Filter] when you need custom SQL (for example
// GROUP BY, which requires 2 [Filter] clauses to construct the SQL):
//
//	where := (&tidal.Filter{}).Where("status", tidal.Eq, "active")
//	rest := (&tidal.Filter{}).OrderBy("-created").Limit(20)
//	listFilter := &tidal.CustomFilter{
//		SQL:  where.Clause() + " GROUP BY role " + rest.Clause(),
//		Args: where.Params(),
//	}
//	cursor, err := crud.List(tx, listFilter)
package filter

import (
	"database/sql"
	"strings"

	"go.rtnl.ai/tidal/filter/builder"
)

//============================================================================
// ListFilter Interface
//============================================================================

// ListFilters are an interface that allows for the construction of complex queries
// for listing models that include filtering, sorting, limiting and pagination.
type ListFilter interface {
	// Returns the complete filtering SQL clause including placeholders in the correct
	// order for ANSI SQL standard.
	Clause() string

	// Returns the parameters for the filter clause.
	Params() []sql.NamedArg
}

// Where builds a grouped WHERE expression inside [Filter.AndGroup] and [Filter.OrGroup].
type Where = builder.Where

//============================================================================
// Filter Implementation
//============================================================================

// Implements the [ListFilter] interface. This filter does not perform any
// constraint or database-specific checks before returning the SQL clause, so
// may not be suitable for all database backends. It ensures that an ANSI SQL
// clause is returned and that the parameters in the query are in the correct
// order and named.
type Filter struct {
	// Clauses

	whereClause *builder.Where
	limit       *builder.Limit
	offset      *builder.Offset
	ordering    builder.Ordering

	// Cache; use f.resetCache() to clear whenever the filter is modified.

	sql    string
	params []sql.NamedArg
}

func (f *Filter) resetCache() {
	f.sql = ""
	f.params = nil
}

//============================================================================
// Where Building Methods
//============================================================================

// Calling Where replaces any previously built WHERE clause and starts a new one.
func (f *Filter) Where(field string, op builder.WhereOp, value any) *Filter {
	f.ensureWhere().Set(field, op, value)
	return f
}

// Appends an AND condition to the current WHERE clause.
func (f *Filter) And(field string, op builder.WhereOp, value any) *Filter {
	f.ensureWhere().And(field, op, value)
	return f
}

// Appends an OR condition to the current WHERE clause.
func (f *Filter) Or(field string, op builder.WhereOp, value any) *Filter {
	f.ensureWhere().Or(field, op, value)
	return f
}

// Appends a parenthesized group joined with AND to the current WHERE clause.
func (f *Filter) AndGroup(fn func(*Where)) *Filter {
	f.ensureWhere().AndGroup(fn)
	return f
}

// Appends a parenthesized group joined with OR to the current WHERE clause.
func (f *Filter) OrGroup(fn func(*Where)) *Filter {
	f.ensureWhere().OrGroup(fn)
	return f
}

// Ensures that the WHERE clause is initialized and resets the [Filter] cache;
// if you modify the WHERE clause functionality on [Filter], make sure to call
// this method or call resetCache() after modifying the [Filter].
func (f *Filter) ensureWhere() *builder.Where {
	f.resetCache()
	if f.whereClause == nil {
		f.whereClause = &builder.Where{}
	}
	return f.whereClause
}

func (f *Filter) renderWhere() (string, []sql.NamedArg) {
	if f.whereClause == nil {
		return "", nil
	}
	return f.whereClause.Render()
}

//============================================================================
// Ordering Building Methods
//============================================================================

// Pass in the column names to order by. To clear the ordering pass in an empty slice.
// Note that the ordering is overwritten if this method is called multiple times.
//
// You can pass in ordering in multiple ways. If you pass a single column, it will be
// ordered ascending by default. You can also use -column to order descending. If you
// pass in multiple columns, the list will first be ordered by the first column, then by
// by subsequent columns to break ordering ties.
func (f *Filter) OrderBy(columns ...string) *Filter {
	f.resetCache()
	f.ordering = nil
	if len(columns) > 0 {
		for _, column := range columns {
			direction := builder.OrderASC
			if strings.HasPrefix(column, "-") {
				direction = builder.OrderDESC
			}
			f.ordering = append(f.ordering, builder.OrderBy{Column: strings.TrimPrefix(column, "-"), Direction: direction})
		}
	}
	return f
}

//============================================================================
// Limit and Offset Building Methods
//============================================================================

// Adds or overwrites the limit for the filter. To remove a limit already set on a
// filter pass in n=-1, which will clear the limit.
func (f *Filter) Limit(n int) *Filter {
	f.resetCache()
	if n < 0 {
		f.limit = nil
	} else {
		l := builder.Limit(n)
		f.limit = &l
	}
	return f
}

// Adds or overwrites the offset for the filter. To remove an offset already set on a
// filter pass in n=-1, which will clear the offset.
func (f *Filter) Offset(n int) *Filter {
	f.resetCache()
	if n < 0 {
		f.offset = nil
	} else {
		o := builder.Offset(n)
		f.offset = &o
	}
	return f
}

//============================================================================
// FilterList Implementation for Filter
//============================================================================

// Returns the SQL clause string for the filter.
func (f *Filter) Clause() string {
	if f.sql == "" {
		sb := strings.Builder{}
		concat := false

		// Build the WHERE clause.
		if whereSQL, _ := f.renderWhere(); whereSQL != "" {
			sb.WriteString(whereSQL)
			concat = true
		}

		// Build the ORDER BY clause.
		if len(f.ordering) > 0 {
			if concat {
				sb.WriteString(" ")
			}

			sb.WriteString(f.ordering.String())
			concat = true
		}

		// Build the LIMIT clause.
		if f.limit != nil {
			if concat {
				sb.WriteString(" ")
			}

			sb.WriteString(f.limit.String())
			concat = true
		}

		// Build the OFFSET clause.
		if f.offset != nil {
			if concat {
				sb.WriteString(" ")
			}

			sb.WriteString(f.offset.String())
		}

		// Render and cache the SQL clause.
		f.sql = sb.String()
	}
	return f.sql
}

// Returns the parameters for the filter clause.
func (f *Filter) Params() []sql.NamedArg {
	if len(f.params) == 0 {
		var params []sql.NamedArg

		// Render the WHERE parameters.
		_, params = f.renderWhere()

		// Cache the parameters.
		f.params = params
	}
	return f.params
}
