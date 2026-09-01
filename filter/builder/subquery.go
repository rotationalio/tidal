package builder

import "strings"

// Subquery marks SQL that should be rendered as a subquery rather than bound as
// a string parameter. The SQL is inserted unchanged, so callers must ensure it
// is trusted and valid for the target database. Values passed directly as
// strings that begin with SELECT or WITH are recognized the same way.
type Subquery string

// Builds a trusted subquery value for use with [In], [NotIn], [Any], or [All].
func Subselect(query string) Subquery {
	return Subquery(query)
}

// Returns the SQL text when value represents a trusted subquery.
func subqueryValue(value any) (string, bool) {
	switch query := value.(type) {
	case Subquery:
		return string(query), true
	case string:
		// Accept the concise ticket/API form while preserving ordinary scalar
		// strings as bound values for IN and NOT IN.
		trimmed := strings.TrimSpace(query)
		upper := strings.ToUpper(trimmed)
		if strings.HasPrefix(upper, "SELECT ") || strings.HasPrefix(upper, "WITH ") {
			return trimmed, true
		}
	}
	return "", false
}
