package conn

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/stdlib"
	"go.rtnl.ai/x/dsn"
)

// Opens a Postgres connection using pgx and applies pool options from uri.
func openPostgres(_ context.Context, uri *dsn.DSN, defaults map[string]string) (*sql.DB, error) {
	connStr, pgopts, err := dsn.PGConnectionOptions(uri, defaults)
	if err != nil {
		return nil, errors.Join(ErrConnectionOptions, err)
	}

	cfg, err := pgx.ParseConfig(connStr)
	if err != nil {
		return nil, errors.Join(ErrConnectionOptions, err)
	}

	// TODO: allow the user to set this option in the future, for now we default to UTC times
	db := stdlib.OpenDB(*cfg, stdlib.OptionAfterConnect(utcTimestamptzAfterConnect))

	db.SetMaxIdleConns(pgopts.MaxIdleConns)
	db.SetMaxOpenConns(pgopts.MaxOpenConns)
	db.SetConnMaxLifetime(pgopts.ConnMaxLifetime)
	db.SetConnMaxIdleTime(pgopts.ConnMaxIdleTime)
	return db, nil
}

// Sets the time zone to UTC for the timestamptz type for pgx connections.
func utcTimestamptzAfterConnect(ctx context.Context, conn *pgx.Conn) error {
	conn.TypeMap().RegisterType(&pgtype.Type{
		Name:  "timestamptz",
		OID:   pgtype.TimestamptzOID,
		Codec: &pgtype.TimestamptzCodec{ScanLocation: time.UTC},
	})
	return nil
}
