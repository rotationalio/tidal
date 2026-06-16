package suite

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.rtnl.ai/tidal"
	"go.rtnl.ai/x/dsn"
)

//============================================================================
// Suite Environment
//============================================================================

const (
	DATABASE_URL          = "DATABASE_URL"
	TEST_DATABASE_URL     = "TEST_DATABASE_URL"
	TIDAL_DATABASE_URL    = "TIDAL_DATABASE_URL"
	SQLITE_DATABASE_URL   = "SQLITE_DATABASE_URL"
	POSTGRES_DATABASE_URL = "POSTGRES_DATABASE_URL"
)

//============================================================================
// Suite Errors
//============================================================================

var (
	ErrInvalidProvider  = errors.New("invalid database provider for this suite")
	ErrSqliteRequired   = errors.New("valid sqlite3 database url is required for this suite")
	ErrPostgresRequired = errors.New("valid postgres database url is required for this suite")
	ErrNoDatabaseURL    = errors.New("could not resolve or load database URL from the environment")
)

//============================================================================
// Suite Defaults and Configuration
//============================================================================

const (
	DefaultTimeout = 20 * time.Second
)

type TeardownStrategy int

const (
	// TeardownTruncate clears table rows and preserves schema/migrations.
	TeardownTruncate TeardownStrategy = iota
	// TeardownDropAndMigrate resets by dropping tables and re-running migrations.
	TeardownDropAndMigrate
	// TeardownNone only cancels the test context.
	TeardownNone
)

//============================================================================
// Suite Entry Point
//============================================================================

// Run takes a testing suite and runs all of the tests attached to it.
func Run(t *testing.T, s suite.TestingSuite) {
	suite.Run(t, s)
}

type DatabaseSuite struct {
	suite.Suite
	Provider
	*tidal.DB // use SQLDB() to access the underlying sql.DB

	DatabaseURL string
	Migrations  Migrations
	Timeout     time.Duration
	Teardown    TeardownStrategy

	mu         sync.RWMutex // protects dsn and contexts
	dsn        *dsn.DSN
	ctx        context.Context    // Suite-wide context
	cancel     context.CancelFunc // Cancel func for suite-wide context
	testCtx    context.Context    // Per-test context
	testCancel context.CancelFunc // Cancel func for per-test context
	subCancel  context.CancelFunc // Cancel func for sub-contexts within tests
}

//============================================================================
// Suite Lifecycle
//============================================================================

// SetupSuite resolves the configured database URL or fetches it from the environment,
// connects to the database, creates the database, and applies the migrations.
func (s *DatabaseSuite) SetupSuite() {
	require := s.Require()
	require.NotNil(s.Provider, "cannot setup database suite without a provider")

	// Step 0: Initialization
	if s.Timeout == 0 {
		s.Timeout = DefaultTimeout
	}

	// Step one: resolve the DSN either from the setting on the suite or from the
	// environment. If this errors, the test suite will fail immediately.
	var err error
	if s.dsn, err = s.ResolveDSN(s.DatabaseURL); err != nil {
		s.T().Fatalf("failed to resolve database URL: %s", err.Error())
		return
	}
	ctx, cancel := s.context()
	defer cancel()

	// Step two: create the database.
	s.T().Log("creating database")
	if s.dsn, err = s.CreateDB(ctx, s.dsn); err != nil {
		s.T().Fatalf("failed to create database: %s", err.Error())
		return
	}

	// Step three: connect to the database.
	s.T().Logf("connecting to database: %s", s.dsn.String())
	if s.DB, err = tidal.Open(ctx, s.dsn); err != nil {
		s.T().Fatalf("failed to connect to database: %s", err.Error())
		return
	}

	// Step four: apply the migrations if there are any.
	if s.Migrations != nil {
		s.T().Log("applying migrations")
		if err = s.Migrations.Apply(ctx, s.DB, "test"); err != nil {
			s.T().Fatalf("failed to apply migrations: %s", err.Error())
			return
		}
	}
}

// TearDownSuite drops the database, closes the connection, and cleans up the context.
func (s *DatabaseSuite) TearDownSuite() {
	if s.DB != nil {
		s.T().Log("closing database connection")
		if err := s.DB.Close(); err != nil {
			s.T().Logf("failed to close database connection: %s", err.Error())
			return
		}

		// Drop the database
		s.T().Log("dropping database")
		ctx, cancel := s.context()
		defer cancel()
		if err := s.DropDB(ctx, s.dsn); err != nil {
			s.T().Logf("failed to drop database: %s", err.Error())
			return
		}

		// Clean up the database connection
		s.DB = nil
		s.dsn = nil

		if s.cancel != nil {
			s.cancel()
			s.cancel = nil
			s.ctx = nil
		}
	} else {
		s.T().Log("cannot tear down the database test suite because s.DB is nil")
	}
}

//============================================================================
// Test Context Lifecycle
//============================================================================

// Creates a new per-test context. Takes a write lock on mu.
func (s *DatabaseSuite) SetupTest() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Parent context for the whole test; subtests derive child contexts.
	s.testCtx, s.testCancel = s.context()
	s.subCancel = nil
	s.ctx, s.cancel = s.testCtx, s.testCancel
}

// Resets the database and cancels the context. Takes a write lock on mu.
func (s *DatabaseSuite) TearDownTest() {
	if !s.ReadOnly() {
		s.tearDownData()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Cancel child first, then parent.
	if s.subCancel != nil {
		s.subCancel()
		s.subCancel = nil
	}
	if s.testCancel != nil {
		s.testCancel()
	}
	// Clear references so stale contexts are never reused.
	s.ctx, s.cancel = nil, nil
	s.testCtx, s.testCancel, s.subCancel = nil, nil, nil
}

// Creates a new per-subtest context. Takes a write lock on mu.
func (s *DatabaseSuite) SetupSubTest() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Defensive fallback if SetupTest was not run.
	if s.testCtx == nil {
		s.testCtx, s.testCancel = s.context()
	}
	if s.subCancel != nil {
		s.subCancel()
	}
	// Subtests get a child context so cancellation/timeouts stay local.
	s.ctx, s.subCancel = context.WithTimeout(s.testCtx, s.Timeout)
	s.cancel = s.subCancel
}

// Cancels the per-subtest context. Takes a write lock on mu.
func (s *DatabaseSuite) TearDownSubTest() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// End child context, then restore parent for between-subtest code.
	if s.subCancel != nil {
		s.subCancel()
		s.subCancel = nil
	}
	s.ctx, s.cancel = s.testCtx, s.testCancel
}

//============================================================================
// Context and Transaction Utilities
//============================================================================

// Returns an uneditable copy of the DSN currently being used by the suite.
// Takes a read lock on mu.
func (s *DatabaseSuite) DSN() dsn.DSN {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return *s.dsn
}

// Checks the DSN to see if the database is read only. Takes a read lock on mu.
func (s *DatabaseSuite) ReadOnly() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.dsn.Options.ReadOnly()
}

// Returns a context with a timeout. Takes a read lock on mu.
func (s *DatabaseSuite) Context() context.Context {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.ctx
}

// Returns a context with the suite timeout. Does not acquire mu.
func (s *DatabaseSuite) context() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), s.Timeout)
}

// Starts a new transaction on the database with the current context. Callers
// must ensure that the transaction is rolled back or committed. Takes a read
// lock on mu.
func (s *DatabaseSuite) BeginTx(opts *sql.TxOptions) tidal.Tx {
	s.mu.RLock()
	defer s.mu.RUnlock()

	require := s.Require()
	require.NotNil(s.DB, "cannot begin a transaction because s.DB is nil")
	require.NotNil(s.ctx, "cannot begin a transaction because s.Context is nil")

	tx, err := s.DB.BeginTx(s.ctx, opts)
	require.NoError(err, "could not begin transaction")

	// Safety net for leaked test transactions.
	s.T().Cleanup(func() {
		_ = tx.Rollback()
	})

	return tx
}

func (s *DatabaseSuite) tearDownData() {
	switch s.Teardown {
	case TeardownDropAndMigrate:
		s.ResetDB()
	case TeardownNone:
		// Skip data teardown.
	default:
		s.TruncateTables()
	}
}

//============================================================================
// Database Utilities
//============================================================================

// Applies the migrations to the database. Does not acquire mu.
func (s *DatabaseSuite) Migrate() {
	if s.Migrations != nil && s.DB != nil {
		ctx, cancel := s.context()
		defer cancel()

		s.Require().NoError(s.Migrations.Apply(ctx, s.DB, "test"), "failed to apply migrations")
	} else {
		// NOTE: this is not necessarily an error; for example a test suite may not
		// use these fields for every test.
		s.T().Log("cannot migrate the database because s.Migrations is nil or s.DB is nil")
	}
}

// ResetDB calls DropTables and then Migrate to reset the database back to its
// initial state. Does not acquire mu.
func (s *DatabaseSuite) ResetDB() {
	if s.DB != nil {
		s.DropTables()
		s.Migrate()
	} else {
		s.T().Log("cannot reset the database because s.DB is nil")
	}
}

// DropTables drops all tables from the database. Does not acquire mu.
func (s *DatabaseSuite) DropTables() {
	if s.DB != nil {
		ctx, cancel := s.context()
		defer cancel()

		s.Require().NoError(s.Provider.DropTables(ctx, s.DB), "failed to drop tables")
	} else {
		s.T().Log("cannot drop tables because s.DB is nil")
	}
}

// TruncateTables truncates all tables in the database. Does not acquire mu.
func (s *DatabaseSuite) TruncateTables() {
	if s.DB != nil {
		ctx, cancel := s.context()
		defer cancel()

		s.Require().NoError(s.Provider.TruncateTables(ctx, s.DB), "failed to truncate tables")
	} else {
		s.T().Log("cannot truncate tables because s.DB is nil")
	}
}
