# ADR-0002: pgx + sqlc + goose; no ORM

Date: 2026-11-12 · Status: accepted

## Context

The Go backend talks to PostgreSQL (its own app-db, and as the reference
implementation for the Go + Postgres curriculum module). The choices must themselves be
idiomatic examples.

## Decision

- `github.com/jackc/pgx/v5` with `pgxpool` as the driver/connection pool.
- `sqlc` generates type-safe store code from SQL files (`server/internal/store`).
- `goose` for migrations, as SQL files embedded with `embed.FS`.

## Alternatives

- **`database/sql` + an ORM (GORM, ent, sqlc alternatives like sqlx)**: `database/sql`
  is taught as a comparison lesson, but native pgx is what a serious Go+Postgres service
  uses (pg types, `CopyFrom`, `LISTEN/NOTIFY`, and the simple protocol the SQL runner
  uses to run lesson statements one by one, as psql does). ORMs generate exactly the SQL the lessons want the learner to read
  and write, and hide NULL/scan semantics.
- **Hand-written store structs**: maximal clarity but drifts from the sqlc workflow the
  curriculum explicitly teaches (§8.1 module 13).

## Consequences

- Queries live as `.sql` files — readable, reviewable, and the sqlc workflow
  (`make gen`) is itself a lesson.
- sqlc's type generation occasionally needs `jsonb`/array type overrides; recorded in
  `sqlc.yaml` with comments when it happens.
- Migration files are `embed`ded so the backend binary migrates its database at startup.

## Update (2026-11-14)

sqlc was never adopted for the app. Its few queries are hand-written pgx calls in
`internal/runstore` and `internal/learningstore` (the "hand-written store" alternative
above); there is no `sqlc.yaml` and no `server/internal/store`, and `make gen` doesn't
run sqlc. The sqlc workflow is taught in lesson `go.postgres.migrations-sqlc` instead.
pgx and goose are as decided: goose runs as a library (`platform.Migrate`, goose's
`Provider`) over the migrations embedded in `cmd/stacklab`.
