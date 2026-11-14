---
id: sql.transactions.deadlocks-locks
title: "Deadlocks (cause and prevention via lock ordering), advisory locks"
track: sql
module: transactions
order: 6
level: advanced
minutes: 13
since: pg9.1
prerequisites: [sql.transactions.row-locks]
compare_with: [go.concurrency.sync, sql.transactions.serializable, go.postgres.migrations-sqlc]
tags: [transactions, deadlocks, lock-ordering, advisory-locks, multisession]
summary: >
  Two transactions that lock the same rows in opposite order deadlock;
  Postgres detects the cycle after deadlock_timeout and aborts one with 40P01.
  Locking in a consistent order prevents it. Advisory locks serialize work
  that has no row to lock.
examples:
  - id: deadlock
    file: examples/01_deadlock.sql
    title: "Opposite transfers deadlock, and Postgres breaks the cycle"
    runtime: sql-multisession
    setup: |
      CREATE TABLE accounts (id text PRIMARY KEY, balance int NOT NULL);
      INSERT INTO accounts VALUES ('a', 100), ('b', 100);
    check: exact
    annotations:
      - lines: "5-6"
        note: >
          Each UPDATE locks its source row (FOR NO KEY UPDATE) until its
          transaction ends: A holds a, B holds b.
      - lines: "9-10"
        note: >
          How the runner shows a wait: a step still running after 400 ms is
          printed as "waiting" and the next step is sent; the result appears
          later under the same number. A now waits for b, held by B; B waits
          for a, held by A. Neither wait can ever end on its own.
      - lines: "12-14"
        note: >
          A waiting session checks for a deadlock after deadlock_timeout (1 s
          by default). A started waiting first, so its check runs first, finds
          the cycle, and A's UPDATE fails with 40P01; its transaction is
          aborted until this ROLLBACK. The docs warn that which transaction is
          aborted is hard to predict; this runner just makes it repeatable.
      - lines: "15-16"
        note: >
          A's abort released its lock, so B's update went through and B
          commits: a = 120, b = 80. A's transfer is simply gone; the
          application has to retry it (40P01 is a retryable error).
  - id: lock-ordering
    file: examples/02_lock_ordering.sql
    title: "Locking both rows in key order turns the deadlock into a short wait"
    runtime: sql-multisession
    setup: |
      CREATE TABLE accounts (id text PRIMARY KEY, balance int NOT NULL);
      INSERT INTO accounts VALUES ('a', 100), ('b', 100);
    check: exact
    annotations:
      - lines: "5"
        note: >
          A locks a, then b, before changing anything. ORDER BY fixes the lock
          order, and FOR NO KEY UPDATE is the mode the UPDATEs need anyway, so
          the first lock taken is already the strongest one needed, as the
          docs recommend.
      - lines: "6"
        note: >
          B asks in the same order, so it waits at row a while holding
          nothing. A wait without a cycle just ends when the holder commits.
      - lines: "9-11"
        note: >
          A's updates find their rows already locked by A itself: no waiting.
          COMMIT releases both, and B's lock request is granted.
      - lines: "13-16"
        note: >
          B's transfer runs after A's, and both are applied: a = 110,
          b = 90. Same work as example 1, no deadlock, no retry.
  - id: advisory
    file: examples/03_advisory.sql
    title: "Advisory locks: transaction-level and session-level"
    runtime: sql-multisession
    check: exact
    annotations:
      - lines: "6-7"
        note: >
          The try variants answer at once. B's false means "someone else is
          running the report": skip it. The variants without try wait, like
          a row lock, and take part in deadlock detection like one.
      - lines: "8-9"
        note: >
          COMMIT released A's lock. B's statement runs in autocommit, so its
          transaction-level lock lasted exactly one statement: it got the
          lock and released it again.
      - lines: "13-16"
        note: >
          A session-level lock ignores transactions: the docs say a lock taken
          in a transaction that later rolls back is still held after the
          rollback. B still can't get it.
      - lines: "17-19"
        note: >
          It ends only with an explicit unlock (one per lock call: they stack)
          or when the connection closes. With a connection pool, a forgotten
          unlock stays with the pooled connection, not with your request.
use_cases:
  - "Transfers and any transaction that updates several rows of one table: lock them in key order first"
  - "Run-once work across many app instances: one nightly report, one migration run, one cache rebuild (advisory locks)"
  - "Serializing work on something that has no row yet, such as 'one import per customer at a time' before the import row exists"
strengths:
  - "Deadlocks are detected and broken automatically, unlike two Go goroutines locking mutexes in opposite order, which hang until something kills them"
  - "The victim gets its own SQLSTATE (40P01), which is safe to retry"
  - "Advisory locks are cheap, need no table and no cleanup: the server releases them at transaction or session end"
  - "Advisory waits take part in the same deadlock detection as row and table locks"
weaknesses:
  - "Nothing happens until deadlock_timeout (1 s by default) has passed, and the aborted transaction's work is lost"
  - "You don't choose which transaction is aborted; the docs say not to rely on it"
  - "Advisory keys are bare numbers shared by every session in the database: nothing stops two features from picking the same one"
  - "Session-level advisory locks ignore transactions and can outlive the request that took them when connections are pooled"
pitfalls:
  - title: "Locking rows in the order the request lists them"
    detail: "A transfer from a to b and one from b to a lock in opposite order. Sort the keys and lock them in one statement before writing."
    bad: "UPDATE accounts SET balance = balance - $3 WHERE id = $1;\nUPDATE accounts SET balance = balance + $3 WHERE id = $2;"
    good: "SELECT id FROM accounts WHERE id = ANY($1) ORDER BY id FOR NO KEY UPDATE;\n-- then both UPDATEs, in any order"
  - title: "Trusting a multi-row UPDATE to lock in a sensible order"
    detail: "UPDATE ... WHERE id IN (...) locks rows in whatever order the plan visits them, which can differ between two statements. Lock explicitly with ORDER BY first when several transactions update overlapping sets."
  - title: "Session-level advisory locks with a pool"
    detail: "pg_advisory_lock outlives COMMIT and ROLLBACK. If the code that should unlock never runs, the pooled connection keeps the lock and the next request that borrows it holds a lock it doesn't know about. Prefer the _xact_ variants; if you need session-level, unlock in a defer on the same connection."
    bad: "SELECT pg_advisory_lock(4242);  -- through a pool, unlock in some other request"
    good: "BEGIN;\nSELECT pg_try_advisory_xact_lock(4242);\n-- ... work ...\nCOMMIT;  -- releases it"
  - title: "Calling lock functions in a query with LIMIT"
    detail: "The docs' example: SELECT pg_advisory_lock(id) FROM foo WHERE id > 12345 LIMIT 100 may lock more ids than it returns, because LIMIT isn't guaranteed to apply before the function runs. Select the ids in a subquery with the LIMIT, and lock in the outer query."
quiz:
  - q: "Transaction A updates row a then b; B updates b then a, concurrently. What happens?"
    options: ["Both wait until lock_timeout", "After deadlock_timeout, one is aborted with 40P01 and the other continues", "Postgres reorders the updates"]
    answer: 1
    why: "The waiting sessions run the deadlock check after deadlock_timeout; one transaction is aborted so the other can proceed."
  - q: "What prevents the deadlock?"
    options: ["A longer deadlock_timeout", "Every transaction locks the rows it needs in the same order, for example by key, before writing", "Running both at SERIALIZABLE"]
    answer: 1
    why: "A cycle needs two transactions that each hold a lock the other wants; a single global order makes that impossible."
  - q: "A session runs BEGIN; SELECT pg_advisory_lock(7); ROLLBACK;. Is lock 7 still held?"
    options: ["No, ROLLBACK released it", "Yes: session-level advisory locks ignore transactions", "Only until the next statement"]
    answer: 1
    why: "Session-level locks are held until pg_advisory_unlock or the end of the session; only the _xact_ variants follow the transaction."
  - q: "Five app instances run the same nightly job; exactly one should do it and the others skip. Which call fits?"
    options: ["pg_advisory_lock(42) at startup", "pg_try_advisory_xact_lock(42) at the start of the job's transaction, skipping on false", "SELECT ... FOR UPDATE NOWAIT on a random row"]
    answer: 1
    why: "The try variant doesn't wait, and the transaction-level lock is released automatically when the job's transaction ends, even if the instance crashes."
further_reading:
  - title: "PostgreSQL docs: Explicit Locking (Deadlocks, Advisory Locks)"
    url: "https://www.postgresql.org/docs/current/explicit-locking.html"
  - title: "PostgreSQL docs: Advisory Lock Functions"
    url: "https://www.postgresql.org/docs/current/functions-admin.html#FUNCTIONS-ADVISORY-LOCKS"
  - title: "PostgreSQL docs: Lock Management (deadlock_timeout)"
    url: "https://www.postgresql.org/docs/current/runtime-config-locks.html"
  - title: "PostgreSQL docs: Serialization Failure Handling (retrying 40P01)"
    url: "https://www.postgresql.org/docs/current/mvcc-serialization-failure-handling.html"
---

## Concept

A **deadlock** is a cycle of waits: A holds a lock B wants while B holds a lock A wants.
Row locks are enough to form one, without any explicit locking: two transfers that update
the same two accounts in opposite order will do it.

Postgres doesn't let it hang. A waiting session checks for a cycle after
`deadlock_timeout` (1 second by default), and if it finds one, one transaction is aborted
with SQLSTATE 40P01 (`deadlock_detected`) so the others can continue. The aborted work is
lost and has to be retried.

The fix is ordering: if every transaction acquires the locks it needs in the same order,
no cycle can form. The docs add a second rule: take the strongest lock you will need on an
object first, rather than upgrading later.

**Advisory locks** are locks on numbers rather than rows, for coordinating things the
database doesn't model: "only one instance runs the nightly report", "one import per
customer at a time". They come in transaction-level (`pg_advisory_xact_lock`, released at
commit or rollback) and session-level (`pg_advisory_lock`, held until unlocked or
disconnect) forms, each with a `try` variant that returns false instead of waiting.

## Coming from C/C++/Java/Python/JS

- **Go mutexes:** the same rule you'd follow with two `sync.Mutex`es (always lock them in
  one order) applies to rows. The difference: Go doesn't detect a deadlock between two
  goroutines while others still run; the runtime only reports one when every goroutine is
  asleep. Postgres detects any cycle among waiting transactions.
- **Java:** `jstack` reports deadlocked threads but doesn't break the cycle. Postgres
  breaks it, and that turns your deadlock into an error your code must handle.
- **Distributed locks (Redis, ZooKeeper):** an advisory lock is a lock service you
  already run. Its lifetime is tied to a transaction or a connection, so a crashed client
  can't leave it held; there's no TTL to tune.

## How it works

Waiting is cheap and optimistic: a blocked session sleeps on the lock, and only after
`deadlock_timeout` does it run the (relatively expensive) check of the wait-for graph.
Only superusers, or roles granted the right to set it, can change `deadlock_timeout`; it
isn't a tuning knob for application code.

Advisory locks live in the same shared-memory lock table as table locks, so they show up
in `pg_locks` with `locktype = 'advisory'`, and the table is finite (sized by
`max_locks_per_transaction` and `max_connections`): don't take millions of them. The
`bigint` key and the `(int, int)` key pair are separate key spaces; the pair is handy for
"(kind of thing, id)".

## Design notes

Deadlock prevention, in order of preference:

1. **Fewer locks per transaction.** A transfer written as a single statement still locks
   two rows, but short transactions leave less room for interleaving.
2. **A global lock order.** Sort ids before locking (`slices.Sort(ids)` in Go, then `WHERE
   id = ANY($1) ORDER BY id FOR NO KEY UPDATE`). Across tables, pick an order too, for
   example parent before child.
3. **Retry 40P01** like 40001, in the same wrapper (the serializable lesson's
   `inSerializableTx` already does). Deadlocks you didn't foresee will still happen.

Advisory locks: prefer the transaction-level `try` variant around a unit of work, so a
crash or a panic can't leave the lock behind. Keep the keys in one place in the code (a
`const` block), since collisions are silent. The session-level form is for work that must
stay exclusive across several transactions, such as a migration run that commits each
step: take it on one dedicated connection and unlock in a `defer`.
