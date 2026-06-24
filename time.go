package tidal

import (
	"time"

	"go.rtnl.ai/x/dsn"
)

// Databases store timestamps at different precisions that are generally less granular
// than Golang's nanosecond precision. This function returns the precision of the
// timestamps stored in the database for a given provider.
func TimePrecision(provider string) time.Duration {
	switch provider {
	case dsn.Honu:
		return time.Microsecond
	case dsn.Postgres:
		return time.Microsecond
	case dsn.MySQL:
		return time.Second
	case dsn.SQLite, dsn.SQLite3:
		return time.Millisecond
	case "mssql":
		return 100 * time.Nanosecond
	default:
		return time.Millisecond
	}
}

// NormalizeTime normalizes a time.Time value to the precision of the database provider
// and in the UTC timezone if any other timezone is specified.
func NormalizeTime(t time.Time, provider string) time.Time {
	return t.UTC().Truncate(TimePrecision(provider))
}
