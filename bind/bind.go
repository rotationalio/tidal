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
//
// Benchmark note:
// Run `go test ./bind -run '^$' -bench '^BenchmarkRewrite$' -benchmem -count=1`
// to capture local Rewrite parser performance.
//
// Last observed benchmark on darwin/arm64 (Apple M2):
// - OrderedSimple: 937.9 ns/op, 198 B/op, 6 allocs/op
// - OrderedComplex: 1407 ns/op, 308 B/op, 5 allocs/op
// - PositionalSimple: 549.4 ns/op, 256 B/op, 4 allocs/op
func Rewrite(query string, args []sql.NamedArg, ph PlaceholderType) (*BoundQuery, error) {
	switch ph {
	case Named:
		out := make([]any, len(args))
		for i, arg := range args {
			out[i] = arg
		}
		return &BoundQuery{query: query, values: out}, nil
	case Ordered:
		return rewriteQuery(query, args, orderedPlaceholder, true, dsn.Postgres)
	case Positional:
		return rewriteQuery(query, args, positionalPlaceholder, false, "")
	case AtP:
		return rewriteQuery(query, args, atpPlaceholder, true, "")
	default:
		return nil, ErrUnsupportedPlaceholder
	}
}

//============================================================================
// Query Rewriting
//============================================================================

// rewriteQuery replaces :name tokens in left-to-right order and builds matching args.
// When reuseByName is true, numbered placeholders ($1, @p1) reuse the same index for
// repeated :name tokens. Anonymous placeholders cannot reuse a single arg slot, so
// reuseByName must be false for positional only placeholders (like '?').
// Placeholder parsing is quote-aware and comment-aware. Provider-specific branches
// are keyed from provider so parser behavior can grow per backend over time.
//
// NOTE: Use BenchmarkRewrite in bind_benchmark_test.go to monitor parser
// performance and compare runs manually across local changes.
func rewriteQuery(query string, args []sql.NamedArg, ph placeholderFunc, reuseByName bool, provider string) (*BoundQuery, error) {
	var (
		b      strings.Builder
		values []any
	)

	// Most output bytes are copied from the input query.
	b.Grow(len(query) + len(args)*2)
	values = make([]any, 0, len(args))

	// there is specific behavior for some providers
	isPostgres := provider == dsn.Postgres

	var (
		byName      map[string]any
		stateByName map[string]namedArgState
	)

	if reuseByName {
		// Reuse mode keeps both value and assigned placeholder index in one map.
		stateByName = make(map[string]namedArgState, len(args))
		for _, arg := range args {
			stateByName[arg.Name] = namedArgState{value: arg.Value}
		}
	} else {
		// Positional mode only needs value lookup by name.
		byName = make(map[string]any, len(args))
		for _, arg := range args {
			byName[arg.Name] = arg.Value
		}
	}

	// Scan through each character in the input query string.
	for i := 0; i < len(query); {
		// SQL comments are copied verbatim and never parsed for placeholders.
		if next, ok := copyLineComment(query, i, &b); ok {
			i = next
			continue
		}
		if next, ok := copyBlockComment(query, i, &b); ok {
			i = next
			continue
		}

		// PostgreSQL E'...' supports backslash escapes and needs its own branch.
		if isPostgres {
			if next, ok := copyPostgresEscapedString(query, i, &b); ok {
				i = next
				continue
			}

			// PostgreSQL dollar-quoted strings can contain ':' without binding.
			if next, ok := copyPostgresDollarQuotedString(query, i, &b); ok {
				i = next
				continue
			}
		}

		// Standard SQL single-quoted strings: :name is treated as literal text.
		if query[i] == '\'' {
			i = copySingleQuotedString(query, i, &b, false)
			continue
		}

		// Parse canonical :name placeholders in normal SQL code regions only.
		if query[i] == ':' && i+1 < len(query) {
			// Keep postgres casts (expr::type) untouched.
			if isPostgres && query[i+1] == ':' {
				j := i + 2
				for j < len(query) && isParamIdentContinue(query[j]) {
					j++
				}
				b.WriteString(query[i:j])
				i = j
				continue
			}

			// Placeholder names must start with [A-Za-z_] and then continue with [A-Za-z0-9_].
			if !isParamIdentStart(query[i+1]) {
				b.WriteByte(query[i])
				i++
				continue
			}

			j := i + 2
			for j < len(query) && isParamIdentContinue(query[j]) {
				j++
			}

			name := query[i+1 : j]
			// Reuse placeholder index for repeated :name where driver supports it.
			if reuseByName {
				state, ok := stateByName[name]
				if !ok {
					return nil, MissingArgument(name)
				}

				if state.idx > 0 {
					b.WriteString(ph(state.idx))
				} else {
					values = append(values, state.value)
					state.idx = len(values)
					stateByName[name] = state
					b.WriteString(ph(state.idx))
				}
			} else {
				value, ok := byName[name]
				if !ok {
					return nil, MissingArgument(name)
				}

				values = append(values, value)
				b.WriteString(ph(len(values)))
			}

			i = j
			continue
		}

		// Any other byte is copied as-is.
		b.WriteByte(query[i])
		i++
	}

	return &BoundQuery{query: b.String(), values: values}, nil
}

type namedArgState struct {
	value any
	idx   int
}

// isParamIdentStart reports whether a byte may start a :name identifier.
func isParamIdentStart(r byte) bool {
	return unicode.IsLetter(rune(r)) || r == '_'
}

// isParamIdentContinue reports whether a byte may continue a :name identifier.
func isParamIdentContinue(r byte) bool {
	return unicode.IsLetter(rune(r)) || unicode.IsDigit(rune(r)) || r == '_'
}

// copyLineComment copies a SQL line comment beginning with "--" up to newline/end.
func copyLineComment(query string, i int, b *strings.Builder) (int, bool) {
	if i+1 >= len(query) || query[i] != '-' || query[i+1] != '-' {
		return i, false
	}

	for i < len(query) {
		b.WriteByte(query[i])
		i++
		if i > 0 && query[i-1] == '\n' {
			break
		}
	}
	return i, true
}

// copyBlockComment copies a SQL block comment beginning with "/*" and ending "*/".
func copyBlockComment(query string, i int, b *strings.Builder) (int, bool) {
	if i+1 >= len(query) || query[i] != '/' || query[i+1] != '*' {
		return i, false
	}

	b.WriteString("/*")
	i += 2
	for i < len(query) {
		b.WriteByte(query[i])
		if query[i] == '*' && i+1 < len(query) && query[i+1] == '/' {
			b.WriteByte(query[i+1])
			i += 2
			return i, true
		}
		i++
	}
	return i, true
}

// copySingleQuotedString copies a single-quoted SQL string literal.
// If backslashEscapes is true, backslash escapes are respected (Postgres E'...').
func copySingleQuotedString(query string, i int, b *strings.Builder, backslashEscapes bool) int {
	b.WriteByte(query[i]) // opening quote
	i++

	for i < len(query) {
		b.WriteByte(query[i])

		if backslashEscapes && query[i] == '\\' && i+1 < len(query) {
			b.WriteByte(query[i+1])
			i += 2
			continue
		}

		if query[i] == '\'' {
			// SQL escaped quote via doubled single quote.
			if i+1 < len(query) && query[i+1] == '\'' {
				b.WriteByte(query[i+1])
				i += 2
				continue
			}
			i++
			return i
		}

		i++
	}

	return i
}

// copyPostgresEscapedString copies a Postgres E'...' string with backslash escapes.
func copyPostgresEscapedString(query string, i int, b *strings.Builder) (int, bool) {
	if i+1 >= len(query) {
		return i, false
	}
	if (query[i] != 'E' && query[i] != 'e') || query[i+1] != '\'' {
		return i, false
	}
	if i > 0 && isParamIdentContinue(query[i-1]) {
		return i, false
	}

	b.WriteByte(query[i]) // E/e prefix
	i++
	return copySingleQuotedString(query, i, b, true), true
}

// copyPostgresDollarQuotedString copies a Postgres dollar-quoted string literal.
func copyPostgresDollarQuotedString(query string, i int, b *strings.Builder) (int, bool) {
	delim, next, ok := parseDollarQuoteDelimiter(query, i)
	if !ok {
		return i, false
	}

	b.WriteString(delim)
	i = next
	for i < len(query) {
		if strings.HasPrefix(query[i:], delim) {
			b.WriteString(delim)
			i += len(delim)
			return i, true
		}
		b.WriteByte(query[i])
		i++
	}
	return i, true
}

// parseDollarQuoteDelimiter parses a postgres dollar-quote opener: $$ or $tag$.
func parseDollarQuoteDelimiter(query string, i int) (string, int, bool) {
	if i >= len(query) || query[i] != '$' {
		return "", i, false
	}

	j := i + 1
	for j < len(query) && query[j] != '$' {
		if !isParamIdentContinue(query[j]) {
			return "", i, false
		}
		j++
	}
	if j >= len(query) || query[j] != '$' {
		return "", i, false
	}

	return query[i : j+1], j + 1, true
}

//============================================================================
// Placeholders
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
