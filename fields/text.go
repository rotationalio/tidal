package fields

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"slices"
)

// StringArray stores a list of strings as a JSON array in a NOT NULL column.
type StringArray []string

// NullStringArray stores a list of strings in a nullable column. Check Valid after scanning.
type NullStringArray struct {
	Valid       bool
	StringArray StringArray
}

//============================================================================
// StringArray Methods
//============================================================================

// Scan implements [database/sql.Scanner].
func (s *StringArray) Scan(src any) (err error) {
	if src == nil {
		*s = nil
		return nil
	}

	switch src := src.(type) {
	case []byte:
		if err = json.Unmarshal(src, s); err != nil {
			return err
		}
	case string:
		if err = json.Unmarshal([]byte(src), s); err != nil {
			return err
		}
	default:
		return fmt.Errorf("cannot scan type %T into StringArray", src)
	}

	return nil
}

// Value implements [database/sql/driver.Valuer].
func (s StringArray) Value() (driver.Value, error) {
	if len(s) == 0 {
		return nil, nil
	}
	return json.Marshal(s)
}

// Equal compares two arrays, treating nil and empty arrays as equivalent.
func (s StringArray) Equal(other StringArray) bool {
	if len(s) == 0 && len(other) == 0 {
		return true
	}
	if len(s) != len(other) {
		return false
	}

	// Compare as multisets so element order does not affect equality.
	a := slices.Clone([]string(s))
	b := slices.Clone([]string(other))
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

//============================================================================
// NullStringArray Methods
//============================================================================

// Scan implements [database/sql.Scanner].
func (n *NullStringArray) Scan(src any) (err error) {
	if src == nil {
		n.StringArray, n.Valid = nil, false
		return nil
	}

	if err = n.StringArray.Scan(src); err != nil {
		n.Valid = false
		return err
	}

	n.Valid = len(n.StringArray) > 0
	return nil
}

// Value implements [database/sql/driver.Valuer].
func (n NullStringArray) Value() (driver.Value, error) {
	if !n.Valid {
		return nil, nil
	}
	return n.StringArray.Value()
}

// Equal compares nullable arrays, normalizing empty arrays to SQL NULL semantics.
func (n NullStringArray) Equal(other NullStringArray) bool {
	if len(n.StringArray) == 0 {
		n.Valid = false
	}
	if len(other.StringArray) == 0 {
		other.Valid = false
	}

	if n.Valid != other.Valid {
		return false
	}
	return n.StringArray.Equal(other.StringArray)
}
