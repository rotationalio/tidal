package migrations

import (
	"bytes"
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"go.rtnl.ai/x/typecase"
)

var (
	ErrNoMigrations     = errors.New("no migrations found")
	ErrUnboundMigration = errors.New("migration has been instantiated incorrectly without an underlying fs.FS")
	ErrInvalidID        = errors.New("migration IDs must be greater than 0")
	ErrDuplicateID      = errors.New("duplicate migration IDs detected")
	ErrDuplicateName    = errors.New("duplicate migration names detected")
	ErrInvalidSequence  = errors.New("migration IDs must be monotonically increasing")
)

// Process migration file names
var pathre = regexp.MustCompile(`^(\d+)_(\w+)\.sql$`)

// Load Migrations from the file system. Migrations contain the SQL commands from SQL
// files named like NNNN_name_of_migration.sql. The NNNN is the sequence ID of the
// migration and is used to determine the order in which to apply the migrations.
// Generally NNNN is zero padded to 4 digits to ensure the lexical order of the
// migrations matches the sequence order.
//
// Generally speaking you'll embed the migrations into your package as follows:
//
// //go:embed migrations/*.sql
// var migrationFS embed.FS
//
// Then you can load the migrations into a Migrations slice like this:
//
// migrations := migrations.Load(migrationFS)
//
// Migrations are used in the initialization of the database schema.
func Load(files fs.FS) (migrations Migrations, err error) {
	migrations = make(Migrations, 0)
	fs.WalkDir(files, ".", func(path string, entry fs.DirEntry, err error) error {
		// Stop walking if there is an error
		if err != nil {
			return err
		}

		// Check for hidden directories (names starting with a dot excluding ".")
		if entry.IsDir() && entry.Name() != "." && strings.HasPrefix(entry.Name(), ".") {
			return fs.SkipDir
		}

		// Skip directories
		if entry.IsDir() {
			return nil
		}

		// Evaluate the regex against the file name
		if !pathre.MatchString(entry.Name()) {
			return nil
		}

		// Path matches, create the new migration
		groups := pathre.FindStringSubmatch(entry.Name())
		if len(groups) != 3 {
			return fmt.Errorf("invalid migration file name %q", entry.Name())
		}

		migration := &Migration{
			Path: path,
			fs:   files,
		}

		if migration.ID, err = strconv.Atoi(groups[1]); err != nil {
			return fmt.Errorf("invalid migration file name %q: %w", entry.Name(), err)
		}

		migration.Name = typecase.Title(strings.Join(strings.Split(groups[2], "_"), " "))
		migrations = append(migrations, migration)
		return nil
	})

	// Sort the migrations by ID
	migrations.Sort()

	// Validate the migrations
	if err = migrations.Validate(); err != nil {
		return nil, err
	}

	return migrations, nil
}

type Migrations []*Migration

// Ensure the migrations are valid and can be applied to the database.
// This method checks to ensure that all IDs are > 0 and that the IDs are unique.
func (m Migrations) Validate() error {
	if len(m) == 0 {
		return ErrNoMigrations
	}

	prev := 0
	seenNames := make(map[string]struct{})

	for _, migration := range m {
		if migration.ID <= 0 {
			return ErrInvalidID
		}

		if migration.ID <= prev {
			return ErrInvalidSequence
		}
		prev = migration.ID

		if _, ok := seenNames[migration.Name]; ok {
			return ErrDuplicateName
		}

		seenNames[migration.Name] = struct{}{}
	}
	return nil
}

func (m Migrations) Sort() {
	slices.SortFunc(m, func(a, b *Migration) int {
		return cmp.Compare(a.ID, b.ID)
	})
}

const lastAppliedSQL = `
SELECT id, name, version, applied FROM migrations
	ORDER BY id DESC LIMIT 1;
`

// Retrieve the last applied migration info from the migrations table.
func LastApplied(ctx context.Context, db *sql.DB) (migration *Migration, err error) {
	var tx *sql.Tx
	if tx, err = db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	migration = &Migration{}
	row := tx.QueryRow(lastAppliedSQL)

	if err = row.Scan(&migration.ID, &migration.Name, &migration.Version, &migration.Applied); err != nil {
		return nil, fmt.Errorf("could not retrieve last applied migration: %w", err)
	}
	return migration, nil
}

// Migration is used to represent both a SQL migration from the embedded file system and
// a migration record in the database. These records are compared to ensure the database
// is as up to date as possible before the application starts.
type Migration struct {
	ID      int       // The unique sequence ID of the migration
	Name    string    // The human readable name of the migration
	Version string    // The package version when the migration was applied
	Applied time.Time // The timestamp when the migration was applied
	Path    string    // The path to the migration file in the filesystem

	fs fs.FS // The file system to read the migration file from
}

// SQL loads the sql query from the embedded file system.
func (m *Migration) SQL() (_ string, err error) {
	if m.fs == nil {
		return "", ErrUnboundMigration
	}

	if m.Path == "" {
		return "", fmt.Errorf("cannot read sql for migration %d", m.ID)
	}

	var f fs.File
	if f, err = m.fs.Open(m.Path); err != nil {
		return "", fmt.Errorf("cannot open migration file %q: %w", m.Path, err)
	}
	defer f.Close()

	var data []byte
	if data, err = io.ReadAll(f); err != nil {
		return "", fmt.Errorf("cannot read sql for migration %d: %w", m.ID, err)
	}

	return string(bytes.TrimSpace(data)), nil
}
