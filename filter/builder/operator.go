package builder

//============================================================================
// WhereOp
//============================================================================

// WhereOp is a comparison operator in a WHERE condition.
type WhereOp uint8

const (
	// Eq represents the SQL '=' operator for equality comparison.
	Eq WhereOp = iota

	// Ne represents the SQL '!=' operator for inequality comparison (not equal).
	Ne

	// Gt represents the SQL '>' operator for greater-than comparison.
	Gt

	// Lt represents the SQL '<' operator for less-than comparison.
	Lt

	// Gte represents the SQL '>=' operator for greater-than-or-equal-to comparison.
	Gte

	// Lte represents the SQL '<=' operator for less-than-or-equal-to comparison.
	Lte

	// Like represents the SQL 'LIKE' operator for pattern matching.
	Like

	// ILike represents the SQL 'ILIKE' operator for case-insensitive pattern matching.
	// Not all database providers support it.
	ILike

	// IsNull is the SQL 'IS NULL' operator for checking if a value is NULL.
	// Deprecated: use [Is] with [Null] instead.
	IsNull

	// IsNotNull is the SQL 'IS NOT NULL' operator for checking if a value is not NULL.
	// Deprecated: use [IsNot] with [Null] instead.
	IsNotNull

	// In represents the SQL 'IN' operator for matching any value in a set/list.
	In

	// Is is the SQL 'IS' operator. Use with a [Literal] for NULL and boolean predicates
	// (for example, field IS NULL), or with any other value for null-safe equality
	// (for example, field IS :param). Not all database providers support every form;
	// invalid combinations fail at query time.
	Is

	// IsNot is the SQL 'IS NOT' operator. Use with a [Literal] for NULL and boolean
	// predicates (for example, field IS NOT NULL), or with any other value for
	// null-safe inequality. Not all database providers support every form;
	// invalid combinations fail at query time.
	IsNot

	// IsDistinctFrom is the SQL 'IS DISTINCT FROM' operator for null-safe inequality.
	// Not all database providers support it.
	IsDistinctFrom

	// IsNotDistinctFrom is the SQL 'IS NOT DISTINCT FROM' operator for null-safe equality.
	// Not all database providers support it.
	IsNotDistinctFrom
)

// Returns the operator as a string for ANSI SQL.
func (op WhereOp) String() string {
	switch op {
	case Eq:
		return "="
	case Ne:
		return "!="
	case Gt:
		return ">"
	case Lt:
		return "<"
	case Gte:
		return ">="
	case Lte:
		return "<="
	case Like:
		return "LIKE"
	case ILike:
		return "ILIKE"
	case IsNull:
		// Deprecated
		return "IS NULL"
	case IsNotNull:
		// Deprecated
		return "IS NOT NULL"
	case In:
		return "IN"
	case Is:
		return "IS"
	case IsNot:
		return "IS NOT"
	case IsDistinctFrom:
		return "IS DISTINCT FROM"
	case IsNotDistinctFrom:
		return "IS NOT DISTINCT FROM"
	default:
		return "unknown"
	}
}

//============================================================================
// Literal
//============================================================================

// Literal is a SQL keyword used as the right-hand side of [Is] and [IsNot]
// predicates. PostgreSQL accepts NULL, TRUE, FALSE, and UNKNOWN. SQLite also
// accepts arbitrary values via a bound parameter instead of a Literal.
type Literal uint8

const (
	// Null is the NULL keyword (for example field IS NULL).
	Null Literal = iota
	// True is the TRUE keyword (for example field IS TRUE).
	True
	// False is the FALSE keyword (for example field IS FALSE).
	False
	// Unknown is the UNKNOWN keyword (for example field IS UNKNOWN).
	Unknown
)

// Returns the keyword as a string for ANSI SQL.
func (lit Literal) String() string {
	switch lit {
	case Null:
		return "NULL"
	case True:
		return "TRUE"
	case False:
		return "FALSE"
	case Unknown:
		return "UNKNOWN"
	default:
		return "unknown"
	}
}
