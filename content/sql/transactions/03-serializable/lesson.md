---
id: sql.transactions.serializable
title: "Serializable isolation and retry loops"
track: sql
module: transactions
order: 3
level: advanced
minutes: 14
since: pg9.1
prerequisites: [sql.transactions.isolation-levels]
compare_with: [go.postgres.transactions, design.architecture.reliability, sql.transactions.row-locks]
tags: [transactions, serializable, ssi, write-skew, retries, multisession]
summary: >
  REPEATABLE READ still lets two transactions each make a correct decision and
  together break a rule (write skew). SERIALIZABLE detects that pattern without
  blocking anyone and rolls one transaction back with SQLSTATE 40001, so the
  application must retry it from the start.
examples:
  - id: write-skew
    file: examples/01_write_skew.sql
    title: "Write skew under REPEATABLE READ: both checks pass, the rule breaks"
    runtime: sql-multisession
    setup: |
      CREATE TABLE doctors (name text PRIMARY KEY, on_call boolean NOT NULL);
      INSERT INTO doctors VALUES ('alice', true), ('bob', true);
    check: exact
    annotations:
      - lines: "3-4"
        note: >
          Each transaction gets one snapshot, taken at its first query. This is
          what many databases call snapshot isolation, and it already prevents
          the non-repeatable read from the previous lesson.
      - lines: "7-8"
        note: >
          Both see two doctors on call. A plain read takes no lock and leaves
          no trace, so nothing records that A's decision depends on Bob's row.
      - lines: "11-14"
        note: >
          REPEATABLE READ fails a transaction only when it writes or locks a
          row that another transaction changed after its snapshot. A and B
          write different rows, so no step waits and both commits succeed.
      - lines: "17"
        note: >
          Each transaction was correct on its own; together they leave nobody
          on call. This is write skew, a serialization anomaly: the docs'
          isolation table lists it as possible under REPEATABLE READ.
  - id: serializable-retry
    file: examples/02_serializable_retry.sql
    title: "SERIALIZABLE aborts one of them with 40001; the retry decides again"
    runtime: sql-multisession
    setup: |
      CREATE TABLE doctors (name text PRIMARY KEY, on_call boolean NOT NULL);
      INSERT INTO doctors VALUES ('alice', true), ('bob', true);
    check: exact
    annotations:
      - lines: "4-5"
        note: >
          Under SERIALIZABLE each read also leaves an SIRead lock, a note of
          what was read (visible in pg_locks). It blocks nobody. This query
          scans the whole table, so the note covers the whole table.
      - lines: "6-7"
        note: >
          A writes a row B read, and B writes a row A read: a read/write
          dependency in each direction. Still nothing waits; the server only
          records the dependencies.
      - lines: "11-12"
        note: >
          No serial order of A and B can produce this result, so one of them
          must go. Here A commits first and B's COMMIT fails with 40001. In
          other interleavings the error can come at an earlier statement; the
          handling is the same. A failed COMMIT ends the transaction, so B can
          begin again at once.
      - lines: "15-18"
        note: >
          The retry runs the whole transaction again, check included, and the
          check now sees one doctor on call. Replaying only the UPDATE would
          reproduce the bug: the decision depends on the read, which is why
          the docs say to retry the complete transaction.
      - lines: "20"
        note: >
          The rule held, and no transaction ever waited for a lock. The price
          was one transaction's work thrown away, plus a retry loop in the
          application.
use_cases:
  - "Rules that span several rows and that no constraint can express: at least one on call, a budget spread across line items, a quota counted over many rows"
  - "Complex transactions where finding the exact rows to lock is error-prone"
  - "Teams that prefer one rule for every write path ('SERIALIZABLE plus a retry wrapper') to reasoning about locks in each one"
strengths:
  - "Nothing blocks: SIRead locks never make anyone wait and can't take part in a deadlock, unlike SELECT ... FOR UPDATE"
  - "Correctness doesn't depend on knowing which rows to lock; the database detects the dangerous pattern for you"
  - "Switching is one clause in BEGIN or one field in pgx.TxOptions; the SQL stays the same"
weaknesses:
  - "Every transaction needs a retry loop, and its side effects outside the database (e-mails, HTTP calls) must wait until after commit or be safe to repeat"
  - "False positives: a sequential scan predicate-locks the whole table, and fine-grained locks are combined into coarser ones when lock memory runs short, so transactions that didn't really conflict can fail"
  - "Under heavy contention many transactions fail and redo their work; a hot row is often better served by an explicit row lock"
  - "The guarantee covers SERIALIZABLE transactions only, and doesn't extend to hot standby or logical replicas"
pitfalls:
  - title: "Retrying the statement instead of the transaction"
    detail: "The failed transaction made decisions from data that has since changed. Re-run everything from BEGIN, reads included, so the decision is made again on fresh data. PostgreSQL offers no automatic retry for exactly this reason."
  - title: "Mixing isolation levels"
    detail: "The docs promise consistency when SERIALIZABLE is used for all writes and for all reads that need a consistent view. A READ COMMITTED writer on the same tables slips past the checks. Set default_transaction_isolation = 'serializable' for the application role rather than relying on every call site."
    good: "ALTER ROLE app SET default_transaction_isolation = 'serializable';"
  - title: "Long serializable transactions"
    detail: "SIRead locks are kept past commit until the overlapping transactions finish, so one long transaction raises the failure rate for everyone. Keep them short, and declare READ ONLY where you can."
  - title: "A long report that keeps failing"
    detail: "A read-only report can still be cancelled by a serialization failure. SERIALIZABLE READ ONLY DEFERRABLE waits for a safe snapshot when it starts, then runs with no risk of failure and without SSI overhead."
    good: "BEGIN ISOLATION LEVEL SERIALIZABLE, READ ONLY, DEFERRABLE;"
quiz:
  - q: "Two REPEATABLE READ transactions each count the on-call doctors, then each takes a different doctor off call. What happens?"
    options: ["The second UPDATE waits for the first transaction", "Both commit and nobody is on call", "The second COMMIT fails with 40001"]
    answer: 1
    why: "REPEATABLE READ only conflicts on rows both transactions write. Disjoint writes based on overlapping reads are write skew, which it allows."
  - q: "Under SERIALIZABLE, what does B's SELECT do to A's later UPDATE of a row B read?"
    options: ["It blocks the UPDATE until B commits", "Nothing visible: it leaves an SIRead lock that only matters for conflict detection", "It takes a FOR SHARE lock on the rows"]
    answer: 1
    why: "Predicate (SIRead) locks never block. They record reads so the server can spot read/write dependencies when the writes happen."
  - q: "B fails with SQLSTATE 40001. What should the application do?"
    options: ["Re-send the COMMIT", "Start a new transaction and run the whole unit of work again, reads and decisions included", "Retry the same transaction at READ COMMITTED"]
    answer: 1
    why: "The error means B's view of the data is no longer valid. Only re-running the logic on a new snapshot gives a correct decision."
  - q: "Why can a SERIALIZABLE transaction fail even though no other transaction wrote the rows it cares about?"
    options: ["It's a bug in SSI", "Predicate locks can be coarse: a sequential scan locks the whole table, and many fine locks can be combined into a page or table lock", "SERIALIZABLE takes table locks on every read"]
    answer: 1
    why: "The docs warn that relation-level predicate locks increase the failure rate. Detection errs on the safe side; indexes and short transactions reduce false positives."
further_reading:
  - title: "PostgreSQL docs: Transaction Isolation, Serializable"
    url: "https://www.postgresql.org/docs/current/transaction-iso.html#XACT-SERIALIZABLE"
  - title: "PostgreSQL docs: Serialization Failure Handling"
    url: "https://www.postgresql.org/docs/current/mvcc-serialization-failure-handling.html"
  - title: "PostgreSQL docs: Data Consistency Checks at the Application Level"
    url: "https://www.postgresql.org/docs/current/applevel-consistency.html"
  - title: "PostgreSQL docs: SET TRANSACTION (READ ONLY, DEFERRABLE)"
    url: "https://www.postgresql.org/docs/current/sql-set-transaction.html"
  - title: "pgx: BeginTxFunc and TxOptions"
    url: "https://pkg.go.dev/github.com/jackc/pgx/v5"
---

## Concept

REPEATABLE READ gives a transaction one snapshot and fails it if it writes a row that
someone changed after that snapshot. That prevents lost updates on the same row, but not
**write skew**: two transactions read overlapping data, each writes a *different* row, and
together they break a rule that each one checked.

SERIALIZABLE in PostgreSQL (true serializability since 9.1) is **Serializable Snapshot
Isolation (SSI)**. Transactions read from snapshots exactly as in REPEATABLE READ; in
addition the server records what each one read (SIRead or predicate locks) and notices when
another concurrent serializable transaction writes that data. When the reads and writes
form a pattern that no one-at-a-time order could produce, one transaction is rolled back
with SQLSTATE 40001 (`serialization_failure`).

Nothing waits. You pay with failed transactions instead, and the application must retry
them.

## Coming from C/C++/Java/Python/JS

- **Optimistic concurrency:** like a compare-and-swap loop in Go (`atomic.CompareAndSwapInt64`)
  or JPA's `@Version` check: do the work, detect the conflict at the end, retry. SSI goes
  further: it detects conflicts on data you only *read*, which a version column on the row
  you write can't see.
- **Python/JS/Go drivers and ORMs:** none of them retry for you, and neither does
  Postgres: the docs explain it can't, because the retry must re-run your decision logic.
- **Java `synchronized` / Go `sync.Mutex`:** pessimistic, you wait up front. SERIALIZABLE
  is the opposite trade: no waiting, occasional redo. Row locks (next lesson) are the
  pessimistic tool in SQL.

## How it works

In example 2, A read Bob's row and B later wrote it; B read Alice's row and A later wrote
it. Each of these is a *read/write dependency*: the reader must come before the writer in
any equivalent serial order. Two of them pointing in opposite directions form a cycle, so
no serial order exists and one transaction has to be rolled back.

SIRead locks live in shared memory, as tuple, page or relation locks. A sequential scan
always takes a relation-level one, and when memory for them runs short, many fine-grained
locks are combined into coarser ones. Coarser locks mean more transactions look like they
depend on each other, so some fail that were actually fine. The docs list the remedies:
declare `READ ONLY` where possible, keep transactions short, use a connection pool, help
the planner choose index scans, and raise `max_pred_locks_per_transaction` (and its
per-relation and per-page siblings) if promotion is the problem.

The two-session runner never shows `waiting` in this lesson: SSI never blocks.

## Design notes

Before reaching for SERIALIZABLE, check whether a **constraint** can state the rule:
uniqueness, a partial unique index ("one active subscription per user") or an exclusion
constraint ("no overlapping bookings") is enforced for every isolation level and never
needs a retry. "At least one doctor on call" can't be written as a constraint, so the
choice is between:

- **Explicit locks:** under READ COMMITTED, `SELECT name FROM doctors WHERE on_call FOR
  UPDATE` makes the second transaction wait, then return only the rows that still match,
  and decide on those. Correct, but you must find every code path that touches the rule.
- **SERIALIZABLE everywhere** plus one retry wrapper. Every path is covered, at the cost of
  occasional redo.

The wrapper, with pgx, in one place:

```go
// inSerializableTx runs fn in a SERIALIZABLE transaction and runs it again
// from the start when Postgres reports a serialization failure or a deadlock.
// fn may run several times: it must not have side effects outside the tx.
func inSerializableTx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	const maxAttempts = 5
	for attempt := 1; ; attempt++ {
		err := pgx.BeginTxFunc(ctx, pool, pgx.TxOptions{IsoLevel: pgx.Serializable}, fn)
		var pgErr *pgconn.PgError
		retryable := errors.As(err, &pgErr) &&
			(pgErr.Code == "40001" || pgErr.Code == "40P01") // serialization_failure, deadlock_detected
		if !retryable || attempt == maxAttempts {
			return err
		}
		backoff := time.Duration(attempt) * 20 * time.Millisecond
		select {
		case <-time.After(backoff + rand.N(backoff)): // jitter: don't collide again in lockstep
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
```

`BeginTxFunc` commits when `fn` returns nil and rolls back otherwise; a 40001 raised at
`COMMIT` comes back as its error too. Wrap errors inside `fn` with `%w`, or `errors.As`
won't find the `*pgconn.PgError`. The retry must contain the reads: if `fn` receives values
read before the transaction began, retrying it decides on stale data again.
