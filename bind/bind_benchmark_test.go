package bind

import (
	"database/sql"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/x/dsn"
)

//============================================================================
// Benchmark Configuration
//============================================================================

var benchmarkRewriteResult *BoundQuery

type rewriteBenchmarkCase struct {
	name  string // stable benchmark name; also used as key in baseline JSON.
	query string
	args  []sql.NamedArg
	// placeholder + reuseByName mirror how Rewrite() configures each placeholder type.
	placeholder placeholderFunc
	reuseByName bool
	// provider toggles provider-specific parser branches (postgres comments/casts/strings).
	provider string
}

type rewriteBenchmarkBaseline struct {
	// Threshold is an allowed slowdown ratio; 0.10 means 10%.
	Threshold float64 `json:"threshold"`
	// NsPerOp stores the accepted baseline by benchmark case name.
	NsPerOp map[string]float64 `json:"ns_per_op"`
	// AllocsPerOp stores the accepted allocation baseline by benchmark case name.
	AllocsPerOp map[string]uint64 `json:"allocs_per_op"`
}

const (
	rewriteRegressionThreshold  = 0.10
	rewriteBaselineFilenameJSON = "rewrite_benchmark_baseline.json"
	rewriteRegressionSamples    = 5
)

// rewriteBenchmarkCases is the canonical workload set for both:
// - BenchmarkRewrite (human-readable perf measurements)
// - TestRewriteBenchmarkRegression (automated slowdown guard)
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
		tc := tc
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				benchmarkRewriteResult = rewriteCase(b, tc)
			}
		})
	}
}

//============================================================================
// Regression Test
//============================================================================

// TestRewriteBenchmarkRegression records and enforces a local benchmark baseline.
// - runs on every test execution (never skipped).
// - fails if baseline file is missing.
// - fails if any case is >10% slower than baseline.
// - delete/recreate testdata/rewrite_benchmark_baseline.json when intentionally re-baselining.
func TestRewriteBenchmarkRegression(t *testing.T) {
	currentNs := make(map[string]float64, len(rewriteBenchmarkCases))
	currentAllocs := make(map[string]uint64, len(rewriteBenchmarkCases))
	for _, tc := range rewriteBenchmarkCases {
		// Sample multiple times and use best metrics to reduce transient host noise.
		ns, allocs := measureBestBenchmark(tc, rewriteRegressionSamples)
		currentNs[tc.name] = ns
		currentAllocs[tc.name] = allocs
		t.Logf("%s: %.1f ns/op, %d allocs/op", tc.name, ns, allocs)
	}

	baselinePath := rewriteBenchmarkBaselinePath(t)
	baseline, err := readRewriteBenchmarkBaseline(baselinePath)
	require.NoError(t, err, "could not read benchmark baseline")
	require.NotNilf(t, baseline, "missing benchmark baseline at %s; re-baseline by recreating %s", baselinePath, rewriteBaselineFilenameJSON)

	threshold := baseline.Threshold
	if threshold <= 0 {
		threshold = rewriteRegressionThreshold
	}

	for _, name := range sortedKeys(baseline.NsPerOp) {
		baselineNs := baseline.NsPerOp[name]
		current, ok := currentNs[name]
		require.Truef(t, ok, "missing benchmark case %q in current run", name)

		maxAllowed := baselineNs * (1 + threshold)
		require.LessOrEqualf(t, current, maxAllowed, "%s regression: %.1f ns/op > %.1f ns/op (baseline %.1f, threshold %.0f%%)",
			name, current, maxAllowed, baselineNs, threshold*100,
		)
	}

	require.NotNil(t, baseline.AllocsPerOp, "missing allocs_per_op in benchmark baseline")
	for _, name := range sortedKeys(baseline.NsPerOp) {
		baselineAllocs, ok := baseline.AllocsPerOp[name]
		require.Truef(t, ok, "missing alloc baseline for benchmark case %q", name)

		current, ok := currentAllocs[name]
		require.Truef(t, ok, "missing current alloc data for benchmark case %q", name)
		require.LessOrEqualf(t, current, baselineAllocs, "%s alloc regression: %d allocs/op > %d allocs/op baseline",
			name, current, baselineAllocs,
		)
	}
}

//============================================================================
// Local Baseline File Helpers
//============================================================================

func rewriteBenchmarkBaselinePath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "could not resolve benchmark path")
	return filepath.Join(filepath.Dir(filename), "testdata", rewriteBaselineFilenameJSON)
}

func readRewriteBenchmarkBaseline(path string) (*rewriteBenchmarkBaseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var baseline rewriteBenchmarkBaseline
	if err = json.Unmarshal(data, &baseline); err != nil {
		return nil, err
	}
	return &baseline, nil
}

//============================================================================
// Internal Helpers
//============================================================================

func rewriteCase(tb testing.TB, tc rewriteBenchmarkCase) *BoundQuery {
	tb.Helper()

	bound, err := rewriteQuery(tc.query, tc.args, tc.placeholder, tc.reuseByName, tc.provider)
	if err != nil {
		tb.Fatal(err)
	}
	return bound
}

func sortedKeys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func measureBestBenchmark(tc rewriteBenchmarkCase, samples int) (float64, uint64) {
	bestNs := math.MaxFloat64
	bestAllocs := ^uint64(0)
	for i := 0; i < samples; i++ {
		res := testing.Benchmark(func(b *testing.B) {
			for b.Loop() {
				benchmarkRewriteResult = rewriteCase(b, tc)
			}
		})

		ns := float64(res.NsPerOp())
		if ns < bestNs {
			bestNs = ns
		}

		allocs := uint64(res.AllocsPerOp())
		if allocs < bestAllocs {
			bestAllocs = allocs
		}
	}
	return bestNs, bestAllocs
}
