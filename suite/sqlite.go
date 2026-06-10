package suite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.rtnl.ai/x/dsn"

	_ "github.com/mattn/go-sqlite3"
)

type SQLiteSuite struct {
	DatabaseSuite
}

func (s *SQLiteSuite) SetupSuite() {
	s.DatabaseSuite.Provider = &SQLiteProvider{}
	s.DatabaseSuite.SetupSuite()
}

type SQLiteProvider struct {
	dsn    *dsn.DSN
	tmpDir string
}

func (p *SQLiteProvider) ResolveDSN(databaseURL string) (_ *dsn.DSN, err error) {
	// If the database URL is not specified load it from the environment variable.
	if databaseURL == "" {
		databaseURL = DatabaseURL(SQLITE_DATABASE_URL, TEST_DATABASE_URL, TIDAL_DATABASE_URL)
	}

	// Attempt to parse the database URL.
	if databaseURL != "" {
		if p.dsn, err = dsn.Parse(databaseURL); err != nil {
			return nil, err
		}

		if p.dsn.Provider != dsn.SQLite3 {
			return nil, errors.Join(ErrInvalidProvider, ErrSqliteRequired)
		}

		return p.dsn, nil
	}

	// Otherwise create a dedicated temp directory so parallel test packages do not
	// share the same database file.
	if p.dsn == nil {
		if p.tmpDir, err = os.MkdirTemp("", "tidal-test-*"); err != nil {
			return nil, fmt.Errorf("could not create temporary directory: %w", err)
		}
		p.dsn = &dsn.DSN{
			Provider: dsn.SQLite3,
			Path:     filepath.Join(p.tmpDir, "tidal-test.db"),
		}
	}
	return p.dsn, nil
}

func (p *SQLiteProvider) Connect(ctx context.Context, uri *dsn.DSN) (db *sql.DB, err error) {
	if uri.Provider != dsn.SQLite3 {
		return nil, errors.Join(ErrInvalidProvider, ErrSqliteRequired)
	}

	if db, err = sql.Open("sqlite3", uri.Path); err != nil {
		return nil, fmt.Errorf("could not connect to database: %w", err)
	}

	if err = db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("could not ping database: %w", err)
	}

	return db, nil
}

func (p *SQLiteProvider) CreateDB(ctx context.Context, uri *dsn.DSN) (*dsn.DSN, error) {
	// The database is created by the SQLite driver when the connection is opened.
	// No additional action is needed to create the database.
	return uri, nil
}

func (p *SQLiteProvider) DropDB(ctx context.Context, uri *dsn.DSN) (err error) {
	if p.dsn != nil {
		if rmerr := os.Remove(p.dsn.Path); rmerr != nil && !os.IsNotExist(rmerr) {
			err = errors.Join(err, rmerr)
		}
	}

	// Remove the per-suite temp directory created by ResolveDSN.
	if p.tmpDir != "" {
		if rmerr := os.RemoveAll(p.tmpDir); rmerr != nil {
			err = errors.Join(err, rmerr)
		}
		p.tmpDir = ""
	}
	return err
}

func (p *SQLiteProvider) DropTables(ctx context.Context, conn *sql.DB) (err error) {
	var tx *sql.Tx
	if tx, err = conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: false, Isolation: sql.LevelSerializable}); err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	var tables []string
	if tables, err = p.listTables(tx); err != nil {
		return fmt.Errorf("could not list tables: %w", err)
	}

	// Disable foreign key constraints.
	var fk bool
	if fk, err = p.foreignKeysEnabled(tx); err != nil {
		return fmt.Errorf("could not check foreign keys: %w", err)
	}

	if fk {
		// Temporarily disable foreign key constraints.
		if _, err = tx.Exec("PRAGMA foreign_keys = OFF"); err != nil {
			return fmt.Errorf("could not set foreign keys to off: %w", err)
		}
	}

	for _, table := range tables {
		if _, err = tx.Exec("DROP TABLE IF EXISTS " + table); err != nil {
			return fmt.Errorf("could not drop table %s: %w", table, err)
		}
	}

	if fk {
		// Re-enable foreign key constraints.
		if _, err = tx.Exec("PRAGMA foreign_keys = ON"); err != nil {
			return fmt.Errorf("could not set foreign keys to on: %w", err)
		}
	}

	return tx.Commit()
}

func (p *SQLiteProvider) TruncateTables(ctx context.Context, conn *sql.DB) (err error) {
	var tx *sql.Tx
	if tx, err = conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: false, Isolation: sql.LevelSerializable}); err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	var tables []string
	if tables, err = p.listTables(tx); err != nil {
		return fmt.Errorf("could not list tables: %w", err)
	}

	// Disable foreign key constraints.
	var fk bool
	if fk, err = p.foreignKeysEnabled(tx); err != nil {
		return fmt.Errorf("could not check foreign keys: %w", err)
	}

	if fk {
		// Temporarily disable foreign key constraints.
		if _, err = tx.Exec("PRAGMA foreign_keys = OFF"); err != nil {
			return fmt.Errorf("could not set foreign keys to off: %w", err)
		}
	}

	for _, table := range tables {
		// Delete the table contents.
		if _, err = tx.Exec("DELETE FROM " + table); err != nil {
			return fmt.Errorf("could not truncate table %s: %w", table, err)
		}

		// Restart the auto-increment counter if it exists.
		if _, err = tx.Exec("DELETE FROM sqlite_sequence WHERE name = ?", table); err != nil {
			return fmt.Errorf("could not restart auto-increment counter for table %s: %w", table, err)
		}
	}

	if fk {
		// Re-enable foreign key constraints.
		if _, err = tx.Exec("PRAGMA foreign_keys = ON"); err != nil {
			return fmt.Errorf("could not set foreign keys to on: %w", err)
		}
	}

	return tx.Commit()
}

const rowIDCountQuery = `SELECT COUNT(rowid) AS count FROM `

func (p *SQLiteProvider) Count(tx *sql.Tx, table string) (count int, err error) {
	query := rowIDCountQuery + table
	row := tx.QueryRow(query)
	err = row.Scan(&count)
	return count, err
}

const sqliteListTablesQuery = `SELECT name FROM sqlite_schema WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name`

func (p *SQLiteProvider) listTables(tx *sql.Tx) (tables []string, err error) {
	var rows *sql.Rows
	if rows, err = tx.Query(sqliteListTablesQuery); err != nil {
		return nil, err
	}
	defer rows.Close()

	tables = make([]string, 0)
	for rows.Next() {
		var table string
		if err = rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	return tables, rows.Err()
}

func (p *SQLiteProvider) foreignKeysEnabled(tx *sql.Tx) (enabled bool, err error) {
	row := tx.QueryRow("PRAGMA foreign_keys")
	err = row.Scan(&enabled)
	return enabled, err
}
