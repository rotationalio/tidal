package errors_test

import (
	"context"
	"database/sql"
	"testing"

	. "go.rtnl.ai/tidal/errors"
	"go.rtnl.ai/tidal/suite"
	"go.rtnl.ai/tidal/suite/fixtures"
)

type PostgresTests struct {
	suite.PostgresSuite
}

func TestPostgres(t *testing.T) {
	s := &PostgresTests{}
	s.Migrations = fixtures.File("errors/postgres_schema.sql")
	suite.Run(t, s)
}

func (s *PostgresTests) TestNil() {
	s.Require().Nil(PostgresError(nil))
}

func (s *PostgresTests) TestNotFound() {
	err := s.DB.QueryRow("SELECT * FROM authors WHERE id = $1", 999).Scan(&struct{}{})
	s.Require().ErrorIs(PostgresError(err), ErrNotFound)
}

func (s *PostgresTests) TestAlreadyExists() {
	_, err := s.DB.Exec("INSERT INTO authors (name, email) VALUES ($1, $2)", "Zelda Sayre", "f.scott.fitzgerald@example.com")
	s.Require().ErrorIs(PostgresError(err), ErrAlreadyExists)
}

func (s *PostgresTests) TestMissingReference() {
	_, err := s.DB.Exec("INSERT INTO books (title, author_id) VALUES ($1, $2)", "Where the Red Fern Grows", 999)
	s.Require().ErrorIs(PostgresError(err), ErrMissingReference)
}

func (s *PostgresTests) TestNotNull() {
	_, err := s.DB.Exec("INSERT INTO authors (name) VALUES ($1)", nil)
	s.Require().ErrorIs(PostgresError(err), ErrNotNull)
}

func (s *PostgresTests) TestConstraint() {
	_, err := s.DB.Exec("INSERT INTO books (title, author_id, price) VALUES ($1, $2, $3)", "Zelda Sayre", 1, -14.99)
	s.Require().ErrorIs(PostgresError(err), ErrConstraint)
}

func (s *PostgresTests) TestDeleteRestricted() {
	require := s.Require()

	var authorID int64
	err := s.DB.QueryRow("SELECT id FROM authors WHERE email = $1", "f.scott.fitzgerald@example.com").Scan(&authorID)
	require.NoError(err)

	_, err = s.DB.Exec("DELETE FROM authors WHERE id = $1", authorID)
	require.ErrorIs(PostgresError(err), ErrDeleteRestricted)
}

func (s *PostgresTests) TestReadOnly() {
	require := s.Require()
	tx, err := s.DB.BeginTx(context.Background(), &sql.TxOptions{ReadOnly: true})
	require.NoError(err)
	_, err = tx.Exec(
		"UPDATE authors SET email = :email WHERE id = :id",
		sql.Named("email", "f.scott.fitzgerald@example.com"),
		sql.Named("id", 1),
	)
	s.Require().ErrorIs(PostgresError(err), ErrReadOnly)
}

func (s *PostgresTests) TestSyntaxError() {
	_, err := s.DB.Exec("UPDATE missing SET email='foo' where age>18")
	s.Require().EqualError(PostgresError(err), `postgres: ERROR: relation "missing" does not exist (SQLSTATE 42P01)`)
}
