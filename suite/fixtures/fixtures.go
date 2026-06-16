// Package fixtures loads SQL files from disk and applies them in database test suites.
//
// Use [File] to reference SQL under suite/testdata, or construct a [Fixture] from any
// path. Fixtures implement the [suite.Migrations] interface.
//
// Example:
//
//	s.Migrations = fixtures.File("fields/sqlite_schema.sql")
package fixtures

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"go.rtnl.ai/tidal/conn"
)

// Fixtures are similar to migrations, they load a SQL query from a file on disk and
// can be used to execute the SQL query against the database. The fixture has the same
// Apply method interface as migrations to be used in test suites. Fixtures are much
// simpler than migrations and are intended to be used for testing purposes.
//
// The fixture is simply a string to a path on disk, e.g. "testdata/fixtures/users.sql"
type Fixture string

// File returns a fixture for a SQL file under suite/testdata.
func File(name string) Fixture {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("fixtures: could not resolve testdata directory")
	}
	return Fixture(filepath.Join(filepath.Dir(file), "..", "testdata", name))
}

func (path Fixture) SQL() (_ string, err error) {
	var f *os.File
	if f, err = os.Open(string(path)); err != nil {
		return "", errors.Join(err, fmt.Errorf("could not open fixture file %q", path), err)
	}
	defer f.Close()

	var data []byte
	if data, err = io.ReadAll(f); err != nil {
		return "", errors.Join(err, fmt.Errorf("could not read fixture file %q", path), err)
	}
	return string(data), nil
}

// Implements the suite.Migrations interface so that a fixture can be used as a migration.
func (path Fixture) Apply(ctx context.Context, db conn.Beginner, _ string) (err error) {
	var query string
	if query, err = path.SQL(); err != nil {
		return err
	}

	var tx *sql.Tx
	if tx, err = db.SQLDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: false, Isolation: sql.LevelSerializable}); err != nil {
		return errors.Join(err, fmt.Errorf("could not begin transaction to apply fixture %q", path), err)
	}
	defer tx.Rollback()

	if _, err = tx.Exec(query); err != nil {
		return errors.Join(err, fmt.Errorf("could not apply fixture %q", path), err)
	}

	return tx.Commit()
}

// Fixtures is an ordered list of fixtures to apply to the database all at once.
type Fixtures []Fixture

func Glob(pattern string) (fixtures Fixtures, err error) {
	var matches []string
	if matches, err = filepath.Glob(pattern); err != nil {
		return nil, errors.Join(err, fmt.Errorf("could not glob fixtures %q", pattern), err)
	}

	fixtures = make(Fixtures, 0, len(matches))
	for _, match := range matches {
		fixtures = append(fixtures, Fixture(match))
	}

	return fixtures, nil
}

func (fs Fixtures) Apply(ctx context.Context, db conn.Beginner, version string) (err error) {
	var tx *sql.Tx
	if tx, err = db.SQLDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: false, Isolation: sql.LevelSerializable}); err != nil {
		return errors.Join(err, fmt.Errorf("could not begin transaction to apply fixtures"), err)
	}
	defer tx.Rollback()

	for _, fixture := range fs {
		var query string
		if query, err = fixture.SQL(); err != nil {
			return err
		}

		if _, err = tx.Exec(query); err != nil {
			return errors.Join(err, fmt.Errorf("could not apply fixture %q", fixture), err)
		}
	}

	return tx.Commit()
}
