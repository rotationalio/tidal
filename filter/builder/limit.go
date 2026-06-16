package builder

import "fmt"

//============================================================================
// Limit
//============================================================================

// Limit is a row cap rendered as a LIMIT clause.
type Limit int

// Returns the limit as a string for ANSI SQL.
func (s Limit) String() string {
	if s >= 0 {
		return fmt.Sprintf("LIMIT %d", s)
	}
	return ""
}

//============================================================================
// Offset
//============================================================================

// Offset is a row skip rendered as an OFFSET clause.
type Offset int

// Returns the offset as a string for ANSI SQL.
func (s Offset) String() string {
	if s >= 0 {
		return fmt.Sprintf("OFFSET %d", s)
	}
	return ""
}
