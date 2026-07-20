# ADR-002: SQLite by default, behind a Store interface, migrated with goose

Date: 2026-07-21
Status: accepted

## Context

The target deployment is a $5 VPS run by one person (PRD §1.4.5). Postgres is listed as
optional. The options were: SQLite only; Postgres only; SQLite now behind an interface with
Postgres later; or an ORM to abstract both from the start.

An ORM was rejected on principle — the query surface here is small and mostly analytical
(histograms, per-source funnels), which is where ORMs are worst. Postgres-first was rejected
because requiring a second container contradicts the single-artifact promise in ADR-001.

## Decision

`internal/store` exposes a `Store` interface. The MVP ships one implementation, SQLite via
`modernc.org/sqlite` (pure Go — no cgo, so cross-compilation in goreleaser stays trivial).
Migrations are goose, embedded in the binary and run automatically at startup.

SQL is hand-written. Queries that differ between SQLite and Postgres live behind interface
methods, not behind string templating.

## Consequences

- Cross-compilation stays free; no cgo toolchain in CI.
- SQLite's single-writer model means the pipeline must not hold a write transaction open across
  an LLM call. Writes are short and happen after scoring returns.
- Adding Postgres in v0.2 is a new implementation of an existing interface plus a second set of
  migration files — additive, and the interface having been exercised by only one implementation
  is the main risk (we will discover SQLite-shaped assumptions then).
- Automatic migration on startup means a downgrade is not supported. Acceptable for a
  single-operator tool; documented in the README rather than solved.
