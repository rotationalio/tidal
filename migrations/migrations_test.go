package migrations_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	. "go.rtnl.ai/tidal/migrations"
)

func TestMigrations_Load(t *testing.T) {
	t.Run("Postgres", func(t *testing.T) {
		migrations, err := Load(postgresFS)
		require.NoError(t, err)
		require.NotNil(t, migrations)

		require.Equal(t, 3, len(migrations))
		require.Equal(t, 1, migrations[0].ID)
		require.Equal(t, "Initial Schema", migrations[0].Name)
		require.Empty(t, migrations[0].Version)
		require.Empty(t, migrations[0].Applied)
		require.Equal(t, "testdata/postgres/0001_initial_schema.sql", migrations[0].Path)

		require.Equal(t, 2, migrations[1].ID)
		require.Equal(t, "Taxonomy", migrations[1].Name)
		require.Empty(t, migrations[1].Version)
		require.Empty(t, migrations[1].Applied)
		require.Equal(t, "testdata/postgres/0002_taxonomy.sql", migrations[1].Path)

		require.Equal(t, 3, migrations[2].ID)
		require.Equal(t, "Post Meta", migrations[2].Name)
		require.Empty(t, migrations[2].Version)
		require.Empty(t, migrations[2].Applied)
		require.Equal(t, "testdata/postgres/0003_post_meta.sql", migrations[2].Path)
	})

	t.Run("SQLite", func(t *testing.T) {
		migrations, err := Load(sqliteFS)
		require.NoError(t, err)
		require.NotNil(t, migrations)

		require.Equal(t, 3, len(migrations))
		require.Equal(t, 1, migrations[0].ID)
		require.Equal(t, "Initial Schema", migrations[0].Name)
		require.Empty(t, migrations[0].Version)
		require.Empty(t, migrations[0].Applied)
		require.Equal(t, "testdata/sqlite/0001_initial_schema.sql", migrations[0].Path)

		require.Equal(t, 2, migrations[1].ID)
		require.Equal(t, "Taxonomy", migrations[1].Name)
		require.Empty(t, migrations[1].Version)
		require.Empty(t, migrations[1].Applied)
		require.Equal(t, "testdata/sqlite/0002_taxonomy.sql", migrations[1].Path)

		require.Equal(t, 3, migrations[2].ID)
		require.Equal(t, "Post Meta", migrations[2].Name)
		require.Empty(t, migrations[2].Version)
		require.Empty(t, migrations[2].Applied)
		require.Equal(t, "testdata/sqlite/0003_post_meta.sql", migrations[2].Path)
	})
}

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

func TestMigration_SQL(t *testing.T) {
	migrations, err := Load(postgresFS)
	require.NoError(t, err)
	require.NotNil(t, migrations)

	t.Run("Happy Path", func(t *testing.T) {
		for _, migration := range migrations {
			sql, err := migration.SQL()
			require.NoError(t, err)
			require.NotEmpty(t, sql)
		}
	})

	t.Run("Unbound", func(t *testing.T) {
		migration := &Migration{}
		sql, err := migration.SQL()
		require.ErrorIs(t, err, ErrUnboundMigration)
		require.Empty(t, sql)
	})

	t.Run("NoPath", func(t *testing.T) {
		migration := &Migration{
			ID:   1,
			Name: "Initial Schema",
		}
		migration.WithFS(postgresFS)

		sql, err := migration.SQL()
		require.ErrorContains(t, err, fmt.Sprintf("cannot read sql for migration %d", migration.ID))
		require.Empty(t, sql)
	})

	t.Run("FileNotFound", func(t *testing.T) {
		migration := &Migration{
			ID:   1,
			Name: "Initial Schema",
			Path: "0008_does_not_exist.sql",
		}
		migration.WithFS(postgresFS)

		sql, err := migration.SQL()
		require.ErrorContains(t, err, fmt.Sprintf("cannot open migration file %q: open %s: file does not exist", migration.Path, migration.Path))
		require.Empty(t, sql)
	})
}
