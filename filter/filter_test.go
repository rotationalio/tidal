package filter

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"go.rtnl.ai/tidal/filter/builder"
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
			Where("status", builder.Eq, "active").
			Where("role", builder.Eq, "admin")

		require.Equal(t, "WHERE role = :w1", f.Clause())
		require.Equal(t, []sql.NamedArg{sql.Named("w1", "admin")}, f.Params())
	})

	t.Run("WhereAndOrAppend", func(t *testing.T) {
		f := (&Filter{}).
			Where("status", builder.Eq, "active").
			And("age", builder.Gte, 18).
			Or("role", builder.Eq, "admin")

		require.Equal(t, "WHERE status = :w1 AND age >= :w2 OR role = :w3", f.Clause())
		require.Equal(t, []sql.NamedArg{
			sql.Named("w1", "active"),
			sql.Named("w2", 18),
			sql.Named("w3", "admin"),
		}, f.Params())
	})

	t.Run("AndGroup", func(t *testing.T) {
		f := (&Filter{}).
			Where("status", builder.Eq, "active").
			AndGroup(func(g *Where) {
				g.Where("role", builder.Eq, "admin").Or("role", builder.Eq, "editor")
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
			Where("status", builder.Eq, "active").
			OrGroup(func(g *Where) {
				g.Where("role", builder.Eq, "admin").And("verified", builder.Eq, true)
			})

		require.Equal(t, "WHERE status = :w1 OR (role = :w2 AND verified = :w3)", f.Clause())
	})

	t.Run("WhereWithOrderLimitOffset", func(t *testing.T) {
		f := (&Filter{}).
			Where("status", builder.Eq, "active").
			OrderBy("-created").
			Limit(20).
			Offset(10)

		require.Equal(t, "WHERE status = :w1 ORDER BY created DESC LIMIT 20 OFFSET 10", f.Clause())
	})

	t.Run("LikeOperators", func(t *testing.T) {
		f := (&Filter{}).
			Where("name", builder.Like, "%ada%").
			And("email", builder.ILike, "%@example.com")

		require.Equal(t, "WHERE name LIKE :w1 AND email ILIKE :w2", f.Clause())
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
		f := (&Filter{}).Or("role", builder.Eq, "admin")
		require.Equal(t, "WHERE role = :w1", f.Clause())
	})

	t.Run("AndBeforeWhereStartsFirstCondition", func(t *testing.T) {
		// And on an empty filter acts like the first condition; no AND keyword is emitted.
		f := (&Filter{}).And("age", builder.Gte, 18)
		require.Equal(t, "WHERE age >= :w1", f.Clause())
	})

	t.Run("OrGroupBeforeWhereBecomesRootGroup", func(t *testing.T) {
		// An empty root accepts the group directly; no leading OR is emitted.
		f := (&Filter{}).OrGroup(func(g *Where) {
			g.Where("role", builder.Eq, "admin")
		})
		require.Equal(t, "WHERE (role = :w1)", f.Clause())
	})

	t.Run("AndGroupBeforeWhereBecomesRootGroup", func(t *testing.T) {
		// An empty root accepts the group directly; no leading AND is emitted.
		f := (&Filter{}).AndGroup(func(g *Where) {
			g.Where("role", builder.Eq, "admin")
		})
		require.Equal(t, "WHERE (role = :w1)", f.Clause())
	})

	t.Run("OrThenWhereReplacesPriorCondition", func(t *testing.T) {
		// Where replaces the entire clause
		f := (&Filter{}).
			Or("legacy", builder.Eq, true).
			Where("status", builder.Eq, "active")
		require.Equal(t, "WHERE status = :w1", f.Clause())
	})

	t.Run("OrGroupThenWhereReplacesEntireClause", func(t *testing.T) {
		// Where replaces the entire clause
		f := (&Filter{}).
			OrGroup(func(g *Where) {
				g.Where("role", builder.Eq, "admin")
			}).
			Where("status", builder.Eq, "active")
		require.Equal(t, "WHERE status = :w1", f.Clause())
	})

	t.Run("EmptyOrGroupThenWhere", func(t *testing.T) {
		// Where replaces the entire clause
		f := (&Filter{}).
			OrGroup(func(g *Where) {}).
			Where("status", builder.Eq, "active")
		require.Equal(t, "WHERE status = :w1", f.Clause())
	})

	t.Run("EmptyOrGroupOnly", func(t *testing.T) {
		// An empty group is ignored; no OR keyword is emitted.
		f := (&Filter{}).OrGroup(func(g *Where) {})
		require.Empty(t, f.Clause())
		require.Empty(t, f.Params())
	})

	t.Run("OrOrWithoutWhere", func(t *testing.T) {
		// First Or starts the expression; the second Or appends normally.
		f := (&Filter{}).
			Or("a", builder.Eq, 1).
			Or("b", builder.Eq, 2)
		require.Equal(t, "WHERE a = :w1 OR b = :w2", f.Clause())
	})

	t.Run("OrGroupOrBeforeWhereInGroup", func(t *testing.T) {
		// Inside a group, Or starts the sub-expression; Where replaces it.
		f := (&Filter{}).OrGroup(func(g *Where) {
			g.Or("role", builder.Eq, "admin").Where("status", builder.Eq, "active")
		})
		require.Equal(t, "WHERE (status = :w1)", f.Clause())
	})

	t.Run("OrOrGroupThenWhereReplacesAll", func(t *testing.T) {
		// Where replaces the entire clause
		f := (&Filter{}).
			Or("legacy", builder.Eq, true).
			OrGroup(func(g *Where) {
				g.Where("role", builder.Eq, "admin")
			}).
			Where("status", builder.Eq, "active")
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
		Where("status", builder.Eq, "active").
		And("deleted_at", builder.Eq, nil).
		And("organization_id", builder.Ne, "").
		AndGroup(func(g *Where) {
			g.Where("role", builder.Eq, "admin").
				Or("role", builder.Eq, "editor").
				Or("role", builder.Eq, "viewer").
				AndGroup(func(g *Where) {
					g.Where("verified", builder.Eq, true).
						And("mfa_enabled", builder.Eq, true).
						OrGroup(func(g *Where) {
							g.Where("sso_provider", builder.Eq, "okta").
								And("sso_external_id", builder.Ne, "")
						})
				})
		}).
		OrGroup(func(g *Where) {
			g.Where("tier", builder.Gte, 2).
				And("name", builder.Like, "%pro%").
				AndGroup(func(g *Where) {
					g.Where("billing_status", builder.Eq, "current").
						Or("trial_ends_at", builder.Gt, "2026-01-01")
				})
		}).
		Or("legacy", builder.Eq, true).
		And("age", builder.Gte, 18).
		And("age", builder.Lte, 65).
		And("score", builder.Lt, 100).
		And("rating", builder.Gte, 3.5).
		OrGroup(func(g *Where) {
			g.Where("region", builder.Eq, "us").
				Or("region", builder.Eq, "eu").
				And("featured", builder.Eq, true).
				AndGroup(func(g *Where) {
					g.Where("campaign", builder.Eq, "spring").
						Or("campaign", builder.Eq, "summer").
						Or("tags", builder.ILike, "%launch%")
				})
		}).
		OrGroup(func(g *Where) {
			g.Where("invited_by", builder.Ne, "").
				And("invite_accepted_at", builder.Eq, nil).
				OrGroup(func(g *Where) {
					g.Where("signup_source", builder.Eq, "referral").
						And("referral_code", builder.Like, "REF-%")
				})
		}).
		And("email", builder.ILike, "%@example.com").
		Or("vip", builder.Eq, true).
		OrderBy("-created", "name", "-score").
		Limit(50).
		Offset(100)

	// Expected SQL clause:
	//
	// WHERE status = :w1
	//   AND deleted_at = :w2
	//   AND organization_id != :w3
	//   AND (
	//     role = :w4 OR role = :w5 OR role = :w6
	//     AND (
	//       verified = :w7 AND mfa_enabled = :w8
	//       OR (sso_provider = :w9 AND sso_external_id != :w10)
	//     )
	//   )
	//   OR (
	//     tier >= :w11 AND name LIKE :w12
	//     AND (billing_status = :w13 OR trial_ends_at > :w14)
	//   )
	//   OR legacy = :w15
	//   AND age >= :w16 AND age <= :w17
	//   AND score < :w18 AND rating >= :w19
	//   OR (
	//     region = :w20 OR region = :w21 AND featured = :w22
	//     AND (campaign = :w23 OR campaign = :w24 OR tags ILIKE :w25)
	//   )
	//   OR (
	//     invited_by != :w26 AND invite_accepted_at = :w27
	//     OR (signup_source = :w28 AND referral_code LIKE :w29)
	//   )
	//   AND email ILIKE :w30 OR vip = :w31
	// ORDER BY created DESC, name ASC, score DESC
	// LIMIT 50 OFFSET 100
	require.Equal(t,
		"WHERE status = :w1 AND deleted_at = :w2 AND organization_id != :w3 AND (role = :w4 OR role = :w5 OR role = :w6 AND (verified = :w7 AND mfa_enabled = :w8 OR (sso_provider = :w9 AND sso_external_id != :w10))) OR (tier >= :w11 AND name LIKE :w12 AND (billing_status = :w13 OR trial_ends_at > :w14)) OR legacy = :w15 AND age >= :w16 AND age <= :w17 AND score < :w18 AND rating >= :w19 OR (region = :w20 OR region = :w21 AND featured = :w22 AND (campaign = :w23 OR campaign = :w24 OR tags ILIKE :w25)) OR (invited_by != :w26 AND invite_accepted_at = :w27 OR (signup_source = :w28 AND referral_code LIKE :w29)) AND email ILIKE :w30 OR vip = :w31 ORDER BY created DESC, name ASC, score DESC LIMIT 50 OFFSET 100",
		f.Clause(),
	)
	require.Equal(t, []sql.NamedArg{
		sql.Named("w1", "active"),
		sql.Named("w2", nil),
		sql.Named("w3", ""),
		sql.Named("w4", "admin"),
		sql.Named("w5", "editor"),
		sql.Named("w6", "viewer"),
		sql.Named("w7", true),
		sql.Named("w8", true),
		sql.Named("w9", "okta"),
		sql.Named("w10", ""),
		sql.Named("w11", 2),
		sql.Named("w12", "%pro%"),
		sql.Named("w13", "current"),
		sql.Named("w14", "2026-01-01"),
		sql.Named("w15", true),
		sql.Named("w16", 18),
		sql.Named("w17", 65),
		sql.Named("w18", 100),
		sql.Named("w19", 3.5),
		sql.Named("w20", "us"),
		sql.Named("w21", "eu"),
		sql.Named("w22", true),
		sql.Named("w23", "spring"),
		sql.Named("w24", "summer"),
		sql.Named("w25", "%launch%"),
		sql.Named("w26", ""),
		sql.Named("w27", nil),
		sql.Named("w28", "referral"),
		sql.Named("w29", "REF-%"),
		sql.Named("w30", "%@example.com"),
		sql.Named("w31", true),
	}, f.Params())
}
