package mock

import "database/sql"

const (
	LastInsertId = "LastInsertId"
	RowsAffected = "RowsAffected"
)

type Result struct {
	Mock
	OnLastInsertId func() (int64, error)
	OnRowsAffected func() (int64, error)
}

var _ sql.Result = (*Result)(nil)

//============================================================================
// Helper Methods
//============================================================================

func (r *Result) Reset() {
	r.Mock.Reset()
	r.OnLastInsertId = nil
	r.OnRowsAffected = nil
}

func (r *Result) LastInsertId() (int64, error) {
	r.increment(LastInsertId)
	if r.OnLastInsertId == nil {
		return 42, nil
	}
	return r.OnLastInsertId()
}

func (r *Result) RowsAffected() (int64, error) {
	r.increment(RowsAffected)
	if r.OnRowsAffected == nil {
		return 1, nil
	}
	return r.OnRowsAffected()
}
