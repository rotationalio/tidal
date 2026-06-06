package migrations_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	. "go.rtnl.ai/tidal/migrations"
)

func TestMigrations_Validate(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		testCases := []Migrations{
			{
				{
					ID:   1,
					Name: "Initial Schema",
					Path: "0001_initial_schema.sql",
				},
			},
			{
				{
					ID:   1,
					Name: "Initial Schema",
					Path: "0001_initial_schema.sql",
				},
				{
					ID:   2,
					Name: "Add Users Table",
					Path: "0002_add_users_table.sql",
				},
				{
					ID:   3,
					Name: "Add Posts Table",
					Path: "0003_add_posts_table.sql",
				},
			},
			{
				{
					ID:   1001,
					Name: "Initial Schema",
					Path: "1001_initial_schema.sql",
				},
			},
		}

		for i, testCase := range testCases {
			err := testCase.Validate()
			require.NoError(t, err, "test case %d failed", i)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		testCases := []struct {
			err   error
			input Migrations
		}{
			{
				err:   ErrNoMigrations,
				input: Migrations{},
			},
			{
				err: ErrInvalidID,
				input: Migrations{
					{
						ID:   0,
						Name: "Initial Schema",
						Path: "0001_initial_schema.sql",
					},
				},
			},
			{
				err: ErrInvalidID,
				input: Migrations{
					{
						ID:   -8,
						Name: "Initial Schema",
						Path: "0008_initial_schema.sql",
					},
				},
			},
			{
				err: ErrInvalidSequence,
				input: Migrations{
					{
						ID:   2,
						Name: "Add Users Table",
						Path: "0002_add_users_table.sql",
					},
					{
						ID:   1,
						Name: "Initial Schema",
						Path: "0001_initial_schema.sql",
					},
				},
			},
			{
				err: ErrInvalidSequence,
				input: Migrations{
					{
						ID:   1,
						Name: "Initial Schema",
						Path: "0001_initial_schema.sql",
					},
					{
						ID:   1,
						Name: "Add Users Table",
						Path: "0001_add_users_table.sql",
					},
				},
			},
			{
				err: ErrInvalidSequence,
				input: Migrations{
					{
						ID:   1,
						Name: "Initial Schema",
						Path: "0001_initial_schema.sql",
					},
					{
						ID:   2,
						Name: "Add Users Table",
						Path: "0002_add_users_table.sql",
					},
					{
						ID:   1,
						Name: "Add Posts Table",
						Path: "0001_add_posts_table.sql",
					},
				},
			},
			{
				err: ErrInvalidSequence,
				input: Migrations{
					{
						ID:   1,
						Name: "Initial Schema",
						Path: "1_initial_schema.sql",
					},
					{
						ID:   15,
						Name: "Add Users Table",
						Path: "15_add_users_table.sql",
					},
					{
						ID:   2,
						Name: "Add Posts Table",
						Path: "2_add_posts_table.sql",
					},
				},
			},
			{
				err: ErrDuplicateName,
				input: Migrations{
					{
						ID:   1,
						Name: "Initial Schema",
						Path: "0001_initial_schema.sql",
					},
					{
						ID:   2,
						Name: "Initial Schema",
						Path: "0002_initial_schema.sql",
					},
				},
			},
		}

		for i, testCase := range testCases {
			err := testCase.input.Validate()
			require.ErrorIs(t, err, testCase.err, "test case %d failed", i)
		}
	})
}

func TestMigrations_Sort(t *testing.T) {
	assertSorted := func(t *testing.T, migrations Migrations) {
		t.Helper()
		for i, migration := range migrations {
			require.Equal(t, i+1, migration.ID)
		}
	}

	m1 := Migrations{
		{
			ID:   1,
			Name: "Initial Schema",
			Path: "0001_initial_schema.sql",
		},
		{
			ID:   2,
			Name: "Add Users Table",
			Path: "0002_add_users_table.sql",
		},
		{
			ID:   3,
			Name: "Add Posts Table",
			Path: "0003_add_posts_table.sql",
		},
		{
			ID:   4,
			Name: "Add Comments Table",
			Path: "0004_add_comments_table.sql",
		},
		{
			ID:   5,
			Name: "Add Likes Table",
			Path: "0005_add_likes_table.sql",
		},
		{
			ID:   6,
			Name: "Add Tags Table",
			Path: "0006_add_tags_table.sql",
		},
		{
			ID:   7,
			Name: "Add Categories Table",
			Path: "0007_add_categories_table.sql",
		},
	}

	m1.Sort()
	assertSorted(t, m1)

	m2 := Migrations{
		{
			ID:   4,
			Name: "Add Comments Table",
			Path: "0004_add_comments_table.sql",
		},
		{
			ID:   2,
			Name: "Add Users Table",
			Path: "0002_add_users_table.sql",
		},
		{
			ID:   1,
			Name: "Initial Schema",
			Path: "0001_initial_schema.sql",
		},
		{
			ID:   3,
			Name: "Add Posts Table",
			Path: "0003_add_posts_table.sql",
		},
		{
			ID:   5,
			Name: "Add Likes Table",
			Path: "0005_add_likes_table.sql",
		},
		{
			ID:   7,
			Name: "Add Categories Table",
			Path: "0007_add_categories_table.sql",
		},
		{
			ID:   6,
			Name: "Add Tags Table",
			Path: "0006_add_tags_table.sql",
		},
	}

	m2.Sort()
	assertSorted(t, m2)

	m3 := Migrations{
		{
			ID:   7,
			Name: "Add Categories Table",
			Path: "0007_add_categories_table.sql",
		},
		{
			ID:   6,
			Name: "Add Tags Table",
			Path: "0006_add_tags_table.sql",
		},
		{
			ID:   5,
			Name: "Add Likes Table",
			Path: "0005_add_likes_table.sql",
		},
		{
			ID:   4,
			Name: "Add Comments Table",
			Path: "0004_add_comments_table.sql",
		},
		{
			ID:   3,
			Name: "Add Posts Table",
			Path: "0003_add_posts_table.sql",
		},
		{
			ID:   2,
			Name: "Add Users Table",
			Path: "0002_add_users_table.sql",
		},
		{
			ID:   1,
			Name: "Initial Schema",
			Path: "0001_initial_schema.sql",
		},
	}

	m3.Sort()
	assertSorted(t, m3)
}
