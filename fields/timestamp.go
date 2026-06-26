package fields

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

const ISO8601 = "2006-01-02T15:04:05.000Z07:00"

// Creates a new Timestamp with the given time.Time value.
func Time(ts time.Time) Timestamp {
	t := Timestamp{}
	t.Set(ts)
	return t
}

// Timestamp wraps a time.Time value in order to ensure that the underlying database
// always has a timestamp stored in the UTC timezone and truncated to millisecond
// precision. It also treats zero valued timestamps as NULL values so there is no need
// for a NullTime type. This timestamp is not well suited for all use cases, but
// generally can unify timestamp behavior across different databases.
type Timestamp struct {
	ts time.Time
}

// Correctly set the timestamp to the UTC timezone and truncated to millisecond precision.
func (t *Timestamp) Set(ts time.Time) {
	t.ts = ts.UTC().Truncate(time.Millisecond)
}

// Sets the timestamp to the current time.
func (t *Timestamp) Now() {
	t.ts = time.Now().UTC().Truncate(time.Millisecond)
}

// Returns the timestamp as a string in the RFC3339 format.
func (t Timestamp) String() string {
	return t.ts.Format(ISO8601)
}

// Returns true if the timestamp is zero (considered null).
func (t Timestamp) IsZero() bool {
	return t.ts.IsZero()
}

// Equal mirrors [time.Time.Equal] for Timestamp operands.
func (t Timestamp) Equal(other Timestamp) bool {
	return t.ts.Equal(other.ts)
}

// Compare mirrors [time.Time.Compare] for Timestamp operands.
func (t Timestamp) Compare(other Timestamp) int {
	return t.ts.Compare(other.ts)
}

// Time returns the underlying [time.Time] value (already normalized when set).
func (t Timestamp) Time() time.Time {
	return t.ts
}

// Add mirrors [time.Time.Add] and returns a new normalized Timestamp.
func (t Timestamp) Add(d time.Duration) Timestamp {
	return Time(t.ts.Add(d))
}

// Sub mirrors [time.Time.Sub] for Timestamp operands.
func (t Timestamp) Sub(other Timestamp) time.Duration {
	return t.ts.Sub(other.ts)
}

//============================================================================
// SQL Methods
//============================================================================

func (t *Timestamp) Scan(src any) (err error) {
	if src == nil {
		*t = Timestamp{ts: time.Time{}}
		return nil
	}

	switch src := src.(type) {
	case time.Time:
		t.Set(src)
		return nil
	case string:
		var ts time.Time
		if ts, err = time.Parse(ISO8601, src); err != nil {
			return err
		}
		t.Set(ts)
		return nil
	default:
		return fmt.Errorf("invalid timestamp type: %T", src)
	}
}

func (t Timestamp) Value() (driver.Value, error) {
	if t.ts.IsZero() {
		return nil, nil
	}
	return t.ts.Format(ISO8601), nil
}

//============================================================================
// JSON Methods
//============================================================================

func (t Timestamp) MarshalJSON() ([]byte, error) {
	if t.ts.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.String())
}

func (t *Timestamp) UnmarshalJSON(data []byte) (err error) {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	// If s is the empty string, then it is the null value.
	if s == "" {
		t.ts = time.Time{}
		return nil
	}

	// Otherwise, parse the string as a time.Time.
	var ts time.Time
	if ts, err = time.Parse(ISO8601, s); err != nil {
		return fmt.Errorf("cannot parse %q as an ISO-8601 timestamp", s)
	}

	t.Set(ts)
	return nil
}
