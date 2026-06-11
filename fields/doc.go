// Package fields provides column types for JSON and string arrays.
//
// Use them as fields on [tidal.Model] structs. They implement [database/sql.Scanner]
// and [database/sql/driver.Valuer], so [tidal.CRUD] queries read and write them
// without extra conversion code. Backing columns should store JSON (JSONB, BYTEA,
// BLOB, or TEXT depending on your database).
package fields
