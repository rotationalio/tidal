// Package fields provides [tidal.Model] column types for JSON, string arrays,
// and normalized timestamps.
//
// Use them as fields on [tidal.Model] structs. They implement [database/sql.Scanner]
// and [database/sql/driver.Valuer], so [tidal.CRUD] queries read and write them
// without extra conversion code. Backing columns should store JSON (JSONB, BYTEA,
// BLOB, or TEXT depending on your database).
//
// Example:
//
//	type Document struct {
//		tidal.BaseModel
//		Metadata fields.JSONB
//		Tags     fields.StringArray
//		SeenAt   fields.Timestamp
//	}
package fields
