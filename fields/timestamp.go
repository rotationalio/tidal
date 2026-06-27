package fields

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

const ISO8601Milli = "2006-01-02T15:04:05.000Z07:00"

// Timestamp wraps a time.Time value in order to ensure that the underlying database
// always has a timestamp stored in the UTC timezone and truncated to millisecond
// precision. It also treats zero valued timestamps as NULL values so there is no need
// for a NullTime type. This timestamp is not well suited for all use cases, but
// generally can unify timestamp behavior across different databases.
type Timestamp struct {
	ts time.Time
}

// --- Constructors ---

// Creates a new Timestamp with the given time.Time value.
func Time(ts time.Time) Timestamp {
	t := Timestamp{}
	t.Set(ts)
	return t
}

// Returns a new Timestamp with the current time.
func Now() Timestamp {
	return Time(time.Now())
}

// Parse parses a time string in the given layout into a Timestamp.
func Parse(layout, value string) (Timestamp, error) {
	ts, err := time.Parse(layout, value)
	if err != nil {
		return Timestamp{}, err
	}
	return Time(ts), nil
}

// --- Setters ---

// Set the timestamp to the given time in the UTC timezone, truncated to
// millisecond precision.
func (t *Timestamp) Set(ts time.Time) {
	t.ts = ts.UTC().Truncate(time.Millisecond)
}

// Sets the timestamp to the current UTC time.
func (t *Timestamp) Now() {
	t.ts = time.Now().UTC().Truncate(time.Millisecond)
}

// --- Formatters ---

// Returns the timestamp as a string in the RFC3339 format.
func (t Timestamp) String() string {
	return t.ts.Format(ISO8601Milli)
}

// Returns the timestamp as a formatted string.
func (t Timestamp) Format(layout string) string {
	return t.ts.Format(layout)
}

// --- Comparison ---

// Returns true if the timestamp is zero (considered null).
func (t Timestamp) IsZero() bool {
	return t.ts.IsZero()
}

// Returns the result of comparing the two timestamps (0 if equal, -1 if less,
// 1 if greater).
func (t Timestamp) Compare(other Timestamp) int {
	return t.ts.Compare(other.ts)
}

// Returns true if the timestamp is equal to the other timestamp.
func (t Timestamp) Equal(other Timestamp) bool {
	return t.ts.Equal(other.ts)
}

// Returns true if the timestamp is before the other timestamp.
func (t Timestamp) Before(other Timestamp) bool {
	return t.ts.Before(other.ts)
}

// Returns true if the timestamp is after the other timestamp.
func (t Timestamp) After(other Timestamp) bool {
	return t.ts.After(other.ts)
}

// --- Getters ---

// Returns the underlying [time.Time] value (already normalized when set).
func (t Timestamp) Time() time.Time {
	return t.ts
}

// Returns the underlying [time.Time] value in the UTC timezone.
func (t Timestamp) UTC() time.Time {
	return t.ts // ts is already in UTC by construction
}

// Returns the unix epoch seconds of the timestamp.
func (t Timestamp) Unix() int64 {
	return t.ts.Unix()
}

// Returns the unix milliseconds of the timestamp.
func (t Timestamp) UnixMilli() int64 {
	return t.ts.UnixMilli()
}

// No nanoseconds available, so no UnixNano method.

// --- Operators ---

// Returns a new normalized Timestamp by adding the given duration.
func (t Timestamp) Add(d time.Duration) Timestamp {
	return Time(t.ts.Add(d))
}

// Returns the duration between the two timestamps.
func (t Timestamp) Sub(other Timestamp) time.Duration {
	return t.ts.Sub(other.ts)
}

// Returns the duration since the other timestamp. Equivalent to [Timestamp.Sub].
func (t Timestamp) Since(other Timestamp) time.Duration {
	return t.Sub(other)
}

// Returns the duration until the other timestamp. Equivalent to argument-inverted
// [Timestamp.Sub].
func (t Timestamp) Until(other Timestamp) time.Duration {
	return other.Sub(t)
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
		if ts, err = time.Parse(ISO8601Milli, src); err != nil {
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
	return t.ts.Format(ISO8601Milli), nil
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
	if ts, err = time.Parse(ISO8601Milli, s); err != nil {
		return fmt.Errorf("cannot parse %q as an ISO-8601 timestamp", s)
	}

	t.Set(ts)
	return nil
}
