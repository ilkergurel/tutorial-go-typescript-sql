---
id: sql.tables-types.alter
title: "ALTER TABLE done safely"
track: sql
module: tables-types
order: 5
level: intermediate
minutes: 15
since: pg18
prerequisites: [sql.tables-types.ids]
compare_with: [sql.security-ops.backup-migrations, go.postgres.migrations-sqlc, sql.transactions.acid-mvcc, sql.constraints.generated]
tags: [ddl, alter-table, migrations, locks, table-rewrite, not-valid]
summary: >
  Some ALTER TABLE forms only touch the catalog; others rewrite every row and index
  while holding a lock that blocks even SELECT. Learn which is which, and the safe
  order for a required column: add nullable, constrain NOT VALID, backfill in
  batches, VALIDATE.
examples:
  - id: rewrites
    file: examples/01_rewrites.sql
    title: "Which changes rewrite the table or its index"
    runtime: sql
    check: exact
    annotations:
      - lines: "10-11"
        note: >
          relfilenode names the file a table or index is stored in, and files()
          reads both. A rewrite writes a new file and swaps it in, so the number
          changes. The numbers themselves differ on every run, so the log
          compares them instead of printing them.
      - lines: "15-18"
        note: >
          Catalog-only. A column without a default reads as NULL in old rows,
          and a non-volatile default is evaluated once and stored in the table's
          metadata (since PostgreSQL 11). Fast on a billion rows.
      - lines: "19"
        note: >
          A volatile default (uuidv7(), clock_timestamp(), random()) differs per
          row, so every row is rewritten, and every index with it. now() is only
          stable: it is evaluated once, costs no rewrite, and gives every
          existing row the same timestamp.
      - lines: "21"
        note: >
          A virtual generated column (the default kind since PostgreSQL 18) is
          computed when read, so adding one never rewrites. A STORED generated
          column, like an identity column, would.
      - lines: "23-26"
        note: >
          Raising a varchar limit or dropping it is binary coercible: no rewrite,
          and because varchar and text sort identically, no index rebuild either.
          Shrinking a limit (varchar(200) to varchar(20)) rewrites both, since
          every value has to be checked.
      - lines: "27"
        note: >
          A collation change leaves the table alone but rebuilds every index on
          the column, because the sort order changes. Switching a big table to
          Turkish ordering (tr-x-icu) costs index builds, not a table copy.
      - lines: "29"
        note: >
          integer to bigint changes every row's bytes: a full rewrite of table
          and indexes under ACCESS EXCLUSIVE, for as long as it takes. On a large
          table, add a bigint column, backfill it, and swap names in one short
          transaction instead.
      - lines: "31"
        note: >
          DROP COLUMN only hides the column. The space comes back as rows are
          updated over time, or at the next rewrite.
  - id: lock-levels
    file: examples/02_lock_levels.sql
    title: "The lock each change holds, from pg_locks"
    runtime: sql-nontx
    check: exact
    annotations:
      - lines: "1-4"
        note: >
          In the sql runtime the table would be created in the same transaction,
          and CREATE TABLE already holds ACCESS EXCLUSIVE on it, which would hide
          what the ALTER takes. Here each statement commits on its own, so the
          table exists first, as in production.
      - lines: "5-7"
        note: >
          pg_locks lists every lock; this view keeps the ones held by this
          session on accounts. Table locks are held until the transaction ends,
          so a query after the ALTER still sees them.
      - lines: "10-11"
        note: >
          AccessExclusiveLock for a catalog-only change. It conflicts with every
          mode, plain SELECT included. The change itself is instant, but the ALTER
          must first wait for every open transaction that touched the table, and
          every query arriving after it queues behind it.
      - lines: "15-16"
        note: >
          NOT VALID skips the scan of existing rows, but adding the constraint
          still needs ACCESS EXCLUSIVE, just for a moment.
      - lines: "20-21"
        note: >
          VALIDATE CONSTRAINT takes only SHARE UPDATE EXCLUSIVE, which doesn't
          conflict with reads or with INSERT, UPDATE and DELETE. The long scan
          runs alongside normal traffic.
      - lines: "25-26"
        note: >
          A plain CREATE INDEX holds SHARE: reads continue, but every write
          waits until the build finishes.
      - lines: "31-32"
        note: >
          SQLSTATE 25001. CREATE INDEX CONCURRENTLY doesn't block writes, but it
          works in several transactions of its own, so it can't run inside one.
          Migration tools that wrap each file in a transaction need an opt-out
          for it.
  - id: safe-order
    file: examples/03_safe_order.sql
    title: "Adding a required column without a long lock"
    runtime: sql
    check: exact
    annotations:
      - lines: "11"
        note: >
          Catalog-only. If one constant were right for every existing row,
          ADD COLUMN country text NOT NULL DEFAULT 'TR' would be catalog-only too,
          and you'd be done. Here the value depends on each row's phone number.
      - lines: "15-16"
        note: >
          PostgreSQL 18 stores NOT NULL as a named constraint that can be added
          NOT VALID. It is enforced at once for new rows and for every updated
          row, even an update that doesn't touch country, so the application must
          already write the column before this step runs.
      - lines: "19-24"
        note: >
          1000 rows without a country, and convalidated is false: the database
          knows the rule doesn't hold for old rows yet.
      - lines: "27-30"
        note: >
          In production this is a loop over id ranges, committing after each
          batch, so no transaction holds thousands of row locks for long.
      - lines: "33"
        note: >
          The scan happens here, under SHARE UPDATE EXCLUSIVE. If a NULL were
          left, VALIDATE would fail and change nothing; fix the rows and run it
          again.
      - lines: "35-40"
        note: >
          Zero rows without a country and a validated constraint: a regular
          NOT NULL now. Before PostgreSQL 18 the same effect needed a
          CHECK (country IS NOT NULL) NOT VALID, VALIDATE, then SET NOT NULL
          (which skips its scan when a valid CHECK proves the column has no
          NULLs), then dropping the CHECK.
      - lines: "43"
        note: >
          SQLSTATE 23502. This insert would have failed at any point after
          step 2, which is what kept the backfill from racing new rows.
use_cases:
  - "Adding columns and constraints to tables that serve traffic while the migration runs"
  - "Reviewing a migration: predicting which statements rewrite or scan, and how long they hold which lock"
  - "Planning type changes (integer to bigint, a collation switch) on large tables"
strengths:
  - "Many common changes are catalog-only: nullable columns, constant defaults, virtual generated columns, raising varchar limits, dropping columns"
  - "DDL is transactional: a migration's ALTERs, data fixes and constraint changes commit or roll back together (CREATE INDEX CONCURRENTLY is a documented exception)"
  - "NOT VALID plus VALIDATE splits a constraint into a brief exclusive step and a long step that runs alongside reads and writes"
  - "Rewrite behaviour is observable (relfilenode) and lock levels are documented per form, so migrations can be reasoned about before they run"
weaknesses:
  - "Even catalog-only ALTERs need ACCESS EXCLUSIVE, which blocks SELECT and queues all later queries behind it"
  - "Type changes such as integer to bigint rewrite the whole table and its indexes while holding that lock"
  - "CREATE INDEX CONCURRENTLY can't run in a transaction block and can leave an invalid index behind when it fails"
  - "Space from dropped columns isn't returned until rows are rewritten"
pitfalls:
  - title: "DDL without lock_timeout"
    detail: "An ALTER waiting behind a long-running query holds its place in the lock queue, and every query after it waits too: a metadata-only change becomes an outage. Set a short lock_timeout before DDL and retry on failure."
    good: "SET lock_timeout = '2s'; ALTER TABLE accounts ADD COLUMN note text;"
  - title: "A volatile default on a big table"
    detail: "DEFAULT uuidv7(), gen_random_uuid() or clock_timestamp() rewrites the table. Add the column without a default, set the default separately (it then applies to new rows only), and backfill old rows in batches."
    bad: "ALTER TABLE events ADD COLUMN token uuid NOT NULL DEFAULT uuidv7();"
    good: "ALTER TABLE events ADD COLUMN token uuid; ALTER TABLE events ALTER COLUMN token SET DEFAULT uuidv7(); -- then backfill, then NOT NULL NOT VALID + VALIDATE"
  - title: "Adding a constraint without NOT VALID"
    detail: "ADD CONSTRAINT scans every row while holding ACCESS EXCLUSIVE (or SHARE ROW EXCLUSIVE for foreign keys, which still blocks writes). Add it NOT VALID, then VALIDATE in a separate transaction."
  - title: "Plain CREATE INDEX on a live table"
    detail: "It blocks all writes for the duration of the build. Use CREATE INDEX CONCURRENTLY outside a transaction block, and check for an INVALID index if it fails."
  - title: "NOT VALID before the application writes the column"
    detail: "A NOT VALID constraint is enforced for every new and updated row. If the old application version still writes rows without the column, its updates start failing. Deploy the writer first."
quiz:
  - q: "Which change rewrites the whole table on PostgreSQL 18?"
    options: ["ADD COLUMN plan text NOT NULL DEFAULT 'free'", "ALTER COLUMN name TYPE varchar(200) (from varchar(100))", "ADD COLUMN token uuid DEFAULT uuidv7()", "DROP COLUMN note"]
    answer: 2
    why: "uuidv7() is volatile, so each existing row needs its own value; the others are catalog-only."
  - q: "Your ALTER TABLE ... ADD COLUMN is instant, yet the app stalls while it waits. Why?"
    options: ["ADD COLUMN always rewrites", "It waits for ACCESS EXCLUSIVE behind a long transaction, and new queries queue behind it", "Postgres locks the whole database for DDL"]
    answer: 1
    why: "ACCESS EXCLUSIVE conflicts even with SELECT; the waiting ALTER blocks everything that arrives after it. A short lock_timeout limits the damage."
  - q: "Which lock does ALTER TABLE ... VALIDATE CONSTRAINT take on the table?"
    options: ["ACCESS EXCLUSIVE", "SHARE UPDATE EXCLUSIVE", "No lock"]
    answer: 1
    why: "Validation only checks existing rows (new ones are already checked), so it can let reads and writes continue."
  - q: "What is the safe order for a new required column whose value is computed per row?"
    options: ["Add nullable, add NOT NULL NOT VALID, backfill in batches, VALIDATE", "ADD COLUMN ... NOT NULL, then backfill", "Backfill first, then ADD COLUMN"]
    answer: 0
    why: "Adding NOT NULL directly fails on existing rows (or scans under ACCESS EXCLUSIVE); NOT VALID guards new writes while old rows are fixed in batches."
further_reading:
  - title: "PostgreSQL docs: ALTER TABLE (see Notes on rewrites and locks)"
    url: "https://www.postgresql.org/docs/current/sql-altertable.html"
  - title: "PostgreSQL docs: Explicit Locking (table-level lock modes)"
    url: "https://www.postgresql.org/docs/current/explicit-locking.html"
  - title: "PostgreSQL docs: CREATE INDEX (Building Indexes Concurrently)"
    url: "https://www.postgresql.org/docs/current/sql-createindex.html"
  - title: "PostgreSQL docs: pg_locks"
    url: "https://www.postgresql.org/docs/current/view-pg-locks.html"
---

## Concept

Every `ALTER TABLE` has two costs, and they are independent:

- **Work.** Catalog-only (instant), a **scan** of every row (adding a constraint, `SET NOT
  NULL`), or a **rewrite** of every row and index (most type changes, volatile defaults,
  stored generated and identity columns).
- **Lock.** Unless the docs say otherwise for a form, `ALTER TABLE` takes ACCESS
  EXCLUSIVE, the one mode that blocks plain `SELECT`. A few forms take less:
  `VALIDATE CONSTRAINT` and `SET STATISTICS` take SHARE UPDATE EXCLUSIVE, `ADD FOREIGN
  KEY` takes SHARE ROW EXCLUSIVE.

A rewrite under ACCESS EXCLUSIVE is downtime proportional to the table's size. Safe
migrations are built from catalog-only steps, scans under weak locks, and batched updates.

## Coming from C/C++/Java/Python/JS

- **ORM migrations (Django, Rails, Flyway):** they generate the DDL, but not the lock
  analysis. "Add a non-null column with a default" can be instant or a rewrite depending on
  whether the default is volatile; the migration file looks the same.
- **Java/Go deployments:** a migration runs while the old version of your service is still
  serving traffic. Expand first (add nullable, write both), contract later (constraints,
  drops), the same discipline as changing a public API.
- **Mutexes:** ACCESS EXCLUSIVE is a writer lock that also excludes readers, and the queue
  is fair: once a writer waits, later readers wait behind it, even though they wouldn't
  conflict with the current holder. The docs call this a "soft block" (under
  `pg_blocking_pids`, System Information Functions): a session waiting for a conflicting
  lock and "ahead of it in the wait queue" blocks the later one. On PostgreSQL 18.7, with
  one session holding ACCESS SHARE and an `ALTER TABLE` waiting, a third session's
  `SELECT` waited too, and `pg_blocking_pids` named the `ALTER`, not the reader.

## Design notes

Before running a migration, ask of each statement: catalog, scan or rewrite? Which lock,
and for how long? The rewrite example gives the common answers for PostgreSQL 18; the
ALTER TABLE notes in the docs are the reference.

Habits that follow:

- `SET lock_timeout` (a few seconds) before DDL, and retry. Waiting forever for ACCESS
  EXCLUSIVE is how a catalog-only change takes a site down.
- Keep each migration transaction short; Postgres DDL is transactional, so a failed
  migration rolls back, but a long transaction holds its locks to the end.
- Constraints: add NOT VALID, VALIDATE separately. Indexes: CREATE INDEX CONCURRENTLY,
  outside a transaction block.
- Type changes on big tables: new column, backfill in batches, swap in a short
  transaction. Choose `bigint` and `text` up front and you rarely need this.

This sandbox runs every example in a throwaway transaction (or, for the lock example,
throwaway schema), so you can try any of these forms and look at `pg_locks` and
`relfilenode` yourself without affecting anything.
