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

	// In represents the SQL 'IN' operator for matching any value in a set/list.
	In

	// NotIn represents the SQL 'NOT IN' operator for excluding values in a set/list.
	NotIn

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

	// BitAnd is the SQL '&' operator for bitwise AND.
	BitAnd

	// BitOr is the SQL '|' operator for bitwise OR.
	BitOr

	// BitXor is the SQL '^' operator for bitwise XOR.
	BitXor

	// BitNot is the SQL '~' operator for bitwise NOT.
	BitNot
)

const (
	quantifierAny  WhereOp = 1 << 6
	quantifierAll  WhereOp = 1 << 7
	quantifierMask         = quantifierAny | quantifierAll

	anyEq    = quantifierAny
	anyNe    = quantifierAny | Ne
	anyGt    = quantifierAny | Gt
	anyLt    = quantifierAny | Lt
	anyGte   = quantifierAny | Gte
	anyLte   = quantifierAny | Lte
	anyLike  = quantifierAny | Like
	anyILike = quantifierAny | ILike

	allEq    = quantifierAll
	allNe    = quantifierAll | Ne
	allGt    = quantifierAll | Gt
	allLt    = quantifierAll | Lt
	allGte   = quantifierAll | Gte
	allLte   = quantifierAll | Lte
	allLike  = quantifierAll | Like
	allILike = quantifierAll | ILike
)

// Builds an ANY comparison from a comparison operator. Unsupported operators
// are rendered as provided and may fail when the database executes the query.
func Any(op WhereOp) WhereOp {
	switch op {
	case Eq:
		return anyEq
	case Ne:
		return anyNe
	case Gt:
		return anyGt
	case Lt:
		return anyLt
	case Gte:
		return anyGte
	case Lte:
		return anyLte
	case Like:
		return anyLike
	case ILike:
		return anyILike
	default:
		return quantifierAny | op
	}
}

// Builds an ALL comparison from a comparison operator. Unsupported operators
// are rendered as provided and may fail when the database executes the query.
func All(op WhereOp) WhereOp {
	switch op {
	case Eq:
		return allEq
	case Ne:
		return allNe
	case Gt:
		return allGt
	case Lt:
		return allLt
	case Gte:
		return allGte
	case Lte:
		return allLte
	case Like:
		return allLike
	case ILike:
		return allILike
	default:
		return quantifierAll | op
	}
}

// Splits a quantified operator into its comparison and quantifier components.
func quantifiedOperator(op WhereOp) (WhereOp, string, bool) {
	var quantifier string
	switch op & quantifierMask {
	case quantifierAny:
		quantifier = "ANY"
	case quantifierAll:
		quantifier = "ALL"
	case quantifierMask:
		quantifier = "ANY ALL"
	default:
		return op, "", false
	}
	return op &^ quantifierMask, quantifier, true
}

// Reports whether an operator is a standard scalar comparison operator.
func isComparisonOperator(op WhereOp) bool {
	switch op {
	case Eq, Ne, Gt, Lt, Gte, Lte, Like, ILike:
		return true
	default:
		return false
	}
}

// Returns the operator as a string for ANSI SQL.
func (op WhereOp) String() string {
	if comparison, quantifier, ok := quantifiedOperator(op); ok {
		return comparison.String() + " " + quantifier
	}

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
	case In:
		return "IN"
	case NotIn:
		return "NOT IN"
	case Is:
		return "IS"
	case IsNot:
		return "IS NOT"
	case IsDistinctFrom:
		return "IS DISTINCT FROM"
	case IsNotDistinctFrom:
		return "IS NOT DISTINCT FROM"
	case BitAnd:
		return "&"
	case BitOr:
		return "|"
	case BitXor:
		return "^"
	case BitNot:
		return "~"
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
