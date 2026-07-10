package store_test

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/tidal/conn"
	"go.rtnl.ai/tidal/model"
	"go.rtnl.ai/tidal/store"
	"go.rtnl.ai/tidal/suite/fixtures"
	"go.rtnl.ai/tidal/suite/mock"
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

// Exercises [tidal.Filter] WHERE, ORDER BY, LIMIT, and OFFSET against a real database.
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

	f := (&tidal.Filter{}).
		Where("name", tidal.Like, "filter-%").
		OrderBy("name").
		Limit(2).
		Offset(1)

	cursor, err := crud.List(tx, f)
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

	// Begin transaction ensuring its rolled back no matter what to prevent other
	// tests from failing because of a hanging write transaction.
	tx := s.BeginTx(nil)
	defer tx.Rollback()

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

	// Begin transaction ensuring its rolled back no matter what to prevent other
	// tests from failing because of a hanging write transaction.
	tx := s.BeginTx(nil)
	defer tx.Rollback()

	user := &fixtures.User{}
	rows, err := tx.Query("SELECT " + joinFields(user.Fields(tidal.List)) + " FROM users LIMIT 1")
	require.NoError(err)

	cursor := store.Rows[*fixtures.User](tx, rows)
	require.NoError(cursor.CloseRows())

	_, err = tx.Exec("SELECT 1")
	require.NoError(err)
}

//============================================================================
// Test Query Construction
//============================================================================

func TestQueries(t *testing.T) {
	var (
		lastQuery string
		lastArgs  []sql.NamedArg
	)

	tx := &mock.Tx{
		OnQuery: func(query string, args ...sql.NamedArg) (*sql.Rows, error) {
			lastQuery = query
			lastArgs = args
			return nil, nil
		},
		OnQueryRow: func(query string, args ...sql.NamedArg) *conn.Row {
			lastQuery = query
			lastArgs = args
			return &conn.Row{}
		},
		OnExec: func(query string, args ...sql.NamedArg) (sql.Result, error) {
			lastQuery = query
			lastArgs = args
			return &mock.Result{}, nil
		},
		OnRollback: func() error {
			return nil
		},
		OnCommit: func() error {
			return nil
		},
	}

	reset := func() {
		lastQuery = ""
		lastArgs = nil
		mock.Reset()
		tx.ResetCalls()
	}

	t.Run("DefaultIdentifier", func(t *testing.T) {
		crud := store.New[*mock.Model]("mock")
		reset()

		t.Run("List", func(t *testing.T) {
			t.Cleanup(reset)
			crud.List(tx, nil)

			tx.AssertCalledOnce(t, mock.Query)
			tx.AssertNotCalled(t, mock.QueryRow)
			tx.AssertNotCalled(t, mock.Exec)
			tx.AssertNotCalled(t, mock.Rollback)
			tx.AssertNotCalled(t, mock.Commit)

			require.Equal(t, "SELECT foo, bar FROM mock", lastQuery)
			require.Empty(t, lastArgs)
		})

		t.Run("Create", func(t *testing.T) {
			t.Cleanup(reset)
			model := &mock.Model{
				OnValidate: func(op model.Operation) error {
					return nil
				},
				OnPrepare: func(op model.Operation) {},
			}
			t.Cleanup(model.Reset)

			_, err := crud.Create(tx, model)
			require.NoError(t, err)
			tx.AssertCalledOnce(t, mock.Exec)
			tx.AssertNotCalled(t, mock.QueryRow)
			tx.AssertNotCalled(t, mock.Query)
			tx.AssertNotCalled(t, mock.Rollback)
			tx.AssertNotCalled(t, mock.Commit)

			model.AssertCalledOnce(t, mock.Validate)
			model.AssertCalledOnce(t, mock.Prepare)
			model.AssertCalledOnce(t, mock.Params)
			model.AssertNotCalled(t, mock.Fields)
			model.AssertNotCalled(t, mock.Scan)

			require.Equal(t, "INSERT INTO mock (foo, bar) VALUES (:foo, :bar)", lastQuery)
			require.Equal(t, []sql.NamedArg{
				{Name: "foo", Value: "qux"},
				{Name: "bar", Value: "baz"},
			}, lastArgs)
		})

		t.Run("Retrieve", func(t *testing.T) {
			t.Cleanup(reset)

			_, err := crud.Retrieve(tx, sql.NamedArg{Name: "id", Value: 42})
			require.NoError(t, err)
			tx.AssertCalledOnce(t, mock.QueryRow)
			tx.AssertNotCalled(t, mock.Query)
			tx.AssertNotCalled(t, mock.Exec)
			tx.AssertNotCalled(t, mock.Rollback)
			tx.AssertNotCalled(t, mock.Commit)

			require.Equal(t, "SELECT foo, bar FROM mock WHERE id=:id", lastQuery)
			require.Equal(t, []sql.NamedArg{
				{Name: "id", Value: 42},
			}, lastArgs)
		})

		t.Run("Update", func(t *testing.T) {
			t.Cleanup(reset)

			obj := &mock.Model{
				OnValidate: func(model.Operation) error {
					return nil
				},
				OnPrepare: func(model.Operation) {},
			}
			obj.OnParams = func(model.Operation) []sql.NamedArg {
				return []sql.NamedArg{{Name: "id", Value: 42}, {Name: "foo", Value: "qux"}, {Name: "bar", Value: "baz"}}
			}
			t.Cleanup(obj.Reset)

			err := crud.Update(tx, obj)
			require.NoError(t, err)
			tx.AssertCalledOnce(t, mock.Exec)
			obj.AssertCalledOnce(t, mock.Validate)
			obj.AssertCalledOnce(t, mock.Prepare)
			obj.AssertCalledOnce(t, mock.Params)
			tx.AssertNotCalled(t, mock.QueryRow)
			tx.AssertNotCalled(t, mock.Query)
			tx.AssertNotCalled(t, mock.Rollback)
			tx.AssertNotCalled(t, mock.Commit)
			obj.AssertNotCalled(t, mock.Fields)
			obj.AssertNotCalled(t, mock.Scan)

			require.Equal(t, "UPDATE mock SET foo=:foo, bar=:bar WHERE id=:id", lastQuery)
			require.Equal(t, []sql.NamedArg{
				{Name: "id", Value: 42},
				{Name: "foo", Value: "qux"},
				{Name: "bar", Value: "baz"},
			}, lastArgs)
		})

		t.Run("Delete", func(t *testing.T) {
			t.Cleanup(reset)

			_, err := crud.Delete(tx, sql.NamedArg{Name: "id", Value: 42})
			require.NoError(t, err)
			tx.AssertCalledOnce(t, mock.Exec)
			tx.AssertNotCalled(t, mock.QueryRow)
			tx.AssertNotCalled(t, mock.Query)
			tx.AssertNotCalled(t, mock.Rollback)
			tx.AssertNotCalled(t, mock.Commit)

			require.Equal(t, "DELETE FROM mock WHERE id=:id", lastQuery)
			require.Equal(t, []sql.NamedArg{
				{Name: "id", Value: 42},
			}, lastArgs)
		})

		t.Run("DefaultIdentifierRequired", func(t *testing.T) {
			t.Cleanup(reset)

			crud := store.New[*mock.Model]("mock")
			reset()

			obj := &mock.Model{
				OnValidate: func(model.Operation) error {
					return nil
				},
				OnPrepare: func(model.Operation) {},
			}

			err := crud.Update(tx, obj)
			require.Error(t, err, "expected an error when the default identifier field is not in the params")
			require.EqualError(t, err, "default identifier field \"id\" not found in update parameters")
		})
	})

	t.Run("IdentifiersInterface", func(t *testing.T) {
		crud := store.New[*mock.Identifier]("mock")
		reset()

		t.Run("SingleUnknown", func(t *testing.T) {
			t.Cleanup(reset)

			obj := &mock.Identifier{
				OnIdentifier: func() []sql.NamedArg {
					return []sql.NamedArg{
						{Name: "slug", Value: "test-model"},
					}
				},
			}

			crud.Update(tx, obj)
			tx.AssertCalledOnce(t, mock.Exec)
			tx.AssertNotCalled(t, mock.QueryRow)
			tx.AssertNotCalled(t, mock.Query)
			tx.AssertNotCalled(t, mock.Rollback)
			tx.AssertNotCalled(t, mock.Commit)

			obj.AssertCalledOnce(t, mock.Identifiers)
			obj.AssertCalledOnce(t, mock.Params)
			obj.AssertNotCalled(t, mock.Fields)
			obj.AssertNotCalled(t, mock.Scan)

			require.Equal(t, "UPDATE mock SET foo=:foo, bar=:bar WHERE slug=:slug", lastQuery)
			require.Equal(t, []sql.NamedArg{
				{Name: "foo", Value: "qux"},
				{Name: "bar", Value: "baz"},
				{Name: "slug", Value: "test-model"},
			}, lastArgs)
		})

		t.Run("SingleParam", func(t *testing.T) {
			t.Cleanup(reset)

			obj := &mock.Identifier{
				OnIdentifier: func() []sql.NamedArg {
					return []sql.NamedArg{
						{Name: "foo", Value: "qux"},
					}
				},
			}

			crud.Update(tx, obj)
			tx.AssertCalledOnce(t, mock.Exec)
			tx.AssertNotCalled(t, mock.QueryRow)
			tx.AssertNotCalled(t, mock.Query)
			tx.AssertNotCalled(t, mock.Rollback)
			tx.AssertNotCalled(t, mock.Commit)

			obj.AssertCalledOnce(t, mock.Identifiers)
			obj.AssertCalledOnce(t, mock.Params)
			obj.AssertNotCalled(t, mock.Fields)
			obj.AssertNotCalled(t, mock.Scan)

			require.Equal(t, "UPDATE mock SET foo=:foo, bar=:bar WHERE foo=:foo", lastQuery)
			require.Equal(t, []sql.NamedArg{
				{Name: "foo", Value: "qux"},
				{Name: "bar", Value: "baz"},
			}, lastArgs)
		})

		t.Run("CompositeUnknown", func(t *testing.T) {
			t.Cleanup(reset)

			obj := &mock.Identifier{
				OnIdentifier: func() []sql.NamedArg {
					return []sql.NamedArg{
						{Name: "slug", Value: "test-model"},
						{Name: "parent", Value: 42},
					}
				},
			}

			crud.Update(tx, obj)
			tx.AssertCalledOnce(t, mock.Exec)
			tx.AssertNotCalled(t, mock.QueryRow)
			tx.AssertNotCalled(t, mock.Query)
			tx.AssertNotCalled(t, mock.Rollback)
			tx.AssertNotCalled(t, mock.Commit)

			obj.AssertCalledOnce(t, mock.Identifiers)
			obj.AssertCalledOnce(t, mock.Params)
			obj.AssertNotCalled(t, mock.Fields)
			obj.AssertNotCalled(t, mock.Scan)

			require.Equal(t, "UPDATE mock SET foo=:foo, bar=:bar WHERE slug=:slug AND parent=:parent", lastQuery)
			require.Equal(t, []sql.NamedArg{
				{Name: "foo", Value: "qux"},
				{Name: "bar", Value: "baz"},
				{Name: "slug", Value: "test-model"},
				{Name: "parent", Value: 42},
			}, lastArgs)
		})

		t.Run("CompositeKnown", func(t *testing.T) {
			t.Cleanup(reset)

			obj := &mock.Identifier{
				OnIdentifier: func() []sql.NamedArg {
					return []sql.NamedArg{
						{Name: "foo", Value: "qux"},
						{Name: "bar", Value: "baz"},
					}
				},
			}

			crud.Update(tx, obj)
			tx.AssertCalledOnce(t, mock.Exec)
			tx.AssertNotCalled(t, mock.QueryRow)
			tx.AssertNotCalled(t, mock.Query)
			tx.AssertNotCalled(t, mock.Rollback)
			tx.AssertNotCalled(t, mock.Commit)

			obj.AssertCalledOnce(t, mock.Identifiers)
			obj.AssertCalledOnce(t, mock.Params)
			obj.AssertNotCalled(t, mock.Fields)
			obj.AssertNotCalled(t, mock.Scan)

			require.Equal(t, "UPDATE mock SET foo=:foo, bar=:bar WHERE foo=:foo AND bar=:bar", lastQuery)
			require.Equal(t, []sql.NamedArg{
				{Name: "foo", Value: "qux"},
				{Name: "bar", Value: "baz"},
			}, lastArgs)
		})

		t.Run("Multiple", func(t *testing.T) {
			t.Cleanup(reset)

			obj := &mock.Identifier{
				OnIdentifier: func() []sql.NamedArg {
					return []sql.NamedArg{
						{Name: "slug", Value: "test-model"},
						{Name: "parent", Value: 42},
						{Name: "category", Value: "test-category"},
					}
				},
			}

			crud.Update(tx, obj)
			tx.AssertCalledOnce(t, mock.Exec)
			tx.AssertNotCalled(t, mock.QueryRow)
			tx.AssertNotCalled(t, mock.Query)
			tx.AssertNotCalled(t, mock.Rollback)
			tx.AssertNotCalled(t, mock.Commit)

			obj.AssertCalledOnce(t, mock.Identifiers)
			obj.AssertCalledOnce(t, mock.Params)
			obj.AssertNotCalled(t, mock.Fields)
			obj.AssertNotCalled(t, mock.Scan)

			require.Equal(t, "UPDATE mock SET foo=:foo, bar=:bar WHERE slug=:slug AND parent=:parent AND category=:category", lastQuery)
			require.Equal(t, []sql.NamedArg{
				{Name: "foo", Value: "qux"},
				{Name: "bar", Value: "baz"},
				{Name: "slug", Value: "test-model"},
				{Name: "parent", Value: 42},
				{Name: "category", Value: "test-category"},
			}, lastArgs)
		})

		t.Run("IdentifiersRequired", func(t *testing.T) {
			t.Cleanup(reset)

			crud := store.New[*mock.Identifier]("mock")
			reset()

			obj := &mock.Identifier{
				OnIdentifier: func() []sql.NamedArg {
					return nil
				},
			}

			err := crud.Update(tx, obj)
			require.Error(t, err, "expected an error when the default identifier field is not in the params")
			require.EqualError(t, err, "no identifiers found for model update")
		})
	})
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
