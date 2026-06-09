package suite

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"go.rtnl.ai/x/dsn"

	_ "github.com/lib/pq"
)

type PostgresSuite struct {
	DatabaseSuite

	// dbName is a dedicated database for this suite instance so parallel test
	// packages do not share the same PostgreSQL database.
	dbName string
}

func (s *PostgresSuite) SetupSuite() {
	s.T().Log("setting up the postgres test suite")
	s.CreateDB("")
}

func (s *PostgresSuite) TearDownSuite() {
	s.T().Log("tearing down the postgres test suite")

	require := s.Require()
	if s.DB != nil {
		s.DropTables()
		require.NoError(s.Close(), "could not close connection to database")
		s.DB = nil
	}

	if s.dbName != "" {
		s.dropSuiteDatabase()
		s.dbName = ""
	}

	s.dsn = nil
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

	if databaseURL == "" {
		if s.dbName == "" {
			s.provisionSuiteDatabase()
		} else {
			s.dsn.Path = s.dbName
		}
	}

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

func (s *PostgresSuite) provisionSuiteDatabase() {
	require := s.Require()

	var suffix [4]byte
	_, err := rand.Read(suffix[:])
	require.NoError(err, "could not generate database name suffix")

	baseDB := s.dsn.Path
	s.dbName = fmt.Sprintf("%s_%s", baseDB, hex.EncodeToString(suffix[:]))

	admin := s.dsn.Clone()
	admin.Path = "postgres"

	connStr, _, err := dsn.PGConnectionOptions(admin, nil)
	require.NoError(err, "could not get PostgreSQL admin connection options")

	adminDB, err := sql.Open("postgres", connStr)
	require.NoError(err, "could not open admin database connection")
	defer adminDB.Close()

	require.NoError(adminDB.Ping(), "could not ping admin database")
	_, err = adminDB.Exec("CREATE DATABASE " + s.dbName)
	require.NoError(err, "could not create suite database")

	s.dsn.Path = s.dbName
}

func (s *PostgresSuite) dropSuiteDatabase() {
	require := s.Require()

	admin := s.dsn.Clone()
	admin.Path = "postgres"

	connStr, _, err := dsn.PGConnectionOptions(admin, nil)
	require.NoError(err, "could not get PostgreSQL admin connection options")

	adminDB, err := sql.Open("postgres", connStr)
	require.NoError(err, "could not open admin database connection")
	defer adminDB.Close()

	require.NoError(adminDB.Ping(), "could not ping admin database")
	_, err = adminDB.Exec("DROP DATABASE IF EXISTS " + s.dbName)
	require.NoError(err, "could not drop suite database")
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

const dropTableQuery = `
DO $$
DECLARE
	l_stmt TEXT;
BEGIN
	SELECT 'DROP TABLE IF EXISTS ' || string_agg(format('%I.%I', schemaname, tablename), ', ') || ' CASCADE'
	INTO l_stmt
	FROM pg_tables
	WHERE schemaname = 'public';

	IF l_stmt IS NOT NULL THEN
		EXECUTE l_stmt;
	END IF;
END $$;
`

func (s *PostgresSuite) DropTables() {
	if s.DB != nil {
		require := s.Require()
		_, err := s.DB.Exec(dropTableQuery)
		require.NoError(err, "could not drop tables")
	} else {
		s.T().Log("cannot drop tables because s.DB is nil")
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
