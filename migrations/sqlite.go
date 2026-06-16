package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"go.rtnl.ai/tidal/conn"
)

const sqliteMigrationTable = `
-- These migrations target an embedded sqlite3 database. The migrations table allows a
-- booting node to determine which version its schema is at so that it can quickly make
-- changes to its data store when the node starts or during runtime.
--
-- The migrations table stores the migrations applied to arrive at the current schema
-- of the database. The application checks this table for the version the db is at and
-- applies any later migrations as needed.
CREATE TABLE IF NOT EXISTS migrations (
    id      INTEGER PRIMARY KEY,
    name    TEXT NOT NULL,
    version TEXT NOT NULL,
    applied DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

const (
	sq3LastAppliedSQL     = `SELECT id FROM migrations ORDER BY id DESC LIMIT 1;`
	sq3InsertMigrationSQL = `INSERT INTO migrations (id, name, version) VALUES (:id, :name, :version);`
)

// Applies any unapplied migrations to the SQLite database and should be run when
// the database is first connected to. This method checks that the migrations table
// exists and if not, it creates the table. Because SQLite is a single file database
// a write lock is used to ensure that only one goroutine can apply migrations.
func (m Migrations) ApplySQLite(ctx context.Context, db conn.Beginner, version string) (err error) {
	// Create the migrations table if it does not exist.
	if _, err = db.SQLDB().ExecContext(ctx, sqliteMigrationTable); err != nil {
		return fmt.Errorf("could not create migrations table: %w", err)
	}

	// Start a transaction to apply the migrations.
	var tx *sql.Tx
	if tx, err = db.SQLDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: false}); err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the last applied migration.
	var lastApplied int
	if lastApplied, err = sq3LastApplied(tx); err != nil {
		return err
	}

	// Apply any unapplied migrations.
	for _, migration := range m {
		if migration.ID > lastApplied {
			var query string
			if query, err = migration.SQL(); err != nil {
				return err
			}

			if _, err = tx.Exec(query); err != nil {
				return fmt.Errorf("could not apply migration %d: %w", migration.ID, err)
			}

			params := []any{
				sql.Named("id", migration.ID),
				sql.Named("name", migration.Name),
				sql.Named("version", version),
			}

			if _, err = tx.Exec(sq3InsertMigrationSQL, params...); err != nil {
				return fmt.Errorf("could not insert migration record for %d: %w", migration.ID, err)
			}
		}
	}

	// Commit the transaction.
	return tx.Commit()
}

func sq3LastApplied(tx *sql.Tx) (lastApplied int, err error) {
	if row := tx.QueryRow(sq3LastAppliedSQL); row != nil {
		if err = row.Scan(&lastApplied); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, nil
			}
			return 0, fmt.Errorf("could not retrieve last applied migration: %w", err)
		}
	}
	return lastApplied, nil
}
