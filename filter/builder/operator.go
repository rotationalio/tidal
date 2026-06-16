package builder

//============================================================================
// WhereOp
//============================================================================

// WhereOp is a comparison operator in a WHERE condition.
type WhereOp uint8

const (
	Eq WhereOp = iota
	Ne
	Gt
	Lt
	Gte
	Lte
	Like
	ILike
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
	default:
		return "unknown"
	}
}
