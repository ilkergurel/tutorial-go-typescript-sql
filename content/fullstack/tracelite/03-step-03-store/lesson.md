---
id: fullstack.tracelite.step-03-store
title: "Step 3: The store layer with sqlc, and transactions"
track: fullstack
module: tracelite
order: 3
level: intermediate
minutes: 15
since: go1.27
prerequisites: [fullstack.tracelite.step-02-schema, go.postgres.transactions, go.postgres.migrations-sqlc, go.postgres.repository-testing]
compare_with: [go.design.layers, sql.transactions.row-locks, sql.transactions.isolation-levels, ts.services.openapi]
tags: [tracelite, sqlc, pgx, store, transactions, withtx, uuid, row-locks, testing]
summary: >
  internal/store is sqlc's generated Queries plus one hand-written Store: a WithTx that
  binds the same queries to a transaction, one method per use case that needs one, and
  the translation of SQLSTATEs and constraint names into domain errors.
examples:
  - id: generated
    file: examples/01_generated.go
    title: "The code sqlc generates for one requirement query, run on PostgreSQL"
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "15-23"
        note: >
          sqlc writes DBTX, the three methods its queries call. *pgx.Conn,
          *pgxpool.Pool and pgx.Tx all have them, so New takes any of the
          three, and the generated WithTx (example 2) rebinds existing Queries
          to a transaction.
      - lines: "25-32"
        note: >
          The model mirrors the table, columns in table order. uuid.UUID is the
          standard library's (Go 1.27), through the overrides in sqlc.yaml;
          nullable parent_id becomes a pointer, which encodes as JSON null
          without a custom type. The real table has nine columns, and the
          timestamptz override turns created_at and updated_at into time.Time.
      - lines: "34-38"
        note: >
          The input: named parameters (@name is sqlc.arg(name)) and RETURNING *,
          which sqlc expands into the column list at generation time (line 41),
          so a new column can't shift the Scan targets of an old query. The
          names become the fields of the Params struct.
      - lines: "39-42"
        note: >
          The SQL is kept with its name comment, so pg_stat_activity and the
          server log show which query is running. Parameters are $1..$4 and
          travel separately from the text.
      - lines: "44-49"
        note: >
          More than one parameter means a Params struct; ParentID *uuid.UUID
          sends NULL when nil. pgx encodes uuid.UUID because its underlying type
          is [16]byte and pgx treats a renamed type like its underlying type:
          no adapter, no driver.Valuer.
      - lines: "68"
        note: >
          Here Queries gets a connection; cmd/tracelite hands it the pool, a
          test a transaction. Nothing in the generated code knows which, which is what
          makes WithTx in example 2 possible.
      - lines: "70-71"
        note: >
          RETURNING * comes back as a full model: status is 'draft', the
          column default, and ParentID is nil for a top-level row.
      - lines: "72-73"
        note: >
          Generated code returns pgx's errors untouched: a *pgconn.PgError with
          Code 23505 and ConstraintName requirements_code_key. Turning that
          into a store error is the Store's job (example 2).
  - id: withtx
    file: examples/02_withtx.go
    title: "WithTx: all of a use case or none of it, and the statement that escapes"
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "21"
        note: >
          sqlc's generated WithTx: the same queries, bound to a transaction
          instead of the pool.
      - lines: "31-34"
        note: >
          The Store keeps what can begin a transaction and the generated
          Queries bound to the pool. The real Store takes anything with a Begin
          method (How it works): Begin on a pgx.Tx creates a savepoint, so a
          test can wrap a whole Store in a transaction it rolls back.
      - lines: "36-39"
        note: >
          pgx.BeginFunc is go.postgres.transactions' helper, ready-made: it
          commits when fn returns nil and rolls back when fn returns an error
          (or panics). fn receives Queries bound to the transaction, so every
          statement written with q is part of it.
      - lines: "41-47"
        note: >
          The store's vocabulary. Callers test errors.Is(err,
          store.ErrDuplicateCode) and never see SQLSTATEs or *pgconn.PgError;
          the HTTP layer maps these errors to statuses in one place (step 4).
          The match is on the constraint name, never on the message text.
      - lines: "61-69"
        note: >
          A use case with several writes that must happen together, like step
          7's CSV import. It takes the Queries to write through as a parameter
          only so that the two runs below differ in one word.
      - lines: "71-73"
        note: >
          Through q: the third row fails, fn returns the error, BeginFunc rolls
          back, and the first two rows go with it.
      - lines: "75-77"
        note: >
          The bug: store.q is bound to the pool, so each INSERT commits on its
          own, outside the transaction. The use case reports the same error,
          but two of its three rows stay.
  - id: project-lock
    file: examples/03_project_lock.sql
    title: "Two moves at once: check-then-write races, the project lock serializes"
    runtime: sql-multisession
    setup: |
      CREATE TABLE projects (id uuid PRIMARY KEY DEFAULT uuidv7(), name text NOT NULL);
      CREATE TABLE requirements (
        id uuid PRIMARY KEY DEFAULT uuidv7(), project_id uuid NOT NULL REFERENCES projects,
        parent_id uuid, code text NOT NULL);
      INSERT INTO projects (name) VALUES ('Ödeme');
      INSERT INTO requirements (project_id, code) SELECT id, unnest(ARRAY['AUTH', 'EXPORT']) FROM projects;
    check: exact
    annotations:
      - lines: "5-10"
        note: >
          Step 2's walk up from the new parent, as a function both sessions can
          call. The store's IsAncestorOrSelf is the same walk by id. It starts
          at the new parent itself, so a row moved under itself is caught here
          too, before the requirements_not_own_parent CHECK would be.
      - lines: "15-20"
        note: >
          Both checks run before either move is committed, so both see the old
          tree and both answer false. The two UPDATEs change different rows, so
          neither waits for the other.
      - lines: "21"
        note: >
          AUTH under EXPORT and EXPORT under AUTH: a loop with no root. Every
          row still has a parent in the same project, so no constraint objects,
          and the tree query would lose both requirements.
      - lines: "27-28"
        note: >
          FOR UPDATE OF p locks the project's row, not the requirement: B moves
          a different requirement but needs the same lock, so it waits. Moves in
          other projects don't wait at all. No row at all would mean the
          requirement doesn't exist: ErrNotFound.
      - lines: "31-32"
        note: >
          A's commit releases the lock and B's SELECT returns. Under READ
          COMMITTED each statement sees what was committed before it started,
          so B's check, a statement that starts after the lock was granted,
          sees A's move and answers true: ErrCycle, and B rolls back.
use_cases:
  - "Every query whose shape is fixed at compile time: CRUD, lists, the tree and cycle queries, reports"
  - "Use cases that read and then write: moving a requirement, linking with a check, recording a run and updating a summary"
  - "Translating constraint violations into errors the service and the HTTP layer understand"
  - "Tests against the real schema, with each test isolated by a transaction or by its own project"
strengths:
  - "A misspelled column fails sqlc generate, not a request; the Scan targets can't drift from the SELECT list because both are generated together"
  - "One Queries type works on a pool, a connection and a transaction, so moving code into a transaction changes one line, unlike passing a Tx through every repository method by hand"
  - "Generated code is plain pgx without reflection; a debugger steps through it like your own code"
  - "Unlike Spring's @Transactional, the transaction boundary is a visible function call: the statements inside WithTx are exactly the ones in the closure"
weaknesses:
  - "Dynamic SQL (optional filters, user-chosen sort columns) doesn't fit sqlc's one-query-one-method model; it stays hand-written beside the generated code"
  - "A row lock per project serializes all moves in that project; with heavy concurrent editing, SERIALIZABLE with retries would allow more parallelism"
  - "Error translation by constraint name couples the store to the migration's names; renaming a constraint is an API change for the store"
  - "Generated files must be regenerated and committed with every query or migration change"
pitfalls:
  - title: "Using the pool inside WithTx"
    detail: "A statement sent through s.q or the pool instead of the q passed to fn runs outside the transaction: it can't see the uncommitted rows and isn't rolled back. In example 2, two rows of a failed import stay."
    bad: "s.WithTx(ctx, func(q *Queries) error { return s.q.SetParent(ctx, arg) })"
    good: "s.WithTx(ctx, func(q *Queries) error { return q.SetParent(ctx, arg) })"
  - title: "Check, then act, without a lock"
    detail: "Two moves that each pass the cycle check can together close a loop, because neither sees the other's uncommitted update (example 3, part 1). Lock something both must lock (the project row) before checking, or use SERIALIZABLE and retry."
  - title: "Matching error messages"
    detail: "Message texts are for people and can be translated (lc_messages). Match the SQLSTATE and the constraint name: 23505 on requirements_code_key, 23503 on requirements_parent_fkey, 23001 on test_runs_test_case_fkey."
    bad: "strings.Contains(err.Error(), \"duplicate key\")"
    good: "pgErr.Code == \"23505\" && pgErr.ConstraintName == \"requirements_code_key\""
  - title: "pgtype values leaking out of the store"
    detail: "Without overrides, sqlc's pgx/v5 output uses pgx's own types, such as pgtype.UUID and pgtype.Text, and every caller has to check .Valid. The overrides in sqlc.yaml give uuid.UUID, *uuid.UUID and time.Time instead."
quiz:
  - q: "Why does MoveRequirement lock the project row rather than the requirement being moved?"
    options: ["Locking a requirement row is not allowed", "Two moves of different requirements can close a loop together; only a lock both must take serializes them", "Project rows are smaller"]
    answer: 1
    why: "AUTH under EXPORT and EXPORT under AUTH touch different rows. The shared project row makes the second move wait and then see the first."
  - q: "What does Store.db.Begin do when db is a pgx.Tx?"
    options: ["Fails: transactions can't nest", "Starts a savepoint inside the outer transaction", "Opens a second connection"]
    answer: 1
    why: "pgx implements Begin on a Tx as a pseudo nested transaction, a SAVEPOINT; that's what lets a test wrap the whole Store in a transaction it rolls back."
  - q: "sqlc reads the schema at generation time. What happens if a query names a column that the migrations don't have?"
    options: ["The generated method fails at run time", "sqlc generate fails", "The column is scanned as NULL"]
    answer: 1
    why: "sqlc parses the migrations and the queries together; an unknown column is a generation error, before any code runs."
  - q: "Deleting a test case with runs fails. Which SQLSTATE does the store see?"
    options: ["23503 foreign_key_violation", "23001 restrict_violation", "23505 unique_violation"]
    answer: 1
    why: "test_runs_test_case_fkey is ON DELETE RESTRICT, which raises 23001; NO ACTION would raise 23503. Step 2 shows both."
further_reading:
  - title: "package uuid (Go 1.27)"
    url: "https://pkg.go.dev/uuid"
  - title: "pgx v5: Tx and BeginFunc"
    url: "https://pkg.go.dev/github.com/jackc/pgx/v5#Tx"
  - title: "pgx v5: pgconn.PgError"
    url: "https://pkg.go.dev/github.com/jackc/pgx/v5/pgconn#PgError"
  - title: "sqlc module page (README links the manual: overrides, query annotations)"
    url: "https://pkg.go.dev/github.com/sqlc-dev/sqlc"
  - title: "PostgreSQL docs: Explicit locking, row-level locks"
    url: "https://www.postgresql.org/docs/current/explicit-locking.html#LOCKING-ROWS"
  - title: "PostgreSQL docs: Error codes (SQLSTATE)"
    url: "https://www.postgresql.org/docs/current/errcodes-appendix.html"
---

## Concept

TraceLite's store has two halves in one package, `internal/store`:

- **Generated** by sqlc from `migrations/*.sql` and `queries/*.sql`: the `DBTX` interface,
  `Queries`, one model struct per table and one typed method per named query. Nobody
  edits these files.
- **Written by hand** in `store.go`: `Store`, which owns a `*pgxpool.Pool` and the
  generated `Queries`; `WithTx`, which runs a function against `Queries` bound to one
  transaction; one method per use case that needs more than one statement
  (`MoveRequirement`); and the store's errors.

A use case that is a single statement (create, get, list) needs no transaction of its
own: PostgreSQL runs every statement atomically. A use case that reads and then writes,
or writes several times, gets one `WithTx` call, and every statement inside goes through
the `q` it receives.

## Coming from C/C++/Java/Python/JS

- **Java/Spring Data**: a repository interface whose implementation is generated is
  similar, but Spring generates at run time from method names or JPQL. sqlc generates
  source code from SQL at build time, and `@Transactional`'s proxy becomes an explicit
  `WithTx(ctx, func(q) error)`.
- **Python/SQLAlchemy**: `with session.begin():` is the same boundary. The difference is
  that there is no session tracking objects: nothing is flushed implicitly, every
  statement is a method call you can see.
- **JS/Prisma**: `prisma.$transaction(async (tx) => ...)` has the same shape as `WithTx`;
  forgetting to use `tx` instead of `prisma` inside it is the same bug as using the pool
  inside `WithTx`.
- **C++ RAII**: the deferred `tx.Rollback(ctx)` inside `pgx.BeginFunc` plays the role of a
  destructor that rolls back unless `Commit` ran first.

## How it works

The project files for this step:

```text
sqlc.yaml
migrations/001_init.sql          # step 2, goose format
queries/requirements.sql         # -- name: ... :one / :many / :exec
internal/store/db.go             # generated: DBTX, New, Queries, WithTx
internal/store/models.go         # generated: Project, Requirement, TestCase, TraceLink, TestRun
internal/store/*.sql.go          # generated: one method per query
internal/store/store.go          # hand-written: Store, WithTx, use cases, errors (below)
```

`sqlc.yaml`. The two `uuid` overrides follow the pattern sqlc's "Overriding types" page
shows for `github.com/google/uuid` (one override for non-null columns, one with
`nullable: true`), pointed at Go 1.27's standard `uuid` package instead (it has no
`NullUUID`, so nullable columns become `*uuid.UUID`); the `timestamptz` override replaces pgx's own time type,
and `emit_pointers_for_null_types` turns the other nullable columns (`duration_ms`) into
pointers:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "migrations"
    queries: "queries"
    gen:
      go:
        package: "store"
        out: "internal/store"
        sql_package: "pgx/v5"
        emit_pointers_for_null_types: true
        overrides:
          - db_type: "uuid"
            go_type: "uuid.UUID"
          - db_type: "uuid"
            nullable: true
            go_type:
              import: "uuid"
              type: "UUID"
              pointer: true
          - db_type: "pg_catalog.timestamptz"
            go_type: "time.Time"
```

The store's errors and what produces them. Later steps add to this list rather than
inventing a second mechanism:

- `ErrNotFound`: `pgx.ErrNoRows` from a `:one` query, or a project foreign key
  (`requirements_project_id_fkey`) when the project in the URL doesn't exist.
- `ErrDuplicateCode`: 23505 on `requirements_code_key` or `test_cases_code_key`.
- `ErrParentNotFound`: 23503 on `requirements_parent_fkey` during insert or update.
- `ErrHasChildren`: 23503 on `requirements_parent_fkey` during delete.
- `ErrCycle`: the move check in `MoveRequirement`.
- `ErrHasRuns`: 23001 on `test_runs_test_case_fkey` when a test case is deleted.
- `ErrNotInProject`: 23503 on `trace_links_test_case_fkey` or
  `trace_links_requirement_fkey`, a link across projects.

Times are `timestamptz`; pgx returns them in the process's local zone, and the HTTP layer
converts to UTC before encoding (step 4).

The one use case of this step that needs a transaction is moving a requirement. Its three
queries in `queries/requirements.sql` (example 3 runs the same statements by code, in two
sessions at once):

```sql
-- name: LockProjectOf :one
SELECT p.id FROM projects p JOIN requirements r ON r.project_id = p.id
WHERE r.id = @requirement_id FOR UPDATE OF p;

-- name: IsAncestorOrSelf :one
WITH RECURSIVE up AS (
  SELECT id, parent_id FROM requirements WHERE id = @node::uuid
  UNION ALL SELECT r.id, r.parent_id FROM requirements r JOIN up ON r.id = up.parent_id
) CYCLE id SET is_cycle USING path
SELECT EXISTS (SELECT FROM up WHERE id = @ancestor::uuid);

-- name: SetParent :exec
UPDATE requirements SET parent_id = @parent_id, updated_at = now() WHERE id = @id;
```

And the hand-written half, `store.go`, minus the error variables listed above:

```go
// Store is what the service holds: one transaction per use case, the store's errors out.
type Store struct {
	db interface {
		Begin(context.Context) (pgx.Tx, error)
	} // the pool; a pgx.Tx in tests
	q *Queries // New(pool)
}

// WithTx commits when fn returns nil and rolls back otherwise.
func (s *Store) WithTx(ctx context.Context, fn func(q *Queries) error) error {
	return pgx.BeginFunc(ctx, s.db, func(tx pgx.Tx) error { return fn(s.q.WithTx(tx)) })
}

// MoveRequirement re-parents id under parent; nil makes it top level.
func (s *Store) MoveRequirement(ctx context.Context, id uuid.UUID, parent *uuid.UUID) error {
	return s.WithTx(ctx, func(q *Queries) error {
		if _, err := q.LockProjectOf(ctx, id); errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		} else if err != nil {
			return err
		}
		if parent != nil {
			cycle, err := q.IsAncestorOrSelf(ctx, IsAncestorOrSelfParams{Node: *parent, Ancestor: id})
			if err != nil {
				return err
			}
			if cycle {
				return ErrCycle
			}
		}
		err := q.SetParent(ctx, SetParentParams{ParentID: parent, ID: id})
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.ConstraintName == "requirements_parent_fkey" {
			return ErrParentNotFound // no such row, or one in another project
		}
		return err
	})
}
```

The order is the point. The lock comes first, in its own statement; the check runs after
the lock is granted, so it sees every move committed before; the write comes last. The
parent's existence isn't checked with a SELECT: the composite foreign key decides at the
write, where a concurrent delete can't slip in between, and the store translates its
constraint name.

The `::uuid` casts in `IsAncestorOrSelf` are for sqlc, not PostgreSQL. Inside this
recursive CTE, sqlc 1.31.1 can't resolve `id` to infer a parameter's type and stops with
`column reference "id" is ambiguous`, although PostgreSQL runs the query as is; a cast
gives sqlc the type without the lookup. Generated with sqlc 1.31.1 on 2026-11-14, these
queries give `IsAncestorOrSelfParams{Ancestor, Node uuid.UUID}` returning `(bool, error)`,
`SetParentParams{ParentID *uuid.UUID; ID uuid.UUID}` and
`LockProjectOf(ctx, requirementID uuid.UUID) (uuid.UUID, error)`, all importing the
standard `uuid` package; sqlc accepts the `CYCLE` clause and `FOR UPDATE OF p` as they are.

## Design notes

**Why a hand-written Store at all.** Generated `Queries` would be enough for single
statements, and the service could call them directly. The Store exists for the two
things generation can't give you: a transaction boundary around several queries, and one
place that turns database errors into the store's errors. Store methods that would only
forward to `Queries` are still written, because they are where the translation happens;
what is never written is a second, hand-made copy of a query.

**One transaction per use case, owned by the store.** The alternative is a service that
begins a transaction and passes it down. That leaks pgx types into the service and lets
any caller forget the boundary. TraceLite's rule: a use case that needs atomicity is one
Store method, and the service calls it once (go.design.layers calls this shaping the port
like the use case).

**Locking.** Moves lock the project row with `SELECT ... FOR UPDATE`. That serializes
tree edits per project, which is fine for a tool where a project is edited by a few
people. `SERIALIZABLE` would let independent moves run in parallel, at the price of
retry loops on SQLSTATE 40001; the transactions module covers both.

**Testing.** Store tests run against PostgreSQL with the real migrations, because the
rules that matter (composite keys, CHECKs, collation) live there. Single-statement tests
hand `Queries` a transaction and roll it back; tests of transactional behavior (commits,
concurrent moves) seed their own project and let the private schema be dropped
afterwards. Step 14 builds the full test setup on these two patterns. A Go test of two
concurrent moves can only assert the outcome: without the lock the race window is a few
milliseconds and the transactions still usually run one after the other. Forcing the bad
interleaving of example 3 takes a hook between check and write, as in go.design.layers.
