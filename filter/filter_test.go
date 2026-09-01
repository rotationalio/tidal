package filter

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

//============================================================================
// ORDER BY Clause Builder
//============================================================================

// Check ascending and descending order, including replacing an old order.
func TestFilterOrder(t *testing.T) {
	t.Run("OrderByAscDesc", func(t *testing.T) {
		// Bare column is ASC; -prefix is DESC.
		f := (&Filter{}).OrderBy("name", "-created")
		require.Equal(t, "ORDER BY name ASC, created DESC", f.Clause())
	})

	t.Run("OrderByOverwrite", func(t *testing.T) {
		// Each OrderBy call replaces the previous ordering entirely.
		f := (&Filter{}).OrderBy("name").OrderBy("-email")
		require.Equal(t, "ORDER BY email DESC", f.Clause())
	})
}

//============================================================================
// LIMIT / OFFSET Clause Builder
//============================================================================

// Check limit and offset, including removing them with negative values.
func TestFilterLimitOffset(t *testing.T) {
	t.Run("LimitOffset", func(t *testing.T) {
		// Set an order, limit, and offset, then check the full SQL clause.
		f := (&Filter{}).OrderBy("id").Limit(10).Offset(5)
		require.Equal(t, "ORDER BY id ASC LIMIT 10 OFFSET 5", f.Clause())
	})

	t.Run("LimitOffsetWithoutOrder", func(t *testing.T) {
		// Set a limit and offset without an order, then check they work on their
		// own.
		f := (&Filter{}).Limit(10).Offset(5)
		require.Equal(t, "LIMIT 10 OFFSET 5", f.Clause())
	})

	t.Run("ClearLimitOffset", func(t *testing.T) {
		// n=-1 clears a previously set limit or offset.
		f := (&Filter{}).Limit(10).Offset(5).Limit(-1).Offset(-1)
		require.Empty(t, f.Clause())
	})
}

//============================================================================
// WHERE Clause Builder
//============================================================================

// Check conditions, groups, operators, subqueries, filter changes, and
// parameter order.
func TestFilterWhere(t *testing.T) {
	t.Run("WhereAppends", func(t *testing.T) {
		// Add two conditions and check that both remain with parameters in order.
		f := (&Filter{}).
			Where("status", Eq, "active").
			Where("role", Eq, "admin")

		require.Equal(t, "WHERE status = :w1 AND role = :w2", f.Clause())
		require.Equal(t, []sql.NamedArg{
			sql.Named("w1", "active"),
			sql.Named("w2", "admin"),
		}, f.Params())
	})

	t.Run("WhereAndOrAppend", func(t *testing.T) {
		// Add AND and OR conditions and check their order and parameters.
		f := (&Filter{}).
			Where("status", Eq, "active").
			And("age", Gte, 18).
			Or("role", Eq, "admin")

		require.Equal(t, "WHERE status = :w1 AND age >= :w2 OR role = :w3", f.Clause())
		require.Equal(t, []sql.NamedArg{
			sql.Named("w1", "active"),
			sql.Named("w2", 18),
			sql.Named("w3", "admin"),
		}, f.Params())
	})

	t.Run("AndGroup", func(t *testing.T) {
		// Add an AND group with AND and OR terms, then check its parentheses and
		// parameter order.
		f := (&Filter{}).
			Where("status", Eq, "active").
			AndGroup(func(g *WhereGroup) {
				g.Where("role", Eq, "admin").Or("role", Eq, "editor")
			})

		require.Equal(t, "WHERE status = :w1 AND (role = :w2 OR role = :w3)", f.Clause())
		require.Equal(t, []sql.NamedArg{
			sql.Named("w1", "active"),
			sql.Named("w2", "admin"),
			sql.Named("w3", "editor"),
		}, f.Params())
	})

	t.Run("OrGroup", func(t *testing.T) {
		// Add an OR group and check that it has its own parentheses.
		f := (&Filter{}).
			Where("status", Eq, "active").
			OrGroup(func(g *WhereGroup) {
				g.Where("role", Eq, "admin").And("verified", Eq, true)
			})

		require.Equal(t, "WHERE status = :w1 OR (role = :w2 AND verified = :w3)", f.Clause())
	})

	t.Run("WhereWithOrderLimitOffset", func(t *testing.T) {
		// Set a condition, order, limit, and offset, then check their SQL order.
		f := (&Filter{}).
			Where("status", Eq, "active").
			OrderBy("-created").
			Limit(20).
			Offset(10)

		require.Equal(t, "WHERE status = :w1 ORDER BY created DESC LIMIT 20 OFFSET 10", f.Clause())
	})

	t.Run("LikeOperator", func(t *testing.T) {
		// Add two LIKE conditions and check that both patterns are parameters.
		f := (&Filter{}).
			Where("name", Like, "%ada%").
			And("email", Like, "%@example.com")

		require.Equal(t, "WHERE name LIKE :w1 AND email LIKE :w2", f.Clause())
	})

	t.Run("ILikeOperator", func(t *testing.T) {
		// Add a case-insensitive LIKE condition and check its operator and
		// pattern parameter.
		f := (&Filter{}).Where("name", ILike, "%ada%")

		require.Equal(t, "WHERE name ILIKE :w1", f.Clause())
		require.Equal(t, []sql.NamedArg{sql.Named("w1", "%ada%")}, f.Params())
	})

	t.Run("InOperator", func(t *testing.T) {
		// Add one normal condition and an IN list, then check all values get
		// placeholders in order.
		f := (&Filter{}).
			Where("status", Eq, "active").
			And("id", In, []int64{1, 2, 3})

		require.Equal(t, "WHERE status = :w1 AND id IN (:w2, :w3, :w4)", f.Clause())
		require.Equal(t, []sql.NamedArg{
			sql.Named("w1", "active"),
			sql.Named("w2", int64(1)),
			sql.Named("w3", int64(2)),
			sql.Named("w4", int64(3)),
		}, f.Params())
	})

	t.Run("InEmptySliceFalse", func(t *testing.T) {
		// Add an empty IN list and check it becomes false instead of removing
		// the filter.
		f := (&Filter{}).
			Where("status", Eq, "active").
			And("id", In, []string{})

		require.Equal(t, "WHERE status = :w1 AND 1=0", f.Clause())
		require.Equal(t, []sql.NamedArg{sql.Named("w1", "active")}, f.Params())
	})

	t.Run("NotInAnyAndAll", func(t *testing.T) {
		// Add a NOT IN list and ANY/ALL conditions, then check the SQL and
		// parameters for each one.
		f := (&Filter{}).
			Where("id", NotIn, []int64{1, 2}).
			And("status", Any(Eq), Subselect("SELECT status FROM archived_statuses")).
			And("priority", All(Gt), []int64{1, 2})

		require.Equal(t, "WHERE id NOT IN (:w1, :w2) AND status = ANY (SELECT status FROM archived_statuses) AND priority > ALL (:w3)", f.Clause())
		require.Equal(t, []sql.NamedArg{
			sql.Named("w1", int64(1)),
			sql.Named("w2", int64(2)),
			sql.Named("w3", []int64{1, 2}),
		}, f.Params())
	})

	t.Run("SubselectParameters", func(t *testing.T) {
		// Use a SELECT string with a named parameter and check that the SELECT
		// stays as SQL while its value is bound separately.
		f := (&Filter{}).
			Where("experiment_id", NotIn, "SELECT experiment_id FROM task_versions WHERE task_id=:task_id").
			Param("task_id", 42)

		require.Equal(t, "WHERE experiment_id NOT IN (SELECT experiment_id FROM task_versions WHERE task_id=:task_id)", f.Clause())
		require.Equal(t, []sql.NamedArg{sql.Named("task_id", 42)}, f.Params())
	})

	t.Run("CloneAndClearWhere", func(t *testing.T) {
		// Make and clone a filter, change the clone, and check that the original
		// is unchanged and the clone keeps its order and limit after clearing
		// WHERE.
		base := (&Filter{}).
			Where("tenant_id", Eq, 7).
			OrderBy("-created").
			Limit(10)
		clone := base.Clone().
			Where("status", Eq, "active").
			Prefix("t")

		require.Equal(t, "WHERE tenant_id = :w1 ORDER BY created DESC LIMIT 10", base.Clause())
		require.Equal(t, "WHERE t.tenant_id = :w1 AND t.status = :w2 ORDER BY t.created DESC LIMIT 10", clone.Clause())

		clone.ClearWhere()
		require.Equal(t, "ORDER BY t.created DESC LIMIT 10", clone.Clause())
	})

	t.Run("CloneDeepCopiesGroupsAndParameters", func(t *testing.T) {
		// Make and cache a filter with a nested group and named parameter, then
		// change the clone. The original must keep its fields and value.
		base := (&Filter{}).
			Where("tenant_id", Eq, 7).
			AndGroup(func(g *WhereGroup) {
				g.Where("status", Eq, "active").Or("status", Eq, "pending")
			}).
			Where("id", In, Subselect("SELECT id FROM memberships WHERE owner=:owner")).
			Param("owner", "ada")
		_ = base.Clause()
		_ = base.Params()

		clone := base.Clone().Prefix("t").Param("owner", "grace")

		require.Equal(t, "WHERE tenant_id = :w1 AND (status = :w2 OR status = :w3) AND id IN (SELECT id FROM memberships WHERE owner=:owner)", base.Clause())
		require.Equal(t, []sql.NamedArg{
			sql.Named("w1", 7),
			sql.Named("w2", "active"),
			sql.Named("w3", "pending"),
			sql.Named("owner", "ada"),
		}, base.Params())
		require.Equal(t, "WHERE t.tenant_id = :w1 AND (t.status = :w2 OR t.status = :w3) AND t.id IN (SELECT id FROM memberships WHERE owner=:owner)", clone.Clause())
		require.Equal(t, []sql.NamedArg{
			sql.Named("w1", 7),
			sql.Named("w2", "active"),
			sql.Named("w3", "pending"),
			sql.Named("owner", "grace"),
		}, clone.Params())
	})

	t.Run("IsIsNotLiterals", func(t *testing.T) {
		// Use SQL literals and check that they appear directly without
		// parameters.
		f := (&Filter{}).
			Where("revoked", Is, Null).
			And("deleted_at", IsNot, Null).
			And("active", Is, True).
			And("disabled", IsNot, False).
			And("maybe", Is, Unknown)

		require.Equal(t, "WHERE revoked IS NULL AND deleted_at IS NOT NULL AND active IS TRUE AND disabled IS NOT FALSE AND maybe IS UNKNOWN", f.Clause())
		require.Empty(t, f.Params())
	})

	t.Run("IsWithValue", func(t *testing.T) {
		// Use normal values with IS and IS NOT and check that they use
		// parameters.
		f := (&Filter{}).
			Where("status", Is, "active").
			And("legacy", IsNot, true)

		require.Equal(t, "WHERE status IS :w1 AND legacy IS NOT :w2", f.Clause())
		require.Equal(t, []sql.NamedArg{
			sql.Named("w1", "active"),
			sql.Named("w2", true),
		}, f.Params())
	})

	t.Run("IsDistinctFromOperators", func(t *testing.T) {
		// Use both distinctness operators and check that their values are
		// bound in order, including nil.
		f := (&Filter{}).
			Where("a", IsDistinctFrom, nil).
			And("b", IsNotDistinctFrom, "active")

		require.Equal(t, "WHERE a IS DISTINCT FROM :w1 AND b IS NOT DISTINCT FROM :w2", f.Clause())
		require.Equal(t, []sql.NamedArg{
			sql.Named("w1", nil),
			sql.Named("w2", "active"),
		}, f.Params())
	})

}

//============================================================================
// WHERE Misuse / Edge Cases
//============================================================================

// Check what happens when WHERE methods are called in unusual orders.
// Filter does not return errors, so these cases define the current behavior.
func TestFilterWhereMisuse(t *testing.T) {
	t.Run("OrBeforeWhereStartsFirstCondition", func(t *testing.T) {
		// OR on an empty filter becomes the first condition without an OR keyword.
		f := (&Filter{}).Or("role", Eq, "admin")
		require.Equal(t, "WHERE role = :w1", f.Clause())
	})

	t.Run("AndBeforeWhereStartsFirstCondition", func(t *testing.T) {
		// AND on an empty filter becomes the first condition without an AND keyword.
		f := (&Filter{}).And("age", Gte, 18)
		require.Equal(t, "WHERE age >= :w1", f.Clause())
	})

	t.Run("OrGroupBeforeWhereBecomesRootGroup", func(t *testing.T) {
		// An OR group on an empty filter becomes the first group without an OR
		// keyword.
		f := (&Filter{}).OrGroup(func(g *WhereGroup) {
			g.Where("role", Eq, "admin")
		})
		require.Equal(t, "WHERE (role = :w1)", f.Clause())
	})

	t.Run("AndGroupBeforeWhereBecomesRootGroup", func(t *testing.T) {
		// An AND group on an empty filter becomes the first group without an AND
		// keyword.
		f := (&Filter{}).AndGroup(func(g *WhereGroup) {
			g.Where("role", Eq, "admin")
		})
		require.Equal(t, "WHERE (role = :w1)", f.Clause())
	})

	t.Run("OrThenWhereAppends", func(t *testing.T) {
		// WHERE still adds an AND condition after OR; it does not replace the
		// existing condition.
		f := (&Filter{}).
			Or("legacy", Eq, true).
			Where("status", Eq, "active")
		require.Equal(t, "WHERE legacy = :w1 AND status = :w2", f.Clause())
	})

	t.Run("OrGroupThenWhereAppends", func(t *testing.T) {
		// WHERE adds an AND condition after an OR group; replacing it requires
		// an explicit call to ReplaceWhere.
		f := (&Filter{}).
			OrGroup(func(g *WhereGroup) {
				g.Where("role", Eq, "admin")
			}).
			Where("status", Eq, "active")
		require.Equal(t, "WHERE (role = :w1 AND status = :w2)", f.Clause())
	})

	t.Run("EmptyOrGroupThenWhere", func(t *testing.T) {
		// An empty group is ignored, so WHERE becomes the first condition.
		f := (&Filter{}).
			OrGroup(func(g *WhereGroup) {}).
			Where("status", Eq, "active")
		require.Equal(t, "WHERE status = :w1", f.Clause())
	})

	t.Run("EmptyOrGroupOnly", func(t *testing.T) {
		// An empty group is ignored and produces no SQL.
		f := (&Filter{}).OrGroup(func(g *WhereGroup) {})
		require.Empty(t, f.Clause())
		require.Empty(t, f.Params())
	})

	t.Run("OrOrWithoutWhere", func(t *testing.T) {
		// The first OR starts the filter; the second OR adds another condition.
		f := (&Filter{}).
			Or("a", Eq, 1).
			Or("b", Eq, 2)
		require.Equal(t, "WHERE a = :w1 OR b = :w2", f.Clause())
	})

	t.Run("OrGroupOrBeforeWhereInGroup", func(t *testing.T) {
		// Inside a group, OR starts the group and WHERE adds an AND condition,
		// just as it does on Filter.
		f := (&Filter{}).OrGroup(func(g *WhereGroup) {
			g.Or("role", Eq, "admin").Where("status", Eq, "active")
		})
		require.Equal(t, "WHERE (role = :w1 AND status = :w2)", f.Clause())
	})

	t.Run("OrOrGroupThenWhereAppends", func(t *testing.T) {
		// WHERE adds an AND condition after existing OR terms; use ReplaceWhere
		// when the old filter should be discarded.
		f := (&Filter{}).
			Or("legacy", Eq, true).
			OrGroup(func(g *WhereGroup) {
				g.Where("role", Eq, "admin")
			}).
			Where("status", Eq, "active")
		require.Equal(t, "WHERE legacy = :w1 OR (role = :w2) AND status = :w3", f.Clause())
	})

	t.Run("ReplaceWhere", func(t *testing.T) {
		// Cache a filter with a subquery parameter and pagination, replace WHERE,
		// and check that pagination remains while the old condition and parameter
		// are removed.
		f := (&Filter{}).
			Where("legacy_id", NotIn, Subselect("SELECT id FROM legacy WHERE owner=:old_owner")).
			Param("old_owner", "ada").
			OrderBy("-created").
			Limit(20).
			Offset(5)
		_ = f.Clause()
		_ = f.Params()

		f.ReplaceWhere("status", Eq, "active")

		require.Equal(t, "WHERE status = :w1 ORDER BY created DESC LIMIT 20 OFFSET 5", f.Clause())
		require.Equal(t, []sql.NamedArg{sql.Named("w1", "active")}, f.Params())
	})

	t.Run("ClearWhereAndClear", func(t *testing.T) {
		// Cache a filter with every clause, clear WHERE, and check that order,
		// limit, and offset remain while WHERE parameters are removed.
		f := (&Filter{}).
			Where("id", In, Subselect("SELECT id FROM records WHERE owner=:owner")).
			Param("owner", "ada").
			OrderBy("id").
			Limit(10).
			Offset(5)
		_ = f.Clause()
		_ = f.Params()

		f.ClearWhere()
		require.Equal(t, "ORDER BY id ASC LIMIT 10 OFFSET 5", f.Clause())
		require.Empty(t, f.Params())

		// Clear changes the filter itself, so callers do not need to assign its
		// return value.
		cleared := f.Where("status", Eq, "active").Clear()
		require.Same(t, f, cleared)
		require.Equal(t, &Filter{}, f)
		require.Empty(t, f.Clause())
		require.Empty(t, f.Params())
	})
}

//============================================================================
// Comprehensive Clause Builder
//============================================================================

// Check a filter that combines WHERE, ORDER BY, LIMIT, and OFFSET.
func TestFilterComprehensive(t *testing.T) {
	f := (&Filter{}).
		Where("status", Eq, "active").
		And("deleted_at", Eq, nil).
		And("organization_id", Ne, "").
		And("plan_id", In, []string{"starter", "pro", "enterprise"}).
		AndGroup(func(g *WhereGroup) {
			g.Where("role", Eq, "admin").
				Or("role", Eq, "editor").
				Or("role", Eq, "viewer").
				AndGroup(func(g *WhereGroup) {
					g.Where("verified", Eq, true).
						And("mfa_enabled", Eq, true).
						OrGroup(func(g *WhereGroup) {
							g.Where("sso_provider", Eq, "okta").
								And("sso_external_id", Ne, "")
						})
				})
		}).
		OrGroup(func(g *WhereGroup) {
			g.Where("tier", Gte, 2).
				And("name", Like, "%pro%").
				AndGroup(func(g *WhereGroup) {
					g.Where("billing_status", Eq, "current").
						Or("trial_ends_at", Gt, "2026-01-01")
				})
		}).
		Or("legacy", Eq, true).
		And("age", Gte, 18).
		And("age", Lte, 65).
		And("score", Lt, 100).
		And("rating", Gte, 3.5).
		OrGroup(func(g *WhereGroup) {
			g.Where("region", Eq, "us").
				Or("region", Eq, "eu").
				And("featured", Eq, true).
				AndGroup(func(g *WhereGroup) {
					g.Where("campaign", Eq, "spring").
						Or("campaign", Eq, "summer").
						Or("tags", Like, "%launch%")
				})
		}).
		OrGroup(func(g *WhereGroup) {
			g.Where("invited_by", Ne, "").
				And("invite_accepted_at", Eq, nil).
				OrGroup(func(g *WhereGroup) {
					g.Where("signup_source", Eq, "referral").
						And("referral_code", Like, "REF-%")
				})
		}).
		And("email", Like, "%@example.com").
		Or("vip", Eq, true).
		OrderBy("-created", "name", "-score").
		Limit(50).
		Offset(100)

	// Expected SQL clause:
	//
	// WHERE status = :w1
	//   AND deleted_at = :w2
	//   AND organization_id != :w3
	//   AND plan_id IN (:w4, :w5, :w6)
	//   AND (
	//     role = :w7 OR role = :w8 OR role = :w9
	//     AND (
	//       verified = :w10 AND mfa_enabled = :w11
	//       OR (sso_provider = :w12 AND sso_external_id != :w13)
	//     )
	//   )
	//   OR (
	//     tier >= :w14 AND name LIKE :w15
	//     AND (billing_status = :w16 OR trial_ends_at > :w17)
	//   )
	//   OR legacy = :w18
	//   AND age >= :w19 AND age <= :w20
	//   AND score < :w21 AND rating >= :w22
	//   OR (
	//     region = :w23 OR region = :w24 AND featured = :w25
	//     AND (campaign = :w26 OR campaign = :w27 OR tags LIKE :w28)
	//   )
	//   OR (
	//     invited_by != :w29 AND invite_accepted_at = :w30
	//     OR (signup_source = :w31 AND referral_code LIKE :w32)
	//   )
	//   AND email LIKE :w33 OR vip = :w34
	// ORDER BY created DESC, name ASC, score DESC
	// LIMIT 50 OFFSET 100
	require.Equal(t,
		"WHERE status = :w1 AND deleted_at = :w2 AND organization_id != :w3 AND plan_id IN (:w4, :w5, :w6) AND (role = :w7 OR role = :w8 OR role = :w9 AND (verified = :w10 AND mfa_enabled = :w11 OR (sso_provider = :w12 AND sso_external_id != :w13))) OR (tier >= :w14 AND name LIKE :w15 AND (billing_status = :w16 OR trial_ends_at > :w17)) OR legacy = :w18 AND age >= :w19 AND age <= :w20 AND score < :w21 AND rating >= :w22 OR (region = :w23 OR region = :w24 AND featured = :w25 AND (campaign = :w26 OR campaign = :w27 OR tags LIKE :w28)) OR (invited_by != :w29 AND invite_accepted_at = :w30 OR (signup_source = :w31 AND referral_code LIKE :w32)) AND email LIKE :w33 OR vip = :w34 ORDER BY created DESC, name ASC, score DESC LIMIT 50 OFFSET 100",
		f.Clause(),
	)
	require.Equal(t, []sql.NamedArg{
		sql.Named("w1", "active"),
		sql.Named("w2", nil),
		sql.Named("w3", ""),
		sql.Named("w4", "starter"),
		sql.Named("w5", "pro"),
		sql.Named("w6", "enterprise"),
		sql.Named("w7", "admin"),
		sql.Named("w8", "editor"),
		sql.Named("w9", "viewer"),
		sql.Named("w10", true),
		sql.Named("w11", true),
		sql.Named("w12", "okta"),
		sql.Named("w13", ""),
		sql.Named("w14", 2),
		sql.Named("w15", "%pro%"),
		sql.Named("w16", "current"),
		sql.Named("w17", "2026-01-01"),
		sql.Named("w18", true),
		sql.Named("w19", 18),
		sql.Named("w20", 65),
		sql.Named("w21", 100),
		sql.Named("w22", 3.5),
		sql.Named("w23", "us"),
		sql.Named("w24", "eu"),
		sql.Named("w25", true),
		sql.Named("w26", "spring"),
		sql.Named("w27", "summer"),
		sql.Named("w28", "%launch%"),
		sql.Named("w29", ""),
		sql.Named("w30", nil),
		sql.Named("w31", "referral"),
		sql.Named("w32", "REF-%"),
		sql.Named("w33", "%@example.com"),
		sql.Named("w34", true),
	}, f.Params())
}

// Verify prefixing updates both WHERE fields and ORDER BY columns after the
// filter has already rendered and cached its output.
func TestFilterPrefix(t *testing.T) {
	// Build and render a filter first so this test also exercises cache
	// invalidation when prefixing is applied afterward.
	f := New().
		Where("color", Eq, "red").
		And("age", Gte, 25).
		OrderBy("-age", "created").
		Limit(10).
		Offset(30)
	_ = f.Clause()
	_ = f.Params()
	f.Prefix("t")

	// Assert that both WHERE fields and ORDER BY columns receive the prefix,
	// while pagination and parameter numbering remain unchanged.
	require.Equal(t,
		"WHERE t.color = :w1 AND t.age >= :w2 ORDER BY t.age DESC, t.created ASC LIMIT 10 OFFSET 30",
		f.Clause(),
		"filtering didn't work as expected",
	)
}
