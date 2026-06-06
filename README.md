# Tidal

[![CI Tests](https://github.com/rotationalio/tidal/actions/workflows/tests.yaml/badge.svg)](https://github.com/rotationalio/tidal/actions/workflows/tests.yaml)
[![Go Doc](https://pkg.go.dev/badge/go.rtnl.ai/tidal)](https://pkg.go.dev/go.rtnl.ai/tidal)
[![Go Report Card](https://goreportcard.com/badge/go.rtnl.ai/tidal)](https://goreportcard.com/report/go.rtnl.ai/tidal)

**SQL Database Store.**

Tidal provides internal mechanisms for managing SQL databases in Rotational applications. It provides a migrations mechanism for storing schema versions inside the database and automatically applying schema changes. It also provides a CRUD and Model interface for use with direct SQL statements rather than ORM functionality. Tidal is not meant to be generally used but implements the Rotational SQL pattern.
