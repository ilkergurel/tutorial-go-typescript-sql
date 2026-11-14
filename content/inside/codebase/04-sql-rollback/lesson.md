---
id: inside.codebase.sql-rollback
title: "The SQL runner's transaction-and-rollback technique"
track: inside
module: codebase
order: 4
level: advanced
minutes: 14
since: pg13
prerequisites: [sql.transactions.acid-mvcc, go.postgres.pgx]
compare_with: [go.postgres.transactions, sql.security-ops.roles, inside.codebase.runner-sandbox, sql.transactions.vacuum]
tags: [transactions, rollback, transactional-ddl, schemas, search_path, sandbox, pgx]
summary: >
  For the sql runtime, internal/sqlrun sends BEGIN, CREATE SCHEMA and SET LOCAL as
  one message, runs your statements one by one, and always ends with ROLLBACK, so
  even your CREATE TABLE never existed. A guard refuses COMMIT and ROLLBACK so your
  script can't end the sandbox early.
examples:
  - id: inside-the-sandbox
    file: examples/01_inside_the_sandbox.sql
    title: "Where your script actually runs"
    runtime: sql
    check: exact
    annotations:
      - lines: "1-5"
        note: >
          `sqlrun.Run` sends this as one simple-protocol message through its
          `quiet` helper, so none of it appears in your output. `sandboxSQL`
          builds the SET lines; scope LOCAL means they end with the transaction.
      - lines: "8-10"
        note: >
          The name is `run_<unix nanos>_<8 hex>`: the time lets the janitor age
          leftovers, the random part keeps two runs in the same clock tick
          apart (Windows' wall clock is coarse). current_setting shows the SET
          LOCAL values; 5000 ms displays as 5s.
      - lines: "13-14"
        note: >
          An xid exists because CREATE SCHEMA wrote catalog rows.
          transaction_timestamp() is fixed at BEGIN; statement_timestamp() is
          this statement's arrival. They differ because the runner's BEGIN was
          an earlier message.
      - lines: "17-22"
        note: >
          search_path resolves unqualified names, so the table lands in the
          run's schema. Nothing here names the schema, which is what lets one
          example file run unchanged on every run.
      - lines: "25"
        note: >
          PostgreSQL answers a nested BEGIN with a WARNING and keeps the one
          transaction going. Notices print under their statement in the text
          output. The dangerous counterpart, your own ROLLBACK or COMMIT, is
          refused: example 3.
  - id: invisible-to-others
    file: examples/02_invisible_to_others.sql
    title: "Uncommitted DDL is invisible, and its names are reserved until it ends"
    runtime: sql-multisession
    check: exact
    annotations:
      - lines: "2-4"
        note: >
          Session A plays sqlrun.Run: a transaction that creates a table and
          rows. This runtime itself sandboxes differently: its run_ schema is
          committed and dropped afterwards, because two sessions must share it.
      - lines: "7"
        note: >
          B uses the same schema, yet the table doesn't exist for it: catalog
          rows are MVCC rows like any other, and A's aren't committed.
          `to_regclass` returns NULL where `'scratch'::regclass` would raise.
      - lines: "10-11"
        note: >
          B's CREATE TABLE has to insert the same (name, schema) key into the
          system catalogs' unique indexes, where A's uncommitted entry already
          sits. B waits for A's outcome. A rolls back, so B goes ahead; had A
          committed, B would have failed with a duplicate-key error.
      - lines: "14"
        note: >
          A sees B's new, empty table: A's two rows were never visible to
          anyone, and after ROLLBACK they are dead tuples waiting for VACUUM.
  - id: your-rollback
    file: examples/03_your_rollback_is_refused.sql
    title: "Your own ROLLBACK is refused; savepoints still work"
    runtime: sql
    check: exact
    annotations:
      - lines: "6-9"
        note: >
          ROLLBACK TO SAVEPOINT undoes the DROP and stays inside the one
          transaction. Its command tag is also ROLLBACK, which is why the guard
          reads the statement's leading keywords, not the tag the server sends
          back.
      - lines: "12"
        note: >
          `refuseTxControl` (internal/sqlrun/guard.go) matches the first
          keyword after any comments and refuses the statement before it is
          sent. The result carries the reason, a DETAIL and a HINT, and
          `runPhase` stops the phase as it does for any error.
      - lines: "13"
        note: >
          Never sent. Before the guard, the ROLLBACK above ended the runner's
          transaction, and this statement ran in autocommit mode in public,
          without timeouts: anything it created would have persisted.
use_cases:
  - "Running untrusted or exploratory SQL against a shared database without leaving anything behind"
  - "Test isolation: one transaction per test, rolled back at the end, instead of truncating tables"
  - "Per-run schemas when a transaction is not possible (VACUUM, CREATE INDEX CONCURRENTLY, two sessions): create, use, DROP SCHEMA ... CASCADE"
strengths:
  - "PostgreSQL's DDL is transactional: CREATE TABLE, ALTER and CREATE SCHEMA roll back with the data, unlike MySQL or Oracle, where DDL commits implicitly"
  - "Cleanup can't be forgotten: there is nothing to delete, and if the backend dies mid-run the server rolls back when the connection drops"
  - "No naming collisions between runs: each one resolves unqualified names into its own schema through search_path"
  - "Statement-at-a-time execution on the simple protocol attributes every result, notice and error to its statement, and stops at the first error, as psql with ON_ERROR_STOP does"
weaknesses:
  - "The sandbox is the transaction, so a `sql` example can't show transaction control: COMMIT and ROLLBACK are refused, and lessons about transactions use `sql-nontx` or `sql-multisession`"
  - "The guard is a keyword list (COMMIT, END, ABORT, ROLLBACK without TO, PREPARE TRANSACTION). The transaction-status check behind it catches what the list misses, but only after that statement has run"
  - "Rolled-back is not free: every run writes catalog rows that become dead tuples, and burns a transaction ID. Harmless at one user's pace, worth knowing at scale"
  - "In Docker mode the playground role is a superuser (the postgres image creates `POSTGRES_USER` as one), so lesson SQL could write server files with COPY, which no rollback undoes"
pitfalls:
  - title: "COMMIT or ROLLBACK in a `sql` runtime example"
    detail: "It would end the runner's transaction, so sqlrun refuses it and stops the run. Use `sql-nontx` for scripts with their own transaction control, and `SAVEPOINT` / `ROLLBACK TO` inside a `sql` example."
    bad: "-- runtime: sql\nBEGIN;\nINSERT INTO t VALUES (1);\nROLLBACK; -- refused: the run stops here"
    good: "-- runtime: sql\nSAVEPOINT s;\nINSERT INTO t VALUES (1);\nROLLBACK TO SAVEPOINT s;"
  - title: "Expecting statements after an error to run"
    detail: "The first error aborts the transaction, so sqlrun stops the phase there. A later statement could only fail with 25P02 (current transaction is aborted)."
  - title: "Plain SET instead of SET LOCAL inside the sandbox"
    detail: "Inside a transaction both are undone by ROLLBACK, but SET outlives a COMMIT and stays for the session. SET LOCAL is the one that is scoped to the transaction by definition."
quiz:
  - q: "Why can the sql runtime roll back a script that creates tables and a schema?"
    options:
      - "PostgreSQL's DDL is transactional: catalog changes commit or abort with the transaction"
      - "sqlrun records every CREATE and drops it afterwards"
      - "The playground database is recreated after every run"
    answer: 0
    why: "Catalog rows are MVCC rows. Example 2 shows another session can't see uncommitted DDL, and ROLLBACK makes it never have existed."
  - q: "A `sql` example ends with `COMMIT; CREATE TABLE t (id int);`. What happens?"
    options:
      - "Both run, and t is rolled back with everything else"
      - "The COMMIT is refused before it is sent, the run stops there, and nothing persists"
      - "t lands in public permanently"
    answer: 1
    why: "sqlrun's guard refuses statements that would end the sandbox transaction. Without it, the COMMIT would have committed the run_ schema, ended SET LOCAL, and let t autocommit in public."
  - q: "In example 2, why does B's CREATE TABLE wait instead of succeeding or failing at once?"
    options:
      - "B holds a table lock A needs"
      - "A's uncommitted catalog entry occupies the same unique-index key, and whether it's a duplicate depends on A's outcome"
      - "lock_timeout delays every DDL statement"
    answer: 1
    why: "A unique-index insert that meets an in-progress entry waits for that transaction. Rollback frees the key, commit makes it a duplicate-key error."
  - q: "Why does sqlrun open its own connection per run instead of using a pool?"
    options:
      - "pgx pools don't support transactions"
      - "Pools are slower for short scripts"
      - "Transaction state and SET LOCAL live on a connection; a dedicated one guarantees the run sees no one else's state, and closing it rolls back whatever is left"
    answer: 2
    why: "Session.Close's comment says it: any open transaction is rolled back by the server. A pooled connection would need resetting before reuse."
further_reading:
  - title: "PostgreSQL: SET (SET LOCAL scope)"
    url: "https://www.postgresql.org/docs/current/sql-set.html"
  - title: "PostgreSQL: Schemas and the search path"
    url: "https://www.postgresql.org/docs/current/ddl-schemas.html"
  - title: "PostgreSQL: Concurrency control (MVCC) introduction"
    url: "https://www.postgresql.org/docs/current/mvcc-intro.html"
  - title: "PostgreSQL: statement_timeout, lock_timeout, search_path"
    url: "https://www.postgresql.org/docs/current/runtime-config-client.html"
---

## Concept

The promise in `internal/sqlrun/sqlrun.go`'s package comment is "nothing ever persists".
For the `sql` runtime, `Run` keeps it like this:

1. runexec opens a fresh `pgx.Conn` to the playground database, as the non-superuser
   `playground` role (local mode). One connection per run, no pool.
2. `Run` sends one message: `BEGIN; CREATE SCHEMA run_<nanos>_<hex>; SET LOCAL search_path
   TO it; SET LOCAL statement_timeout = 5000; SET LOCAL lock_timeout = 2000;`.
3. It defers `ROLLBACK`, with `context.WithoutCancel` and a 5 s budget, so the cleanup
   runs even when the run's own context has expired.
4. It splits the lesson's `setup`, then your script, at top-level semicolons (psql's
   rules, dollar quotes and `BEGIN ATOMIC` function bodies included) and sends each
   statement on its own, after refusing
   any that would end the transaction (`guard.go`). The first error ends the phase.
5. It renders the results as text (command tag, notices, table) for the expected files.

There is no cleanup code that deletes what you created, because there is nothing to
delete: PostgreSQL's DDL is transactional, so the schema, your tables and your rows are
uncommitted catalog and heap rows that ROLLBACK turns into dead tuples nobody ever saw.

## Coming from C/C++/Java/Python/JS

- **Java/Spring tests**: `@Transactional` on a test class rolls back after each test,
  the same technique. Here the schema is part of the transaction too, which is only
  possible because PostgreSQL's DDL is transactional; MySQL's and Oracle's commit
  implicitly.
- **Python/Django**: `TestCase` wraps each test in a transaction for the same reason, and
  `TransactionTestCase` exists for code that needs real commits: StackLab's `sql-nontx`.
- **JS**: test runners often truncate tables between tests. A rollback is cheaper and
  needs no list of tables to clean.
- **C/C++ with libpq**: the multi-statement first message is `PQexec` with several
  statements; pgx's `PgConn().Exec(...).ReadAll()` is the same simple-protocol call.

## How it works

The three SQL modes share one idea, a `run_` schema per run, and differ in how it goes
away:

- **`sql`** (`Run`): schema and settings inside the transaction (`SET LOCAL`); ROLLBACK
  removes everything.
- **`sql-nontx`** (`RunNonTx`): autocommit, needed for `VACUUM` and
  `CREATE INDEX CONCURRENTLY`, which refuse to run in a transaction block. Settings are
  session-level; a deferred `dropSchema` sends ROLLBACK (in case your script left a
  transaction open) and `DROP SCHEMA ... CASCADE`.
- **`sql-multisession`** (`RunMulti`): the schema is committed so both sessions see it,
  setup commits, and cleanup rolls back B first, since B may hold locks the DROP would
  wait on.

Go lessons that import pgx get the same treatment through `ScratchSchema`: an empty
committed schema, passed to the program as `DATABASE_URL` with a `search_path`
parameter, dropped after the run. Committed schemas can outlive a crash, so the
non-transactional modes start with `dropStaleSchemas`: drop every `run_` schema this role
owns whose name-encoded timestamp is older than ten minutes. `Run` doesn't call it,
because a rolled-back run can't leave one behind.

If the backend dies mid-run, nothing is left either: the connection drops and the server
aborts its open transaction.

## Design notes

The technique had one real hole: the sandbox and the learner's script share a
transaction. A `ROLLBACK` or `COMMIT` in a `sql` example ended the sandbox, and every
later statement autocommitted in `public`, without timeouts, as a role allowed to
create there. `internal/sqlrun/guard.go` now closes it in two layers:

- **Refuse transaction control in `sql` mode.** A statement whose leading keyword is
  `COMMIT`, `END`, `ABORT`, `ROLLBACK` (but not `ROLLBACK TO`) or `PREPARE TRANSACTION`
  is never sent; the result explains why and points to `sql-nontx`. Example 3 shows it.
- **Check after each statement** that the session is still in a transaction. The server
  reports it with every ReadyForQuery message, and pgx exposes it as
  `PgConn().TxStatus()`: `'T'` in a transaction, `'E'` in a failed one, `'I'` when the
  script has ended it. On `'I'` the run stops and `Run` drops the schema, which may have
  been committed.

The same file refuses `COPY ... FROM STDIN` in every mode: the server would wait for
the client to stream rows, which this runner can't do, so the run used to hang until
its time budget ran out. A third layer sits in the database itself. Since PostgreSQL 15,
`public` is owned by `pg_database_owner`, a role whose only member is the current
database's owner, and `playground` owns its database, so it could create in `public`. A
`REVOKE` can't take that away from the owner, so `scripts/setup-local-db.sql` ends with
`ALTER SCHEMA public OWNER TO postgres`: `playground` keeps `USAGE` and loses `CREATE`.
An escape that slipped past both checks would fall back to the default search path, and
its unqualified `CREATE` would now fail instead of persisting.

The Docker mode caveat belongs here too. ADR-0003 says SQL runs "with a non-superuser
role owning only sandbox objects", but `docker-compose.yml` creates `playground` through
the postgres image's `POSTGRES_USER`, which the image makes a superuser. A rollback
can't undo what a superuser does outside the database, such as `COPY ... TO` a file.
Local mode's `scripts/setup-local-db.sql` gets this right with `NOSUPERUSER`.
