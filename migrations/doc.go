// Package migrations loads versioned SQL files from an [io/fs.FS] and applies any
// that have not been run yet.
//
// Name files NNNN_description.sql (for example 0001_initial_schema.sql), embed them
// with //go:embed, load with [Load], then call [Migrations.ApplyPostgres],
// [Migrations.ApplySQLite], or [Migrations.Apply] after connecting. Pass your app
// version string so each applied migration is recorded with the release that ran it.
package migrations
