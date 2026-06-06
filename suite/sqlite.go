package suite

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"

	"go.rtnl.ai/x/dsn"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteSuite struct {
	DatabaseSuite
}

func (s *SQLiteSuite) SetupSuite() {
	s.CreateDB("")
}

func (s *SQLiteSuite) AfterTest(suiteName, testName string) {
	s.ResetDB()
}

// Create a new SQlite3 database with the specified database URL. If the database URL
// is empty, it will be loaded from the environment variable SQLITE_DATABASE_URL. If
// there is no environment variable, it will create a new database in a temporary
// directory.
func (s *SQLiteSuite) CreateDB(databaseURL string) {
	var err error
	require := s.Require()

	s.dsn, err = s.ResolveDSN(databaseURL)
	require.NoError(err, "could not resolve database URL")

	s.T().Logf("database dsn resolved to %s", s.dsn.String())

	s.DB, err = sql.Open("sqlite3", s.dsn.Path)
	require.NoError(err, "could not open database")

	require.NoError(s.DB.Ping(), "could not ping database")
}

// Resets the SQlite3 database by deleting the database file and calling CreateDB again.
func (s *SQLiteSuite) ResetDB() {
	if s.dsn != nil && s.DB != nil {
		require := s.Require()
		require.NoError(s.Close(), "could not close connection to database")
		require.NoError(os.Remove(s.dsn.Path), "could not delete database file")
		s.CreateDB(s.dsn.String())
	} else {
		s.T().Log("cannot reset the sqlite database because s.dsn or s.DB is nil")
	}
}

// Resolves the database URL from the environment variable SQLITE_DATABASE_URL. If
// the database URL is not specified, it will be loaded from the environment variable
// TEST_DATABASE_URL or TIDAL_DATABASE_URL or DATABASE_URL. If none are found, it will
// create a new DSN with a SQLite3 provider and a path in a temporary directory.
func (s *SQLiteSuite) ResolveDSN(databaseURL string) (uri *dsn.DSN, err error) {
	// If the database URL is not specified load it from the environment variable.
	if databaseURL == "" {
		databaseURL = DatabaseURL(SQLITE_DATABASE_URL, TEST_DATABASE_URL, TIDAL_DATABASE_URL)
	}

	// Attempt to parse the database URL.
	if databaseURL != "" {
		if uri, err = dsn.Parse(databaseURL); err != nil {
			return nil, err
		}

		if uri.Provider != dsn.SQLite3 {
			return nil, errors.Join(ErrInvalidProvider, ErrSqliteRequired)
		}

		return uri, nil
	}

	// Otherwise create a new database in a temporary directory.
	return &dsn.DSN{
		Provider: "sqlite3",
		Path:     filepath.Join(os.TempDir(), "tidal-sqlite-test.db"),
	}, nil
}

const rowIDCountQuery = `SELECT COUNT(rowid) FROM `

func (s *SQLiteSuite) Count(table string) (count int) {
	require := s.Require()
	query := rowIDCountQuery + table

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
