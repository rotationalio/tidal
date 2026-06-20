package errors_test

import (
	"context"
	"database/sql"
	"errors"

	. "go.rtnl.ai/tidal/errors"
	"go.rtnl.ai/tidal/suite"
	"go.rtnl.ai/x/dsn"
)

type ConvertError func(error) error

type ErrorsTests struct {
	suite.DatabaseSuite
	DatabaseError ConvertError
}

func (s *ErrorsTests) TestNil() {
	s.Require().Nil(s.DatabaseError(nil))
}

func (s *ErrorsTests) TestNotFound() {
	require := s.Require()
	err := s.DB.QueryRow("SELECT * FROM authors WHERE id = $1", 999).Scan(&struct{}{})
	require.Error(err, "expected an error from this operation")
	require.ErrorIs(s.DatabaseError(err), ErrNotFound)
}

func (s *ErrorsTests) TestAlreadyExists() {
	require := s.Require()
	_, err := s.DB.Exec("INSERT INTO authors (name, email) VALUES ($1, $2)", "Zelda Sayre", "f.scott.fitzgerald@example.com")
	require.Error(err, "expected an error from this operation")
	require.ErrorIs(s.DatabaseError(err), ErrAlreadyExists)
}

func (s *ErrorsTests) TestMissingReference() {
	require := s.Require()
	_, err := s.DB.Exec("INSERT INTO books (title, author_id) VALUES ($1, $2)", "Where the Red Fern Grows", 999)
	require.Error(err, "expected an error from this operation")

	dberr := s.DatabaseError(err)
	require.ErrorIs(dberr, ErrMissingReference, "got code %+v from %T, %+v", dberr.(*Error).Code, err, err)
}

func (s *ErrorsTests) TestNotNull() {
	require := s.Require()
	_, err := s.DB.Exec("INSERT INTO authors (name) VALUES ($1)", nil)
	require.Error(err, "expected an error from this operation")
	require.ErrorIs(s.DatabaseError(err), ErrNotNull)
}

func (s *ErrorsTests) TestConstraint() {
	s.Run("DuplicateID", func() {
		require := s.Require()

		var authorID int64
		err := s.DB.QueryRow("SELECT id FROM authors WHERE email = $1", "f.scott.fitzgerald@example.com").Scan(&authorID)
		require.NoError(err)

		_, err = s.DB.Exec("INSERT INTO books (title, author_id, price) VALUES ($1, $2, $3)", "Zelda Sayre", authorID, -14.99)
		require.Error(err, "expected an error from this operation")

		dberr := s.DatabaseError(err)
		require.ErrorIs(dberr, ErrConstraint, "got code %+v from %T, %+v", dberr.(*Error).Code, err, err)
	})

	s.Run("PriceBelowZero", func() {
		require := s.Require()

		_, err := s.DB.Exec("INSERT INTO books (title, author_id, price) VALUES ($1, $2, $3)", "Zelda Sayre", 99, -14.99)
		require.Error(err, "expected an error from this operation")

		dberr := s.DatabaseError(err)
		require.ErrorIs(dberr, ErrConstraint, "got code %+v from %T, %+v", dberr.(*Error).Code, err, err)
	})
}

func (s *ErrorsTests) TestDeleteRestricted() {
	require := s.Require()

	tx, err := s.DB.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: false})
	require.NoError(err)
	defer tx.Rollback()

	var authorID int64
	err = tx.QueryRow("SELECT id FROM authors WHERE email = :email", sql.Named("email", "f.scott.fitzgerald@example.com")).Scan(&authorID)
	require.NoError(err)
	require.NotZero(authorID)

	_, err = tx.Exec("DELETE FROM authors WHERE id = :authorID", sql.Named("authorID", authorID))
	require.Error(err, "expected an error from this operation")

	dberr := s.DatabaseError(err)
	require.ErrorIs(dberr, ErrDeleteRestricted, "got code %+v from %T, %+v", dberr.(*Error).Code, err, err)
}

func (s *ErrorsTests) TestReadOnly() {
	s.Run("Postgres", func() {
		if s.DB.Provider() != dsn.Postgres {
			s.T().Skip("this is a postgres specific test")
			return
		}

		require := s.Require()
		tx, err := s.DB.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
		require.NoError(err)
		defer tx.Rollback()

		_, err = tx.Exec(
			"UPDATE authors SET email = :email WHERE id = :id",
			sql.Named("email", "f.scott.fitzgerald@example.com"),
			sql.Named("id", 1),
		)
		require.ErrorIs(s.DatabaseError(err), ErrReadOnly)
	})

	s.Run("SQLite3", func() {
		if s.DB.Provider() != dsn.SQLite3 {
			s.T().Skip("this is a sqlite3 specific test")
			return
		}

		// Execute the pragma to set the read-only flag
		require := s.Require()
		_, err := s.DB.Exec("PRAGMA query_only = ON")
		require.NoError(err)

		// Ensure that we go back to read-write mode after the test
		defer func() {
			_, err := s.DB.Exec("PRAGMA query_only = OFF")
			require.NoError(err)
		}()

		// Execute a write transaction that should fail in read-only mode
		_, err = s.DB.Exec("UPDATE authors SET email = 'foo' WHERE id = 1")
		require.Error(err, "expected an error from this operation")

		dberr := s.DatabaseError(err)
		require.ErrorIs(dberr, ErrReadOnly, "got code %+v from %T, %+v", dberr.(*Error).Code, err, err)
	})
}

func (s *ErrorsTests) TestSyntaxError() {
	require := s.Require()

	// Execute a query with a syntax error (no relation/table named "missing")
	_, err := s.DB.Exec("UPDATE missing SET email='foo' where age>18")
	err = s.DatabaseError(err)

	// Determine the provider specific error message
	var expected string
	dberr, ok := errors.AsType[*Error](err)
	require.True(ok, "error was not converted correctly")

	switch dberr.Provider {
	case "postgres":
		expected = `postgres: ERROR: relation "missing" does not exist (SQLSTATE 42P01)`
	case "sqlite3+modernc":
		expected = `sqlite3+modernc: SQL logic error: no such table: missing (1)`
	case "sqlite3+mattn":
		expected = `sqlite3+mattn: no such table: missing`
	case "sqlite3+ncruces":
		expected = `sqlite3+ncruces: sqlite3: SQL logic error: no such table: missing`
	}

	require.NotEmpty(expected, "could not determine expected error message for provider %q", dberr.Provider)
	require.EqualError(err, expected)
}
