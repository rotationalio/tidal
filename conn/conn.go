// Package conn opens database connections and runs transactions with :name placeholders.
//
// Use [Open] to connect from a DSN, or [Wrap] when you already have a [sql.DB].
// [DB.BeginTx] returns a [Tx] that rewrites :name SQL for Postgres ($1, $2, …) or
// SQLite automatically.
//
// Example:
//
//	package db
//
//	import (
//		"context"
//		"database/sql"
//		"os"
//
//		"go.rtnl.ai/tidal/conn"
//		"go.rtnl.ai/x/dsn"
//	)
//
//	func Connect(ctx context.Context) (*conn.DB, error) {
//		uri, err := dsn.Parse(os.Getenv("DATABASE_URL"))
//		if err != nil {
//			return nil, err
//		}
//		return conn.Open(ctx, uri)
//	}
//
//	tx, err := db.BeginTx(ctx, nil)
//	if err != nil {
//		return err
//	}
//	defer tx.Rollback()
//
//	_, err = tx.Exec(
//		"INSERT INTO users (id, email) VALUES (:id, :email)",
//		sql.Named("id", id),
//		sql.Named("email", email),
//	)
package conn
