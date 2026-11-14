---
id: sql.transactions.isolation-levels
title: "Isolation levels with live anomalies"
track: sql
module: transactions
order: 2
level: advanced
minutes: 14
since: pg1.0
prerequisites: [sql.transactions.acid-mvcc, sql.querying.null-logic]
compare_with: [go.concurrency.select, design.paradigms.concurrency, ts.async.combinators]
tags: [transactions, isolation, mvcc, multisession]
summary: >
  Under READ COMMITTED (the default), another transaction's commit can change
  what your next query sees. This lesson runs two sessions side by side so you
  watch the anomaly happen, then prevent it.
examples:
  - id: non-repeatable-read
    file: examples/01_anomaly.sql
    title: Session B's read changes between statements in one transaction
    runtime: sql-multisession
    setup: |
      CREATE TABLE accounts (id text PRIMARY KEY, balance int NOT NULL);
      INSERT INTO accounts VALUES ('a', 100), ('b', 0);
    check: exact
    annotations:
      - lines: "5-6"
        note: >
          Session A opens a transaction and reads a's balance: 100. Under READ
          COMMITTED, each statement takes a fresh snapshot.
      - lines: "9-11"
        note: >
          Session B, in its own transaction, moves 40 out and commits. No
          conflict: A only read.
      - lines: "14"
        note: >
          A reads again inside the same transaction and sees 60. That is a
          non-repeatable read: one query, two answers.
      - lines: "18-19"
        note: >
          REPEATABLE READ takes one snapshot for the whole transaction, at its
          first statement rather than at BEGIN. A now reads 60.
      - lines: "21-25"
        note: >
          B commits again (60 becomes 20), yet A's re-read still returns 60: the
          snapshot doesn't move. SERIALIZABLE goes further and aborts one
          transaction when concurrent reads and writes could not have happened
          in any serial order; that is the next lesson.
use_cases:
  - "Reports that must see one consistent snapshot (REPEATABLE READ)"
  - "Transfers and invariants that reads alone must not violate (SELECT ... FOR UPDATE, next lesson)"
strengths:
  - "Under MVCC plain reads never wait for writes and writes never wait for reads; only two writers on the same row wait for each other"
  - "Escalating isolation is one line — no code restructure"
  - "Anomalies are teachable because each level forbids a named, testable behavior"
weaknesses:
  - "READ COMMITTED makes every statement its own consistency unit: multi-statement invariants leak"
  - "REPEATABLE READ/SERIALIZABLE can fail with a serialization error and need a retry loop"
pitfalls:
  - title: "Re-reading inside a transaction and assuming consistency"
    detail: "Under READ COMMITTED each statement gets a fresh snapshot. Use one snapshot (REPEATABLE READ) or lock rows."
  - title: "Retrying nothing under SERIALIZABLE"
    detail: "Serializable failures (40001) are normal: catch and retry the transaction."
quiz:
  - q: "Under READ COMMITTED, when does A's snapshot refresh?"
    options: ["Once per transaction", "Once per statement", "Never"]
    answer: 1
    why: "Read committed takes a new snapshot for each statement — that's exactly the anomaly source."
  - q: "Which level prevents the non-repeatable read above?"
    options: ["READ UNCOMMITTED", "REPEATABLE READ", "It can't be prevented"]
    answer: 1
    why: "A transaction-pinned snapshot returns the same rows it read before."
further_reading:
  - title: "PostgreSQL docs: Transaction Isolation"
    url: "https://www.postgresql.org/docs/current/transaction-iso.html"
  - title: "PostgreSQL docs: MVCC"
    url: "https://www.postgresql.org/docs/current/mvcc.html"
---

## Concept

Postgres implements isolation with MVCC: writers create new row versions, readers see
snapshots. READ COMMITTED (default) takes a snapshot per statement; REPEATABLE READ and
SERIALIZABLE pin one snapshot per transaction and reject what would break it.

## Coming from C/C++/Java/Python/JS

- Mutexes: MVCC is "readers never block writers" — no lock table for plain reads.
- Goroutines/JS: the interleaving is real concurrency, but the *database defines* which
  interleavings are legal, per isolation level.
- Go memory model: think of isolation levels as happens-before guarantees you opt into.

## Design notes

Choose isolation per transaction, not per database: most traffic is fine at READ COMMITTED,
and the two or three flows with invariants (transfers, stock) either lock rows (FOR UPDATE)
or escalate to REPEATABLE READ/SERIALIZABLE with retries. The multi-session runner exists
because these behaviors only become visible — and only become trusted — when you watch both
sessions in a timeline.
