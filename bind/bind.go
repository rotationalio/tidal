// Package bind rewrites canonical :name SQL and named arguments for database drivers.
//
// Tidal transactions accept :name placeholders regardless of backend. This package
// converts them to the style the driver expects ($1 for Postgres, named for SQLite, etc.).
//
// Example:
//
//	query := "INSERT INTO users (id, email) VALUES (:id, :email)"
//	args := []sql.NamedArg{
//		sql.Named("id", id),
//		sql.Named("email", email),
//	}
//	bound, err := bind.Rewrite(query, args, bind.Ordered)
//	if err != nil {
//		return err
//	}
//	// bound.SQL() => "INSERT INTO users (id, email) VALUES ($1, $2)"
package bind

import (
	"database/sql"
	"fmt"
	"strings"
	"unicode"

	"go.rtnl.ai/x/dsn"
)

//============================================================================
// Rewrite
//============================================================================

// Rewrites canonical :name SQL and named arguments for the given driver style.
func Rewrite(query string, args []sql.NamedArg, ph PlaceholderType) (*BoundQuery, error) {
	switch ph {
	case Named:
		out := make([]any, len(args))
		for i, arg := range args {
			out[i] = arg
		}
		return &BoundQuery{query: query, values: out}, nil
	case Ordered:
		return rewriteQuery(query, args, orderedPlaceholder, true)
	case Positional:
		return rewriteQuery(query, args, positionalPlaceholder, false)
	case AtP:
		return rewriteQuery(query, args, atpPlaceholder, true)
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
// When reuseByName is true, numbered placeholders ($1, @p1) reuse the same index for
// repeated :name tokens. Anonymous placeholders cannot reuse a single arg slot, so
// reuseByName must be false for positional only placeholders (like '?').
//
// NOTE: This function is very performant, and caching is probably not needed; this
// was confirmed in benchmarks with cached vs uncached versions; the cached version
// was complex and only ~5% faster.
func rewriteQuery(query string, args []sql.NamedArg, ph placeholderFunc, reuseByName bool) (*BoundQuery, error) {
	var (
		b           strings.Builder
		values      []any
		indexByName map[string]int
	)

	if reuseByName {
		indexByName = make(map[string]int, len(args))
	}

	byName := make(map[string]any, len(args))
	for _, arg := range args {
		byName[arg.Name] = arg.Value
	}

	// Scan through each character in the input query string.
	for i := 0; i < len(query); {
		// Check if the current character is ':' indicating the start of a named placeholder (e.g., :name)
		if query[i] == ':' && i+1 < len(query) {
			// Postgres cast operator (::type) — copy through without binding.
			if query[i+1] == ':' {
				j := i + 2
				for j < len(query) && isParamIdent(query[j]) {
					j++
				}
				b.WriteString(query[i:j])
				i = j
				continue
			}

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
					return nil, MissingArgument(name)
				}

				// If the same :name token appears twice, reuse the same
				// placeholder index if reuseByName is true.
				if idx, ok := indexByName[name]; ok {
					b.WriteString(ph(idx))
				} else {
					values = append(values, value)
					idx := len(values)
					if reuseByName {
						indexByName[name] = idx
					}
					b.WriteString(ph(idx))
				}

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
