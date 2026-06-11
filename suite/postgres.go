package suite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go.rtnl.ai/tidal"
	"go.rtnl.ai/x/dsn"
	"go.rtnl.ai/x/randstr"
)

type PostgresSuite struct {
	DatabaseSuite
}

func (s *PostgresSuite) SetupSuite() {
	s.DatabaseSuite.Provider = &PostgresProvider{}
	s.DatabaseSuite.SetupSuite()
}

// The PostgresProvider uses the DSN it gets from the environment or the test suite as
// an admin DSN to connect to the database. If the db in the DSN is specifed as
// 'postgres' (e.g. the default admin database), then a new database is created for the
// test suite alone. This prevents multiple suites running in parallel from interfering
// with each other. Otherwise, the test suite will use the existing database.
type PostgresProvider struct {
	dsn   *dsn.DSN // The test database DSN created by the test suite.
	admin *dsn.DSN // The management database DSN used to create the test database.
}

func (p *PostgresProvider) ResolveDSN(databaseURL string) (_ *dsn.DSN, err error) {
	// If the database URL is not specified load it from the environment variable.
	if databaseURL == "" {
		databaseURL = DatabaseURL(POSTGRES_DATABASE_URL, TEST_DATABASE_URL, TIDAL_DATABASE_URL)
	}

	// if the database URL is still empty, then try to get the postgres env vars.
	if databaseURL == "" {
		if p.dsn, err = PostgresEnv(); err != nil {
			return nil, err
		}
	} else {
		if p.dsn, err = dsn.Parse(databaseURL); err != nil {
			return nil, err
		}
	}

	if p.dsn.Provider != dsn.Postgres {
		p.dsn = nil
		return nil, errors.Join(ErrInvalidProvider, ErrPostgresRequired)
	}

	return p.dsn, nil
}

func (p *PostgresProvider) CreateDB(ctx context.Context, uri *dsn.DSN) (_ *dsn.DSN, err error) {
	// If a non-postgres database is specified then return the original DSN.
	if uri.Path != "postgres" {
		return uri, nil
	}

	// If the database is postgres, then create a new test database.
	p.admin = uri
	p.dsn = uri.Clone()
	p.dsn.Path = "tidal_test_" + randstr.AlphaLower(5) // databases in PG must be lower case

	// Create a new database.
	var admin *tidal.DB
	if admin, err = tidal.Open(ctx, p.admin); err != nil {
		return nil, fmt.Errorf("could not connect to admin database: %w", err)
	}
	defer admin.Close()

	stmt := "CREATE DATABASE " + p.dsn.Path
	if p.admin.User != nil && p.admin.User.Username != "" {
		stmt += " WITH OWNER " + p.admin.User.Username
	}

	if _, err = admin.ExecContext(ctx, stmt); err != nil {
		return nil, fmt.Errorf("could not create database: %w", err)
	}

	return p.dsn, nil
}

func (p *PostgresProvider) DropDB(ctx context.Context, uri *dsn.DSN) (err error) {
	if p.admin != nil && p.dsn != nil {
		var admin *tidal.DB
		if admin, err = tidal.Open(ctx, p.admin); err != nil {
			return fmt.Errorf("could not connect to admin database: %w", err)
		}
		defer admin.Close()

		if _, err = admin.ExecContext(ctx, "DROP DATABASE "+p.dsn.Path); err != nil {
			return fmt.Errorf("could not drop database: %w", err)
		}

		p.admin = nil
		p.dsn = nil
	}
	return nil
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

func (p *PostgresProvider) DropTables(ctx context.Context, conn *sql.DB) error {
	_, err := conn.Exec(dropTableQuery)
	return err
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

func (p *PostgresProvider) TruncateTables(ctx context.Context, conn *sql.DB) error {
	_, err := conn.Exec(truncateQuery)
	return err
}

const countQuery = `SELECT COUNT(*) as count FROM `

func (p *PostgresProvider) Count(tx *sql.Tx, table string) (count int, err error) {
	query := countQuery + table
	row := tx.QueryRow(query)
	err = row.Scan(&count)
	return count, err
}
