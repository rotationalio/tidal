// Package builder provides SQL clause fragments for [filter.Filter] list queries.
//
// Types such as [Where], [Ordering], [Limit], and [Offset] render ANSI SQL fragments
// (WHERE, ORDER BY, LIMIT, OFFSET). [filter.Filter] composes them into a single
// [filter.ListFilter] clause.
package builder
