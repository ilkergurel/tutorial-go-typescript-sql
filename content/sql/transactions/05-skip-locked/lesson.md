---
id: sql.transactions.skip-locked
title: "A job queue with FOR UPDATE SKIP LOCKED"
track: sql
module: transactions
order: 5
level: advanced
minutes: 15
since: pg10
prerequisites: [sql.transactions.row-locks]
compare_with: [fullstack.tracelite.step-11-jobs, go.concurrency.patterns, sql.server-side.listen-notify-ext]
tags: [transactions, locking, skip-locked, job-queue, multisession]
summary: >
  A jobs table plus SELECT ... FOR UPDATE SKIP LOCKED gives you a work queue
  in which workers never wait for each other. Then choose how a crashed
  worker's job comes back: hold the lock for the whole job, or commit the
  claim and use a lease.
examples:
  - id: naive-queue
    file: examples/01_naive_queue.sql
    title: "Plain FOR UPDATE: the second worker queues behind the first"
    runtime: sql-multisession
    setup: |
      CREATE TABLE jobs (
        id     bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
        kind   text NOT NULL,
        status text NOT NULL DEFAULT 'queued'
      );
      INSERT INTO jobs (kind) VALUES ('report 1'), ('report 2'), ('report 3');
    check: exact
    annotations:
      - lines: "4"
        note: >
          The row lock is the claim: as long as A's transaction is open, job 1
          is A's. ORDER BY id LIMIT 1 asks for the oldest queued job.
      - lines: "8"
        note: >
          How the runner shows a wait: B's step is still running after 400 ms,
          so it is printed as "waiting" and the script carries on with A. B's
          result appears under the same number once A commits. B is stuck on
          job 1 although jobs 2 and 3 are free; ten workers would all queue
          on the same row.
      - lines: "11-12"
        note: >
          When A commits, B's query re-checks job 1 against WHERE status =
          'queued', finds it done, and moves on to lock job 2. Correct, but B
          was idle for as long as A's job ran.
  - id: skip-locked
    file: examples/02_skip_locked.sql
    title: "SKIP LOCKED: workers never wait; a crash puts the job back"
    runtime: sql-multisession
    setup: |
      CREATE TABLE jobs (
        id     bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
        kind   text NOT NULL,
        status text NOT NULL DEFAULT 'queued'
      );
      INSERT INTO jobs (kind) VALUES ('report 1'), ('report 2'), ('report 3');
    check: exact
    annotations:
      - lines: "7-8"
        note: >
          Rows that can't be locked immediately are skipped, so B gets job 2
          straight away: no "waiting" line anywhere in this transcript.
      - lines: "14-15"
        note: >
          B takes job 3 while job 1 is still running. SKIP LOCKED gives up
          strict FIFO order; for a work queue that's the point.
      - lines: "19"
        note: >
          A killed worker's connection drops and the server rolls its
          transaction back, exactly like this ROLLBACK. The lock goes with it,
          and job 1 was never marked done: no cleanup code needed.
      - lines: "24-26"
        note: >
          Job 1 is retried by B. In this design the lock is the job's
          "running" state, so a crash can never leave a stuck job. The price:
          one open transaction, and one connection, per running job.
  - id: claim-with-lease
    file: examples/03_claim_with_lease.sql
    title: "Claim and commit, with a lease and a reaper for crashed workers"
    runtime: sql-multisession
    setup: |
      CREATE TABLE jobs (
        id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
        kind        text NOT NULL,
        status      text NOT NULL DEFAULT 'queued',
        attempts    int NOT NULL DEFAULT 0,
        lease_until timestamptz
      );
      INSERT INTO jobs (kind) VALUES ('report 1'), ('report 2'), ('report 3');
    check: exact
    annotations:
      - lines: "4-8"
        note: >
          The claim is one statement: the subquery picks and locks the oldest
          queued job it can get, the UPDATE marks it running with a deadline,
          and RETURNING hands it to the worker. After commit, status =
          'running' keeps other workers away; the lock was only needed for
          the moment of the claim.
      - lines: "9-13"
        note: >
          B claims while A's claim is still uncommitted. Job 1 is locked, so
          B skips it and takes job 2 instead of waiting for A.
      - lines: "14"
        note: >
          The transaction ends right after the claim. The job then runs with
          no transaction open: no locks held, no connection pinned, and
          nothing holding back VACUUM, however long the job takes.
      - lines: "19-21"
        note: >
          The cost of committing the claim: when A dies, nothing rolls back
          and job 1 stays 'running'. The lease is the deadline after which
          other workers may assume its owner is dead. The demo moves the
          deadline into the past instead of waiting five minutes.
      - lines: "24-26"
        note: >
          The reaper puts expired jobs back in the queue. Run it every few
          seconds from any worker; it is idempotent.
      - lines: "29-33"
        note: >
          Job 1 comes back with attempts = 2. Cap it: a job that keeps killing
          its worker should end up 'failed' for a human to look at, not
          retried forever.
use_cases:
  - "Background jobs stored next to the data they belong to: reports, e-mails, webhooks, imports (TraceLite's report generation)"
  - "An outbox table: rows written in the business transaction, delivered by a worker pool afterwards"
  - "Any table several consumers drain concurrently, where each row must be handled by one consumer at a time"
strengths:
  - "Enqueueing is an INSERT in the same transaction as the business change: the job exists if and only if the change committed, which a separate message broker can't give you without extra machinery"
  - "Workers never wait for each other: each claim skips rows other workers hold"
  - "Crash recovery comes from transactions (example 2) or from a lease and a reaper (example 3), with no separate broker to run, monitor and back up"
  - "Queue depth, stuck jobs and retry counts are one SQL query away"
weaknesses:
  - "Every job passes through several row versions (queued, running, done), so a busy queue table depends on autovacuum keeping up"
  - "Ordering is only approximately FIFO"
  - "Idle workers have to poll, or be woken with LISTEN/NOTIFY"
  - "Holding a transaction per running job (example 2) pins one connection per worker and holds back VACUUM for as long as jobs run"
pitfalls:
  - title: "Plain FOR UPDATE in the claim"
    detail: "Every worker queues on the oldest job's lock, so N workers process one job at a time. Add SKIP LOCKED."
    bad: "SELECT id FROM jobs WHERE status = 'queued' ORDER BY id LIMIT 1 FOR UPDATE;"
    good: "SELECT id FROM jobs WHERE status = 'queued' ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED;"
  - title: "SKIP LOCKED outside a queue"
    detail: "The docs are explicit: skipping locked rows gives an inconsistent view of the data. Never use it for counts, reports, uniqueness checks or anything that must see every row."
  - title: "Long jobs inside the claim transaction"
    detail: "Example 2's design suits jobs of seconds. A job that runs for minutes keeps a transaction open the whole time, which holds back VACUUM in the whole database and ties up a pooled connection. Commit the claim and use a lease (example 3)."
  - title: "A lease without a fence"
    detail: "A worker that was only slow can finish after its lease expired and the job was claimed again. Make jobs idempotent, or let only the current attempt finish the job, using attempts as a fencing token."
    good: "UPDATE jobs SET status = 'done', lease_until = NULL\nWHERE id = $1 AND attempts = $2 AND status = 'running';"
  - title: "No index for the claim query"
    detail: "Done jobs pile up, and the claim's scan wades through them. A partial index covers only the rows the claim looks at."
    good: "CREATE INDEX jobs_queued ON jobs (id) WHERE status = 'queued';"
quiz:
  - q: "Without SKIP LOCKED, what does a second worker do when the oldest queued job is locked by the first?"
    options: ["Takes the next job", "Waits until the first worker's transaction ends", "Gets an error"]
    answer: 1
    why: "A plain FOR UPDATE waits for the lock holder, so workers process the queue one at a time."
  - q: "In example 2's design (lock held for the whole job) the worker process is killed. What happens to its job?"
    options: ["It stays 'running' until a reaper resets it", "The server rolls the transaction back, the lock is released and the job is queued again", "It is marked failed"]
    answer: 1
    why: "The claim was never committed: the status change and the lock disappear with the transaction."
  - q: "In example 3's design (claim committed) why do you need a lease?"
    options: ["To make the claim faster", "Because a crashed worker leaves the job 'running' and no transaction will undo that", "Because SKIP LOCKED requires a timestamp"]
    answer: 1
    why: "Once the claim commits, only a deadline tells other workers the job's owner is gone."
  - q: "Where is SKIP LOCKED the wrong tool?"
    options: ["Claiming the next outbox row to send", "SELECT count(*) of open orders for a dashboard", "Picking the next import file to process"]
    answer: 1
    why: "A count must see every row; SKIP LOCKED leaves out the ones that happen to be locked."
further_reading:
  - title: "PostgreSQL docs: SELECT, The Locking Clause (NOWAIT, SKIP LOCKED)"
    url: "https://www.postgresql.org/docs/current/sql-select.html"
  - title: "PostgreSQL docs: Row-Level Locks"
    url: "https://www.postgresql.org/docs/current/explicit-locking.html#LOCKING-ROWS"
  - title: "PostgreSQL docs: Partial Indexes"
    url: "https://www.postgresql.org/docs/current/indexes-partial.html"
  - title: "PostgreSQL docs: NOTIFY (waking idle workers)"
    url: "https://www.postgresql.org/docs/current/sql-notify.html"
---

## Concept

A job queue in Postgres is a table plus one query. Workers claim the oldest queued job
with `SELECT ... FOR UPDATE SKIP LOCKED`: rows another worker has locked are skipped
instead of waited for, so every worker gets a different job immediately. `SKIP LOCKED`
(PostgreSQL 9.5+) exists for exactly this; the docs call it unsuitable for general work
but right for "multiple consumers accessing a queue-like table".

The claim is easy. The design decision is what happens when a worker dies mid-job:

- **Hold the lock for the job's duration** (example 2): claim, work, mark done, commit, all
  in one transaction. A crash rolls back and the job reappears by itself.
- **Commit the claim** (example 3): mark the job `running` with a lease, commit, work with
  no transaction open, then mark it done. A crash leaves the job `running` until the lease
  expires and a reaper re-queues it.

## Coming from C/C++/Java/Python/JS

- **Go worker pools:** a buffered channel of jobs hands each job to exactly one goroutine.
  A `SKIP LOCKED` claim is the same "receive" across processes and machines, with the
  queue surviving restarts. The usual shape in Go: N goroutines, each looping claim, run,
  finish, and sleeping (or waiting for a `NOTIFY`) when the claim returns no row.
- **Java `BlockingQueue` / Python `queue.Queue`:** in-memory and lost on a crash. Here the
  queue is durable and transactional with your data.
- **Message brokers:** example 3's lease plays the role of a broker's acknowledgement
  deadline (SQS calls it a visibility timeout): a message that isn't confirmed in time is
  delivered again. What you gain over a broker is the enqueue happening in the same
  transaction as the business change.

## How it works

`SKIP LOCKED` (like `NOWAIT`) only changes how row locks are taken; the table-level lock is
taken normally. With `LIMIT 1`, locking stops once one row is returned. Each candidate row
is read through the snapshot as usual; if its row lock can't be taken immediately, the row
is left out of the result and the scan moves on.

In example 1, B's waiting query shows a READ COMMITTED detail: a row that matched the
`WHERE` clause at the snapshot is locked, but it isn't returned if it was changed meanwhile
and no longer matches. That's why B moves on to job 2 instead of getting job 1 twice.

## Design notes

Example 2's design (lock held for the whole job) is the simplest correct queue and fits
short jobs: there is no state machine to get wrong, and a crash can't leave a stuck job.
Its limits come from holding a transaction open while the job runs: one connection per
running job, and an old snapshot that keeps VACUUM from cleaning up behind it (lesson 7).

Example 3's design (claim, lease, reaper) fits jobs that take minutes, call external
services or must survive a deploy, like TraceLite's report generation. Its rules:

- The claim and the finish are separate short transactions; the job runs between them.
- A lease longer than the longest normal run, an `attempts` counter, and a cap that moves
  jobs to `failed`.
- Jobs must be idempotent, or finished only by the current attempt (the fencing pitfall).
  At-least-once is the honest guarantee.
- A partial index on queued jobs, and autovacuum settings tuned for a table where every
  row is updated several times (lesson 7).

To avoid polling, a worker can `LISTEN jobs` and the enqueuer can `NOTIFY jobs` in the
same transaction as the insert; the notification is delivered only if that transaction
commits. Keep a slow poll as a fallback anyway: notifications aren't stored, so a worker
that was restarting misses them.
