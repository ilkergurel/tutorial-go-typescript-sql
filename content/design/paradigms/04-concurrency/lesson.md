---
id: design.paradigms.concurrency
title: "The same problem: concurrency (goroutines vs the event loop vs MVCC)"
track: design
module: paradigms
order: 4
level: advanced
minutes: 15
since: go1.25
prerequisites: [design.paradigms.side-by-side, go.concurrency.sync, ts.async.event-loop, sql.transactions.acid-mvcc]
compare_with: [go.concurrency.goroutines-channels, go.concurrency.sync, go.concurrency.errgroup, ts.async.event-loop, ts.async.combinators, sql.transactions.acid-mvcc, sql.transactions.isolation-levels, sql.transactions.row-locks]
tags: [concurrency, goroutines, mutex, event-loop, mvcc, lost-update, check-then-act, comparison]
summary: >
  Twenty buyers, ten items: Go guards the counter with a mutex, TypeScript hides the bug
  behind an await on one thread, PostgreSQL fixes it with the check inside the UPDATE.
  Same bug, same cure in all three: make the check and the write one step.
examples:
  - id: go-goroutines
    file: examples/01_goroutines.go
    title: "Go: parallel goroutines, a mutex around check and write"
    runtime: go
    check: exact
    annotations:
      - lines: "9-12"
        note: >
          The mutex sits next to the state it guards, unexported, and only
          Reserve touches it. Callers can't forget to lock what they can't reach.
      - lines: "16-24"
        note: >
          The check and the decrement form one critical section. Goroutines run
          in parallel on several threads and can be preempted almost anywhere, so
          without the lock two of them could both see left == 1: a data race,
          which `go run -race` reports (the race detector needs cgo and, except
          on macOS, a C compiler).
      - lines: "28"
        note: >
          Results travel over a channel instead of a second shared counter:
          "channels orchestrate, mutexes serialize". The buffer of 20 means no
          buyer blocks on sending.
      - lines: "31-33"
        note: >
          wg.Go (Go 1.25) starts each buyer and counts it. Which buyers win
          changes from run to run; how many never does, which is why the output
          prints counts only.
  - id: ts-event-loop
    file: examples/02_event_loop.ts
    title: "TypeScript: one thread, and a race across await"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "8-13"
        note: >
          No data race is possible here: one thread runs one task at a time.
          But await ends the synchronous step, all twenty buyers pass the check
          before the first decrement runs, and stock goes to -10. Nothing
          failed; nothing warned.
      - lines: "16-21"
        note: >
          The fix needs no lock: code between two awaits can't be interrupted, so
          checking and decrementing in the same step is atomic. The cost moves to
          failure handling: a failed payment must put the item back.
      - lines: "25"
        note: >
          Promise.all starts all twenty at once: concurrency without parallelism.
          They interleave only at awaits, which is why the output is the same on
          every run (lesson ts.async.event-loop shows the queue order).
  - id: sql-mvcc
    file: examples/03_mvcc.sql
    title: "PostgreSQL: snapshots for readers, row locks for writers"
    runtime: sql-multisession
    check: exact
    annotations:
      - lines: "9-10"
        note: >
          Both transactions read 1, and neither waits: plain reads take no row
          locks under MVCC. This is the TS check before the await.
      - lines: "11-14"
        note: >
          B's UPDATE waits for A's row lock, then runs once A commits. It writes
          the 0 the application computed from a stale read, so both buyers were
          told yes for one item: a lost update. The CHECK can't help, because 0
          is a valid value.
      - lines: "19-20"
        note: >
          A has written a new version but not committed. B's read doesn't wait:
          its snapshot still shows the committed version, 1. Readers never block
          writers, and writers never block readers.
      - lines: "21-22"
        note: >
          The check is inside the write. B waits for A's lock; after A commits,
          READ COMMITTED re-evaluates B's WHERE against the new version, finds 0,
          and B reports UPDATE 0: sold out. In Go, pgx returns that count as
          tag.RowsAffected().
use_cases:
  - "Stock, seats, quotas, coupon redemptions: any \"take one if any are left\" decision, made in one conditional UPDATE"
  - "Go: in-process shared state (caches, counters, connection bookkeeping) under a mutex next to the data, or owned by one goroutine fed by a channel"
  - "TS: UI state updated after an await, where a request that started earlier can finish later and overwrite newer data"
strengths:
  - "Go: real parallelism on every core, with the mutex making the critical section explicit, and a race detector that ships with the standard toolchain (go test -race)"
  - "TS: run-to-completion makes every synchronous block atomic, so the fix needs no lock and no data race can corrupt memory"
  - "PostgreSQL: readers never wait for writers, and a conditional UPDATE is checked against the latest committed row, across every process and replica of your service"
weaknesses:
  - "Go: nothing stops you from touching shared state without the lock; the race detector only finds races that happen during the run you test"
  - "TS: the race is invisible in the code's shape; moving one await or adding one fetch can turn a correct function into a lost update"
  - "PostgreSQL: under the default READ COMMITTED, read-then-write in the application loses updates silently; you have to choose a conditional write, a row lock or a stricter isolation level"
pitfalls:
  - title: "Shared state without a lock in Go"
    detail: "Called from twenty goroutines, the unguarded version can sell an item twice and is a data race, which the memory model leaves undefined. Run tests with -race."
    bad: "func (s *Stock) Reserve() bool { if s.left == 0 { return false }; s.left--; return true }"
    good: "func (s *Stock) Reserve() bool { s.mu.Lock(); defer s.mu.Unlock(); if s.left == 0 { return false }; s.left--; return true }"
  - title: "Locking the check and the write separately"
    detail: "Two methods that each lock (Available, then Take) are each safe and together racy, exactly like a SELECT followed by an UPDATE or a check followed by an await. Make the operation the caller needs one locked method, one synchronous step, one statement."
    bad: "if s.Available() { s.Take() }"
    good: "if s.Reserve() { ... } // checks and takes under one lock"
  - title: "An in-process lock in a service with several instances"
    detail: "A Go mutex guards one process's memory, and a TS variable one tab's. Two replicas behind a load balancer each keep their own count. Decide in the database: a conditional UPDATE, SELECT ... FOR UPDATE, or an advisory lock (sql.transactions.row-locks)."
  - title: "Holding a transaction open across slow I/O"
    detail: "Awaiting a payment API between BEGIN and COMMIT keeps the row locked and every other buyer waiting. Reserve and commit, call the payment service, then confirm or release the item in a second short transaction."
quiz:
  - q: "In the TS racy version, why do all twenty buyers succeed?"
    options:
      - "Two threads ran reserveRacy at the same time"
      - "Each buyer awaits after its check, so every check runs before any decrement"
      - "Promise.all retries rejected promises"
    answer: 1
    why: "One thread runs the code, but await ends each synchronous step; all twenty checks see left === 10 before the first left-- runs."
  - q: "B's UPDATE ... WHERE ... AND left_qty > 0 waited for A, then reported UPDATE 0. Why?"
    options:
      - "B's snapshot was taken before A's update"
      - "After A committed, B re-checked its WHERE against the new row version, where left_qty was 0"
      - "The CHECK constraint rejected the row silently"
    answer: 1
    why: "Under READ COMMITTED a blocked UPDATE re-evaluates its condition on the updated row once the other transaction commits; no row matched."
  - q: "While A had updated 'fixed' but not committed, B ran SELECT left_qty. What happened?"
    options:
      - "B waited for A's commit, then saw 0"
      - "B saw 1 immediately"
      - "B got a serialization error"
    answer: 1
    why: "MVCC readers take no locks; B's snapshot shows the last committed version."
  - q: "Your Go service runs as three replicas. Where must \"is there an item left?\" be decided?"
    options:
      - "In a sync.Mutex in each replica"
      - "In the browser, before sending the order"
      - "In the database, in the same statement that takes the item"
    answer: 2
    why: "Each replica has its own memory; only the database sees every buyer, and a conditional UPDATE makes the check and the write one step."
further_reading:
  - title: "The Go Memory Model"
    url: "https://go.dev/ref/mem"
  - title: "Data Race Detector"
    url: "https://go.dev/doc/articles/race_detector"
  - title: "MDN: JavaScript execution model"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Execution_model"
  - title: "PostgreSQL docs: Introduction to MVCC"
    url: "https://www.postgresql.org/docs/current/mvcc-intro.html"
  - title: "PostgreSQL docs: Transaction Isolation (Read Committed)"
    url: "https://www.postgresql.org/docs/current/transaction-iso.html#XACT-READ-COMMITTED"
---

## Concept

"Twenty buyers, ten items" is the same problem in all three languages: a check ("is
anything left?") followed by a write ("take one"). If anything can run between the two,
two buyers can pass the check for the last item. Each runtime decides what can run in
between, and that decides what the fix looks like.

- **Go: threads.** Goroutines run in parallel on several OS threads and can be preempted
  at almost any point. Any shared variable written by one goroutine and read by another
  needs synchronization, or it is a data race. The fix is a critical section: the check
  and the write under one mutex, or the state owned by a single goroutine that others
  talk to over a channel.
- **TypeScript: tasks.** One thread runs one task at a time, to completion. Memory can't
  be corrupted, and synchronous code is atomic. But every `await` ends the step, and other
  tasks run before the function continues. The fix is to do the check and the write in
  one synchronous step and await afterwards.
- **PostgreSQL: transactions.** Many sessions run at once, each in its own server
  process, isolated by MVCC: every statement reads a snapshot of committed rows, and
  writers of the same row queue on its lock. The fix is a write that carries its own
  check, `UPDATE ... WHERE left_qty > 0`, which is re-evaluated against the latest
  committed row when it has to wait.

## Coming from C/C++/Java/Python/JS

- **Java/C++**: Go's model is yours, threads plus locks, with cheaper threads and a
  channel type built in. `sync.Mutex` is `ReentrantLock` without reentrancy and without
  `synchronized`; `defer mu.Unlock()` is a `lock_guard`.
- **Python**: the GIL hides some races in CPython threads (a single bytecode such as a
  list append is atomic); Go has no GIL, so the unguarded version breaks quickly.
  `asyncio` is the TypeScript model exactly: one thread, switches only at `await`.
- **JS/Node**: the TS example is plain JavaScript semantics. Node is the same model on
  the server; parallelism comes from Worker threads or more processes, each with its own
  memory unless you share a `SharedArrayBuffer`.
- **Databases you've used through an ORM**: Hibernate's optimistic locking (`@Version`)
  is the conditional UPDATE in disguise: `UPDATE ... WHERE version = ?`, and zero rows
  updated means someone else won.

## How it works

Where each runtime can interleave two buyers, and what makes a step atomic:

```text
             runs                                   atomic step                   guards for shared state
Go           goroutines on all cores, preempted     code between Lock and Unlock  mutex, channel ownership,
             almost anywhere                                                      sync/atomic
TypeScript   one thread, each task to completion    synchronous code between      nothing within a step; a
                                                    two awaits                    promise chain across awaits
PostgreSQL   one server process per connection,    one statement; a transaction  snapshots for reads, row
             all at once                            with row locks or             locks for writes, isolation
                                                    SERIALIZABLE                  levels
```

The same bug shows up in each column: Go's unguarded `left--`, TypeScript's check
before `await`, SQL's SELECT followed by an UPDATE of a computed value. The same cure
fixes all three: move the check into the atomic step.

## Design notes

A real deployment has all three at once: several browser tabs, several Go processes,
one database. In-process tools (a mutex, a single-threaded loop) protect one process's
memory, so any decision that involves shared business state, such as stock, seats or a
unique booking, belongs in the database, made by one statement or under a row lock. Go
and TypeScript keep their own concurrency tools for their own state: caches, in-flight
requests, UI state.

In SQL, prefer the conditional write: `UPDATE ... WHERE left_qty > 0` and look at the
row count. Use `SELECT ... FOR UPDATE` when the decision needs several reads first, and
REPEATABLE READ or SERIALIZABLE with a retry loop when the invariant spans rows
(sql.transactions.serializable). Under REPEATABLE READ, the racy version's second
UPDATE would have failed with SQLSTATE 40001 instead of silently losing an update.

In Go, keep the mutex next to the data, unexported, with methods that lock; pass
ownership through channels when work moves between goroutines. In TypeScript, treat
every `await` as a place where the world changes: re-read what you checked, or do the
check and the change before awaiting.
