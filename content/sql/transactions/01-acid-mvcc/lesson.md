---
id: sql.transactions.acid-mvcc
title: "ACID, savepoints, MVCC: why readers don't block writers"
track: sql
module: transactions
order: 1
level: intermediate
minutes: 13
since: pg13
prerequisites: [sql.modifying.insert-update, sql.constraints.keys]
compare_with: [go.postgres.transactions, go.concurrency.sync, design.paradigms.concurrency]
tags: [transactions, acid, mvcc, savepoints, multisession]
summary: >
  A transaction is all or nothing, and one failed statement aborts all of it
  unless a savepoint contains the damage. Postgres isolates transactions with
  row versions instead of read locks, so readers and writers never wait for
  each other; only two writers on the same row do.
examples:
  - id: row-versions
    file: examples/01_row_versions.sql
    title: "UPDATE writes a new row version; ROLLBACK TO just hides it"
    runtime: sql
    check: exact
    annotations:
      - lines: "7-8"
        note: >
          Transaction IDs change on every run, so xmin is compared, not
          printed. pg_current_xact_id() (PostgreSQL 13+) returns the 64-bit
          xid8; the cast gives the 32-bit form that xmin uses. Both rows were
          written by this script's one transaction, and xmax 0 means no
          transaction has deleted or replaced them.
      - lines: "11-12"
        note: >
          Row a moved from (0,1) to (0,3): 60 is a new version. The 100 is
          still on the page, its xmax set to our transaction. We no longer see
          it, but a concurrent transaction would, until we commit. That old
          version is what lets readers proceed without waiting for writers.
      - lines: "15-18"
        note: >
          SAVEPOINT starts a subtransaction. Its update gets a transaction ID
          of its own and writes version (0,4). ROLLBACK TO marks that
          subtransaction aborted, releases the locks taken since the
          savepoint, and leaves the outer transaction usable.
      - lines: "22"
        note: >
          The docs say a nonzero xmax on a visible row usually means the
          deleting transaction hasn't committed yet, or that an attempted
          deletion was rolled back. Here it's the second case: rolling back
          did not restore the old version, it only marked the newer one's
          writer as aborted, so visibility checks skip it.
      - lines: "25"
        note: >
          The aborted version still occupies slot 4, so the new version lands
          in slot 5. Versions from rolled-back work are garbage that VACUUM
          collects later (the last lesson of this module).
  - id: readers-writers
    file: examples/02_readers_writers.sql
    title: "Two sessions: the reader doesn't wait, the second writer does"
    runtime: sql-multisession
    setup: |
      CREATE TABLE accounts (id text PRIMARY KEY, balance int NOT NULL);
      INSERT INTO accounts VALUES ('a', 100), ('b', 0);
    check: exact
    annotations:
      - lines: "3-5"
        note: >
          A has written new versions of both rows and holds a row lock on each
          until it ends. Nothing is visible to other sessions yet.
      - lines: "8"
        note: >
          B never ran BEGIN, so each of its statements is its own transaction.
          Its snapshot shows the last committed versions, 100 and 0: no
          waiting, and never the in-between state where the 40 has left a but
          not reached b.
      - lines: "11"
        note: >
          How the runner shows a wait: a step still running after 400 ms is
          printed as "[5] B: waiting" and the script moves on, as you would
          switch terminals; the result is printed under the same number when
          it arrives. B needs the row lock A holds on a, so it waits for A's
          commit or rollback.
      - lines: "13-16"
        note: >
          A's COMMIT releases the locks and B's update completes right after
          it. Under READ COMMITTED a blocked UPDATE re-reads the row once the
          other transaction commits, so it adds 1 to A's 60, not to the 100 it
          saw when it started. Nothing was lost: 61 and 40.
  - id: failed-transaction
    file: examples/03_failed_transaction.sql
    title: "One error aborts the transaction; a savepoint contains it"
    runtime: sql-multisession
    setup: |
      CREATE TABLE accounts (id text PRIMARY KEY, balance int NOT NULL);
      INSERT INTO accounts VALUES ('a', 100), ('b', 0);
    check: exact
    annotations:
      - lines: "4-5"
        note: >
          The duplicate key fails (23505), and from then on the whole
          transaction is aborted: every statement is refused with 25P02 until
          it ends. The successful insert of c is doomed too.
      - lines: "6"
        note: >
          COMMIT on an aborted transaction doesn't raise an error. It rolls
          back and reports the command tag ROLLBACK. pgx turns that tag into
          pgx.ErrTxCommitRollback; code that ignores Commit's error believes
          it saved data that is gone.
      - lines: "11-13"
        note: >
          With a savepoint in front of the risky statement, ROLLBACK TO undoes
          just that statement and clears the aborted state. The insert of c,
          made before the savepoint, survives.
      - lines: "18"
        note: >
          A second connection sees c and d from the committed transaction and
          nothing from the failed one: all or nothing, and once COMMIT
          returned, durable.
use_cases:
  - "Any change that spans several statements and must happen completely or not at all: a transfer, an order with its lines, a schema migration"
  - "Imports that should skip bad rows instead of failing: a savepoint around the risky statement, ROLLBACK TO on error, carry on"
  - "Long reads (exports, reports) next to normal write traffic, without either side waiting for the other"
strengths:
  - "Plain reads take no row locks: a long report never stalls updates, unlike a Java ReentrantReadWriteLock, where one writer waits for every reader to finish"
  - "ROLLBACK doesn't copy old values back; it marks the transaction aborted and visibility checks skip what it wrote"
  - "Most DDL is transactional too: a migration's CREATE, ALTER and data fixes commit or roll back together"
  - "Savepoints give statement-level recovery inside a transaction; PL/pgSQL's EXCEPTION blocks use the same subtransaction mechanism"
weaknesses:
  - "Every UPDATE writes a whole new row version, usually with new index entries, so update-heavy tables grow until VACUUM reclaims the old versions"
  - "Any error aborts the entire transaction; recovering needs a savepoint, which costs a subtransaction"
  - "MVCC removes read/write blocking, not write/write blocking: writers to the same row still queue"
  - "Under the default READ COMMITTED each statement takes a fresh snapshot, so two reads in one transaction can disagree (next lesson)"
pitfalls:
  - title: "Ignoring Commit's error"
    detail: "COMMIT of an aborted transaction succeeds as a statement and reports ROLLBACK. pgx returns pgx.ErrTxCommitRollback for it, so the error from tx.Commit is the only signal that your writes are gone."
    bad: "_ = tx.Commit(ctx)"
    good: "if err := tx.Commit(ctx); err != nil {\n\treturn fmt.Errorf(\"commit transfer: %w\", err)\n}"
  - title: "Catching an error and carrying on in the same transaction"
    detail: "After any error every statement fails with 25P02 until ROLLBACK. If you expect a statement to fail (a duplicate during an import), put a savepoint in front of it, or better, write it so it can't fail."
    bad: "INSERT INTO tags VALUES ('go');  -- may be a duplicate: aborts the transaction"
    good: "INSERT INTO tags VALUES ('go') ON CONFLICT DO NOTHING;"
  - title: "A savepoint per row in a big loop"
    detail: "Each savepoint is a subtransaction. The docs warn that overhead grows with the number kept open, and past 64 per backend lookups go to pg_subtrans on disk. Release savepoints you no longer need, and prefer ON CONFLICT or a staging table to a savepoint per row."
  - title: "Leaving a transaction open"
    detail: "An idle transaction keeps its row locks and stops VACUUM from removing versions it might still see. Never wait for user input, an HTTP call or a slow job inside a transaction; idle_in_transaction_session_timeout is the server-side safety net."
quiz:
  - q: "Session A has updated row a but not committed. Session B runs SELECT balance FROM accounts WHERE id = 'a'. What happens?"
    options: ["B waits until A commits or rolls back", "B sees A's uncommitted value", "B immediately gets the last committed value", "B gets a lock error"]
    answer: 2
    why: "B's snapshot doesn't include A's uncommitted version, and plain reads take no row locks, so it reads the previous version without waiting."
  - q: "Inside a transaction an INSERT fails with a unique violation. What does the next SELECT do?"
    options: ["Runs normally: only the INSERT was undone", "Fails with 25P02 until the transaction ends", "Retries the INSERT automatically"]
    answer: 1
    why: "Postgres aborts the whole transaction on an error. Only ROLLBACK (or ROLLBACK TO a savepoint taken before the error) makes it usable again."
  - q: "What does ROLLBACK TO SAVEPOINT s do?"
    options: ["Ends the transaction", "Undoes the work done after s and leaves the transaction open", "Undoes the work done before s"]
    answer: 1
    why: "The savepoint's subtransaction is aborted; everything before it is kept, and the transaction can continue and commit."
  - q: "UPDATE accounts SET balance = 60 WHERE id = 'a' has committed. What happened on disk?"
    options: ["The row was overwritten in place", "A new row version was written; the old one stays until VACUUM removes it once no snapshot can see it", "The old version was moved to an undo log"]
    answer: 1
    why: "Every update creates a new version with a new ctid. The old one is dead but stays in the table until VACUUM, which is what keeps it available to older snapshots."
further_reading:
  - title: "PostgreSQL docs: Concurrency Control, Introduction (MVCC)"
    url: "https://www.postgresql.org/docs/current/mvcc-intro.html"
  - title: "PostgreSQL docs: Tutorial, Transactions"
    url: "https://www.postgresql.org/docs/current/tutorial-transactions.html"
  - title: "PostgreSQL docs: SAVEPOINT"
    url: "https://www.postgresql.org/docs/current/sql-savepoint.html"
  - title: "PostgreSQL docs: System Columns (xmin, xmax, ctid)"
    url: "https://www.postgresql.org/docs/current/ddl-system-columns.html"
  - title: "PostgreSQL docs: Subtransactions"
    url: "https://www.postgresql.org/docs/current/subxacts.html"
  - title: "PostgreSQL docs: Asynchronous Commit (what COMMIT normally guarantees)"
    url: "https://www.postgresql.org/docs/current/wal-async-commit.html"
---

## Concept

A transaction is the unit of correctness. Everything between `BEGIN` and `COMMIT` is:

- **Atomic**: all of it takes effect or none of it does. In Postgres a single failed
  statement aborts the whole transaction; a **savepoint** lets you roll back to a point
  inside it and continue.
- **Consistent**: constraints are checked, so a committed state never violates them.
- **Isolated**: other sessions don't see your changes until you commit, and you don't
  see theirs half-done.
- **Durable**: by default `COMMIT` returns only after the transaction's WAL records are
  flushed to disk, so a reported commit survives a crash.

Postgres isolates with **MVCC** (multiversion concurrency control). An `UPDATE` never
changes a row in place: it writes a new **row version** and marks the old one as replaced.
Each statement reads from a **snapshot** that decides which versions it may see. A reader
therefore never needs a writer's lock: it takes the newest version its snapshot allows.
Only two writers on the same row wait for each other.

Examples 2 and 3 use the two-session runner: steps tagged `A:` and `B:` run on two
connections, in script order, like two psql terminals side by side.

## Coming from C/C++/Java/Python/JS

- **Java/C++ locks:** a `ReadWriteLock` makes writers wait for readers. MVCC works like
  copy-on-write: a writer publishes a new version and readers keep the one they started
  with. In Go terms, it's closer to swapping an `atomic.Pointer` to an immutable struct
  than to a `sync.RWMutex`.
- **Python (psycopg) and Go (pgx):** the "poisoned transaction" is the classic surprise.
  psycopg raises `InFailedSqlTransaction` on the next statement after an error; that is
  SQLSTATE 25P02 from example 3. In pgx, check the error from `tx.Commit`.
- **Connection pools (pgxpool, node-postgres):** a transaction lives on one connection.
  Sending `BEGIN` through the pool and then another statement through the pool may use
  two different connections. Use the pool's transaction API (`pool.Begin` in pgx), which
  pins one connection until `Commit` or `Rollback`.
- **C:** think of `xmin` and `xmax` as a version stamp in every record header: "written by
  transaction X, superseded by transaction Y".

## How it works

Every row version carries the hidden columns from example 1: `xmin` (the transaction that
wrote it), `xmax` (the transaction that deleted or replaced it, or 0) and `ctid` (its
physical place, page and slot). Postgres also records for each transaction whether it
committed or aborted. Roughly, a version is visible to your snapshot if its `xmin`
committed before the snapshot was taken (or is your own transaction), and its `xmax` is
0, aborted, or not yet committed as far as the snapshot knows.

That is why `ROLLBACK` is cheap: it records "aborted" and every version the transaction
wrote becomes invisible. The cost comes later: dead versions stay in the table until
`VACUUM` removes the ones no snapshot can still see (lesson 7).

The `sql` runtime of example 1 runs your script inside one transaction that is always
rolled back, so it can show savepoints but not `COMMIT`; the two-session examples commit
for real, in a throwaway schema.

## Design notes

- **Keep transactions short.** Row locks are held until the end, and an open transaction
  holds back VACUUM. Do the HTTP call, the file upload or the user prompt before `BEGIN`
  or after `COMMIT`, never in between.
- **One place owns the boundary.** In Go, a small `WithTx(ctx, func(pgx.Tx) error)`
  helper begins, rolls back on error, and commits (the Go + PostgreSQL module builds it).
  Repositories take a `pgx.Tx` (or an interface it satisfies) instead of beginning
  their own.
- **Prefer statements that can't fail over savepoints.** `ON CONFLICT DO NOTHING`, `WHERE
  NOT EXISTS`, or a staging table is cheaper and clearer than catching duplicates.
  Savepoints are for genuinely unpredictable failures inside a larger unit of work.
- MVCC only removes read/write blocking. When two transactions write the same rows, you
  still choose how they coordinate: isolation levels (next two lessons) or explicit row
  locks (lesson 4).
