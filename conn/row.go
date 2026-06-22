package conn

import "database/sql"

// Row is the result of calling [Tx.QueryRow] to select a single row. It mirrors
// [sql.Row]: errors from query execution or parameter binding are returned from
// [Row.Scan] and [Row.Err], not from [Tx.QueryRow].
type Row struct {
	row    *sql.Row
	err    error
	mapErr func(error) error
}

// Scan copies the columns from the matching row into dest, delegating to the
// underlying [sql.Row] when the query was executed successfully.
func (r *Row) Scan(dest ...any) error {
	if r.err != nil {
		return r.mapErr(r.err)
	}
	if err := r.row.Scan(dest...); err != nil {
		return r.mapErr(err)
	}
	return nil
}

// Err returns any error deferred while preparing or running the query.
func (r *Row) Err() error {
	if r.err != nil {
		return r.mapErr(r.err)
	}
	if r.row != nil {
		return r.mapErr(r.row.Err())
	}
	return nil
}
