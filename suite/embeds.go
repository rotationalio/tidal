package suite

import "embed"

//go:embed testdata/postgres
var PostgresTestdata embed.FS

//go:embed testdata/sqlite
var SQLiteTestdata embed.FS
