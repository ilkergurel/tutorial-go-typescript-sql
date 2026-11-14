---
id: go.postgres.bulk
title: "CopyFrom bulk loads, batches, context timeouts and cancellation"
track: go
module: postgres
order: 4
level: advanced
minutes: 15
since: go1.25
prerequisites: [go.postgres.transactions, go.concurrency.context]
compare_with: [sql.files-movement.copy, sql.files-movement.csv-staging, go.services.client]
tags: [postgres, pgx, copy, batch, context, timeouts]
summary: >
  CopyFrom streams rows over the COPY protocol and loads 100k rows in one
  all-or-nothing statement; a Batch sends many statements in one round trip. Give
  every query a deadline, and choose whether it is enforced by closing the
  connection, by a cancel request, or by the server's statement_timeout.
examples:
  - id: copy-from
    file: examples/01_copy_from.go
    title: 100,000 rows with CopyFrom, and a load that fails as a whole
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "27-33"
        note: >
          CopyFrom runs COPY ... FROM STDIN in binary format: one statement, rows
          streamed without a round trip each. pgx's own docs say it can beat
          INSERT with as few as 5 rows. The program prints no timing because it
          varies; the run panel's duration covers the whole program, connect
          and summary query included.
      - lines: "30-32"
        note: >
          CopyFromSlice asks for one row at a time, so nothing holds 100k
          []any at once. CopyFromFunc does the same for sources without a
          length, such as a csv.Reader; CopyFromRows wraps a ready [][]any.
      - lines: "36-40"
        note: >
          A single bad row fails the whole COPY and loads nothing: the row
          100001 before the duplicate is gone too. The aborted rows still take
          disk space until VACUUM. COPY's ON_ERROR ignore doesn't help here: it
          only applies to text and csv input, and pgx sends binary.
      - lines: "42-43"
        note: >
          Under this database's C.UTF-8 locale, ORDER BY city puts İstanbul and
          İzmir after Muğla, because İ (U+0130) sorts by code point. The ICU
          collation tr-x-icu gives the order a Turkish reader expects.
  - id: batch
    file: examples/02_batch.go
    title: A Batch is one round trip and one implicit transaction
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "28-36"
        note: >
          Queue only records the statement; nothing is sent yet. The callback
          sees each result in order. Capturing name is safe because since Go
          1.22 every loop iteration has its own variable.
      - lines: "37-38"
        note: >
          SendBatch sends everything at once, and Close reads every result and
          runs the callbacks. Close must happen before the connection is used
          again, and its error is the batch's error.
      - lines: "40-49"
        note: >
          Without an explicit BEGIN, the batch runs as one implicit transaction:
          the duplicate rolls back the "sql" insert queued before it, and the
          DELETE after it never executes, so its callback never runs.
      - lines: "51-53"
        note: >
          The table shows only batch 1's rows. array_agg needs its own ORDER BY
          for a deterministic order, and pgx scans the text[] straight into a
          []string.
  - id: cancel
    file: examples/03_cancel.go
    title: Three ways a slow query ends, and what each leaves behind
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "24"
        note: >
          The default: when the context expires, pgx sends PostgreSQL a cancel
          request and closes the connection. The error wraps
          context.DeadlineExceeded; a pool simply discards the dead connection,
          a lone *pgx.Conn must reconnect.
      - lines: "25-29"
        note: >
          CancelRequestContextWatcherHandler asks the server to cancel and waits
          for its answer, so the connection survives. The error is then the
          server's 57014, not DeadlineExceeded: check ctx.Err() to know a
          deadline caused it. DeadlineDelay caps how long it waits.
      - lines: "30-32"
        note: >
          statement_timeout is enforced by the server alone, with no client
          context involved; RuntimeParams makes it the session default on every
          connection (on a pool: cfg.ConnConfig.RuntimeParams). The connection
          stays healthy.
      - lines: "46-48"
        note: >
          Call cancel as soon as the work is done, not only on error: a context
          with a timer holds resources until it fires or is cancelled.
use_cases:
  - "CopyFrom: imports, seeding test data, nightly loads from another system, copying between databases"
  - "Batch: many small independent statements per request (several inserts, a read after a write) without paying a round trip each"
  - "Context deadlines: every query in a request handler, bounded by the request's own deadline"
  - "statement_timeout: a server-side ceiling for a role or service, so one bad query can't run for hours"
strengths:
  - "COPY is PostgreSQL's fastest way in, and pgx exposes it with Go values; database/sql has no equivalent without Raw"
  - "A batch removes round trips without string-building a multi-row INSERT, and each statement keeps its own parameters"
  - "Cancellation is a context, the same mechanism as for HTTP calls, so a client disconnect can stop the query it started"
  - "Unlike JDBC's setQueryTimeout (whole seconds, set per statement), one context carries a single deadline through every call of a request, HTTP and SQL alike"
weaknesses:
  - "COPY is all or nothing and stops at the first bad row; reporting which rows were bad needs validation first or a staging table"
  - "CopyFrom needs binary encodings for every column: enum and other custom types must be registered with the connection first"
  - "The default cancellation closes the connection, which costs a reconnect; the polite handler opens an extra connection to deliver the cancel request, which PostgreSQL's protocol makes inherently racy"
pitfalls:
  - title: "Queries without a deadline"
    detail: "context.Background() in a handler lets a slow query outlive the request. Use the request's context (r.Context()) and, for long work, context.WithTimeout on top of it."
  - title: "Using the connection before closing the batch"
    detail: "BatchResults must be closed before the connection runs anything else; until then the connection is busy reading the batch's results."
  - title: "Creating a table and using it in the same batch"
    detail: "With the default exec mode pgx may prepare every queued statement before running any of them, so a statement that refers to a table created earlier in the batch can fail. Run DDL first, then the batch."
  - title: "Building one giant INSERT string"
    detail: "Concatenating VALUES (...) rows risks injection and runs into the 65535 bind-parameter limit if you parameterize it. CopyFrom has neither problem."
quiz:
  - q: "The 60,000th row of a 100,000-row CopyFrom violates a CHECK constraint. How many rows are in the table afterwards?"
    options: ["59,999", "0 of the new rows", "99,999"]
    answer: 1
    why: "COPY is one statement: an error aborts it, and the rows it had already written stay invisible until VACUUM removes them."
  - q: "A batch queues INSERT a, INSERT b (which fails), INSERT c, with no BEGIN. What is committed?"
    options: ["a only", "a and c", "Nothing"]
    answer: 2
    why: "SendBatch runs the queued statements in one implicit transaction unless the batch contains its own transaction control."
  - q: "With pgx's default settings, a query's context deadline expires. What happens to the connection?"
    options: ["It stays open and idle", "pgx sends a cancel request and closes it", "PostgreSQL closes it after statement_timeout"]
    answer: 1
    why: "The default handler interrupts the network read and closes the connection, after asking the server to cancel the query."
  - q: "You want slow queries stopped but connections kept. Which options do that?"
    options: ["Only statement_timeout", "statement_timeout, or CancelRequestContextWatcherHandler", "A shorter context deadline"]
    answer: 1
    why: "Both make the server cancel the statement (57014) and leave the connection usable; a context deadline with the default handler closes it."
further_reading:
  - title: "package pgx: CopyFrom, CopyFromSlice, Batch, SendBatch"
    url: "https://pkg.go.dev/github.com/jackc/pgx/v5#Conn.CopyFrom"
  - title: "package pgconn: CancelRequestContextWatcherHandler"
    url: "https://pkg.go.dev/github.com/jackc/pgx/v5/pgconn#CancelRequestContextWatcherHandler"
  - title: "PostgreSQL: COPY"
    url: "https://www.postgresql.org/docs/current/sql-copy.html"
  - title: "PostgreSQL: Populating a database (bulk loading)"
    url: "https://www.postgresql.org/docs/current/populate.html"
  - title: "PostgreSQL: statement_timeout and other client settings"
    url: "https://www.postgresql.org/docs/current/runtime-config-client.html"
---

## Concept

Three tools for doing a lot, or for giving up on time.

**CopyFrom** uses PostgreSQL's COPY protocol: one `COPY ... FROM STDIN` statement, then
the rows streamed in binary. No per-row round trip, no per-row statement. You hand it a
`CopyFromSource`: `CopyFromRows` for a `[][]any`, `CopyFromSlice(n, fn)` to produce row
`i` on demand, `CopyFromFunc` for a stream of unknown length. Being one statement, it is
all or nothing.

**Batch** queues ordinary statements, each with its own parameters, and `SendBatch` sends
them together. Results come back in order, to callbacks or to `BatchResults`. Without its
own `BEGIN`, the batch runs as one implicit transaction.

**Cancellation** is a context. Every pgx call takes one; when it expires, what happens
depends on the connection's context-watcher handler: by default pgx asks the server to
cancel and closes the connection; `CancelRequestContextWatcherHandler` asks and keeps it.
Independently, the server's `statement_timeout` cancels any statement running longer than
the limit, and the connection stays usable either way.

## Coming from C/C++/Java/Python/JS

- Java: CopyFrom is the PgJDBC `CopyManager` with typed values instead of CSV text; a
  Batch is closer to pipelining than to `addBatch`, because each statement can return rows.
  A context replaces `Statement.setQueryTimeout` and `cancel()` with one mechanism.
- Python: psycopg 3's `cursor.copy()` and `executemany` are the equivalents; asyncio's
  `wait_for` around a query plays the context's role.
- Node: `pg-copy-streams` adds COPY to node-postgres; an `AbortSignal` passed down the call
  chain is the closest thing to a context.
- C: libpq's `PQputCopyData` and pipeline mode are the same protocol features, which pgx
  implements natively rather than wrapping libpq; `PQcancel` is the cancel request.

## Design notes

Put deadlines where the work starts: an HTTP handler passes `r.Context()` down, so a
client that disconnects cancels its queries; background jobs set `context.WithTimeout`.
Add `statement_timeout` as a server-side backstop for the whole service, set on the
pool's `ConnConfig.RuntimeParams` or with `ALTER ROLE ... SET statement_timeout`, so a
forgotten context can't run a query for hours. `SET LOCAL statement_timeout` inside a
transaction raises or lowers it for one unit of work.

For imports, validate in Go before CopyFrom, or COPY into a staging table without
constraints and move the good rows with one `INSERT ... SELECT`; the
`sql.files-movement` module shows the staging pattern. Measure with your own data and
indexes: every index on the target table is maintained row by row during the load, which
is why PostgreSQL's bulk-loading advice suggests creating indexes after a large initial
load.
