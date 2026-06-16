// Package builder provides SQL clause fragments for [filter.Filter] list queries.
//
// Types such as [Ordering], [Limit], and [Offset] implement [fmt.Stringer] and render
// ANSI SQL fragments (ORDER BY, LIMIT, OFFSET). [filter.Filter] composes them into a
// single [filter.ListFilter] clause.
package builder
