package errors_test

import (
	"context"
	"database/sql"
	"errors"

	. "go.rtnl.ai/tidal/errors"
	"go.rtnl.ai/tidal/suite"
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
	err := s.DB.QueryRow("SELECT * FROM authors WHERE id = $1", 999).Scan(&struct{}{})
	s.Require().ErrorIs(s.DatabaseError(err), ErrNotFound)
}

func (s *ErrorsTests) TestAlreadyExists() {
	_, err := s.DB.Exec("INSERT INTO authors (name, email) VALUES ($1, $2)", "Zelda Sayre", "f.scott.fitzgerald@example.com")
	s.Require().ErrorIs(s.DatabaseError(err), ErrAlreadyExists)
}

func (s *ErrorsTests) TestMissingReference() {
	_, err := s.DB.Exec("INSERT INTO books (title, author_id) VALUES ($1, $2)", "Where the Red Fern Grows", 999)
	s.Require().ErrorIs(s.DatabaseError(err), ErrMissingReference)
}

func (s *ErrorsTests) TestNotNull() {
	_, err := s.DB.Exec("INSERT INTO authors (name) VALUES ($1)", nil)
	s.Require().ErrorIs(s.DatabaseError(err), ErrNotNull)
}

func (s *ErrorsTests) TestConstraint() {
	s.Run("DuplicateID", func() {
		var authorID int64
		err := s.DB.QueryRow("SELECT id FROM authors WHERE email = $1", "f.scott.fitzgerald@example.com").Scan(&authorID)
		s.Require().NoError(err)

		_, err = s.DB.Exec("INSERT INTO books (title, author_id, price) VALUES ($1, $2, $3)", "Zelda Sayre", authorID, -14.99)
		s.Require().ErrorIs(s.DatabaseError(err), ErrConstraint)
	})

	s.Run("PriceBelowZero", func() {
		_, err := s.DB.Exec("INSERT INTO books (title, author_id, price) VALUES ($1, $2, $3)", "Zelda Sayre", 99, -14.99)
		s.Require().ErrorIs(s.DatabaseError(err), ErrConstraint)
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

	_, err = tx.Exec("DELETE FROM authors WHERE id = :authorID", sql.Named("authorID", authorID))
	require.ErrorIs(s.DatabaseError(err), ErrDeleteRestricted)
}

func (s *ErrorsTests) TestReadOnly() {
	require := s.Require()
	tx, err := s.DB.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	require.NoError(err)
	_, err = tx.Exec(
		"UPDATE authors SET email = :email WHERE id = :id",
		sql.Named("email", "f.scott.fitzgerald@example.com"),
		sql.Named("id", 1),
	)
	s.Require().ErrorIs(s.DatabaseError(err), ErrReadOnly)
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
		expected = `sqlite3+ncruces: no such table: missing`
	}

	require.NotEmpty(expected, "could not determine expected error message for provider %q", dberr.Provider)
	require.EqualError(err, expected)
}
