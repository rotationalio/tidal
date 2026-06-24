package fields

import (
	"database/sql/driver"
	"fmt"
	"time"
)

type Timestamp struct {
	time.Time
}

func (t Timestamp) String() string {
	return t.Format(time.RFC3339)
}

func (t *Timestamp) Scan(src any) error {
	if src == nil {
		*t = Timestamp{Time: time.Time{}}
		return nil
	}

	switch src := src.(type) {
	case time.Time:
		*t = Timestamp{Time: src}
	case string:
		ts, err := time.Parse(time.RFC3339, src)
		if err != nil {
			return err
		}
		*t = Timestamp{Time: ts}
	default:
		return fmt.Errorf("invalid timestamp type: %T", src)
	}

	// Ensure timestamps are always in UTC
	t.Time = t.Time.In(time.UTC)
	return nil
}

func (t Timestamp) Value() (driver.Value, error) {
	return t.String(), nil
}
