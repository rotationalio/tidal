package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const pgMigrationTable = `
-- These migrations target an PostgreSQL database. The migrations table allows a
-- booting node to determine which version its schema is at so that it can quickly make
-- changes to its data store when the node starts or during runtime.
--
-- The migrations table stores the migrations applied to arrive at the current schema
-- of the database. The application checks this table for the version the db is at and
-- applies any later migrations as needed.
CREATE TABLE IF NOT EXISTS migrations (
    id      INTEGER PRIMARY KEY,
    name    VARCHAR(255) NOT NULL,
    version VARCHAR(32) NOT NULL,
    applied TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
`

const (
	pgLastAppliedSQL        = `SELECT id FROM migrations ORDER BY id DESC LIMIT 1;`
	pgInsertMigrationSQL    = `INSERT INTO migrations (id, name, version) VALUES ($1, $2, $3);`
	acquireMigrationLockSQL = `SELECT pg_advisory_lock($1);`
	releaseMigrationLockSQL = `SELECT pg_advisory_unlock($1);`

	AdvisoryLockID = int64(4006367007158143198)
)

// Applies any unapplied migrations to the PostgreSQL database and should be run when
// the database is first connected to. This method checks that the migrations table
// exists and if not, it creates the table. A PostgreSQL advisory lock is used to ensure
// that only one instance of the application can apply migrations at a time.
func (m Migrations) ApplyPostgres(ctx context.Context, db *sql.DB, version string) (err error) {
	// Acquire a single connection so that we can acquire the advisory lock.
	var conn *sql.Conn
	if conn, err = db.Conn(ctx); err != nil {
		return fmt.Errorf("could not acquire connection: %w", err)
	}
	defer conn.Close()

	// Acquire the advisory lock.
	if _, err = conn.ExecContext(ctx, acquireMigrationLockSQL, AdvisoryLockID); err != nil {
		return fmt.Errorf("could not acquire advisory lock: %w", err)
	}

	// Ensure the advisory lock is released.
	defer func() {
		if _, cerr := conn.ExecContext(ctx, releaseMigrationLockSQL, AdvisoryLockID); cerr != nil {
			err = errors.Join(err, cerr)
		}
	}()

	// Create the migrations table if it does not exist.
	if _, err = conn.ExecContext(ctx, pgMigrationTable); err != nil {
		return fmt.Errorf("could not create migrations table: %w", err)
	}

	// Start a transaction to apply the migrations.
	var tx *sql.Tx
	if tx, err = conn.BeginTx(ctx, &sql.TxOptions{ReadOnly: false}); err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the last applied migration.
	var lastApplied int
	if lastApplied, err = pgLastApplied(tx); err != nil {
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

			if _, err = tx.Exec(pgInsertMigrationSQL, migration.ID, migration.Name, version); err != nil {
				return fmt.Errorf("could not insert migration record for %d: %w", migration.ID, err)
			}
		}
	}

	// Commit the transaction.
	return tx.Commit()
}

func pgLastApplied(tx *sql.Tx) (lastApplied int, err error) {
	if row := tx.QueryRow(pgLastAppliedSQL); row != nil {
		if err = row.Scan(&lastApplied); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return 0, nil
			}
			return 0, fmt.Errorf("could not retrieve last applied migration: %w", err)
		}
	}
	return lastApplied, nil
}
