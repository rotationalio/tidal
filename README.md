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
| [`go.rtnl.ai/tidal/filter`](https://pkg.go.dev/go.rtnl.ai/tidal/filter) | `Filter`, `CustomFilter`, list-query pagination |
| [`go.rtnl.ai/tidal/store`](https://pkg.go.dev/go.rtnl.ai/tidal/store) | `CRUD`, `Cursor`, query generation |
| [`go.rtnl.ai/tidal/bind`](https://pkg.go.dev/go.rtnl.ai/tidal/bind) | `:name` placeholder rewriting |
| [`go.rtnl.ai/tidal/migrations`](https://pkg.go.dev/go.rtnl.ai/tidal/migrations) | Versioned SQL schema migrations |
| [`go.rtnl.ai/tidal/fields`](https://pkg.go.dev/go.rtnl.ai/tidal/fields) | JSON, string-array, and normalized timestamp column types for `Model` structs |
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

Pass `tidal.Tx` to [`CRUD`](https://pkg.go.dev/go.rtnl.ai/tidal#CRUD) methods and [`Cursor`](https://pkg.go.dev/go.rtnl.ai/tidal#Cursor) results.

When you need a raw `*sql.DB` — for migrations, third-party libraries, or admin DDL — use the embedded connection (`db.DB`) or [`DB.SQLDB`](https://pkg.go.dev/go.rtnl.ai/tidal#DB.SQLDB).

If you already have an open `*sql.DB`, wrap it with [`tidal.Wrap`](https://pkg.go.dev/go.rtnl.ai/tidal#Wrap). You still need a parsed `*dsn.DSN` so tidal knows which placeholder style to use:

```go
uri, _ := dsn.Parse(os.Getenv("DATABASE_URL"))
sqlDB, _ := sql.Open("sqlite3", uri.Path)
db := tidal.Wrap(sqlDB, uri)
```

## Connection options

Connection URLs are parsed by [`go.rtnl.ai/x/dsn`](https://pkg.go.dev/go.rtnl.ai/x/dsn). Query parameters become `DSN.Options` and are handled in one of three ways: consumed by tidal for pool or connection behavior, read from the DSN at runtime, or passed through to the driver unchanged. See the [dsn package docs](https://pkg.go.dev/go.rtnl.ai/x/dsn) for URL format and Postgres libpq parameters (`sslmode`, `connect_timeout`, and so on).

### Shared

| Option | Description |
| --- | --- |
| `readonly` | When `true`, [`DB.BeginTx`](https://pkg.go.dev/go.rtnl.ai/tidal#DB.BeginTx) defaults to read-only transactions and rejects writes. |

### Postgres

Tidal opens Postgres through [pgx](https://github.com/jackc/pgx) (`database/sql` stdlib driver).

| Option | Description |
| --- | --- |
| `max_idle_conns` | `database/sql` pool setting (default `8`). Removed from the URL before connecting. |
| `max_open_conns` | `database/sql` pool setting (default `16`). Removed from the URL before connecting. |
| `conn_max_lifetime` | `database/sql` pool setting (default `1h`). Removed from the URL before connecting. |
| `conn_max_idle_time` | `database/sql` pool setting (default `30m`). Removed from the URL before connecting. |

All other query parameters are forwarded to Postgres as normal connection options — see [dsn](https://pkg.go.dev/go.rtnl.ai/x/dsn).

On connect, tidal registers a pgx `timestamptz` codec so values scanned into `time.Time` use `time.UTC` as their location (the instant is unchanged). This matches lib/pq behavior and avoids local-timezone surprises in tests and equality checks. Not configurable via DSN yet.

### SQLite3

| Option | Description |
| --- | --- |
| `readonly` | Same as above. SQLite read-only mode is enforced at the transaction level. |

The database file path comes from the DSN path (`sqlite3:///path/to/db.sqlite`). Tidal does not set SQLite pragmas on `Open`; run `PRAGMA foreign_keys = on` or `PRAGMA query_only = on` after connect in application code when you need them.

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

cursor, err := crud.List(tx, tidal.OrderBy("name").Limit(10))
users, err := cursor.List()
```

[`Filter`](https://pkg.go.dev/go.rtnl.ai/tidal#Filter) builds ANSI SQL `WHERE`, `ORDER BY`, `LIMIT`, and `OFFSET` clauses. Calling `Where` replaces any previous WHERE clause (like `OrderBy`, `Limit`, and `Offset`). `And`, `Or`, `AndGroup`, and `OrGroup` append to the current WHERE clause.

Start a filter with a constructor instead of the `&tidal.Filter{}` literal. Each constructor returns a `*Filter` you can keep chaining from:

| Constructor | Creates a filter with |
| --- | --- |
| [`tidal.NewFilter()`](https://pkg.go.dev/go.rtnl.ai/tidal#NewFilter) | no clauses (empty filter) |
| [`tidal.Where(field, op, value)`](https://pkg.go.dev/go.rtnl.ai/tidal#Where) | an initial `WHERE` condition |
| [`tidal.OrderBy(columns...)`](https://pkg.go.dev/go.rtnl.ai/tidal#OrderBy) | an `ORDER BY` clause |
| [`tidal.Limit(n)`](https://pkg.go.dev/go.rtnl.ai/tidal#Limit) | a `LIMIT` clause |
| [`tidal.Offset(n)`](https://pkg.go.dev/go.rtnl.ai/tidal#Offset) | an `OFFSET` clause |

These re-export [`filter.New`](https://pkg.go.dev/go.rtnl.ai/tidal/filter#New), [`filter.Where`](https://pkg.go.dev/go.rtnl.ai/tidal/filter#Where), [`filter.OrderBy`](https://pkg.go.dev/go.rtnl.ai/tidal/filter#OrderBy), [`filter.Limit`](https://pkg.go.dev/go.rtnl.ai/tidal/filter#Limit), and [`filter.Offset`](https://pkg.go.dev/go.rtnl.ai/tidal/filter#Offset) (root uses `NewFilter` because `tidal.New` builds a CRUD store). Import `go.rtnl.ai/tidal/filter` to call them under the `filter.` prefix instead.

WHERE operators: `Eq`, `Ne`, `Gt`, `Lt`, `Gte`, `Lte`, `Like`, `ILike`, `In`, `Is`, `IsNot`, `IsDistinctFrom`, and `IsNotDistinctFrom`. An `In` condition with an empty slice is omitted. `ILike`, `Is`, `IsNot`, `IsDistinctFrom`, and `IsNotDistinctFrom` are not supported by every database provider; invalid combinations fail at query time.

Use `Is` and `IsNot` with a [`Literal`](https://pkg.go.dev/go.rtnl.ai/tidal/filter#Literal) (`Null`, `True`, `False`, `Unknown`) for SQL keyword predicates such as `IS NULL` and `IS NOT TRUE`. Pass any other value to compare with a bound parameter (for example `status IS :w1`). PostgreSQL accepts `Literal` values only with `Is` and `IsNot`; SQLite also allows arbitrary bound values. `IsNull` and `IsNotNull` are deprecated; use `Is` with `Null` or `IsNot` with `Null` instead.

`IsDistinctFrom` and `IsNotDistinctFrom` render null-safe comparisons with a bound parameter (for example `a IS DISTINCT FROM :w1`). Use `ILike` for case-insensitive pattern matching on providers that support it (for example PostgreSQL).

```go
f := tidal.Where("revoked", tidal.Is, tidal.Null).
 And("deleted_at", tidal.IsNot, tidal.Null).
 And("active", tidal.Is, tidal.True)
// WHERE revoked IS NULL AND deleted_at IS NOT NULL AND active IS TRUE
```

Flat `And`/`Or` chains follow SQL precedence: `Where("a", Eq, 1).And("b", Eq, 2).Or("c", Eq, 3)` renders `a = :w1 AND b = :w2 OR c = :w3`, which SQL parses as `(a AND b) OR c`. Use `AndGroup` or `OrGroup` for explicit grouping.

```go
// tidal.Where starts the filter; a second Where replaces the first WHERE clause.
f := tidal.Where("status", tidal.Eq, "active").
 Where("role", tidal.Eq, "admin") // only role = :w1 remains

// And/Or/AndGroup/OrGroup append to the current WHERE clause.
f = tidal.Where("status", tidal.Eq, "active").
 And("id", tidal.In, []int64{1, 2, 3}).
 And("age", tidal.Gte, 18).
 AndGroup(func(g *tidal.WhereGroup) {
  g.Where("role", tidal.Eq, "admin").Or("role", tidal.Eq, "editor")
 }).
 OrderBy("-created").
 Limit(20)

cursor, err := crud.List(tx, f)
```

Use [`CustomFilter`](https://pkg.go.dev/go.rtnl.ai/tidal#CustomFilter) for hand-written SQL when you need clauses `Filter` does not build (for example `GROUP BY`):

```go
filter := &tidal.CustomFilter{
 SQL:  "WHERE status = :status GROUP BY name",
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

`Apply` and `LastApplied` accept any value that implements [`conn.Beginner`](https://pkg.go.dev/go.rtnl.ai/tidal/conn#Beginner) — typically your `*tidal.DB` from `tidal.Open`:

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
| `Timestamp` | struct wrapper around `time.Time` | UTC-normalized timestamp values with stable precision across drivers | Zero value writes/scans as SQL `NULL` |

`JSONB`, `NullJSONB`, `StringArray`, and `NullStringArray` marshal/scan as JSON, so those columns should use JSON-compatible storage (`JSONB` or `BYTEA` in Postgres, `BLOB`/`TEXT` in SQLite). `Timestamp` stores ISO-8601 UTC values and normalizes precision when read/written.

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

### Timestamp

`Timestamp` wraps `time.Time` with driver-friendly normalization: values are stored in UTC and truncated to millisecond precision on assignment/scan. This keeps equality behavior stable across provider round-trips.

API behavior mirrors `time.Time` where possible:

- `Equal(other)` and `Compare(other)` follow `time.Time.Equal` and `time.Time.Compare`.
- `Add(d)` returns a new normalized `Timestamp` (non-mutating, like `time.Time.Add`).
- `Sub(other)` returns a duration (like `time.Time.Sub`).
- `Time()` returns the underlying `time.Time` value.

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
 })
}
```

`DatabaseSuite` creates a per-test context in `SetupTest`. Subtests run with child contexts and restore the parent context between `s.Run(...)` calls, so parent test code can safely call `s.Context()` and `s.BeginTx(nil)` between subtests.

Per-test teardown defaults to truncating tables (`TeardownTruncate`) for fast integration tests. Set `s.Teardown = suite.TeardownDropAndMigrate` for migration/schema reset behavior, or `suite.TeardownNone` to skip data teardown.

`Create` should return a valid insert each time — generate unique values (email, slug, etc.) inside the factory. `Update` receives the same instance that was created and inserted.

Conformance equality now prefers model/field semantic interfaces: implement `Equal(other T) bool` or `Compare(other T) int` on your model (and custom field types) so both Scan and RoundTrip comparisons use domain-aware equality before reflective fallback.

Conformance comparison precedence is:

1. `CRUDConformance.Equal` (deprecated override; replace with model `Equal(other)` implementations)
2. model `Equal(other)` implementation
3. model `Compare(other) == 0` implementation
4. built-in reflective fallback with tidal field/time normalization rules

`CRUDConformance.Equal` is still supported for backward compatibility, but deprecated in favor of interface-based equality.

`CRUDConformance` supports the following fields:

Required:

- `Table string` — database table mapped by the model.
- `Create func() M` — factory for a fresh model instance to insert.
- `Update func(M)` — mutates the created model for update-phase checks.

Optional:

- `Phases []suite.CRUDPhase` — choose which phases to run (default: `Shape`, `Scan`, `RoundTrip`).
- `ScanColumns map[tidal.Operation][]string` — override the exact columns fed into `Scan(op)` in Scan conformance.
- `ScanOps []tidal.Operation` — limit which operations Scan conformance runs (default: `Create`, `Retrieve`, `Update`).
- `FieldMap map[string]string` — map DB column name -> Go struct field name when snake_case matching is not enough (for acronyms like `client_id` -> `ClientID`, etc.).
- `Equal func(a, b M) bool` — deprecated; use model `Equal`/`Compare` methods instead.

When `ScanColumns` is not set for `Update`, conformance falls back to `Fields(Retrieve)` if `Update` is a strict subset of retrieve columns; this matches models where `Scan(Update)` reads full-row projections.

```go
suite.ConformsCRUD(&s.DatabaseSuite, suite.CRUDConformance[*APIKey]{
 Table:  "api_keys",
 Create: newAPIKey,
 Update: func(k *APIKey) { k.Description = sql.NullString{Valid: true, String: "updated"} },
 FieldMap: map[string]string{
  "client_id": "ClientID",
 },
 ScanColumns: map[tidal.Operation][]string{
  tidal.Update: {"id", "description", "client_id", "secret", "created_by", "last_seen", "revoked", "created", "modified"},
 },
})
```

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

To benchmark bind rewrite performance:

```bash
go test ./bind -run '^$' -bench '^BenchmarkRewrite$' -benchmem -count=1
```

Last observed benchmark on darwin/arm64 (Apple M2):

| Case | ns/op | B/op | allocs/op |
| --- | ---: | ---: | ---: |
| OrderedSimple | 937.9 | 198 | 6 |
| OrderedComplex | 1407 | 308 | 5 |
| PositionalSimple | 549.4 | 256 | 4 |

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
