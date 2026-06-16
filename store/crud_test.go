package store_test

import (
	"database/sql"
	"fmt"

	"go.rtnl.ai/tidal"
	"go.rtnl.ai/tidal/store"
	"go.rtnl.ai/tidal/suite/fixtures"
	"go.rtnl.ai/ulid"
)

//============================================================================
// CRUD Error Paths
//============================================================================

// Update on a row that does not exist must return [tidal.ErrNotFound], not succeed silently.
func (s *StoreTestSuite) TestUpdateNotFound() {
	require := s.Require()
	crud := tidal.New[*fixtures.User]("users")

	tx := s.BeginTx(nil)
	defer tx.Rollback()

	user := fixtures.NewConformanceUser()
	user.Prepare(tidal.Create) // valid ID, but never inserted

	require.ErrorIs(crud.Update(tx, user), tidal.ErrNotFound)
}

// Validate fails; zero ID returns [tidal.ErrMissingID].
func (s *StoreTestSuite) TestUpdateValidation() {
	require := s.Require()
	crud := tidal.New[*fixtures.User]("users")

	tx := s.BeginTx(nil)
	defer tx.Rollback()

	user := &fixtures.User{Name: "validation-only"}
	require.ErrorIs(crud.Update(tx, user), tidal.ErrMissingID)
}

//============================================================================
// List + Filter
//============================================================================

// Exercises [tidal.Filter] ORDER BY / LIMIT / OFFSET against a real database.
// WHERE scoping is supplied via [tidal.Clause] until Filter grows WHERE support.
func (s *StoreTestSuite) TestListFilter() {
	require := s.Require()
	crud := tidal.New[*fixtures.User]("users")

	tx := s.BeginTx(nil)
	defer tx.Rollback()

	names := []string{"filter-aaa", "filter-ccc", "filter-bbb"}
	for _, name := range names {
		u := fixtures.NewConformanceUser()
		u.Name = name
		u.Email = fmt.Sprintf("%s-%s@example.com", name, ulid.MakeSecure().String())
		_, err := crud.Create(tx, u)
		require.NoError(err)
	}

	f := (&tidal.Filter{}).OrderBy("name").Limit(2).Offset(1)
	filter := &tidal.Clause{
		SQL:  "WHERE name LIKE :pattern " + f.Clause(),
		Args: []sql.NamedArg{sql.Named("pattern", "filter-%")},
	}

	cursor, err := crud.List(tx, filter)
	require.NoError(err)

	models, err := cursor.List()
	require.NoError(err)
	require.NoError(cursor.CloseRows())

	require.Len(models, 2)
	require.Equal("filter-bbb", models[0].Name)
	require.Equal("filter-ccc", models[1].Name)
}

//============================================================================
// Cursor Lifecycle
//============================================================================

// [tidal.Cursor.Close] rolls back the transaction.
func (s *StoreTestSuite) TestCursorCloseRollsBackTx() {
	require := s.Require()

	tx := s.BeginTx(nil)

	user := &fixtures.User{}
	rows, err := tx.Query("SELECT " + joinFields(user.Fields(tidal.List)) + " FROM users LIMIT 1")
	require.NoError(err)

	cursor := store.Rows[*fixtures.User](tx, rows)
	require.NoError(cursor.Close())

	_, err = tx.Exec("SELECT 1")
	require.Error(err, "transaction should be rolled back after cursor.Close")
}

// [tidal.Cursor.CloseRows] does not roll back the transaction.
func (s *StoreTestSuite) TestCursorCloseRowsDoesNotRollBackTx() {
	require := s.Require()

	tx := s.BeginTx(nil)

	user := &fixtures.User{}
	rows, err := tx.Query("SELECT " + joinFields(user.Fields(tidal.List)) + " FROM users LIMIT 1")
	require.NoError(err)

	cursor := store.Rows[*fixtures.User](tx, rows)
	require.NoError(cursor.CloseRows())

	_, err = tx.Exec("SELECT 1")
	require.NoError(err)
}

//============================================================================
// Test Helpers
//============================================================================

func joinFields(fields []string) string {
	out := fields[0]
	for _, f := range fields[1:] {
		out += ", " + f
	}
	return out
}
