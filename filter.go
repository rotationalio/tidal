package tidal

import (
	"database/sql"
	"fmt"
	"strings"
)

// ListFilters are an interface that allows for the construction of complex queries
// for listing models that include filtering, sorting, limiting and pagination.
type ListFilter interface {
	// Returns the complete filtering SQL clause including placeholders in the correct
	// order for ANSI SQL standard.
	Clause() string

	// Returns the parameters for the filter clause.
	Params() []sql.NamedArg
}

// Clause is a manual filtering mechanism that implements the ListFilter interface, but
// requires the user to manually construct the SQL clause and parameters.
type Clause struct {
	SQL  string
	Args []sql.NamedArg
}

func (c *Clause) Clause() string {
	return c.SQL
}

func (c *Clause) Params() []sql.NamedArg {
	return c.Args
}

// Simple Filter that implements the ListFilter interface. This filter does not perform
// any constraint or database-specific checks before returning the SQL clause, so may
// not be suitable for all database backends. It ensures that an ANSI SQL clause is
// returned and that the parameters in the query are in the correct order.
//
// TODO: Handle WHERE clauses.
type Filter struct {
	limit    *Limit
	offset   *Offset
	ordering Ordering
	params   []sql.NamedArg
}

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
			if strings.HasPrefix(column, "-") {
				f.ordering = append(f.ordering, OrderBy{field: strings.TrimPrefix(column, "-"), direction: OrderDESC})
			} else {
				f.ordering = append(f.ordering, OrderBy{field: column, direction: OrderASC})
			}
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
		l := Limit(n)
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
		o := Offset(n)
		f.offset = &o
	}
	return f
}

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

	return sb.String()
}

func (f *Filter) Params() []sql.NamedArg {
	return f.params
}

//============================================================================
// Ordering
//============================================================================

type Ordering []OrderBy

func (o Ordering) String() string {
	sb := strings.Builder{}
	sb.WriteString("ORDER BY ")
	for i, order := range o {
		sb.WriteString(order.String())
		if i < len(o)-1 {
			sb.WriteString(", ")
		}
	}
	return sb.String()
}

type OrderBy struct {
	field     string
	direction OrderDirection
}

func (o OrderBy) String() string {
	return fmt.Sprintf("%s %s", o.field, o.direction)
}

type OrderDirection uint8

const (
	OrderASC OrderDirection = iota
	OrderDESC
)

func (o OrderDirection) String() string {
	switch o {
	case OrderASC:
		return "ASC"
	case OrderDESC:
		return "DESC"
	default:
		return "unknown"
	}
}

//============================================================================
// Limit and Offset
//============================================================================

type Limit int

type Offset int

func (s Limit) String() string {
	if s >= 0 {
		return fmt.Sprintf("LIMIT %d", s)
	}
	return ""
}

func (s Offset) String() string {
	if s >= 0 {
		return fmt.Sprintf("OFFSET %d", s)
	}
	return ""
}
