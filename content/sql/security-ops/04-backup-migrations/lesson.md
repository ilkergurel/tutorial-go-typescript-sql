---
id: sql.security-ops.backup-migrations
title: "Backups and PITR (concepts); zero-downtime migrations (add, backfill, NOT VALID + VALIDATE)"
track: sql
module: security-ops
order: 4
level: advanced
minutes: 15
since: pg13
prerequisites: [sql.tables-types.alter, sql.transactions.row-locks]
compare_with: [sql.tables-types.alter, go.postgres.migrations-sqlc, sql.files-movement.dump-generate, sql.security-ops.monitoring, sql.transactions.vacuum]
tags: [backup, pitr, wal, pg_dump, pg_basebackup, migrations, create-index-concurrently, not-valid, backfill]
summary: >
  A dump is a snapshot; a base backup plus archived WAL can be rolled forward to any
  moment, which is what you need after a bad DELETE. Schema changes go out in small
  steps: expand, add constraints NOT VALID, build indexes concurrently, backfill in
  committed batches, validate, and only then contract.
examples:
  - id: wal
    file: examples/01_wal.sql
    title: "What the WAL records, and what recovery can bring back"
    runtime: sql
    check: exact
    annotations:
      - lines: "4-6"
        note: >
          relpersistence p (permanent) and u (unlogged). An unlogged table's data
          never enters the WAL: it is truncated after a crash, not replicated to
          standbys, and left out of base backups except for an empty
          initialization fork. Fine for scratch data you can reload, never for
          an audit log.
      - lines: "9-14"
        note: >
          EXPLAIN (ANALYZE, WAL) counts the WAL a statement generated. Only the
          record count is returned: the byte count includes full-page images,
          which depend on when the last checkpoint happened.
      - lines: "16-19"
        note: >
          2003 records for 1000 rows: one per heap insert, one per index entry,
          and a few more. The unlogged table wrote none. (With wal_level =
          minimal, a table created in the same transaction would skip row logging
          too; the default, replica, is the minimum for archiving.)
      - lines: "22-24"
        note: >
          A logical backup is different: pg_dump reads tables with SQL, so it
          includes unlogged tables' rows unless you pass
          --no-unlogged-table-data. Physical backups and PITR see only what the
          WAL and the data files hold.
      - lines: "27"
        note: >
          SQLSTATE 42501: restricted to superusers by default, though EXECUTE can
          be granted, for example to a deploy role. recovery_target_name later
          stops replay at that mark. Keep names unique: recovery stops at the
          first restore point with a matching name.
  - id: cic
    file: examples/02_cic.sql
    title: "A concurrent unique index: the failed build, the cleanup, and the wait"
    runtime: sql-multisession
    setup: |
      CREATE TABLE users (
        id    bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
        email text NOT NULL
      );
      INSERT INTO users (email)
      SELECT 'user' || g || '@example.com' FROM generate_series(1, 1000) AS g;
      INSERT INTO users (email) VALUES ('ayse@example.com'), ('Ayse@example.com');
    check: exact
    annotations:
      - lines: "2"
        note: >
          CONCURRENTLY takes SHARE UPDATE EXCLUSIVE instead of SHARE, so inserts,
          updates and deletes go on during the build. It pays with two table
          scans, several waits, and no transaction block. SQLSTATE 23505:
          ayse@ and Ayse@ collide under lower().
      - lines: "4-5"
        note: >
          The failed build left an invalid index behind. indisready is false
          because it failed in the first scan, so it isn't even maintained. A
          failure in the later validation scan would leave it ready: updated on
          every write and, the docs warn, still enforcing uniqueness. Either way
          queries ignore it, and psql's \d marks it INVALID.
      - lines: "6"
        note: >
          INSERT 0 1: an index that isn't ready enforces nothing, so a third
          spelling of the same address got in. Leftovers are worth a monitoring
          query: SELECT indexrelid::regclass FROM pg_index WHERE NOT indisvalid.
      - lines: "9-10"
        note: >
          DROP INDEX CONCURRENTLY removes the leftover without blocking the table.
          The data fix is the real migration step; here the mixed-case
          duplicates are deleted, in TraceLite you would merge the accounts.
      - lines: "13-14"
        note: >
          REPEATABLE READ keeps the snapshot of its first query until COMMIT, like
          a long report or a pg_dump. B never touches the index or writes
          anything.
      - lines: "15-16"
        note: >
          A waits. Before it can mark the index valid, a concurrent build waits
          for every transaction holding a snapshot older than its last scan, a
          read-only one included. pg_stat_progress_create_index names the phase.
      - lines: "17-19"
        note: >
          B commits, A finishes, the index is valid and enforces uniqueness. One
          forgotten idle-in-transaction session can stall every concurrent index
          build: look at pg_stat_activity before you start one.
  - id: fk-backfill
    file: examples/03_fk_backfill.sql
    title: "A foreign key on a live table: NOT VALID, batched backfill, VALIDATE"
    runtime: sql-nontx
    check: exact
    annotations:
      - lines: "6-9"
        note: >
          Table locks are held until the transaction ends, so a query right after
          each ALTER, inside the same transaction, shows what the ALTER took.
      - lines: "12"
        note: >
          Expand. A nullable column without a default is a catalog-only change
          (the ALTER TABLE lesson lists what rewrites and what doesn't).
      - lines: "16-18"
        note: >
          ADD FOREIGN KEY takes SHARE ROW EXCLUSIVE on both tables, which blocks
          writes to both; NOT VALID skips the scan of existing rows, so that lasts
          a moment. From here on every insert and update must reference an
          existing project. AccessShareLock, a plain SELECT's mode, conflicts only
          with ACCESS EXCLUSIVE.
      - lines: "22-37"
        note: >
          A procedure may COMMIT inside its loop when CALL runs in its own
          transaction (PostgreSQL 11+). Each batch's row locks are released at
          once, a lock wait or failure costs one batch, and autovacuum can clean
          up behind it. Real backfills use batches of a few thousand rows and
          often pause between them.
      - lines: "41-42"
        note: >
          VALIDATE takes SHARE UPDATE EXCLUSIVE on requirements, which lets reads
          and writes continue, and only ROW SHARE on projects, exactly as the
          ALTER TABLE docs list. The long scan of existing rows happens here, off
          the critical path.
use_cases:
  - "Recovering from a bad DELETE or a destructive migration to the minute before it happened (PITR)"
  - "Moving one database between servers or major versions, or keeping a portable copy, with pg_dump/pg_restore"
  - "Deploying schema changes to tables that serve traffic: new columns, foreign keys, unique indexes, type changes"
  - "Long data backfills that must not hold locks or bloat one giant transaction"
strengths:
  - "Continuous archiving gives recovery to any moment since the base backup, not just to the last nightly copy"
  - "pg_dump produces a consistent snapshot of a live database without blocking readers or writers"
  - "NOT VALID + VALIDATE and CONCURRENTLY split a blocking operation into a short strong lock and a long weak one, so most changes need no maintenance window"
  - "Transactional DDL means a failed migration step rolls back cleanly, unlike MySQL, where DDL commits implicitly"
weaknesses:
  - "PITR needs setup in advance (wal_level, archive_mode, a reliable archive), and restore time grows with the amount of WAL to replay since the base backup"
  - "pg_dump is per database and doesn't include roles; a restore also needs pg_dumpall --globals-only and rebuilds every index"
  - "CREATE INDEX CONCURRENTLY can fail halfway and leave an invalid index, and waits for every transaction holding an older snapshot"
  - "Zero-downtime changes take several deploys (expand, backfill, contract) and the application must work with both schemas in between"
pitfalls:
  - title: "Backups that were never restored"
    detail: "A backup is only proven by a restore. Restore to a scratch server on a schedule, replay WAL to a target, and run a few sanity queries; measure how long it took."
  - title: "Migrations without lock_timeout"
    detail: "Even the brief SHARE ROW EXCLUSIVE of ADD FOREIGN KEY waits behind a long transaction, and every query queues behind the waiting ALTER. Set a short lock_timeout and retry."
    good: "SET lock_timeout = '3s';\nALTER TABLE requirements ADD CONSTRAINT requirements_project_fk FOREIGN KEY (project_id) REFERENCES projects NOT VALID;"
  - title: "One huge UPDATE as the backfill"
    detail: "A single UPDATE of ten million rows holds every row lock until it commits, doubles the table with dead versions before VACUUM can act, and loses everything on failure. Commit in batches."
    bad: "UPDATE requirements r SET project_id = p.id FROM projects p WHERE p.code = r.project_code;"
    good: "CALL backfill_project_id(5000);  -- COMMIT after each batch"
  - title: "Ignoring an invalid index after a failed CREATE INDEX CONCURRENTLY"
    detail: "Retrying with the same name fails because the leftover exists; an index left ready keeps costing every write. Drop it (DROP INDEX CONCURRENTLY) or REINDEX INDEX CONCURRENTLY, after fixing the data."
  - title: "Contracting too early"
    detail: "Dropping the old column in the same deploy that stops writing it breaks the old application version still serving requests. Drop it in a later deploy, after a backup or restore point."
quiz:
  - q: "A DROP TABLE committed at 14:32 yesterday must be undone. Which setup lets you recover the database as it was at 14:31?"
    options: ["A nightly pg_dump at 02:00", "A base backup plus a continuous WAL archive", "A streaming replica"]
    answer: 1
    why: "PITR restores the base backup and replays archived WAL up to a recovery target. The dump loses everything after 02:00; the replica replayed the DROP within moments."
  - q: "CREATE UNIQUE INDEX CONCURRENTLY fails on duplicate values. What is left?"
    options: ["Nothing: it rolled back", "An invalid index that must be dropped (or rebuilt) after fixing the data", "A valid index without the duplicate rows"]
    answer: 1
    why: "The build runs in several transactions; a failure leaves the catalog entry behind, marked invalid."
  - q: "Which lock does ALTER TABLE requirements ADD FOREIGN KEY (project_id) REFERENCES projects NOT VALID take on projects?"
    options: ["None", "ROW SHARE", "SHARE ROW EXCLUSIVE, which briefly blocks writes to projects"]
    answer: 2
    why: "ADD FOREIGN KEY locks the referenced table too. ROW SHARE is what VALIDATE CONSTRAINT takes there later."
  - q: "Why does the backfill commit after each batch?"
    options: ["PostgreSQL limits an UPDATE to a few thousand rows", "To release row locks quickly, let VACUUM keep up, and lose only one batch on failure", "VALIDATE CONSTRAINT requires it"]
    answer: 1
    why: "One giant transaction holds every lock and every dead row version until it ends."
further_reading:
  - title: "PostgreSQL docs: Backup and Restore (SQL dump, file system, continuous archiving)"
    url: "https://www.postgresql.org/docs/current/backup.html"
  - title: "PostgreSQL docs: Continuous Archiving and Point-in-Time Recovery"
    url: "https://www.postgresql.org/docs/current/continuous-archiving.html"
  - title: "PostgreSQL docs: pg_dump"
    url: "https://www.postgresql.org/docs/current/app-pgdump.html"
  - title: "PostgreSQL docs: pg_basebackup"
    url: "https://www.postgresql.org/docs/current/app-pgbasebackup.html"
  - title: "PostgreSQL docs: Write Ahead Log settings (wal_level, archiving, recovery targets)"
    url: "https://www.postgresql.org/docs/current/runtime-config-wal.html"
  - title: "PostgreSQL docs: CREATE INDEX (Building Indexes Concurrently)"
    url: "https://www.postgresql.org/docs/current/sql-createindex.html#SQL-CREATEINDEX-CONCURRENTLY"
  - title: "PostgreSQL docs: Transaction Management in PL/pgSQL"
    url: "https://www.postgresql.org/docs/current/plpgsql-transactions.html"
---

## Concept

**Backups** come in two families:

- **Logical:** `pg_dump` writes one database as SQL or as an archive for `pg_restore`.
  It reads through a consistent snapshot and doesn't block readers or writers. Roles
  and tablespaces are cluster-wide and need `pg_dumpall --globals-only`. A dump is a
  copy as of the moment it started, nothing later.
- **Physical:** `pg_basebackup` copies the whole cluster's files while it runs. On its
  own that's another snapshot. Combined with **continuous archiving** of the WAL
  (`archive_mode`, `archive_command` or `archive_library`, `wal_level` at least
  `replica`), it becomes **point-in-time recovery**: restore the base backup, then
  replay archived WAL up to a target time, transaction, LSN or named restore point.

**Zero-downtime migrations** follow expand, migrate, contract. Add what the new code
needs without breaking the old code; move the data in small committed batches; add
constraints `NOT VALID` and validate them later; build indexes `CONCURRENTLY`; remove
the old shape only when no running version uses it. Each step should be catalog-only,
a short strong lock, or a long weak lock. A long strong lock is downtime.

## Coming from C/C++/Java/Python/JS

- **Git:** a base backup is a checkout of an old commit, the WAL archive is every commit
  since, and PITR is `git checkout <time>` by replaying them. A `pg_dump` is a tarball of
  one working tree.
- **Rails and Django migrations** run each migration in a transaction; concurrent index
  builds need `disable_ddl_transaction!` (Rails) or a non-atomic migration with Django's
  `AddIndexConcurrently`. goose's equivalent is `-- +goose NO TRANSACTION`.
- **MySQL:** DDL commits implicitly, so a failed migration can leave half its changes.
  PostgreSQL's DDL is transactional; `CREATE INDEX CONCURRENTLY`, `VACUUM` and a few
  others are the exceptions.
- **API versioning:** expand/contract is the database version of supporting v1 and v2
  of an endpoint until every client has moved.

## How it works

PITR, end to end:

1. Ahead of time: `wal_level = replica` (the default), `archive_mode = on` (a restart),
   and an `archive_command` or `archive_library` that copies each completed WAL segment
   somewhere safe and reports success only when it really succeeded.
2. Regularly: `pg_basebackup` (a role with `REPLICATION` and a replication entry in
   `pg_hba.conf`). Since PostgreSQL 17, `pg_basebackup --incremental` with
   `summarize_wal = on` sends only changed blocks, and `pg_combinebackup` rebuilds a full
   backup from the chain.
3. To recover: restore the base backup into an empty data directory, set
   `restore_command` and one recovery target (`recovery_target_time`, `_name`, `_xid` or
   `_lsn`), create `recovery.signal`, start the server. It replays to the target and, by
   default, pauses there so you can check the result before promoting. A new timeline
   starts, so the old future and the new one don't mix.

`pg_dump` details worth knowing: the custom (`-Fc`) and directory (`-Fd`) formats restore
selectively with `pg_restore`, and the directory format dumps in parallel (`-j`). It
takes shared locks on the tables it dumps, so a migration's `ALTER TABLE` waits until
the dump finishes; `--lock-wait-timeout` keeps the dump from waiting forever on the
reverse. Since PostgreSQL 18 it can also dump planner statistics (`--statistics`), not
done by default, so run `ANALYZE` after a restore.

## Design notes

- **Decide RPO and RTO first.** How much data may be lost (the WAL archive's lag, versus
  a day for nightly dumps) and how long a restore may take (replaying a week of WAL is
  slow; take base backups often enough). Keep the archive off the database host.
- **Test restores** on a schedule, into a scratch server. A backup nobody restored is a
  hope, not a backup.
- **Backups contain personal data.** Encrypt them, restrict who can restore them, and
  remember that a row deleted today lives in every backup until that backup expires:
  the retention period is part of how long you keep personal data under KVKK.
- **Migration discipline:** `SET lock_timeout` and retry; one risky statement per
  transaction; `NOT VALID` then `VALIDATE`; `CONCURRENTLY` in a file of its own (goose:
  `-- +goose NO TRANSACTION`); backfills in committed batches; contract in a later deploy.
  Before a destructive step, take a fresh base backup or, with the right grant,
  `pg_create_restore_point`.
- The rewrite and lock tables for individual `ALTER TABLE` forms are in the
  `sql.tables-types.alter` lesson; this one adds what happens across tables (foreign
  keys), across transactions (backfills, concurrent builds) and across time (backups).
