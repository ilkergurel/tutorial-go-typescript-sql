---
id: sql.transactions.vacuum
title: "VACUUM, autovacuum and bloat"
track: sql
module: transactions
order: 7
level: advanced
minutes: 15
since: pg10
prerequisites: [sql.transactions.acid-mvcc]
compare_with: [sql.security-ops.monitoring, sql.indexes.stats-pagination, sql.tables-types.alter, design.paradigms.immutability]
tags: [vacuum, autovacuum, mvcc, bloat, hot-updates, fillfactor]
summary: >
  Every UPDATE and DELETE leaves a dead row version behind. VACUUM makes that
  space reusable but rarely gives it back to the OS, an open transaction can
  stop it from removing anything, and HOT updates with a lower fillfactor
  avoid much of the garbage in the first place.
examples:
  - id: bloat
    file: examples/01_bloat.sql
    title: "Table size through updates, VACUUM, a mass DELETE and VACUUM FULL"
    runtime: sql-nontx
    check: exact
    annotations:
      - lines: "2-4"
        note: >
          Why TEMP: this playground database is shared with other runs, and
          an open transaction anywhere in it could stop VACUUM from removing
          rows (example 2). Other sessions' transactions don't hold back a
          temporary table. Autovacuum can't process temporary tables either,
          so nothing runs behind the script's back.
      - lines: "10-11"
        note: >
          Each UPDATE writes a new version of every row. The pages are full
          (fillfactor 100), so the new versions go to new pages and the table
          doubles. Once the UPDATE commits, the old versions are dead.
      - lines: "12-13"
        note: >
          Plain VACUUM removes the dead versions and records the space as
          free, but the file keeps its size: the free pages are at the start
          of the table, and VACUUM only hands back empty pages at the end.
      - lines: "14-15"
        note: >
          The second round of updates fills the space VACUUM freed: no
          growth. This is the steady state autovacuum aims for, a table larger
          than its live data that stops growing.
      - lines: "16-19"
        note: >
          Deleting half the rows frees space all over the table. VACUUM makes
          it reusable, and the file still doesn't shrink, because live rows
          remain up to the last page.
      - lines: "20-21"
        note: >
          VACUUM FULL copies the live rows into a new file: half the original
          size. Meanwhile it holds ACCESS EXCLUSIVE, which blocks even
          SELECT, and it needs disk space for the copy.
      - lines: "24-26"
        note: >
          Byte counts depend on the page size and the platform; ratios to the
          first measurement are what this lesson is about, and they come out
          the same on every run.
  - id: long-transaction
    file: examples/02_long_transaction.sql
    title: "An open transaction keeps VACUUM from removing anything"
    runtime: sql-multisession
    setup: |
      CREATE TABLE invoices (id int PRIMARY KEY);
      INSERT INTO invoices SELECT generate_series(1, 3);
      CREATE TABLE events (id int PRIMARY KEY, payload text NOT NULL)
        WITH (autovacuum_enabled = off);
      INSERT INTO events SELECT g, repeat('x', 100) FROM generate_series(1, 10000) AS g;
      CREATE TABLE sizes AS SELECT pg_relation_size('events') AS bytes;
    check: exact
    annotations:
      - lines: "2-3"
        note: >
          REPEATABLE READ keeps the snapshot from its first query until the
          transaction ends. A is now idle in transaction with an old snapshot,
          like a report that is still running or a forgotten BEGIN.
      - lines: "7-9"
        note: >
          The deleted rows are dead for everyone except A's snapshot, which
          predates the delete. VACUUM must keep them; VACUUM VERBOSE would say
          "5000 are dead but not yet removable". Nothing shrank.
      - lines: "11-12"
        note: >
          A never read events. VACUUM can't know which tables an open
          snapshot will read next, so an old snapshot holds back cleanup in
          every table of the database. (Setup turned autovacuum off for events
          so it can't interfere with the demo.)
      - lines: "13-15"
        note: >
          With A gone the rows are removable, the last pages become empty, and
          VACUUM truncates them: the one case where plain VACUUM returns space
          to the operating system. It needs a brief ACCESS EXCLUSIVE lock to
          do so, and simply skips the truncation if it can't get it.
  - id: hot-fillfactor
    file: examples/03_hot_fillfactor.sql
    title: "HOT updates and fillfactor: three rounds of updates, no growth, no VACUUM"
    runtime: sql-nontx
    check: exact
    annotations:
      - lines: "4-5"
        note: >
          fillfactor = 50 makes INSERT fill pages only halfway. The rest of
          each page is reserved for updated versions of the rows on that page.
      - lines: "11-15"
        note: >
          packed has no free space, so every update is a regular one: the new
          version goes to another page, with a new index entry. Without
          VACUUM in between, each round adds a full copy: 4.0 after three.
      - lines: "16-18"
        note: >
          roomy's updates are HOT (heap-only tuple): the new version stays on
          the same page and the index isn't touched, because the only indexed
          column, id, didn't change. The next update prunes the old versions
          from the page in passing and reuses the space.
      - lines: "20-23"
        note: >
          roomy didn't grow at all, with no VACUUM. The price is visible in
          the last column: it started twice as big. Lower fillfactor only on
          tables whose rows are updated often.
use_cases:
  - "Diagnosing a table that keeps growing while its row count stays flat"
  - "Tuning autovacuum per table for hot, update-heavy tables: counters, sessions, job queues"
  - "Finding the forgotten open transaction that blocks cleanup"
  - "Getting disk space back after a mass DELETE (VACUUM FULL in a maintenance window)"
strengths:
  - "Cleanup is off the critical path: UPDATE and DELETE don't move old data anywhere, and readers never block"
  - "Plain VACUUM and autovacuum run alongside SELECT, INSERT, UPDATE and DELETE"
  - "HOT updates and page pruning remove many dead versions during normal reads and writes, with no VACUUM at all"
  - "Autovacuum is on by default and can be tuned per table with storage parameters"
weaknesses:
  - "Dead versions occupy space until VACUUM runs, and indexes bloat along with the table"
  - "Plain VACUUM rarely shrinks a file; VACUUM FULL does, but blocks all access to the table while it rewrites it"
  - "One forgotten open transaction stops cleanup in the whole database"
  - "Vacuuming is not optional: every table must be vacuumed (frozen) at least once every two billion transactions, and Postgres forces it even with autovacuum off"
pitfalls:
  - title: "Turning autovacuum off to save I/O"
    detail: "Dead rows pile up, planner statistics go stale, and the anti-wraparound vacuum runs anyway, later and bigger. Tune instead: a lower scale factor makes autovacuum visit a big, busy table sooner."
    good: "ALTER TABLE jobs SET (autovacuum_vacuum_scale_factor = 0.01);"
  - title: "VACUUM FULL on a live table"
    detail: "It holds ACCESS EXCLUSIVE for the whole rewrite: every query on the table waits. Usually you don't need the space back; plain VACUUM keeps the size stable. If you must shrink, plan a maintenance window."
  - title: "Sessions left idle in transaction"
    detail: "A connection that ran BEGIN and then waits on the application holds back VACUUM everywhere. Find the oldest snapshots, and set idle_in_transaction_session_timeout as a safety net."
    good: "SELECT pid, state, xact_start, age(backend_xmin) AS xmin_age\nFROM pg_stat_activity\nWHERE backend_xmin IS NOT NULL\nORDER BY age(backend_xmin) DESC;"
  - title: "Reading n_dead_tup right after a change"
    detail: "pg_stat_user_tables is cumulative statistics: each backend flushes its counts when it goes idle, at most about once a second, so a query right after your DELETE may not see it yet. Use these numbers for monitoring trends, not as a check inside a script."
  - title: "Indexing a column you update constantly"
    detail: "An update that changes any indexed column can't be HOT. An index on last_seen_at turns every heartbeat into a new index entry plus a dead version for VACUUM. Index it only if queries really need it."
quiz:
  - q: "You deleted every other row of a big table and ran plain VACUUM. What happens to the file size?"
    options: ["It halves", "It stays about the same: the space is now reusable but not returned", "It grows"]
    answer: 1
    why: "Plain VACUUM returns space only when whole pages at the end of the table are empty; scattered free space is kept for reuse."
  - q: "VACUUM ran, yet the deleted rows are still there ('dead but not yet removable'). The most likely cause?"
    options: ["Autovacuum is disabled", "A transaction with a snapshot older than the delete is still open", "The table has no primary key"]
    answer: 1
    why: "VACUUM may only remove versions no snapshot can see. One old snapshot anywhere in the database is enough to keep them."
  - q: "Why does fillfactor = 70 help a table whose rows are updated often?"
    options: ["It compresses rows", "It leaves room on each page, so updates can be HOT: same page, no new index entry, cleaned up without VACUUM", "It makes VACUUM run more often"]
    answer: 1
    why: "HOT needs free space on the old row's page and no change to indexed columns; fillfactor reserves that space."
  - q: "When does plain VACUUM give disk space back to the operating system?"
    options: ["Always", "Only when pages at the end of the table become entirely empty and it can briefly lock the table exclusively", "Never"]
    answer: 1
    why: "That truncation is the only shrinking plain VACUUM does; VACUUM FULL rewrites the whole table."
further_reading:
  - title: "PostgreSQL docs: Routine Vacuuming"
    url: "https://www.postgresql.org/docs/current/routine-vacuuming.html"
  - title: "PostgreSQL docs: VACUUM"
    url: "https://www.postgresql.org/docs/current/sql-vacuum.html"
  - title: "PostgreSQL docs: Heap-Only Tuples (HOT)"
    url: "https://www.postgresql.org/docs/current/storage-hot.html"
  - title: "PostgreSQL docs: Vacuuming and autovacuum settings"
    url: "https://www.postgresql.org/docs/current/runtime-config-vacuum.html"
  - title: "PostgreSQL docs: idle_in_transaction_session_timeout"
    url: "https://www.postgresql.org/docs/current/runtime-config-client.html"
---

## Concept

MVCC never overwrites: an `UPDATE` leaves the old row version behind and a `DELETE` only
marks it deleted. Those dead versions take space until **VACUUM** removes them, and it can
remove a version only when no snapshot can still see it. That is the price of readers that
never wait.

What VACUUM does with the space depends on the form:

- **Plain `VACUUM`** (and autovacuum, which runs it for you) marks the space free for
  reuse inside the same table. It runs alongside normal reads and writes, and returns
  space to the OS only when whole pages at the end of the table are empty.
- **`VACUUM FULL`** writes a compact copy of the table and drops the old file. It gets the
  space back but holds ACCESS EXCLUSIVE, which blocks every query on the table, until it
  finishes.

A table that is updated steadily and vacuumed regularly settles at some size above its
live data, and stays there. **Bloat** is when it keeps growing: autovacuum can't keep up,
or something stops it from removing anything.

VACUUM also has jobs beyond space: it updates the visibility map (which makes index-only
scans cheaper), can refresh planner statistics (`VACUUM ANALYZE`), and **freezes** old
rows so that 32-bit transaction IDs can wrap around safely.

## Coming from C/C++/Java/Python/JS

- **Garbage collection:** dead row versions are garbage, VACUUM is the collector, and an
  open snapshot is a GC root: anything it might read stays alive. Like a long-lived
  reference that leaks memory in Java, a long-lived transaction leaks disk.
- **Go's GC vs. fillfactor:** you leave headroom (`GOGC`) so the collector runs less
  often; `fillfactor` leaves headroom on each page so updates can be cleaned up locally,
  without VACUUM.
- **C/C++ allocators:** plain VACUUM is `free()`: the memory goes back to the allocator's
  free lists, not to the OS. `VACUUM FULL` is a compacting copy.

## How it works

**Autovacuum** tries to visit each database once per `autovacuum_naptime` (1 minute) and
vacuums a table once its dead rows exceed

```
min(autovacuum_vacuum_max_threshold,
    autovacuum_vacuum_threshold + autovacuum_vacuum_scale_factor * reltuples)
```

With the defaults (100 million, 50, 0.2), a table of 10 million rows waits for about 2
million dead rows. The 100-million cap is new in PostgreSQL 18; before it, the trigger grew
without limit with the table. All three can be set per table, and lowering the scale
factor on large, busy tables is the most common tuning. A separate insert-based threshold
makes sure append-only tables get vacuumed (and frozen) too.

**What blocks cleanup** is the oldest snapshot or open transaction in the database, as in
example 2. In production that is a long report, a session idle in transaction, a
forgotten prepared transaction (`pg_prepared_xacts`) or a stale replication slot
(`pg_replication_slots`); the docs' wraparound checklist goes through exactly these.
`pg_stat_activity.backend_xmin` shows each session's horizon.

**HOT updates** avoid much of the work. When an update changes no indexed column and the
new version fits on the same page, no index entry is added, and later reads or updates of
that page prune the intermediate versions without VACUUM. `fillfactor` reserves the room
that makes this likely; `pg_stat_user_tables.n_tup_hot_upd` shows how often it happens.

The TEMP tables in examples 1 and 3 aren't held back by other sessions' snapshots, which
keeps these demos deterministic on a shared database. The docs don't say so; the source
does: since PostgreSQL 14, temporary tables get a cleanup horizon of their own that only
looks at the owning session (src/backend/storage/ipc/procarray.c, function
ComputeXidHorizons, temp_oldest_nonremovable; absent from the 13 branch). On PostgreSQL
18.7, with another session holding a REPEATABLE READ snapshot, VACUUM removed all 10,000
deleted rows of a temp table and truncated it to 0 pages, while a regular table reported
"10000 are dead but not yet removable".

## Design notes

Treat autovacuum as part of the schema design for update-heavy tables, not as an
afterthought:

- **Keep transactions short** everywhere (lessons 1 and 5). This is the one thing no
  setting can fix for you. `idle_in_transaction_session_timeout` and `transaction_timeout`
  (PostgreSQL 17+) are the server-side safety nets.
- **Make updates HOT-friendly:** don't index columns that change on every update, and give
  heavily updated tables a `fillfactor` of 70 to 90.
- **Tune per table, not globally:** `ALTER TABLE ... SET (autovacuum_vacuum_scale_factor =
  0.01)` for big, busy tables; job queues and session tables are the usual candidates.
- **Watch the trend:** `n_dead_tup`, `last_autovacuum` and `n_tup_hot_upd` in
  `pg_stat_user_tables`, and the oldest `backend_xmin` in `pg_stat_activity`. The
  monitoring lesson builds this into a routine.
- **Don't reach for `VACUUM FULL`** to fix bloat on a live system. Fix the cause (long
  transactions, autovacuum settings) and let the table's size stabilize; reclaim space in
  a maintenance window if you really need it.
