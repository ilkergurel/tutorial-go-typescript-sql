---
id: go.postgres.migrations-sqlc
title: "goose migrations and the sqlc workflow"
track: go
module: postgres
order: 5
level: intermediate
minutes: 14
since: go1.25
prerequisites: [go.postgres.transactions, go.files-io.embed]
compare_with: [sql.security-ops.backup-migrations, fullstack.tracelite.step-03-store, ts.services.openapi]
tags: [postgres, pgx, migrations, goose, sqlc, code-generation, embed]
summary: >
  Schema changes are numbered SQL files applied in order and recorded in a version
  table; on PostgreSQL each one commits or rolls back as a whole. sqlc turns
  annotated queries into typed Go methods over a DBTX interface, checked against
  that same schema at generation time.
examples:
  - id: migrate
    file: examples/01_migrate.go
    title: What goose does underneath, in 30 lines, over an fs.FS
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "15-21"
        note: >
          The version table is the whole memory of the system: a file runs once
          per database, ever. This demo records file names; goose's table,
          goose_db_version, records the version number parsed from the name
          and when each version was applied.
      - lines: "22-23"
        note: >
          Order comes from the file names, so zero-pad sequential prefixes (010
          after 009, not after 001). goose parses the number and sorts by it;
          goose create uses timestamp prefixes unless you pass -s.
      - lines: "29-37"
        note: >
          Recording the file first, in the same transaction as the change,
          means the two commit or roll back together. A second instance
          migrating at the same moment blocks on the primary key, then finds the
          row and skips. goose offers an advisory lock for this instead.
      - lines: "35"
        note: >
          No arguments, so pgx uses the simple protocol and the whole file runs
          in one call. goose splits files into statements itself, which is why
          function bodies need -- +goose StatementBegin / StatementEnd.
      - lines: "53-55"
        note: >
          A real service embeds the directory and passes fs.Sub(embedded,
          "migrations"), exactly as this app's main.go does. The files keep
          goose's required -- +goose Up header, so goose could run them unchanged.
      - lines: "62-65"
        note: >
          Schema and data change together: add the column nullable, backfill
          it, then require it. On a large table SET NOT NULL scans the table
          under an ACCESS EXCLUSIVE lock; the zero-downtime variants are in the
          sql.security-ops module.
      - lines: "69-72"
        note: >
          The first ALTER succeeds, the second fails, and PostgreSQL rolls back
          both: transactional DDL. MySQL commits each DDL statement implicitly
          and would have left city behind.
      - lines: "74-78"
        note: >
          Ayşe's row got its name from the backfill, and city does not exist:
          migration 003 left nothing behind, and because its row in
          schema_migrations rolled back too, a fixed 003 will run on the next
          deploy.
  - id: sqlc-shape
    file: examples/02_sqlc_shape.go
    title: The code sqlc generates for one :many query, and how you call it
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "14-25"
        note: >
          DBTX is declared by the generated code, the consumer, and *pgx.Conn,
          *pgxpool.Pool and pgx.Tx all satisfy it. WithTx rebinds the same
          queries to a transaction: q.WithTx(tx) inside the WithTx helper from
          lesson 3.
      - lines: "27-33"
        note: >
          Models come from the schema: phone is nullable, so it becomes
          pgtype.Text. With emit_pointers_for_null_types: true in sqlc.yaml it
          would be *string instead.
      - lines: "35-38"
        note: >
          You write the SQL with a name and a command (:one, :many, :exec, plus
          :batchexec for pgx batches and :copyfrom for COPY). The constant keeps the name
          comment, so server logs and pg_stat_activity show which query is
          running.
      - lines: "40-58"
        note: >
          The loop CollectRows would run, spelled out with positional Scan: no
          reflection, and columns and fields can't drift apart because both come
          from the same query. var items stays nil for zero rows, which encodes
          to JSON null.
      - lines: "75-79"
        note: >
          The calling code has no SQL and no Scan, and a typo in a column name
          fails at sqlc generate instead of at run time. The NULL phone prints
          as { false}: pgtype.Text's String and Valid fields. Zero rows give a
          nil slice, as the Muğla line shows.
use_cases:
  - "goose (library mode): a service that migrates itself at startup from embedded files, like this app"
  - "goose (CLI): a separate deploy step that runs before new instances start, when several replicas share one database"
  - "sqlc: CRUD and reporting queries whose shape is known at compile time, which is most of a typical service"
  - "Hand-written pgx next to sqlc: dynamic filters and sort orders, COPY pipelines with custom sources"
strengths:
  - "The SQL that runs in production is the SQL in the pull request, unlike an ORM's generated DDL and queries"
  - "PostgreSQL's transactional DDL makes each migration all or nothing; MySQL commits every DDL statement implicitly"
  - "sqlc checks queries against the schema at generation time: a renamed column breaks sqlc generate, not a request in production"
  - "Generated code is plain pgx with positional Scan: no reflection, no runtime magic, readable in a debugger"
weaknesses:
  - "Queries that change shape at run time (optional filters, user-chosen sorting) don't fit one static SQL string; they need several queries, sqlc.narg tricks, or hand-written code"
  - "One more tool in the build: generated files must be regenerated and committed whenever a query or migration changes"
  - "Down migrations are rarely tested and often can't restore dropped data; in practice a bad migration is fixed by a new one"
pitfalls:
  - title: "Editing a migration that already ran"
    detail: "Its version is recorded, so the edit never runs where the old version did, and databases silently drift apart. Write a new migration; the app's own migrations say so in their header."
  - title: "CREATE INDEX CONCURRENTLY in a migration"
    detail: "It cannot run inside a transaction block, and goose wraps each migration in one. Put -- +goose NO TRANSACTION at the top of that file, and keep it to that one statement."
  - title: "Several replicas migrating at startup"
    detail: "goose's locking is off unless you pass WithSessionLocker (for example lock.NewPostgresSessionLocker(), a PostgreSQL advisory lock). Enable it, or run migrations as a separate deploy step."
  - title: "Editing generated code"
    detail: "The next sqlc generate overwrites it. Change the SQL or sqlc.yaml (type overrides, emit_pointers_for_null_types) and regenerate."
  - title: "Semicolons inside function bodies"
    detail: "goose splits a file on semicolons. Wrap CREATE FUNCTION and DO blocks in -- +goose StatementBegin and -- +goose StatementEnd."
quiz:
  - q: "Migration 007 ran in production yesterday. Today you spot a wrong column type in it. What do you do?"
    options: ["Fix 007 and redeploy", "Write 008 that alters the column", "Delete the 007 row from the version table and redeploy"]
    answer: 1
    why: "007 never runs again where it already ran; only a new version reaches every database the same way."
  - q: "The second statement of a goose migration fails on PostgreSQL. What state is the schema in?"
    options: ["The first statement's change is kept", "As before the migration: it ran in one transaction", "Half-applied until goose down runs"]
    answer: 1
    why: "goose runs each migration in a transaction by default, and PostgreSQL rolls DDL back like any other change."
  - q: "How does sqlc know that users.phone is nullable?"
    options: ["It connects to the database at run time", "It parses the schema, which can be your migrations directory, when you run sqlc generate", "From a struct tag you write"]
    answer: 1
    why: "sqlc reads the DDL (ignoring down migrations) and derives the Go types from it before any code runs."
  - q: "Why does generated code take a DBTX instead of a *pgxpool.Pool?"
    options: ["DBTX is faster", "So the same Queries work on a pool, a single connection or a transaction", "pgxpool can't run queries"]
    answer: 1
    why: "All three satisfy the three-method interface; WithTx swaps a transaction in."
further_reading:
  - title: "package goose (v3): Provider, NewProvider, WithSessionLocker"
    url: "https://pkg.go.dev/github.com/pressly/goose/v3"
  - title: "sqlc module page (README; the full manual is linked from the Concept section)"
    url: "https://pkg.go.dev/github.com/sqlc-dev/sqlc"
  - title: "package embed"
    url: "https://pkg.go.dev/embed"
  - title: "PostgreSQL: ALTER TABLE (SET NOT NULL, locking)"
    url: "https://www.postgresql.org/docs/current/sql-altertable.html"
---

## Concept

**Migrations.** The schema's history is a directory of numbered SQL files in the repo.
A tool applies the ones a database hasn't seen yet, in order, and records each version
in a table. A file that has run is never edited again; a change is a new file. goose is a
common choice: files start with `-- +goose Up` (an optional `-- +goose Down` section
undoes them), it records versions in `goose_db_version`, and on PostgreSQL it runs each
migration in a transaction, so a failure leaves nothing half-applied. It works as a CLI
or as a library inside your binary.

**sqlc.** You write queries in `.sql` files, each named and given a cardinality:

```sql
-- name: ListUsersByCity :many
SELECT id, email, city, phone FROM users WHERE city = $1 ORDER BY email;
```

`sqlc generate` parses them against the schema (it can read your migrations directory
directly) and writes Go: a struct per table, a `Queries` type over a `DBTX` interface, and
one typed method per query. A query that names an unknown table or column fails at
generation time. The generated code is ordinary pgx; there is no runtime library.

The workflow: write a migration, write or change a query, run `sqlc generate`, call the
new method, commit all of it together. sqlc's manual covers the details:
[getting started with PostgreSQL](https://docs.sqlc.dev/en/latest/tutorials/getting-started-postgresql.html),
[using goose migrations as the schema](https://docs.sqlc.dev/en/latest/howto/ddl.html) and
[query annotations](https://docs.sqlc.dev/en/latest/reference/query-annotations.html).

## Coming from C/C++/Java/Python/JS

- Java: goose is Flyway with SQL migrations. sqlc resembles jOOQ's code generation in that
  types come from the schema, but you write plain SQL instead of a query DSL. Unlike
  JPA/Hibernate, nothing writes SQL for you at run time.
- Python: Alembic and Django can generate migration files from models; goose files are
  SQL you write. aiosql (named queries in `.sql` files with `-- name:` comments) is the
  nearest relative of sqlc, minus the generated types.
- JS/TS: Prisma Migrate plus Prisma Client is the closest pair, except that sqlc starts
  from SQL rather than a schema DSL; PgTyped (TypeScript types generated from SQL) is the
  direct analogue.
- C: PostgreSQL's ECPG preprocessor, embedded SQL turned into C calls, is the old form of
  the same idea.

## How it works

This app migrates itself at startup. `server/cmd/stacklab/main.go` embeds the files with
`//go:embed migrations/*.sql` and passes `fs.Sub(embedded, "migrations")`, because goose
only reads the root of the FS it is given. `server/internal/platform/migrate.go` builds a
`*sql.DB` from the pool's connection config (goose speaks database/sql, lesson 1) and
runs:

```go
provider, err := goose.NewProvider(goose.DialectPostgres, sdb, fsys)
// ...
_, err = provider.Up(ctx) // applies every version newer than the database's
```

A minimal `sqlc.yaml` for pgx:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "query.sql"
    schema: "migrations"
    gen:
      go:
        package: "store"
        out: "internal/store"
        sql_package: "pgx/v5"
```

## Design notes

Self-migrating at startup suits a single instance like this app. With several replicas,
either enable goose's session locker or run migrations as their own deploy step, and
write migrations that the previous release can live with: add a column before the code
that uses it ships, drop it only after the code that read it is gone.

sqlc covers the static queries, which is most of them. Keep hand-written pgx (with
`CollectRows`) beside it for genuinely dynamic SQL, and wrap the generated `*Queries` in
your own store type only when you need a boundary, which is the next lesson's question.
