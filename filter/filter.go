// Package filter builds list-query clauses for sorting and pagination.
//
// Use [Filter] for ORDER BY, LIMIT, and OFFSET. Combine with [CustomFilter] for WHERE
// conditions.
//
// Building a composable [Filter] and integrating with a custom [Clause]:
//
//	f := (&filter.Filter{}).OrderBy("-created").Limit(20)
//	listFilter := &filter.Clause{
//		SQL:  "WHERE status = :status " + f.Clause(),
//		Args: []sql.NamedArg{sql.Named("status", "active")},
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

//============================================================================
// Filter Implementation
//============================================================================

// Implements the [ListFilter] interface. This filter does not perform any
// constraint or database-specific checks before returning the SQL clause, so
// may not be suitable for all database backends. It ensures that an ANSI SQL
// clause is returned and that the parameters in the query are in the correct
// order and named.
//
// TODO: Handle WHERE clauses.
type Filter struct {
	limit    *builder.Limit
	offset   *builder.Offset
	ordering builder.Ordering
	params   []sql.NamedArg
}

//============================================================================
// Filter Building Methods
//============================================================================

// Pass in the column names to order by. To clear the ordering pass in an empty slice.
// Note that the ordering is overwritten if this method is called multiple times.
//
// You can pass in ordering in multiple ways. If you pass a single column, it will be
// ordered ascending by default. You can also use -column to order descending. If you
// pass in multiple columns, the list will first be ordered by the first column, then by
// by subsequent columns to break ordering ties.
func (f *Filter) OrderBy(columns ...string) *Filter {
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

// Adds or overwrites the limit for the filter. To remove a limit already set on a
// filter pass in n=-1, which will clear the limit.
func (f *Filter) Limit(n int) *Filter {
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
	sb := strings.Builder{}
	concat := false

	if len(f.ordering) > 0 {
		if concat {
			sb.WriteString(" ")
		}

		sb.WriteString(f.ordering.String())
		concat = true
	}

	if f.limit != nil {
		if concat {
			sb.WriteString(" ")
		}

		sb.WriteString(f.limit.String())
		concat = true
	}

	if f.offset != nil {
		if concat {
			sb.WriteString(" ")
		}

		sb.WriteString(f.offset.String())
	}

	// TODO: cache the SQL clause string; reset the cache when the filter is modified

	return sb.String()
}

// Returns the parameters for the filter clause.
func (f *Filter) Params() []sql.NamedArg {
	return f.params
}
