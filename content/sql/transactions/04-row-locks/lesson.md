---
id: sql.transactions.row-locks
title: "Row locks: FOR UPDATE, FOR NO KEY UPDATE, NOWAIT"
track: sql
module: transactions
order: 4
level: advanced
minutes: 14
since: pg10
prerequisites: [sql.transactions.isolation-levels, sql.constraints.fk-check]
compare_with: [go.concurrency.sync, sql.transactions.serializable, sql.modifying.returning-upsert]
tags: [transactions, locking, for-update, for-no-key-update, nowait, lock-timeout, multisession]
summary: >
  When application code reads a row, decides, and writes it back, a concurrent
  transaction can overwrite the result. SELECT ... FOR UPDATE serializes exactly
  those rows; FOR NO KEY UPDATE does it without blocking foreign-key inserts,
  and NOWAIT or lock_timeout bound the wait.
examples:
  - id: lost-update
    file: examples/01_lost_update.sql
    title: "A lost update, then the same withdrawals with FOR UPDATE"
    runtime: sql-multisession
    setup: |
      CREATE TABLE accounts (id text PRIMARY KEY, balance int NOT NULL);
      INSERT INTO accounts VALUES ('a', 100), ('b', 100);
    check: exact
    annotations:
      - lines: "5-6"
        note: >
          Both read 100. A plain read locks nothing, so nothing stops the
          other transaction from changing the row before you write.
      - lines: "7-10"
        note: >
          Each writes a constant its application computed from a stale read.
          B's UPDATE has nothing to wait for (A already committed) and nothing
          to re-check: SET balance = 50 doesn't depend on the current value.
          A's withdrawal of 30 is overwritten.
      - lines: "15-16"
        note: >
          FOR UPDATE takes the row lock at read time. How the runner shows a
          wait: B's read is still running after 400 ms, so it is printed as
          "waiting" and the script moves on to A's steps; B's result appears,
          under the same step number, as soon as A's COMMIT releases the lock.
      - lines: "17-20"
        note: >
          Under READ COMMITTED, a FOR UPDATE that waited returns the row as
          the other transaction committed it: 70, not 100. B's decision is
          made on current data.
      - lines: "24"
        note: >
          Account a ends at 50 instead of 20; account b, with the same two
          withdrawals, ends at 20.
  - id: key-share
    file: examples/02_key_share.sql
    title: "FOR UPDATE blocks inserts of child rows; FOR NO KEY UPDATE doesn't"
    runtime: sql-multisession
    setup: |
      CREATE TABLE accounts (id text PRIMARY KEY, balance int NOT NULL);
      CREATE TABLE payments (
        id         int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
        account_id text NOT NULL REFERENCES accounts,
        amount     int NOT NULL
      );
      INSERT INTO accounts VALUES ('a', 100);
    check: exact
    annotations:
      - lines: "1-3"
        note: >
          The foreign-key check of an INSERT into payments locks the
          referenced account row in KEY SHARE mode, the weakest of the four:
          it only conflicts with deleting the row or changing its key.
      - lines: "6-8"
        note: >
          An UPDATE that doesn't change a key column takes FOR NO KEY UPDATE,
          which doesn't conflict with KEY SHARE. B's payment goes in while A's
          update is uncommitted.
      - lines: "13-14"
        note: >
          FOR UPDATE conflicts with all four modes, so the payment insert
          waits for A to finish, although A never meant to delete the account
          or change its id.
      - lines: "20-21"
        note: >
          FOR NO KEY UPDATE is the lock A's UPDATE will take anyway. Taking it
          at read time protects the read-modify-write just as well, and
          inserts of child rows keep flowing.
  - id: nowait
    file: examples/03_nowait.sql
    title: "NOWAIT and lock_timeout: fail fast instead of queueing"
    runtime: sql-multisession
    setup: |
      CREATE TABLE accounts (id text PRIMARY KEY, balance int NOT NULL);
      INSERT INTO accounts VALUES ('a', 100);
    check: exact
    annotations:
      - lines: "6"
        note: >
          NOWAIT applies to the row locks of this one statement: it fails with
          SQLSTATE 55P03 (lock_not_available) at once, so the caller can
          answer "busy, try again" instead of hanging.
      - lines: "10-11"
        note: >
          lock_timeout bounds every lock wait, including the implicit row lock
          of a plain UPDATE, which has no NOWAIT clause. Same SQLSTATE, after
          100 ms, still inside the runner's 400 ms window, so no "waiting"
          line. B runs in autocommit, so the error doesn't leave a
          transaction to roll back.
      - lines: "14"
        note: >
          Row locks never block plain reads: B gets the last committed value,
          100, while A still holds the lock.
use_cases:
  - "Read-modify-write when the new value is computed in application code: balances, stock levels, seat counts"
  - "Check-then-act on one row: 'if the order is still pending, ship it'"
  - "Endpoints that should answer 'busy, try again' instead of queueing (NOWAIT, lock_timeout)"
strengths:
  - "Locks only the rows the query returns, and never blocks plain readers, unlike a Java synchronized method that serializes every caller"
  - "No lock table to fill up: row locks are recorded in the row itself, so there's no limit on how many rows one transaction can lock"
  - "Four strengths let you lock exactly as much as you need; foreign-key checks use the weakest, so ordinary updates don't block child inserts"
  - "Waits can be bounded per statement (NOWAIT) or per lock (lock_timeout), and the error has its own SQLSTATE, 55P03"
weaknesses:
  - "Locks are held until the transaction ends; one slow transaction stalls everyone queued behind it"
  - "Locking a row writes to it (the lock is marked in the row), so SELECT ... FOR UPDATE on many rows causes disk writes"
  - "Transactions that lock the same rows in different orders can deadlock (lesson 6)"
  - "Under REPEATABLE READ or SERIALIZABLE, locking a row that changed after your snapshot raises 40001 instead of returning the new version"
pitfalls:
  - title: "Reading, then writing a value computed from the read"
    detail: "If the new value can be computed in SQL, one UPDATE is atomic and needs no explicit lock: a blocked UPDATE re-evaluates on the newest committed version. Use FOR UPDATE when the decision really needs application code."
    bad: "SELECT balance FROM accounts WHERE id = 'a';            -- app computes 100 - 50\nUPDATE accounts SET balance = 50 WHERE id = 'a';"
    good: "UPDATE accounts SET balance = balance - 50\nWHERE id = 'a' AND balance >= 50\nRETURNING balance;"
  - title: "FOR UPDATE where FOR NO KEY UPDATE would do"
    detail: "FOR UPDATE also blocks inserts of rows that reference the locked row, because their foreign-key check needs KEY SHARE. Unless you will delete the row or change its key, lock FOR NO KEY UPDATE."
    bad: "SELECT * FROM orders WHERE id = $1 FOR UPDATE;"
    good: "SELECT * FROM orders WHERE id = $1 FOR NO KEY UPDATE;"
  - title: "Expecting FOR UPDATE to stop readers"
    detail: "A plain SELECT reads the last committed version and never waits. If another code path decides on the same row, it must lock too, or put its condition into its UPDATE's WHERE clause."
  - title: "No bound on lock waits in request handlers"
    detail: "Without a limit a request queues behind a lock for as long as the holder runs. Set lock_timeout for the transaction and turn SQLSTATE 55P03 into a retry or an HTTP 409/503."
    good: "BEGIN;\nSET LOCAL lock_timeout = '2s';\n-- ...\nCOMMIT;"
quiz:
  - q: "A holds FOR UPDATE on account a. B runs SELECT balance FROM accounts WHERE id = 'a' with no locking clause. What happens?"
    options: ["B waits until A ends", "B immediately gets the last committed balance", "B fails with 55P03"]
    answer: 1
    why: "Row locks block writers and lockers only; plain reads use their snapshot."
  - q: "id is the primary key and balance has no index. Which row lock does UPDATE accounts SET balance = 0 WHERE id = 'a' take?"
    options: ["FOR UPDATE", "FOR NO KEY UPDATE", "FOR SHARE", "None"]
    answer: 1
    why: "An UPDATE takes FOR UPDATE only when it changes a column with a unique index usable by a foreign key; otherwise FOR NO KEY UPDATE."
  - q: "Under READ COMMITTED, B's SELECT ... FOR UPDATE waited while A changed the balance from 100 to 70 and committed. What does B's query return?"
    options: ["100, the value when B's statement started", "70, the committed version", "An error, because the row changed"]
    answer: 1
    why: "After the wait it locks and returns the updated row. Under REPEATABLE READ it would fail with 40001 instead."
  - q: "What is the difference between NOWAIT and lock_timeout = '100ms'?"
    options: ["NOWAIT fails at once on a locked row in that statement's locking clause; lock_timeout lets any lock wait up to the limit", "They are the same", "NOWAIT skips locked rows"]
    answer: 0
    why: "NOWAIT is a clause on one SELECT; lock_timeout is a setting that applies to every lock acquisition, including UPDATE's implicit row lock. Skipping is SKIP LOCKED (next lesson)."
further_reading:
  - title: "PostgreSQL docs: Row-Level Locks"
    url: "https://www.postgresql.org/docs/current/explicit-locking.html#LOCKING-ROWS"
  - title: "PostgreSQL docs: SELECT, The Locking Clause"
    url: "https://www.postgresql.org/docs/current/sql-select.html"
  - title: "PostgreSQL docs: Enforcing Consistency with Explicit Blocking Locks"
    url: "https://www.postgresql.org/docs/current/applevel-consistency.html"
  - title: "PostgreSQL docs: lock_timeout and other client settings"
    url: "https://www.postgresql.org/docs/current/runtime-config-client.html"
---

## Concept

MVCC lets readers and writers ignore each other. That's exactly wrong for a
**read-modify-write** done in application code: read a balance, compute in Go, write the
result. Two such transactions read the same value, and the second write silently replaces
the first. This is the **lost update**.

A **row lock** makes the read part of the critical section. `SELECT ... FOR UPDATE` locks
the rows it returns until the transaction ends; a second locker waits, then (under READ
COMMITTED) gets the committed new version. Plain readers are never affected.

There are four strengths, strongest first. Writes take one implicitly; `SELECT` takes one
when you add the clause:

- **`FOR UPDATE`**: taken by `DELETE` and by an `UPDATE` that changes a key column.
  Conflicts with all four modes.
- **`FOR NO KEY UPDATE`**: taken by every other `UPDATE`. Conflicts with all but
  `FOR KEY SHARE`.
- **`FOR SHARE`**: a shared lock; conflicts with the two exclusive modes above.
- **`FOR KEY SHARE`**: taken by foreign-key checks. Conflicts only with `FOR UPDATE`.

"Key column" means a column with a unique index usable by a foreign key. To bound waiting,
`NOWAIT` makes one statement fail immediately and `lock_timeout` caps every lock wait;
both raise SQLSTATE 55P03.

## Coming from C/C++/Java/Python/JS

- **Mutexes:** `FOR UPDATE` is a mutex per row, created on demand and released at
  `COMMIT` or `ROLLBACK` (or `ROLLBACK TO` a savepoint taken before it); there's no
  unlock call. Plain `SELECT`s don't take it, so only the code paths that ask for the
  lock respect it.
- **Java `synchronized` / Go `sync.Mutex` in the service:** stops working as soon as you
  run two instances. The row lock lives in the database, where every instance meets.
- **ORM pessimistic locking (JPA, Django, SQLAlchemy):** these APIs emit `SELECT ... FOR
  UPDATE` or one of its variants. Log the SQL once to see which mode you actually get.
- **Go `TryLock`:** `NOWAIT` is the database's `sync.Mutex.TryLock`, except that it reports
  failure as an error, not a boolean.

## How it works

Row locks aren't kept in a lock table: the locking transaction marks the row itself (its
`xmax`, from the first lesson), which is why there's no limit on how many rows you can lock
and why locking can cause disk writes. A transaction that wants a locked row waits for the
*holder's transaction* to end. To see who blocks whom, ask for each waiting session's
blockers:

```sql
SELECT pid, pg_blocking_pids(pid) AS blocked_by, wait_event_type, query
FROM pg_stat_activity
WHERE cardinality(pg_blocking_pids(pid)) > 0;
```

PostgreSQL 18 adds `log_lock_failures`, which logs `SELECT ... NOWAIT` lock failures, if
you need to see how often your fail-fast paths fire.

## Design notes

Pick the narrowest tool that's correct:

1. **One statement**, when the new value can be computed in SQL: `UPDATE ... SET balance =
   balance - 50 WHERE balance >= 50 RETURNING balance`. Atomic, no explicit lock.
2. **`FOR NO KEY UPDATE`** on the rows you'll update, when the decision needs application
   code. Use `FOR UPDATE` only when you'll delete the row or change its key.
3. **SERIALIZABLE** (previous lesson), when the rule spans rows you can't enumerate up
   front.

Lock late and briefly: read everything that needs no lock first, lock, decide, write,
commit. Every millisecond between the lock and the commit is a millisecond everyone else
waits. And put a bound on that waiting on every user-facing path: `SET LOCAL
lock_timeout` in the transaction, plus a mapping from 55P03 to a clear error.
