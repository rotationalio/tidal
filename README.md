# Tidal

[![CI Tests](https://github.com/rotationalio/tidal/actions/workflows/tests.yaml/badge.svg)](https://github.com/rotationalio/tidal/actions/workflows/tests.yaml)
[![Go Doc](https://pkg.go.dev/badge/go.rtnl.ai/tidal)](https://pkg.go.dev/go.rtnl.ai/tidal)
[![Go Report Card](https://goreportcard.com/badge/go.rtnl.ai/tidal)](https://goreportcard.com/report/go.rtnl.ai/tidal)

**SQL Database Store.**

Tidal provides internal mechanisms for managing SQL databases in Rotational applications. It provides a migrations mechanism for storing schema versions inside the database and automatically applying schema changes. It also provides a CRUD and Model interface for use with direct SQL statements rather than ORM functionality. Tidal is not meant to be generally used but implements the Rotational SQL pattern.

## Migrations

The `go.rtnl.ai/tidal/migrations` package manages your database schema by tracking which schema version the database is at and automatically applying any newer migrations on startup. Migrations are plain SQL files, embedded into your binary, and applied inside a transaction so that the schema is only advanced when every pending migration succeeds.

### Writing Migration Files

Each migration is a `.sql` file named `NNNN_name_of_migration.sql`, where `NNNN` is the sequence ID that determines the order in which migrations are applied. Zero-pad the ID (typically to 4 digits) so the lexical file order matches the sequence order. The name portion is converted to a human-readable title (e.g. `add_users_table` becomes `Add Users Table`).

```text
migrations/
  0001_initial_schema.sql
  0002_add_users_table.sql
  0003_add_posts_table.sql
```

Migration IDs must be greater than zero, unique, and monotonically increasing, and migration names must be unique. These rules are enforced by `Load` (see `Validate`).

### Loading Migrations

Embed the migration files into your package and load them into a `Migrations` slice. `Load` walks the file system, parses the IDs and names, sorts the migrations by ID, and validates them:

```go
package db

import (
 "embed"

 "go.rtnl.ai/tidal/migrations"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func Migrations() (migrations.Migrations, error) {
 return migrations.Load(migrationFS)
}
```

### Applying Migrations

Call `ApplyPostgres` or `ApplySQLite` (depending on your backend) when the database is first connected to. Both methods create the `migrations` bookkeeping table if it does not exist, look up the last applied migration ID, and apply only the migrations whose ID is greater than that. The `version` string you pass is recorded alongside each migration so you can tell which release applied a given schema change.

```go
ctx := context.Background()

m, err := migrations.Load(migrationFS)
if err != nil {
 return err
}

// Postgres: uses an advisory lock so only one instance applies migrations at a time.
if err := m.ApplyPostgres(ctx, db, "v1.4.0"); err != nil {
 return err
}

// SQLite: applies all pending migrations in a single write transaction.
if err := m.ApplySQLite(ctx, db, "v1.4.0"); err != nil {
 return err
}
```

Applying migrations is idempotent: if the database is already up to date, no migrations are applied. Because all pending migrations run inside one transaction, a failure rolls back the entire batch and leaves the schema unchanged.

### Inspecting Applied Migrations

Use `LastApplied` to read the most recently applied migration record (ID, name, version, and the time it was applied) from the `migrations` table:

```go
last, err := migrations.LastApplied(ctx, db)
if err != nil {
 return err
}

fmt.Printf("schema at migration %d (%s), applied %s with %s\n",
 last.ID, last.Name, last.Applied, last.Version)
```

## Fields

The `go.rtnl.ai/tidal/fields` package provides custom column types that implement the `database/sql` [`driver.Valuer`](https://pkg.go.dev/database/sql/driver#Valuer) and [`sql.Scanner`](https://pkg.go.dev/database/sql#Scanner) interfaces. This means they can be used directly as `Model` struct fields and passed to or scanned from the database without any extra conversion code.

| Field | Go type | Use for | Null handling |
| --- | --- | --- | --- |
| `JSONB` | `json.RawMessage` (`[]byte`) | Arbitrary JSON stored in a `JSONB` or `BYTEA` column | Empty/`null` JSON scans to a `nil` slice |
| `NullJSONB` | struct with `Valid bool` and `JSONB` | A nullable JSON column where you must distinguish SQL `NULL` from JSON `null`/`{}` | `Valid` is `false` when the column is `NULL` or the JSON is `null` |
| `StringArray` | `[]string` | A list of strings stored as a JSON array | Empty array scans to a `nil` slice |
| `NullStringArray` | struct with `Valid bool` and `StringArray` | A nullable list of strings | `Valid` is `false` when the column is `NULL` |

All four types marshal/scan their values as JSON, so the backing column should be a JSON-compatible type (`JSONB` or `BYTEA` in Postgres, `BLOB`/`TEXT` in SQLite).

### Defining a Model

A `Model` supplies its values via `Params` (for `INSERT`/`UPDATE`) and reads them back via `Scan` (for `SELECT`). Use the field types directly as struct fields:

```go
package models

import (
 "database/sql"

 "go.rtnl.ai/tidal"
 "go.rtnl.ai/tidal/fields"
)

type Document struct {
 tidal.BaseModel
 Metadata fields.JSONB           // NOT NULL JSON column
 Settings fields.NullJSONB       // nullable JSON column
 Tags     fields.StringArray     // NOT NULL array column
 Authors  fields.NullStringArray // nullable array column
}

// Ensure the model satisfies the tidal.Model interface.
var _ tidal.Model = (*Document)(nil)

func (d *Document) Fields(tidal.Operation) []string {
 return []string{"id", "metadata", "settings", "tags", "authors", "created", "modified"}
}

func (d *Document) Params(op tidal.Operation) []sql.NamedArg {
 return []sql.NamedArg{
  sql.Named("id", d.ID),
  sql.Named("metadata", d.Metadata),
  sql.Named("settings", d.Settings),
  sql.Named("tags", d.Tags),
  sql.Named("authors", d.Authors),
  sql.Named("created", d.Created),
  sql.Named("modified", d.Modified),
 }
}

func (d *Document) Scan(op tidal.Operation, s tidal.Scanner) error {
 return s.Scan(&d.ID, &d.Metadata, &d.Settings, &d.Tags, &d.Authors, &d.Created, &d.Modified)
}
```

Because the field types implement `driver.Valuer` and `sql.Scanner`, the values are passed to and read from the database by reference (in `Scan`) or by value (in `Params`) without additional conversion.

### JSONB

`JSONB` is a `json.RawMessage` that carries raw JSON bytes to and from the database. Use it for columns that are declared `NOT NULL`. A SQL `NULL` or a JSON `null` value scans into a `nil` slice.

Use `MarshalFrom` to populate the field from a Go value and `UnmarshalTo` to decode it into one:

```go
doc := &Document{}

// Encode a Go value into the JSONB field before saving.
if err := doc.Metadata.MarshalFrom(map[string]any{"version": 2, "draft": false}); err != nil {
 return err
}

// ... after loading the record from the database ...

// Decode the JSONB field back into a Go value.
var meta map[string]any
if err := doc.Metadata.UnmarshalTo(&meta); err != nil {
 return err
}
```

Helpers:

- `MarshalFrom(src any) error` — JSON-encodes `src` into the field; a `nil` source produces a `nil` field.
- `UnmarshalTo(dst any) error` — JSON-decodes the field into `dst`; a `nil`/empty field is a no-op.
- `IsNull() bool` — reports whether the field is empty or the literal JSON `null`.
- `Normalize() []byte` — returns canonical JSON bytes (object keys in sorted order), useful for hashing or equality checks.

### NullJSONB

`NullJSONB` wraps a `JSONB` with a `Valid` flag so you can distinguish a SQL `NULL` column from a present-but-empty value. Use it for nullable JSON columns. After scanning, check `Valid` before reading `JSONB`:

```go
doc := &Document{}

// Set a non-null value.
if err := doc.Settings.MarshalFrom(map[string]bool{"public": true}); err != nil {
 return err
}

// ... after loading the record ...

if doc.Settings.Valid {
 var settings map[string]bool
 if err := doc.Settings.UnmarshalTo(&settings); err != nil {
  return err
 }
}
```

`MarshalFrom` automatically sets `Valid` to `false` when the source is `nil` or encodes to JSON `null`, and `true` otherwise. To store an explicit SQL `NULL`, leave the zero value (`NullJSONB{}`) or set `Valid: false`.

### StringArray

`StringArray` is a `[]string` stored as a JSON array. Use it for `NOT NULL` columns. Assign and read it like an ordinary slice; an empty array or SQL `NULL` scans into a `nil` slice:

```go
doc := &Document{
 Tags: fields.StringArray{"go", "sql", "database"},
}

// ... after loading the record ...

for _, tag := range doc.Tags {
 fmt.Println(tag)
}
```

### NullStringArray

`NullStringArray` wraps a `StringArray` with a `Valid` flag for nullable array columns. Set `Valid: true` along with the values you want to store, and check `Valid` after scanning:

```go
doc := &Document{
 Authors: fields.NullStringArray{
  StringArray: fields.StringArray{"alice", "bob"},
  Valid:       true,
 },
}

// ... after loading the record ...

if doc.Authors.Valid {
 for _, author := range doc.Authors.StringArray {
  fmt.Println(author)
 }
}
```

A zero-value `NullStringArray{}` (or one with `Valid: false`) is written to the database as SQL `NULL`.

## Testing

Run the full suite from the repository root:

```bash
go test ./... -race

# Ignore go test cache and use verbose mode:
go test ./... -count=1 -race -v
```

SQLite tests need no setup. Each test suite creates its own database file in a temporary directory.

Postgres tests are skipped unless a database URL is set. Start a local Postgres instance (matching CI):

```bash
docker run -d --name tidal-postgres \
  -e POSTGRES_USER=rotational \
  -e POSTGRES_PASSWORD=theeaglefliesatdawn \
  -e POSTGRES_DB=tidal_test \
  -p 5432:5432 \
  postgres:18
```

Then run the Postgres suites:

```bash
export POSTGRES_DATABASE_URL="postgres://rotational:theeaglefliesatdawn@localhost:5432/tidal_test?sslmode=disable"
go test ./... -race -run Postgres
```

Stop the container when finished:

```bash
docker stop tidal-postgres && docker rm tidal-postgres
```

Database URLs are read from the environment in this order:

- Postgres: `POSTGRES_DATABASE_URL`, then `TEST_DATABASE_URL`, then `TIDAL_DATABASE_URL`, then `DATABASE_URL`
- SQLite: `SQLITE_DATABASE_URL`, then `TEST_DATABASE_URL`, then `TIDAL_DATABASE_URL`, then `DATABASE_URL`
