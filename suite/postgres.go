package suite

import (
	"database/sql"
	"errors"

	"go.rtnl.ai/x/dsn"

	_ "github.com/lib/pq"
)

type PostgresSuite struct {
	DatabaseSuite
}

func (s *PostgresSuite) SetupSuite() {
	s.T().Log("setting up the postgres test suite")
	s.CreateDB("")
}

func (s *PostgresSuite) AfterTest(suiteName, testName string) {
	s.T().Logf("resetting the postgres test suite after test %s.%s", suiteName, testName)
	s.ResetDB()
}

func (s *PostgresSuite) CreateDB(databaseURL string) {
	var err error
	require := s.Require()

	s.dsn, err = s.ResolveDSN(databaseURL)
	require.NoError(err, "could not resolve database URL")

	s.T().Logf("database dsn resolved to %s", s.dsn.String())

	connStr, pgopts, err := dsn.PGConnectionOptions(s.dsn, nil)
	require.NoError(err, "could not get PostgreSQL connection options")

	s.DB, err = sql.Open("postgres", connStr)
	require.NoError(err, "could not open database")

	// Apply any options to the connection string.
	s.DB.SetMaxIdleConns(pgopts.MaxIdleConns)
	s.DB.SetMaxOpenConns(pgopts.MaxOpenConns)
	s.DB.SetConnMaxLifetime(pgopts.ConnMaxLifetime)
	s.DB.SetConnMaxIdleTime(pgopts.ConnMaxIdleTime)

	require.NoError(s.DB.Ping(), "could not ping database")
}

const truncateQuery = `
DO $$
DECLARE
    l_stmt TEXT;
BEGIN
    SELECT 'TRUNCATE TABLE ' || string_agg(format('%I.%I', schemaname, tablename), ', ') || ' RESTART IDENTITY CASCADE'
    INTO l_stmt
    FROM pg_tables
    WHERE schemaname = 'public';

    IF l_stmt IS NOT NULL THEN
        EXECUTE l_stmt;
    END IF;
END $$;
`

func (s *PostgresSuite) ResetDB() {
	if s.DB != nil {
		require := s.Require()

		_, err := s.DB.Exec(truncateQuery)
		require.NoError(err, "could not truncate database")
	} else {
		s.T().Log("cannot reset the postgres database because s.DB is nil")
	}
}

// Resolves the database URL from the environment variable POSTGRES_DATABASE_URL. If
// the database URL is not specified, it will be loaded from the environment variable
// TEST_DATABASE_URL or TIDAL_DATABASE_URL or DATABASE_URL. If none are found, it will
// return an error meaning that the tests should fail or be skipped.
func (s *PostgresSuite) ResolveDSN(databaseURL string) (uri *dsn.DSN, err error) {
	// If the database URL is not specified load it from the environment variable.
	if databaseURL == "" {
		databaseURL = DatabaseURL(POSTGRES_DATABASE_URL, TEST_DATABASE_URL, TIDAL_DATABASE_URL)
	}

	// Attempt to parse the database URL.
	if databaseURL != "" {
		if uri, err = dsn.Parse(databaseURL); err != nil {
			return nil, err
		}

		if uri.Provider != dsn.Postgres {
			return nil, errors.Join(ErrInvalidProvider, ErrPostgresRequired)
		}

		return uri, nil
	}

	// Otherwise return an error that the database URL is not specified.
	return nil, ErrNoDatabaseURL
}

const countQuery = `SELECT COUNT(*) FROM `

func (s *PostgresSuite) Count(table string) (count int) {
	require := s.Require()
	query := countQuery + table

	tx, cancel := s.BeginTx(&sql.TxOptions{
		ReadOnly:  true,
		Isolation: sql.LevelDefault,
	})
	defer cancel()
	defer tx.Rollback()

	row := tx.QueryRow(query)
	require.NoError(row.Scan(&count), "could not scan count")

	return count
}
