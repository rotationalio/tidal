package suite

import (
	"context"
	"database/sql"

	"go.rtnl.ai/x/dsn"
)

// A database provider is used to power a database test suite. The provider is meant to
// implement database-specific functionality for the test suite, e.g. different SQL
// syntaxes, different query patterns, etc.
type Provider interface {
	// Resolve DSN must return a DSN for the database that allows connections while
	// tests are running (possible in parallel with other test suites). It should
	// return an error if the DSN cannot be resolved or the database cannot be connected
	// to with information from the database string or the environment variables.
	ResolveDSN(databaseURL string) (uri *dsn.DSN, err error)

	// Managing the database lifecycle for tests.
	// CreateDB is called first to create the database using a management connection.
	// If CreateDB creates a new DSN connection, it should return the new DSN.
	CreateDB(ctx context.Context, uri *dsn.DSN) (*dsn.DSN, error)

	// Connect is called to connect to the actual test database.
	Connect(ctx context.Context, uri *dsn.DSN) (*sql.DB, error)

	// DropDB is called after Close() to drop the database using a management connection.
	DropDB(ctx context.Context, uri *dsn.DSN) error

	// Managing tables for tests. This is not used by the DatabaseSuite itself, but
	// the user can use it to override how the database is reset for tests.
	DropTables(ctx context.Context, conn *sql.DB) error
	TruncateTables(ctx context.Context, conn *sql.DB) error

	// Helper for counting the number of rows in a table, which users can use to make
	// assertions about the database state.
	Count(tx *sql.Tx, table string) (count int, err error)
}

// Migrations are used to ensure the database meets a set schema.
type Migrations interface {
	Apply(ctx context.Context, provider string, db *sql.DB, version string) error
}
