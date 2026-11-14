---
id: go.postgres.repository-testing
title: "The repository pattern (and when it's overkill); testing DB code on real Postgres"
track: go
module: postgres
order: 6
level: advanced
minutes: 14
since: go1.26
prerequisites: [go.postgres.migrations-sqlc, go.interfaces.implicit]
compare_with: [go.testing.integration, go.testing.fakes-httptest, go.design.layers, design.patterns.anti-patterns]
tags: [postgres, pgx, repository, interfaces, testing, integration-tests]
summary: >
  A repository is a concrete type that owns its SQL and speaks domain values and
  errors; an interface appears only where a consumer needs a substitute. The rules
  that matter live in the database, so test repositories on a real PostgreSQL, one
  rolled-back transaction per test.
examples:
  - id: fake-vs-real
    file: examples/01_fake_vs_real.go
    title: The same service over a fake and over PostgreSQL
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "16-19"
        note: >
          The consumer declares the interface, with only the method it calls.
          The repository doesn't name it and doesn't need to: Go interfaces are
          satisfied implicitly.
      - lines: "28-38"
        note: >
          The repository owns the SQL and translates driver errors at the
          boundary: callers see ErrEmailTaken, never SQLSTATE 23505 or a
          *pgconn.PgError. It is a concrete struct; nothing here needs an
          interface of its own.
      - lines: "40-49"
        note: >
          A fake encodes what you believe the database does: here, exact-match
          uniqueness. It is fast and fine for testing signup's own logic, but it
          can't tell you whether the belief is true.
      - lines: "58-60"
        note: >
          The real rule: a unique index on the expression lower(email). This is
          behaviour of your system that lives only in the schema, along with
          CHECK constraints, NULL semantics, collations and triggers.
      - lines: "65-69"
        note: >
          Same service, two stores, two answers. A test suite built on the fake
          stays green while production returns "email already registered", which
          is why the repository itself must be tested against PostgreSQL.
  - id: users-test
    file: examples/02_users_test.go
    title: A table-driven test on real PostgreSQL, one rolled-back transaction per case
    runtime: go-test
    check: contains
    timeout_s: 15
    annotations:
      - lines: "17-20"
        note: >
          The repository asks for the one method it uses. *pgxpool.Pool,
          *pgx.Conn and pgx.Tx all have it, which is what lets a test hand it a
          transaction instead of the pool.
      - lines: "33-47"
        note: >
          Connect and migrate once per test binary. Without DATABASE_URL the
          tests skip instead of failing, so go test ./... still works on a
          machine without a database. TestMain may return; the wrapper exits
          with m.Run's result.
      - lines: "54-58"
        note: >
          Every test runs inside its own transaction, and Cleanup rolls it back:
          no test sees another's rows and nothing needs deleting. t.Context()
          (Go 1.24) is canceled just before cleanups run, so the rollback gets
          context.Background().
      - lines: "63-68"
        note: >
          Table-driven: each case names the situation and the error it expects.
          The second case is the one the fake in the first example got wrong;
          the unique index answers it.
      - lines: "75"
        note: >
          After the expected 23505 the transaction is aborted (lesson 3). That is
          fine because the case ends here; a test that keeps going after an
          expected error needs a savepoint.
use_cases:
  - "A concrete repository (or sqlc's Queries) called directly: CRUD services with one database and little logic, which is most of them"
  - "A consumer-declared interface plus a fake: business logic with real branching that should be unit-tested in microseconds"
  - "Interfaces with two real implementations, like this app's httpapi.Runner (in-process runner and Docker runner client)"
  - "Integration tests on real PostgreSQL: every repository, every query with constraints, NULLs or collations involved"
strengths:
  - "Unlike Java's interface-per-repository habit, Go's implicit interfaces let the consumer define the seam later, without touching the repository"
  - "Transactional DDL and cheap transactions make rolled-back tests fast enough to run on every change against a local PostgreSQL"
  - "Error translation in the repository keeps driver types out of services and handlers, so storage can change behind a stable API"
  - "Tests against the real database pin down behaviour no fake reproduces: expression indexes, constraint names, collations, NULL comparisons"
weaknesses:
  - "Integration tests need a running PostgreSQL and a migrated schema; CI must provide one"
  - "A transaction per test can't cover code that must commit or that needs two sessions (lock waits, LISTEN/NOTIFY); those need a schema per test or cleanup"
  - "Interfaces added too early hide which implementation runs and multiply files for no gain"
pitfalls:
  - title: "Mocking the driver"
    detail: "A mocked pgx asserts the SQL text you expected to send, not what PostgreSQL does with it, so it passes while a constraint, a typo in a column or a NULL breaks production. Test repositories against PostgreSQL."
  - title: "An interface per struct, by reflex"
    detail: "A UserRepository interface beside its only implementation, declared by the producer, mirrors every method and protects nothing. Let each consumer declare the few methods it needs, when it needs a substitute."
  - title: "Rolling back with t.Context() in Cleanup"
    detail: "t.Context() is already canceled when cleanup functions run, so the ROLLBACK can't be sent and pgx closes the connection instead. Use context.Background() there."
  - title: "Parallel tests that insert the same unique value"
    detail: "Uncommitted rows are invisible to other transactions, but a unique index still makes the second insert wait until the first transaction ends. Give parallel tests distinct values."
  - title: "Tests that depend on committed data"
    detail: "Seeding shared rows that every test reads couples the tests to their order. Create what a test needs inside its own transaction."
quiz:
  - q: "Your fake user store passes every test, but production rejects \"Ayse@Example.com\" after \"ayse@example.com\". Why?"
    options: ["The fake has a bug in its map", "The unique index on lower(email) is a database rule the fake never knew", "pgx lowercases parameters"]
    answer: 1
    why: "A fake encodes your assumptions; the schema holds the actual rule, and only a test on PostgreSQL checks it."
  - q: "A service has one storage implementation (sqlc over PostgreSQL) and almost no logic. Does it need a UserRepository interface?"
    options: ["Yes, always program to an interface", "No: call the concrete store or Queries; add a small interface when a consumer needs a substitute", "Only if it uses transactions"]
    answer: 1
    why: "An interface earns its place with a second implementation or logic worth testing in isolation; otherwise it is indirection."
  - q: "Which context should the rollback in t.Cleanup use?"
    options: ["t.Context()", "context.Background()", "The context passed to Begin"]
    answer: 1
    why: "t.Context() is canceled just before cleanups run, so a rollback using it can't reach the server."
  - q: "Code under test calls WithTx(ctx, db, fn), and the test passes its own pgx.Tx as db. What happens?"
    options: ["It fails: you can't begin inside a transaction", "Begin on a Tx creates a savepoint; the code's commit releases it, and the test's rollback still undoes everything", "The inner commit makes the data permanent"]
    answer: 1
    why: "pgx.Tx.Begin starts a pseudo nested transaction backed by a savepoint, so the outer rollback wins."
further_reading:
  - title: "package testing: TestMain, T.Cleanup, T.Context, T.Skip"
    url: "https://pkg.go.dev/testing"
  - title: "Go wiki: Table-driven tests"
    url: "https://go.dev/wiki/TableDrivenTests"
  - title: "go command: build constraints (for -tags integration)"
    url: "https://pkg.go.dev/cmd/go#hdr-Build_constraints"
  - title: "package pgx: Tx and pseudo nested transactions"
    url: "https://pkg.go.dev/github.com/jackc/pgx/v5#Tx"
  - title: "PostgreSQL: Indexes on expressions"
    url: "https://www.postgresql.org/docs/current/indexes-expressional.html"
---

## Concept

A **repository** is the type that owns the SQL for one part of the domain. It takes
domain values in, returns domain values and domain errors out (`User`, `ErrEmailTaken`),
and nothing above it imports pgx. In Go it is a plain struct with a constructor, often a
thin layer over sqlc's `Queries`.

Whether it also needs an **interface** is a separate question, answered by its consumers.
Add one when a consumer needs a substitute: business logic you want to unit-test with a
fake, or a second real implementation. The consumer declares it, with only the methods it
calls, and the repository satisfies it without knowing. Without such a consumer, an
interface is ceremony: one implementation, a mirror of its methods, and one more hop when
reading the code. That is when the pattern is overkill, and with sqlc's typed methods it
often is.

**Testing.** A repository's behaviour is half Go and half schema: unique indexes,
constraints, NULL handling, collations. Only a real PostgreSQL tests the second half, so
test repositories against one. The standard shape: connect and migrate once, give each
test a transaction, roll it back in `t.Cleanup`, and skip when no database is configured.

## Coming from C/C++/Java/Python/JS

- Java: Spring Data generates an implementation behind `interface UserRepository`; in Go
  you write the struct, and interfaces come from consumers. Testcontainers plus a
  `@Transactional` test (rolled back by default in Spring tests) is the same
  real-database, rolled-back-transaction technique.
- Python: Django's `TestCase` wraps each test in a transaction and rolls it back; pytest
  fixtures with teardown play `t.Cleanup`'s role.
- JS/TS: mocking the database client in Jest tests the mock. The real-database approach
  is the same as here: a test database and a transaction or truncate per test.
- C: swapping in a fake object file at link time is the C version of a fake, with the
  same blind spot.

## Design notes

This app shows both sides. `httpapi.Runner` is an interface declared by its consumer,
the HTTP layer, because two real implementations exist: `runexec.Executor` runs lessons
in-process, `runclient.Client` forwards them to the Docker runner. The stores
(`*learningstore.Store`, `*runstore.Store`) are concrete and used directly, because there
is one database and nothing to swap.

Its database tests follow the integration pattern: `sqlrun_test.go` and
`multisession_test.go` in `server/internal/sqlrun` carry `//go:build integration` and read
`TEST_DATABASE_URL`, skipping when it is unset, while the unit tests in `split_test.go`
beside them have no tag. `go test ./...` stays fast, and `go test -tags integration ./...` runs the honest
suite against the local playground database. This lesson's runtime passes
`DATABASE_URL` instead and has no tags, which is why the example skips on the variable
alone.

A transaction per test covers code that accepts a `DBTX`-style interface. Code that
commits on its own can still run inside it if it begins from something you pass in:
`Begin` on a `pgx.Tx` makes a savepoint. Code that needs two sessions, waits on locks or
uses LISTEN needs committed data instead: a schema per test (what this runner does for
every run) or truncating tables between tests.

Case-insensitive rules carry a locale: `lower()` in this C.UTF-8 database maps `I` to
`i`, so a unique index on `lower(username)` treats `IŞIL` and `ışıl` as different users,
while `lower(username COLLATE "tr-x-icu")` would not. Behaviour like that is exactly what
a test on the real database pins down.
