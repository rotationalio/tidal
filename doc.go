// Package tidal connects to SQL databases and runs typed CRUD queries without an ORM.
//
// Open a connection with [Open], start a transaction with [DB.BeginTx], and pass the
// transaction to [CRUD] methods from [New]. Write SQL with :name placeholders; tidal
// rewrites them for Postgres ($1, $2, …) or SQLite (?).
//
// Implement [Model] on your structs and embed [BaseModel] for IDs and timestamps.
// Use [ListFilter] ([Filter] or [Clause]) to sort and paginate list queries.
package tidal
