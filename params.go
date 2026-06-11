package tidal

import (
	"database/sql"
	"fmt"
	"strings"
	"unicode"

	"go.rtnl.ai/x/dsn"
)

//============================================================================
// QueryParams
//============================================================================

// Rewrites canonical :name SQL and named arguments for the given driver style.
func QueryParams(query string, args []sql.NamedArg, ph PlaceholderType) (*BoundQuery, error) {
	switch ph {
	case Named:
		out := make([]any, len(args))
		for i, arg := range args {
			out[i] = arg
		}
		return &BoundQuery{query: query, values: out}, nil
	case Ordered:
		return rewriteQuery(query, args, orderedPlaceholder)
	case Positional:
		return rewriteQuery(query, args, positionalPlaceholder)
	case AtP:
		return rewriteQuery(query, args, atpPlaceholder)
	default:
		return nil, ErrUnsupportedPlaceholder
	}
}

//============================================================================
// Helper Functions
//============================================================================

// Selects the placeholder type from a DSN provider name.
func PlaceholderFor(provider string) PlaceholderType {
	switch provider {
	case dsn.Postgres:
		return Ordered
	case dsn.SQLite3:
		return Named
	default:
		return UnknownPlaceholder
	}
}

type placeholderFunc func(n int) string

// orderedPlaceholder formats a Postgres positional placeholder (ex: $1, $2, $3, etc.).
func orderedPlaceholder(n int) string { return fmt.Sprintf("$%d", n) }

// positionalPlaceholder formats a SQLite-style positional placeholder (always returns '?').
func positionalPlaceholder(_ int) string { return "?" }

// atpPlaceholder formats an SQL Server-style positional placeholder (ex: @p1, @p2, @p3, etc.).
func atpPlaceholder(n int) string { return fmt.Sprintf("@p%d", n) }

// rewriteQuery replaces :name tokens in left-to-right order and builds matching args.
func rewriteQuery(query string, args []sql.NamedArg, ph placeholderFunc) (*BoundQuery, error) {
	var (
		b      strings.Builder
		values []any
	)

	byName := make(map[string]any, len(args))
	for _, arg := range args {
		byName[arg.Name] = arg.Value
	}

	// Scan through each character in the input query string.
	for i := 0; i < len(query); {
		// Check if the current character is ':' indicating the start of a named placeholder (e.g., :name)
		if query[i] == ':' && i+1 < len(query) {
			// Continue scanning while the characters are valid identifier characters (letters, digits, or '_')
			j := i + 1
			for j < len(query) && isParamIdent(query[j]) {
				j++
			}

			// Extract the parameter name found after ':'
			if name := query[i+1 : j]; name != "" {
				// Look up the named argument value by name in byName map
				value, ok := byName[name]
				if !ok {
					// If not found, return an error indicating the missing argument
					return nil, &MissingArgumentError{Name: name}
				}

				// Add the value to the values slice that will be the args for the query
				values = append(values, value)

				// Write the corresponding placeholder into the query string (e.g., $1, ?, @p, etc.)
				b.WriteString(ph(len(values)))

				// Move index i to just after the parameter name we just substituted
				i = j
				continue
			}
		}
		// If current character is not the start of a placeholder, just copy it to the output buffer
		b.WriteByte(query[i])
		i++
	}

	return &BoundQuery{query: b.String(), values: values}, nil
}

// isParamIdent reports whether a byte may continue a :name placeholder identifier.
func isParamIdent(r byte) bool {
	return unicode.IsLetter(rune(r)) || unicode.IsDigit(rune(r)) || r == '_'
}

//============================================================================
// Placeholder Types
//============================================================================

// PlaceholderType selects how :name placeholders are rewritten for a database driver.
type PlaceholderType uint8

const (
	UnknownPlaceholder PlaceholderType = iota
	Positional
	Ordered
	Named
	AtP
)

//============================================================================
// BoundQuery
//============================================================================

// Holds a query and arguments ready for database/sql execution.
type BoundQuery struct {
	query  string
	values []any
}

// SQL returns the query string with placeholders rewritten for the target driver.
func (b *BoundQuery) SQL() string {
	return b.query
}

// Args returns argument values in the order required by the rewritten query.
func (b *BoundQuery) Args() []any {
	return b.values
}
