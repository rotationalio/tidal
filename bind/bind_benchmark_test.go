package bind

import (
	"database/sql"
	"strings"
	"testing"

	"go.rtnl.ai/x/dsn"
)

//============================================================================
// Benchmark Configuration
//============================================================================

// benchmarkRewriteResult prevents compiler optimization of benchmark work.
var benchmarkRewriteResult *BoundQuery

type rewriteBenchmarkCase struct {
	name  string
	query string
	args  []sql.NamedArg

	// placeholder + reuseByName mirror Rewrite() paths.
	placeholder placeholderFunc
	reuseByName bool
	provider    string
}

// rewriteBenchmarkCases is the canonical workload set for Rewrite performance.
var rewriteBenchmarkCases = []rewriteBenchmarkCase{
	{
		// Typical ordered placeholders used by postgres CRUD queries.
		name:  "OrderedSimple",
		query: "SELECT * FROM users WHERE id = :id AND tenant_id = :tenant_id AND status = :status",
		args: []sql.NamedArg{
			sql.Named("id", "01HZX9WQ8GQ6KVG5KQY8QY0Y7G"),
			sql.Named("tenant_id", "01HZX9WQ8GQ6KVG5KQY8QY0Y7H"),
			sql.Named("status", "active"),
		},
		placeholder: orderedPlaceholder,
		reuseByName: true,
		provider:    dsn.Postgres,
	},
	{
		// Stress ordered parser paths: comments, quotes, E'', dollar strings, and :: casts.
		name: "OrderedComplex",
		query: strings.Join([]string{
			"SELECT id FROM users",
			"/* static block comment */",
			"WHERE display_name = 'Alice''s account'",
			"AND payload = E'a\\nb'",
			"AND note = $tag$plain text$tag$",
			"AND role::text = :role",
			"AND tenant_id = :tenant_id",
			"-- static line comment",
		}, "\n"),
		args: []sql.NamedArg{
			sql.Named("role", "admin"),
			sql.Named("tenant_id", "01HZX9WQ8GQ6KVG5KQY8QY0Y7H"),
		},
		placeholder: orderedPlaceholder,
		reuseByName: true,
		provider:    dsn.Postgres,
	},
	{
		// Positional placeholders represent non-postgres binding behavior.
		name:  "PositionalSimple",
		query: "SELECT * FROM api_keys WHERE owner_id = :owner_id AND revoked = :revoked OR owner_id = :owner_id",
		args: []sql.NamedArg{
			sql.Named("owner_id", "01HZX9WQ8GQ6KVG5KQY8QY0Y7G"),
			sql.Named("revoked", false),
		},
		placeholder: positionalPlaceholder,
		reuseByName: false,
		provider:    "",
	},
}

//============================================================================
// Benchmarks
//============================================================================

func BenchmarkRewrite(b *testing.B) {
	for _, tc := range rewriteBenchmarkCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchmarkRewriteResult = rewriteCase(tc)
			}
		})
	}
}

//============================================================================
// Internal Helpers
//============================================================================

func rewriteCase(tc rewriteBenchmarkCase) *BoundQuery {
	bound, err := rewriteQuery(tc.query, tc.args, tc.placeholder, tc.reuseByName, tc.provider)
	if err != nil {
		panic(err)
	}
	return bound
}
