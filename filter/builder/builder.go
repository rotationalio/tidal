// Package builder provides SQL clause fragments for [filter.Filter] list queries.
package builder

import (
	"slices"
	"strings"
)

type Prefixer interface {
	// Add a table alias to the beginning of the field in the clause using the dot
	// separator. If fields is specified, only the fields in the list will be prefixed,
	// otherwise all the fields in the clause will be prefixed. To remove a prefix, set
	// the tableAlias to an empty string.
	Prefix(tableAlias string, fields ...string)
}

// Adds (or removes) a table alias to the beginning of the field using the dot separator.
// This function should be used to implement the Prefixer interface.
func Prefix(field string, tableAlias string, fields ...string) string {
	unprefixed := field
	if i := strings.Index(field, "."); i != -1 {
		unprefixed = field[i+1:]
	}

	if len(fields) > 0 && !slices.Contains(fields, unprefixed) {
		return field
	}

	if tableAlias != "" {
		return tableAlias + "." + unprefixed
	}

	return unprefixed
}
