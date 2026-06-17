package builder

import "reflect"

// Returns the elements to bind for an [In] condition. Non-slice values are
// treated as a single-element list. Nil and empty slices are treated as empty.
func inValues(value any) []any {
	if value == nil {
		return nil
	}

	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		if rv.Len() == 0 {
			return nil
		}
		out := make([]any, rv.Len())
		for i := range out {
			out[i] = rv.Index(i).Interface()
		}
		return out
	default:
		return []any{value}
	}
}

// Returns true if the condition should be added to the WHERE clause.
func shouldAddCondition(op WhereOp, value any) bool {
	// All operators except In are always added.
	if op != In {
		return true
	}

	// An In condition with an empty slice is omitted.
	return len(inValues(value)) > 0
}
