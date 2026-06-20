package errors_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	. "go.rtnl.ai/tidal/errors"
	"go.rtnl.ai/tidal/suite"
	"go.rtnl.ai/tidal/suite/fixtures"
)

func TestPostgres(t *testing.T) {
	s := &ErrorsTests{}
	s.DatabaseError = PostgresError

	s.Provider = &suite.PostgresProvider{}
	s.Teardown = suite.TeardownNone
	s.Migrations = fixtures.File("errors/postgres_schema.sql")

	suite.Run(t, s)
}

func TestPostgresError(t *testing.T) {
	err := errors.New("testing error")
	dberr := PostgresError(err)

	require.EqualError(t, dberr, "postgres: testing error")

	e, ok := errors.AsType[*Error](dberr)
	require.True(t, ok)
	require.Equal(t, "postgres", e.Provider)
	require.ErrorIs(t, e.Err, err)
}
