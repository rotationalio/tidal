package filter

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

//============================================================================
// ORDER BY Clause Builder
//============================================================================

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

func TestFilterLimitOffset(t *testing.T) {
	t.Run("LimitOffset", func(t *testing.T) {
		f := (&Filter{}).OrderBy("id").Limit(10).Offset(5)
		require.Equal(t, "ORDER BY id ASC LIMIT 10 OFFSET 5", f.Clause())
	})

	t.Run("LimitOffsetWithoutOrder", func(t *testing.T) {
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

func TestFilterWhere(t *testing.T) {
	t.Run("WhereOverwrite", func(t *testing.T) {
		f := (&Filter{}).
			Where("status", Eq, "active").
			Where("role", Eq, "admin")

		require.Equal(t, "WHERE role = :w1", f.Clause())
		require.Equal(t, []sql.NamedArg{sql.Named("w1", "admin")}, f.Params())
	})

	t.Run("WhereAndOrAppend", func(t *testing.T) {
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
		f := (&Filter{}).
			Where("status", Eq, "active").
			OrGroup(func(g *WhereGroup) {
				g.Where("role", Eq, "admin").And("verified", Eq, true)
			})

		require.Equal(t, "WHERE status = :w1 OR (role = :w2 AND verified = :w3)", f.Clause())
	})

	t.Run("WhereWithOrderLimitOffset", func(t *testing.T) {
		f := (&Filter{}).
			Where("status", Eq, "active").
			OrderBy("-created").
			Limit(20).
			Offset(10)

		require.Equal(t, "WHERE status = :w1 ORDER BY created DESC LIMIT 20 OFFSET 10", f.Clause())
	})

	t.Run("LikeOperator", func(t *testing.T) {
		f := (&Filter{}).
			Where("name", Like, "%ada%").
			And("email", Like, "%@example.com")

		require.Equal(t, "WHERE name LIKE :w1 AND email LIKE :w2", f.Clause())
	})

	t.Run("InOperator", func(t *testing.T) {
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

	t.Run("InEmptySliceOmitted", func(t *testing.T) {
		f := (&Filter{}).
			Where("status", Eq, "active").
			And("id", In, []string{})

		require.Equal(t, "WHERE status = :w1", f.Clause())
		require.Equal(t, []sql.NamedArg{sql.Named("w1", "active")}, f.Params())
	})

	t.Run("InEmptySliceOnly", func(t *testing.T) {
		f := (&Filter{}).Where("id", In, []int{})

		require.Empty(t, f.Clause())
		require.Empty(t, f.Params())
	})

	t.Run("NullOperators", func(t *testing.T) {
		f := (&Filter{}).
			Where("revoked", IsNull, nil).
			And("deleted_at", IsNotNull, nil)

		require.Equal(t, "WHERE revoked IS NULL AND deleted_at IS NOT NULL", f.Clause())
		require.Empty(t, f.Params())
	})
}

//============================================================================
// WHERE Misuse / Edge Cases
//============================================================================

// Documents current behavior when WHERE helpers are called out of the usual order.
// [Filter] does not return errors today; these tests lock in what happens instead.
func TestFilterWhereMisuse(t *testing.T) {
	t.Run("OrBeforeWhereStartsFirstCondition", func(t *testing.T) {
		// Or on an empty filter acts like the first condition; no OR keyword is emitted.
		f := (&Filter{}).Or("role", Eq, "admin")
		require.Equal(t, "WHERE role = :w1", f.Clause())
	})

	t.Run("AndBeforeWhereStartsFirstCondition", func(t *testing.T) {
		// And on an empty filter acts like the first condition; no AND keyword is emitted.
		f := (&Filter{}).And("age", Gte, 18)
		require.Equal(t, "WHERE age >= :w1", f.Clause())
	})

	t.Run("OrGroupBeforeWhereBecomesRootGroup", func(t *testing.T) {
		// An empty root accepts the group directly; no leading OR is emitted.
		f := (&Filter{}).OrGroup(func(g *WhereGroup) {
			g.Where("role", Eq, "admin")
		})
		require.Equal(t, "WHERE (role = :w1)", f.Clause())
	})

	t.Run("AndGroupBeforeWhereBecomesRootGroup", func(t *testing.T) {
		// An empty root accepts the group directly; no leading AND is emitted.
		f := (&Filter{}).AndGroup(func(g *WhereGroup) {
			g.Where("role", Eq, "admin")
		})
		require.Equal(t, "WHERE (role = :w1)", f.Clause())
	})

	t.Run("OrThenWhereReplacesPriorCondition", func(t *testing.T) {
		// Where replaces the entire clause
		f := (&Filter{}).
			Or("legacy", Eq, true).
			Where("status", Eq, "active")
		require.Equal(t, "WHERE status = :w1", f.Clause())
	})

	t.Run("OrGroupThenWhereReplacesEntireClause", func(t *testing.T) {
		// Where replaces the entire clause
		f := (&Filter{}).
			OrGroup(func(g *WhereGroup) {
				g.Where("role", Eq, "admin")
			}).
			Where("status", Eq, "active")
		require.Equal(t, "WHERE status = :w1", f.Clause())
	})

	t.Run("EmptyOrGroupThenWhere", func(t *testing.T) {
		// Where replaces the entire clause
		f := (&Filter{}).
			OrGroup(func(g *WhereGroup) {}).
			Where("status", Eq, "active")
		require.Equal(t, "WHERE status = :w1", f.Clause())
	})

	t.Run("EmptyOrGroupOnly", func(t *testing.T) {
		// An empty group is ignored; no OR keyword is emitted.
		f := (&Filter{}).OrGroup(func(g *WhereGroup) {})
		require.Empty(t, f.Clause())
		require.Empty(t, f.Params())
	})

	t.Run("OrOrWithoutWhere", func(t *testing.T) {
		// First Or starts the expression; the second Or appends normally.
		f := (&Filter{}).
			Or("a", Eq, 1).
			Or("b", Eq, 2)
		require.Equal(t, "WHERE a = :w1 OR b = :w2", f.Clause())
	})

	t.Run("OrGroupOrBeforeWhereInGroup", func(t *testing.T) {
		// Inside a group, Or starts the sub-expression; Where replaces it.
		f := (&Filter{}).OrGroup(func(g *WhereGroup) {
			g.Or("role", Eq, "admin").Where("status", Eq, "active")
		})
		require.Equal(t, "WHERE (status = :w1)", f.Clause())
	})

	t.Run("OrOrGroupThenWhereReplacesAll", func(t *testing.T) {
		// Where replaces the entire clause
		f := (&Filter{}).
			Or("legacy", Eq, true).
			OrGroup(func(g *WhereGroup) {
				g.Where("role", Eq, "admin")
			}).
			Where("status", Eq, "active")
		require.Equal(t, "WHERE status = :w1", f.Clause())
	})
}

//============================================================================
// Comprehensive Clause Builder
//============================================================================

// Comprehensive test of all WHERE, ORDER BY, LIMIT, and OFFSET building
// methods with a really, really, REALLY complex filter.
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
