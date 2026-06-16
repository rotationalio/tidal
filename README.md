# Tidal

[![CI Tests](https://github.com/rotationalio/tidal/actions/workflows/tests.yaml/badge.svg)](https://github.com/rotationalio/tidal/actions/workflows/tests.yaml)
[![Go Doc](https://pkg.go.dev/badge/go.rtnl.ai/tidal)](https://pkg.go.dev/go.rtnl.ai/tidal)
[![Go Report Card](https://goreportcard.com/badge/go.rtnl.ai/tidal)](https://goreportcard.com/report/go.rtnl.ai/tidal)

**SQL Database Store.**

Tidal provides internal mechanisms for managing SQL databases in Rotational applications. It provides a migrations mechanism for storing schema versions inside the database and automatically applying schema changes. It also provides a CRUD and Model interface for use with direct SQL statements rather than ORM functionality. Tidal is not meant to be generally used but implements the Rotational SQL pattern.

Import [`go.rtnl.ai/tidal`](https://pkg.go.dev/go.rtnl.ai/tidal) in application code. The root package re-exports the API (`tidal.Open`, `tidal.New`, `tidal.CRUD`, and so on). Subpackages are public when you need a narrower import.

## Packages

| Import | Purpose |
| --- | --- |
| [`go.rtnl.ai/tidal`](https://pkg.go.dev/go.rtnl.ai/tidal) | Main entry point — re-exports connections, models, filters, and CRUD |
| [`go.rtnl.ai/tidal/conn`](https://pkg.go.dev/go.rtnl.ai/tidal/conn) | `DB`, `Tx`, `Open`, `Wrap`, [`Beginner`](https://pkg.go.dev/go.rtnl.ai/tidal/conn#Beginner) |
| [`go.rtnl.ai/tidal/model`](https://pkg.go.dev/go.rtnl.ai/tidal/model) | `Model`, `BaseModel`, `Operation` |
| [`go.rtnl.ai/tidal/filter`](https://pkg.go.dev/go.rtnl.ai/tidal/filter) | `Filter`, `Clause`, list-query pagination |
| [`go.rtnl.ai/tidal/store`](https://pkg.go.dev/go.rtnl.ai/tidal/store) | `CRUD`, `Cursor`, query generation |
| [`go.rtnl.ai/tidal/bind`](https://pkg.go.dev/go.rtnl.ai/tidal/bind) | `:name` placeholder rewriting |
| [`go.rtnl.ai/tidal/migrations`](https://pkg.go.dev/go.rtnl.ai/tidal/migrations) | Versioned SQL schema migrations |
| [`go.rtnl.ai/tidal/fields`](https://pkg.go.dev/go.rtnl.ai/tidal/fields) | JSON and string-array column types for `Model` structs |
| [`go.rtnl.ai/tidal/suite`](https://pkg.go.dev/go.rtnl.ai/tidal/suite) | Database test harness, `ConformsCRUD`, shared `testdata`, and integration tests |
| [`go.rtnl.ai/tidal/suite/fixtures`](https://pkg.go.dev/go.rtnl.ai/tidal/suite/fixtures) | SQL fixture loader for test suites (`fixtures.File`) |

## Connecting

Use [`tidal.Open`](https://pkg.go.dev/go.rtnl.ai/tidal#Open) to connect to a supported database. Pass a [`*dsn.DSN`](https://pkg.go.dev/go.rtnl.ai/x/dsn#DSN) from `go.rtnl.ai/x/dsn` (typically parsed from `DATABASE_URL`):

```go
package db

import (
 "context"
 "os"

 "go.rtnl.ai/tidal"
 "go.rtnl.ai/x/dsn"
)

func Connect(ctx context.Context) (*tidal.DB, error) {
 uri, err := dsn.Parse(os.Getenv("DATABASE_URL"))
 if err != nil {
  return nil, err
 }
 return tidal.Open(ctx, uri)
}
```

`Open` registers the correct SQL driver, applies custom per-db settings from the DSN parameters, and pings the database before returning. It currently supports SQLite3 and Postgres.

The returned [`*tidal.DB`](https://pkg.go.dev/go.rtnl.ai/tidal#DB) embeds `*sql.DB`, so the usual `Close`, `Ping`, and `ExecContext` methods are available directly. The provider is stored on the connection so transactions bind placeholders automatically.

Start a transaction with [`DB.BeginTx`](https://pkg.go.dev/go.rtnl.ai/tidal#DB.BeginTx). It returns a [`tidal.Tx`](https://pkg.go.dev/go.rtnl.ai/tidal#Tx) that accepts canonical `:name` SQL and `sql.NamedArg` arguments regardless of backend and rewrites them for the backend (ex: Postgres placeholders are rewritten to `$1`, `$2`, etc.).

```go
import "database/sql"

db, err := tidal.Open(ctx, uri)
if err != nil {
 return err
}
defer db.Close()

tx, err := db.BeginTx(ctx, nil)
if err != nil {
 return err
}
defer tx.Rollback()

_, err = tx.Exec(
 "INSERT INTO users (id, email) VALUES (:id, :email)",
 sql.Named("id", id),
 sql.Named("email", email),
)
```

Pass `tidal.Tx` to [`CRUD`](https://pkg.go.dev/go.rtnl.ai/tidal#CRUD) methods and [`Rows`](https://pkg.go.dev/go.rtnl.ai/tidal#Rows) cursors.

When you need a raw `*sql.DB` — for migrations, third-party libraries, or admin DDL — use the embedded connection (`db.DB`) or [`DB.SQLDB`](https://pkg.go.dev/go.rtnl.ai/tidal#DB.SQLDB).

If you already have an open `*sql.DB`, wrap it with [`tidal.Wrap`](https://pkg.go.dev/go.rtnl.ai/tidal#Wrap). You still need a parsed `*dsn.DSN` so tidal knows which placeholder style to use:

```go
uri, _ := dsn.Parse(os.Getenv("DATABASE_URL"))
sqlDB, _ := sql.Open("sqlite3", uri.Path)
db := tidal.Wrap(sqlDB, uri)
```

## CRUD

Implement [`Model`](https://pkg.go.dev/go.rtnl.ai/tidal#Model) on your struct and embed [`BaseModel`](https://pkg.go.dev/go.rtnl.ai/tidal#BaseModel) for ULID ids and timestamps. Build a typed store with [`New`](https://pkg.go.dev/go.rtnl.ai/tidal#New) and run operations inside a transaction:

```go
crud := tidal.New[*User]("users")

tx, err := db.BeginTx(ctx, nil)
if err != nil {
 return err
}
defer tx.Rollback()

user := &User{Name: "Ada"}
_, err = crud.Create(tx, user)

loaded, err := crud.Retrieve(tx, sql.Named("id", user.ID))
err = crud.Update(tx, user)
_, err = crud.Delete(tx, sql.Named("id", user.ID))

cursor, err := crud.List(tx, (&tidal.Filter{}).OrderBy("name").Limit(10))
users, err := cursor.List()
```

Use [`Clause`](https://pkg.go.dev/go.rtnl.ai/tidal#Clause) for `WHERE` conditions. [`Filter`](https://pkg.go.dev/go.rtnl.ai/tidal#Filter) adds `ORDER BY`, `LIMIT`, and `OFFSET` — combine both when you need filtering and pagination:

```go
f := (&tidal.Filter{}).OrderBy("-created").Limit(20)
filter := &tidal.Clause{
 SQL:  "WHERE status = :status " + f.Clause(),
 Args: []sql.NamedArg{sql.Named("status", "active")},
}
cursor, err := crud.List(tx, filter)
```

[`Cursor.Close`](https://pkg.go.dev/go.rtnl.ai/tidal#Cursor.Close) rolls back the transaction. Use [`Cursor.CloseRows`](https://pkg.go.dev/go.rtnl.ai/tidal#Cursor.CloseRows) when you want to keep the transaction open for more queries.

See [Fields](#fields) for JSON and array column types.

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

Migration IDs must be greater than zero, strictly increasing, and migration names must be unique. `Load` enforces these rules via `Validate`.

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

Call `Apply` (or `ApplyPostgres` / `ApplySQLite` directly) after connecting with `tidal.Open`. These methods create the `migrations` bookkeeping table if it does not exist, look up the last applied migration ID, and apply only migrations with a higher ID. The `version` string you pass is recorded alongside each migration so you can tell which release applied a given schema change.

`Apply` and `LastApplied` accept any value that implements [`tidal.Beginner`](https://pkg.go.dev/go.rtnl.ai/tidal#Beginner) — typically your `*tidal.DB` from `tidal.Open`:

```go
ctx := context.Background()

db, err := tidal.Open(ctx, uri)
if err != nil {
 return err
}
defer db.Close()

m, err := migrations.Load(migrationFS)
if err != nil {
 return err
}

if err := m.Apply(ctx, db, "v1.4.0"); err != nil {
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

## Model conformance (`ConformsCRUD`)

The `go.rtnl.ai/tidal/suite` package includes [`ConformsCRUD`](https://pkg.go.dev/go.rtnl.ai/tidal/suite#ConformsCRUD), a helper that checks a [`Model`](https://pkg.go.dev/go.rtnl.ai/tidal#Model) implementation against [`tidal.CRUD`](https://pkg.go.dev/go.rtnl.ai/tidal#CRUD). Use it in tests to catch `Fields`, `Params`, and `Scan` mistakes before they show up in production.

It runs three subtests:

1. **Shape** — `Fields` and `Params` line up for each operation, and `tidal.New` produced non-empty SQL. No database access.
2. **Scan** — builds fake row values from `Params`, feeds them through `Scan`, and compares the result to your factory output. No database access.
3. **RoundTrip** — runs create, retrieve, list, update, and delete against the real database inside a transaction that is always rolled back.

Wire it into a [`DatabaseSuite`](https://pkg.go.dev/go.rtnl.ai/tidal/suite#DatabaseSuite) test (the suite connects, applies migrations, and tears down the database for you):

```go
package myapp_test

import (
 "embed"
 "testing"

 "github.com/stretchr/testify/require"
 "go.rtnl.ai/tidal/migrations"
 "go.rtnl.ai/tidal/suite"
)

//go:embed testdata/migrations
var migrationFS embed.FS

type ModelTestSuite struct {
 suite.DatabaseSuite
}

func TestModels(t *testing.T) {
 m, err := migrations.Load(migrationFS)
 require.NoError(t, err)

 s := &ModelTestSuite{}
 s.Provider = &suite.SQLiteProvider{} // or &suite.PostgresProvider{}
 s.Migrations = m

 suite.Run(t, s)
}

func (s *ModelTestSuite) TestUserCRUDConformance() {
 suite.ConformsCRUD(&s.DatabaseSuite, suite.CRUDConformance[*User]{
  Table:  "users",
  Create: newTestUser, // return a fresh row ready to insert
  Update: func(u *User) {
   u.Name = "Updated Name" // mutate the inserted row for the update check
  },
  Equal: nil // optional model comparison function; if nil uses a good default
 })
}
```

`Create` should return a valid insert each time — generate unique values (email, slug, etc.) inside the factory. `Update` receives the same instance that was created and inserted.

Provide `Equal` when the default comparison is not enough (for example JSON fields that need normalization). Otherwise the suite compares list results on the `Fields(List)` column subset and tolerates database timestamp truncation (compares times to the second).

For simple schema setup in tests, use [`suite/fixtures`](https://pkg.go.dev/go.rtnl.ai/tidal/suite/fixtures) instead of full migrations:

```go
import "go.rtnl.ai/tidal/suite/fixtures"

s.Migrations = fixtures.File("fields/sqlite_schema.sql")
```

SQL files live under `suite/testdata/` in this repository.

## Testing

Run the full suite from the repository root:

```bash
go test ./... -race

# Ignore go test cache and use verbose mode:
go test ./... -count=1 -race -v
```

SQLite tests need no setup. Each test suite creates its own database file in a temporary directory.

```bash
go test ./... -race -run SQLite
```

Postgres tests are skipped unless a database URL is set. Start a local Postgres instance (matching CI):

```bash
docker run -d --name tidal-postgres -e POSTGRES_USER=rotational -e POSTGRES_PASSWORD=theeaglefliesatdawn -e POSTGRES_DB=postgres -p 5432:5432 postgres:18
```

Then run the Postgres suites:

```bash
export POSTGRES_DATABASE_URL="postgres://rotational:theeaglefliesatdawn@localhost:5432/postgres?sslmode=disable"
go test ./... -race -run Postgres
```

Stop the container when finished:

```bash
docker stop tidal-postgres && docker rm tidal-postgres
```

Database URLs are read from the environment in this order:

- Postgres: `POSTGRES_DATABASE_URL`, then `TEST_DATABASE_URL`, then `TIDAL_DATABASE_URL`, then `DATABASE_URL`
- SQLite: `SQLITE_DATABASE_URL`, then `TEST_DATABASE_URL`, then `TIDAL_DATABASE_URL`, then `DATABASE_URL`
