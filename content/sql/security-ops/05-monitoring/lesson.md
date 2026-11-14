---
id: sql.security-ops.monitoring
title: "Monitoring essentials"
track: sql
module: security-ops
order: 5
level: advanced
minutes: 15
since: pg10
prerequisites: [sql.transactions.row-locks, sql.indexes.stats-pagination]
compare_with: [sql.indexes.stats-pagination, sql.transactions.vacuum, sql.security-ops.pooling-config, design.architecture.reliability, go.postgres.pgx]
tags: [monitoring, pg_stat_activity, pg_locks, pg_blocking_pids, cumulative-statistics, pg_stat_user_tables, pg_cancel_backend, timeouts]
summary: >
  When the database is slow, look at what is running and waiting right now
  (pg_stat_activity, pg_blocking_pids, pg_locks), then at what has accumulated
  (pg_stat_statements, pg_stat_user_tables, pg_stat_database). Cancel or terminate the
  culprit, and let timeouts do that job before anyone gets paged.
examples:
  - id: blocking
    file: examples/01_blocking.sql
    title: "Who is waiting, on what, and for whom"
    runtime: sql-multisession
    setup: |
      CREATE TABLE accounts (id int PRIMARY KEY, balance int NOT NULL);
      INSERT INTO accounts VALUES (1, 100), (2, 100);
      CREATE TABLE sessions (name text PRIMARY KEY, pid int NOT NULL);
    check: exact
    annotations:
      - lines: "2-3"
        note: >
          Process ids differ on every run, so each session records its own pid
          and the queries below print names instead. In production you work with
          the pids directly.
      - lines: "8-10"
        note: >
          idle in transaction: A has done its UPDATE, holds the row lock, and
          waits for its client (Client / ClientRead). query is the last statement
          it ran, not something running now. Minutes in this state is the most
          common cause of blocked writers and of VACUUM leaving dead rows behind.
      - lines: "13"
        note: >
          B needs the same row and is reported as waiting: from the
          application's side, just a slow query.
      - lines: "14-18"
        note: >
          B is active but waiting on a heavyweight Lock, on a transactionid: the
          lock is recorded in the row itself, so a waiter waits for the holding
          transaction to end. pg_blocking_pids names the blocker. The docs warn
          that it briefly locks the lock manager's state, so poll it with
          restraint.
      - lines: "19-21"
        note: >
          The same wait in pg_locks: a ShareLock on A's transaction id, not
          granted. Everything a session already holds shows granted = true.
      - lines: "22"
        note: >
          A commits and B's UPDATE completes at once. In an incident you end the
          blocker's transaction from its application, or with
          pg_terminate_backend(pid); idle_in_transaction_session_timeout does it
          automatically.
  - id: counters
    file: examples/02_counters.sql
    title: "Cumulative counters: sequential scans give a missing index away"
    runtime: sql
    check: exact
    annotations:
      - lines: "13-16"
        note: >
          pg_stat_xact_user_tables counts what this transaction did and updates
          continuously, which makes it usable in a script. The starting
          seq_scan of 1 reading 0 rows is the primary key's index build at
          CREATE TABLE: index builds read the table and are counted like any
          scan. ANALYZE is not.
      - lines: "19-21"
        note: >
          Two lookups, two full scans of 20,000 rows each for 40 matches. In
          pg_stat_user_tables the same pattern shows as a large
          seq_tup_read / seq_scan on a big table: the signature of a query that
          wants an index.
      - lines: "23-26"
        note: >
          One more seq_scan of 20,000 rows (the index build), then two index
          scans that fetched 80 rows in total. Counters count everything,
          maintenance included, so read them as rates and ratios over time, not
          as raw numbers.
      - lines: "29"
        note: >
          Still zero: a backend reports its counts to the shared statistics after
          the transaction ends (and at most about once a second), so the shared
          views lag behind. Don't use them as an assertion inside a script.
  - id: cancel
    file: examples/03_cancel.sql
    title: "Stopping a runaway statement: by hand, and by timeout"
    runtime: sql-multisession
    setup: |
      CREATE TABLE sessions (name text PRIMARY KEY, pid int NOT NULL);
    check: exact
    annotations:
      - lines: "4"
        note: >
          A long statement. After 400 ms the runner reports it as waiting and
          moves on, as you would switch to a second terminal.
      - lines: "5-6"
        note: >
          active, waiting on Timeout / PgSleep. A real runaway query shows an IO
          wait such as DataFileRead, or no wait event while it uses CPU.
          now() - query_start says how long it has run; it is left out here
          because it differs on every run.
      - lines: "7"
        note: >
          pg_cancel_backend stops the current statement and keeps the session;
          inside a transaction, that transaction is aborted and must be rolled
          back. It is allowed on your own role's sessions, or with
          pg_signal_backend, and never on a superuser's unless you are one. A's
          SQLSTATE 57014 follows. pg_terminate_backend ends the whole session
          instead, which is what releases an idle-in-transaction blocker.
      - lines: "10-11"
        note: >
          statement_timeout is the same cancellation without a human: SQLSTATE
          57014 again, with a different message. Set it per role or per
          transaction; the docs advise against postgresql.conf, where it would
          also hit maintenance and migrations.
      - lines: "12"
        note: >
          A cancel ends a statement, not the connection: the session carries on.
use_cases:
  - "Incident triage: something is slow or stuck, and you need to know within a minute whether it is a lock, a long transaction, I/O or one expensive query"
  - "Finding missing indexes (high seq_tup_read) and unused ones (idx_scan = 0) from the cumulative statistics"
  - "Watching migrations and maintenance: pg_stat_progress_create_index, pg_stat_progress_vacuum and the lock views while they run"
  - "Feeding an external monitoring system with periodic snapshots of the counters, read by a role in pg_monitor"
strengths:
  - "Everything is a view you can query with SQL, join and filter, unlike a fixed admin console"
  - "wait_event_type and wait_event say what a session is waiting for (a lock, the client, I/O, a timeout), which a thread dump in many runtimes doesn't"
  - "pg_blocking_pids answers 'who blocks whom' in one call instead of a self-join over pg_locks"
  - "Cumulative counters cost almost nothing to collect and are always on, so the history is there when you need it"
weaknesses:
  - "Cumulative views lag: counts arrive after a transaction ends, and within a transaction a statistics snapshot stays fixed until pg_stat_clear_snapshot()"
  - "Counters are totals since the last reset; you need periodic snapshots and differences to see a rate"
  - "Without pg_read_all_stats (or pg_monitor) you can't see other roles' query text: they show <insufficient privilege>"
  - "pg_stat_statements and auto_explain need preloading and a restart, so they must be in place before the incident"
pitfalls:
  - title: "Querying pg_stat_activity twice in one transaction"
    detail: "Activity data is collected at the first access in a transaction and then shown unchanged until it ends. A monitoring loop inside one transaction sees a frozen picture. Query in autocommit mode, or call pg_stat_clear_snapshot()."
  - title: "Killing the waiter instead of the blocker"
    detail: "Cancelling the session that waits only moves the queue. Follow pg_blocking_pids to the head of the chain, often an idle-in-transaction session, and end that one."
    good: "SELECT pid, pg_blocking_pids(pid) AS blocked_by, state, left(query, 60) FROM pg_stat_activity WHERE cardinality(pg_blocking_pids(pid)) > 0;"
  - title: "Monitoring as a superuser"
    detail: "A dashboard that logs in as postgres can do anything if its credentials leak. Give it a role with pg_monitor, which can read every statistic and setting but change nothing."
    good: "CREATE ROLE monitoring LOGIN PASSWORD '...'; GRANT pg_monitor TO monitoring;"
  - title: "Slow-query logs full of personal data"
    detail: "With log_min_duration_statement on, bind parameters are logged in full by default (log_parameter_max_length = -1): customer names, TCKNs and e-mail addresses end up in log files. Under KVKK those logs are personal data too; trim or disable parameter logging, or restrict and expire the logs."
    good: "log_parameter_max_length = 0  # postgresql.conf: no bind values in statement logs"
quiz:
  - q: "pg_stat_activity shows a session 'idle in transaction' for 20 minutes, and writers are piling up behind it. What is the usual fix?"
    options: ["pg_cancel_backend on the waiting writers", "End the idle transaction: fix the application, or pg_terminate_backend on that pid, and set idle_in_transaction_session_timeout", "Run VACUUM"]
    answer: 1
    why: "The idle session holds the locks; there is no running statement to cancel. Terminating it releases them; the timeout prevents a repeat."
  - q: "A session's wait_event_type is Lock and wait_event is transactionid. What is it waiting for?"
    options: ["Disk I/O", "Another transaction holding a row it wants to change, to commit or roll back", "Its client to send the next query"]
    answer: 1
    why: "Row locks are recorded in the rows; a waiter waits on the holding transaction's id, and pg_blocking_pids tells you which session that is."
  - q: "pg_stat_user_tables shows seq_scan = 90,000 and seq_tup_read = 4.5 billion for a 50,000-row table. What does it suggest?"
    options: ["Autovacuum is not running", "Queries keep scanning the whole table: a missing or unusable index", "The table is bloated"]
    answer: 1
    why: "About 50,000 rows read per scan, 90,000 times: a frequent query is filtering by full scans. pg_stat_statements then tells you which one."
  - q: "Inside one test transaction you DELETE rows and immediately query pg_stat_user_tables.n_tup_del, which still shows the old value. Why?"
    options: ["The statistics collector is disabled", "Counts reach the shared statistics only after the transaction ends; pg_stat_xact_user_tables shows the current transaction's", "DELETE isn't counted"]
    answer: 1
    why: "Backends flush their counts when they go idle, at most about once a second, so in-progress work isn't in the shared totals yet."
further_reading:
  - title: "PostgreSQL docs: The Cumulative Statistics System (pg_stat_activity, wait events, pg_stat_* views)"
    url: "https://www.postgresql.org/docs/current/monitoring-stats.html"
  - title: "PostgreSQL docs: Viewing Locks (pg_locks)"
    url: "https://www.postgresql.org/docs/current/monitoring-locks.html"
  - title: "PostgreSQL docs: Progress Reporting"
    url: "https://www.postgresql.org/docs/current/progress-reporting.html"
  - title: "PostgreSQL docs: Server Signaling Functions (pg_cancel_backend, pg_terminate_backend)"
    url: "https://www.postgresql.org/docs/current/functions-admin.html#FUNCTIONS-ADMIN-SIGNAL"
  - title: "PostgreSQL docs: Error Reporting and Logging (log_min_duration_statement, log_lock_waits)"
    url: "https://www.postgresql.org/docs/current/runtime-config-logging.html"
  - title: "PostgreSQL docs: auto_explain"
    url: "https://www.postgresql.org/docs/current/auto-explain.html"
---

## Concept

PostgreSQL exposes two kinds of monitoring data, both as views you query with SQL:

- **What is happening now.** `pg_stat_activity` has one row per server process: its
  `state` (`active`, `idle`, `idle in transaction`), what it is waiting for
  (`wait_event_type`, `wait_event`), when its transaction and its current statement
  started, and the statement text. `pg_locks` and `pg_blocking_pids()` show who waits
  for whom; the `pg_stat_progress_*` views show long operations such as `CREATE INDEX`
  and `VACUUM` as they run.
- **What has accumulated.** Counters since the last statistics reset: per query in
  `pg_stat_statements` (the indexes module), per table and index in
  `pg_stat_user_tables` and `pg_stat_user_indexes`, per database in `pg_stat_database`,
  and for I/O and checkpoints in `pg_stat_io` and `pg_stat_checkpointer`.

When it's slow, the first questions are, in order: is something blocked (locks,
idle-in-transaction sessions)? Is something running far too long? And over the last
hour, which queries and which tables did the most work?

## Coming from C/C++/Java/Python/JS

- **JVM thread dumps:** `pg_stat_activity` with `pg_blocking_pids` is a thread dump
  with the monitor owner filled in: every session, its state, and what it's blocked on.
- **Prometheus counters:** the cumulative views are counters in the same sense: totals
  that only mean something as a rate between two snapshots. Exporters do exactly that.
- **APM and profilers:** `pg_stat_statements` is the per-endpoint latency table, and
  `auto_explain` (logging the plan of every statement slower than a threshold) is the
  slow-trace sampler.
- **Cancelling work:** `pg_cancel_backend` is a `context.Context` cancel or an
  `AbortController` for one statement; the server checks for it while the statement
  runs and stops with SQLSTATE 57014.

## How it works

The first-look queries, for a role with `pg_monitor` (in this sandbox you see other
sessions' rows, but not their query text):

```sql
-- What is running or holding a transaction open, oldest transaction first
SELECT pid, usename, application_name, state, wait_event_type, wait_event,
       now() - xact_start AS xact_age, now() - query_start AS query_age,
       left(query, 80) AS query
FROM pg_stat_activity
WHERE backend_type = 'client backend' AND state <> 'idle'
ORDER BY xact_start NULLS LAST;

-- Who is blocked, and by whom
SELECT pid, pg_blocking_pids(pid) AS blocked_by, wait_event_type, wait_event,
       left(query, 60) AS query
FROM pg_stat_activity
WHERE cardinality(pg_blocking_pids(pid)) > 0;

-- Tables read by full scans, most rows first; dead rows and autovacuum
SELECT relname, seq_scan, seq_tup_read, idx_scan, n_live_tup, n_dead_tup,
       last_autovacuum, last_autoanalyze
FROM pg_stat_user_tables
ORDER BY seq_tup_read DESC
LIMIT 10;

-- Indexes no query has used since the last reset (unique ones enforce rules)
SELECT s.relid::regclass AS table_name, s.indexrelid::regclass AS index_name,
       pg_size_pretty(pg_relation_size(s.indexrelid)) AS size
FROM pg_stat_user_indexes s
JOIN pg_index i USING (indexrelid)
WHERE s.idx_scan = 0 AND NOT i.indisunique;

-- The database as a whole: cache hits, temp files, deadlocks, rollbacks
SELECT datname,
       round(100.0 * blks_hit / nullif(blks_hit + blks_read, 0), 2) AS cache_hit_pct,
       temp_files, pg_size_pretty(temp_bytes) AS temp_size,
       deadlocks, xact_commit, xact_rollback, stats_reset
FROM pg_stat_database
WHERE datname = current_database();
```

`blks_hit` counts only PostgreSQL's own buffer cache, so a low hit ratio can still be
served from the operating system's cache; temp files point back at `work_mem` (the
pooling lesson). PostgreSQL 18 added `total_vacuum_time` and friends to
`pg_stat_user_tables`, byte columns and WAL rows to `pg_stat_io`, and per-backend I/O and
WAL statistics (`pg_stat_get_backend_io`, `pg_stat_get_backend_wal`).

The server log is the other half. All of these are off by default and need a superuser
(or a granted `SET` privilege) to change: `log_min_duration_statement` logs every
statement slower than a threshold, `log_lock_waits` logs waits longer than
`deadlock_timeout`, `log_temp_files` logs temporary files, and the `auto_explain` module
logs the plans of slow statements. `log_checkpoints` is on by default, and
`log_autovacuum_min_duration` logs autovacuum runs over 10 minutes.

## Design notes

- **Name your connections.** Set `application_name` in every service's connection
  string (the pgx lesson shows how), so `pg_stat_activity` says which service a session
  belongs to. StackLab's runner doesn't: its sessions show up only as role `playground`
  with an empty `application_name`.
- **Guards before dashboards.** `statement_timeout`, `lock_timeout` and
  `idle_in_transaction_session_timeout` per role turn the classic incidents into errors
  in one request's log. The sandbox runs with exactly such guards: 5 s per statement and
  a 2 s lock timeout (4 s in multisession), set by `sqlrun` for every run.
- **Collect history.** The counters only become useful as rates. Snapshot them
  periodically (an exporter, or a cron job into a table), and keep `pg_stat_statements`
  and `track_io_timing` on, so the next incident starts with data.
- **Least privilege for monitoring.** A `pg_monitor` member reads every statistic and
  setting and can't change data; the roles lesson explains why that matters. Query text
  and logs can contain personal data, so who can read them is part of your KVKK
  measures too.
