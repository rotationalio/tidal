package builder

import (
	"fmt"
	"strings"
)

//============================================================================
// Ordering
//============================================================================

// Ordering is a list of sort columns rendered as an ORDER BY clause.
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

//============================================================================
// OrderBy
//============================================================================

// OrderBy is one column and sort direction in an [Ordering].
type OrderBy struct {
	Column    string
	Direction OrderDirection
}

// Returns the column and direction as a string for ANSI SQL.
func (o OrderBy) String() string {
	return fmt.Sprintf("%s %s", o.Column, o.Direction)
}

//============================================================================
// OrderDirection
//============================================================================

// OrderDirection is ASC or DESC in an [OrderBy].
type OrderDirection uint8

const (
	OrderASC OrderDirection = iota
	OrderDESC
)

// Returns the direction as a string for ANSI SQL.
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
