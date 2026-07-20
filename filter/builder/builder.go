// Package builder provides SQL clause fragments for [filter.Filter] list queries.
//
// Types such as [Where], [Ordering], [Limit], and [Offset] render ANSI SQL fragments
// (WHERE, ORDER BY, LIMIT, OFFSET). [Where] supports Eq, Ne, comparisons, Like,
// ILike, In, Is, IsNot (with [Literal] or a bound value), and IS DISTINCT FROM.
// Deprecated WhereOp values IsNull and IsNotNull remain supported.
// [filter.Filter] composes them into a single [filter.ListFilter] clause.
package builder
