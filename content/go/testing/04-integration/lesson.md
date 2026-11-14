---
id: go.testing.integration
title: "Build-tagged integration tests on a real Postgres"
track: go
module: testing
order: 4
level: advanced
minutes: 13
since: go1.25
prerequisites: [go.testing.table-driven, go.postgres.repository-testing]
compare_with: [sql.transactions.skip-locked, go.testing.fakes-httptest, ts.testing.msw-playwright]
tags: [testing, integration-tests, build-tags, postgres, pgx, testmain]
summary: >
  Put database tests in files tagged //go:build integration, so go test ./...
  stays fast and needs nothing, and go test -tags integration -count=1 ./... runs
  the honest suite against a real PostgreSQL, set up once in TestMain.
examples:
  - id: build-tags
    file: examples/01_build_tags.go
    title: "What -short and -tags integration change, run for real"
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "15-26"
        note: >
          The unit test file has no constraint and is always compiled.
          testing.Short() is the other gate: true under go test -short, for
          tests that are slow but need nothing external.
      - lines: "27-40"
        note: >
          //go:build integration must come before the package clause, followed
          by a blank line. Without -tags integration the file isn't compiled
          at all: its tests don't run, and its compile errors go unnoticed.
      - lines: "36-40"
        note: >
          The tag says "I asked for the database tests"; the variable says
          where the database is. Skipping when it is unset is this app's
          policy. A CI job that sets the tag should fail instead, so a missing
          database can't pass as green.
      - lines: "55-56"
        note: >
          Two runs of the same module. With -short, TestReport skips and the
          integration file is absent. With -tags=integration, TestUsersOnPostgres
          exists and skips because TEST_DATABASE_URL is empty. A plain go test
          would run TestParse and TestReport only.
      - lines: "63"
        note: >
          -count=1 bypasses the test cache. go test can reuse a passing result
          when the binary, files and environment variables it read are
          unchanged; it can't see that the database changed.
      - lines: "66"
        note: >
          TEST_DATABASE_URL= empties the variable for the child, whatever the
          parent environment holds: when a key repeats in Cmd.Env, the last
          value wins.
  - id: queue
    file: examples/02_queue_test.go
    title: "A test only a real database can answer: concurrent workers and SKIP LOCKED"
    runtime: go-test
    check: contains
    annotations:
      - lines: "1"
        note: >
          A single-file runtime can't pass -tags, so this file has no
          constraint. Written for real, it would begin with //go:build
          integration and a blank line, and read TEST_DATABASE_URL instead of
          the DATABASE_URL this app's runner provides.
      - lines: "17-24"
        note: >
          The code under test commits: each claim is one statement, so its own
          transaction. That rules out the rollback-per-test technique, and the
          guarantee lives in PostgreSQL's row locks, which no fake reproduces.
      - lines: "28-42"
        note: >
          TestMain connects and migrates once per test binary instead of once
          per test. No database configured: pool stays nil and every test skips
          itself. TestMain may return; the generated main exits with m.Run's
          result, after the deferred Close.
      - lines: "45-51"
        note: >
          Committed data needs explicit isolation. Truncating before each test,
          not after, keeps a failed test's rows for inspection, and RESTART
          IDENTITY makes ids start at 1 again. With many tests, put the skip
          and the TRUNCATE in one helper that every test calls first.
      - lines: "55-70"
        note: >
          Four goroutines, each running its own statements on its own pool
          connection, race for 50 jobs. Each appends only to its own slice, so
          no mutex is needed. wg.Go (Go 1.25) does the Add and Done. t.Error is
          safe from these goroutines; t.Fatal is not.
      - lines: "72-77"
        note: >
          The property: 50 claims, 50 distinct ids. Remove FOR UPDATE SKIP
          LOCKED and two workers can pick the same row: the second waits for
          the first one's lock and then updates the row again. That only shows
          up under real concurrency on a real server.
use_cases:
  - "Repositories and queries whose correctness lives in the schema: constraints, unique indexes, collations, NULL handling"
  - "Concurrency that relies on the database: SKIP LOCKED queues, advisory locks, serializable retries, upserts under contention"
  - "Migrations: applying the full chain to an empty database in TestMain is itself a test"
  - "A fast default suite (go test ./...) for every save, and a DB suite (-tags integration) before pushing and in CI"
strengths:
  - "The gate is the compiler: tagged files cost nothing in the default run, unlike JUnit categories or pytest markers, which still load and collect the tests"
  - "TestMain gives one place for setup and teardown per test binary, without a framework's lifecycle annotations"
  - "Tests run against the same PostgreSQL version and extensions as production, where an in-memory substitute such as H2 or SQLite in Java and Python stacks differs in SQL dialect and locking"
  - "Plain env vars and flags (TEST_DATABASE_URL, -tags, -count=1) work the same locally, in CI and in a container"
weaknesses:
  - "Tagged files are invisible to go build, go vet and gopls unless they get the tag too, so they can stop compiling without anyone noticing"
  - "go test ./... runs package test binaries in parallel (-p defaults to GOMAXPROCS); packages sharing one database need separate schemas or -p 1"
  - "Committed test data needs explicit cleanup or isolation, which costs time and code that the rollback-per-test technique avoids"
  - "Needs a running PostgreSQL: a service in CI, a local install or a container, plus migrations"
pitfalls:
  - title: "Pointing tests at DATABASE_URL"
    detail: "Tests that TRUNCATE tables must never pick up the development or production connection string by accident. Read a variable only tests use, such as TEST_DATABASE_URL, as this app does."
  - title: "Letting the tagged files rot"
    detail: "go vet ./... and the editor skip files behind the tag. Run go vet -tags integration ./... in CI (it needs no database), and set gopls's buildFlags to [\"-tags=integration\"]."
  - title: "Trusting a cached result"
    detail: "A database test that passed is cached as long as its binary, files and environment are unchanged, even if the schema changed underneath. Run integration tests with -count=1."
  - title: "Skipping silently in CI"
    detail: "If TEST_DATABASE_URL is missing in CI, every database test skips and the job is green. Fail when the tag is set and the variable isn't, or check the skip count."
    bad: "if url == \"\" { t.Skip(\"TEST_DATABASE_URL not set\") }"
    good: "if url == \"\" {\n\tif os.Getenv(\"CI\") != \"\" {\n\t\tt.Fatal(\"TEST_DATABASE_URL not set in CI\")\n\t}\n\tt.Skip(\"TEST_DATABASE_URL not set\")\n}"
  - title: "Old-style constraints"
    detail: "Code from before Go 1.17 has // +build integration lines. gofmt adds the equivalent //go:build line; the //go:build form is the one to write."
quiz:
  - q: "A file starts with //go:build integration. What does plain go test ./... do with it?"
    options: ["Doesn't compile it, so its tests don't exist in that run", "Runs its tests and skips them at runtime", "Fails the build"]
    answer: 0
    why: "Build constraints exclude the file from the package unless the tag is set with -tags integration."
  - q: "Why add -count=1 when running the integration suite?"
    options: ["To run each test once instead of twice", "To bypass the test cache, which can't know the database changed", "To disable parallel tests"]
    answer: 1
    why: "go test caches passing results keyed on the binary, flags, files read and env vars; external state such as a database isn't part of the key."
  - q: "The code under test commits its own transactions. How do you isolate tests?"
    options: ["Wrap each test in a transaction and roll back", "Run go test -short", "Reset the tables (TRUNCATE) or use a schema per test"]
    answer: 2
    why: "A rollback can't undo what the code already committed in its own transactions; committed data has to be cleared or kept in separate schemas."
  - q: "Where should one-time setup such as connecting and migrating go?"
    options: ["In init()", "In TestMain, before m.Run()", "At the start of every test"]
    answer: 1
    why: "TestMain runs once per test binary, wraps all tests, can stop with a clear error before any test runs and can clean up after m.Run; init has no place for teardown."
further_reading:
  - title: "go command: build constraints"
    url: "https://pkg.go.dev/cmd/go#hdr-Build_constraints"
  - title: "go command: testing flags (-count, -short, test caching)"
    url: "https://pkg.go.dev/cmd/go#hdr-Testing_flags"
  - title: "package testing: Main, Short, Skip"
    url: "https://pkg.go.dev/testing#hdr-Main"
  - title: "gopls settings: buildFlags"
    url: "https://go.dev/gopls/settings#buildflags-string"
  - title: "PostgreSQL: SELECT, The Locking Clause (SKIP LOCKED)"
    url: "https://www.postgresql.org/docs/current/sql-select.html#SQL-FOR-UPDATE-SHARE"
---

## Concept

Split the test suite in two along what it needs:

- **Unit tests** need nothing outside the process. They run on every `go test ./...`,
  in seconds, on any machine.
- **Integration tests** need a real PostgreSQL. They live in files whose first line is
  `//go:build integration`, so they are compiled only with
  `go test -tags integration ./...`.

Inside the tagged files, a second gate says where the database is: an environment
variable such as `TEST_DATABASE_URL`, with tests skipping (locally) or failing (in CI)
when it is missing. Expensive setup happens once per package in `TestMain`: connect,
run the migrations, call `m.Run()`, close. Each test then gets a clean state, by a
rolled-back transaction when the code under test accepts one, or by resetting tables
or using its own schema when the code commits.

A third, lighter gate is `testing.Short()`: true under `go test -short`, for tests that
are slow but need nothing external.

## Coming from C/C++/Java/Python/JS

- **Java**: Maven runs `*IT.java` with Failsafe in a later phase, or JUnit 5 `@Tag`s
  filter at runtime. A Go build tag works at compile time: the tests don't exist in the
  default build. Testcontainers starts a database per run; in Go you pass a URL to one
  you already run, or start a container in `TestMain`.
- **Python**: `@pytest.mark.integration` with `-m integration`, and a session-scoped
  fixture for the connection, is the same split; `TestMain` is the session fixture.
- **JS/TS**: separate Vitest or Jest configs or projects for integration tests, often
  with `globalSetup` doing what `TestMain` does here.
- **C/C++**: like a CMake option or `#ifdef` that compiles a test target only when
  asked for.

## How it works

The go command decides which files belong to a package before compiling. A file whose
`//go:build` expression is false for this build is left out entirely, so tags work for
any condition: `integration`, `linux`, `go1.27`. `-tags a,b` adds tags. The constraint
must precede the package clause and be followed by a blank line; one `//go:build` line
per file.

This app does exactly this. `server/internal/sqlrun/sqlrun_test.go`,
`multisession_test.go` and `server/internal/runexec/runexec_integration_test.go` carry
`//go:build integration` above the package clause and skip unless `TEST_DATABASE_URL` is
set, while `split_test.go` beside them has no tag:

```
cd server
go test ./...                                    # unit tests only
# the playground database from .env.example:
$env:TEST_DATABASE_URL = "postgres://playground:playground@127.0.0.1:5432/playground"
go test -tags integration -count=1 ./...         # plus the PostgreSQL suite
```

(`$env:` is PowerShell; in bash it is `TEST_DATABASE_URL=... go test ...`.) The
`Makefile` also runs `go test -tags integration ./internal/gorun/`, where the tag guards
tests that compile and run real programs and so need a Go toolchain rather than a
database.

The lesson runtime has no flags, so example 2 has no tag. It reads `DATABASE_URL`, which
the runner sets to the playground database with an empty schema of its own, dropped
after the run. That is also why the example can commit freely.

## Design notes

Keep the default `go test ./...` hermetic and fast; it is what runs on every save. Make
the integration suite easy to run on demand (one documented command, one environment
variable) and mandatory in CI, where a missing database must fail the job rather than
skip it. Run `go vet -tags integration ./...` in the fast pipeline too: it compiles the
tagged files without needing a database.

Choose isolation by what the code under test does. If it takes a `DBTX`-style interface,
give each test a transaction and roll it back (go.postgres.repository-testing). If it
commits, as a job queue or anything with its own transactions must, reset tables before
each test, or give each test (or each package, when `go test ./...` runs packages in
parallel) its own schema and set `search_path`. A test schema per package also stops
two packages from truncating each other's tables mid-run.

Test the behaviour only the database can give you: constraints, collations (a unique
index on `lower(name)` treats `IŞIL` and `ışıl` as different users in a `C.UTF-8`
database), locks, isolation levels, `SKIP LOCKED`. Everything above that line is
better served by fakes and fast unit tests.
