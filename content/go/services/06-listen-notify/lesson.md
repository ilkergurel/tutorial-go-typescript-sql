---
id: go.services.listen-notify
title: "Postgres LISTEN/NOTIFY as a lightweight message bus"
track: go
module: services
order: 6
level: advanced
minutes: 13
since: go1.26
prerequisites: [go.postgres.transactions, go.concurrency.context]
compare_with: [sql.server-side.listen-notify-ext, sql.transactions.skip-locked, fullstack.tracelite.step-09-realtime]
tags: [postgres, pgx, listen-notify, events, triggers]
summary: >
  NOTIFY delivers a short message to every session LISTENing on a channel, at
  commit and only at commit. Use it as a wake-up signal next to a table that holds
  the real events, so a listener that was away can always catch up.
examples:
  - id: semantics
    file: examples/01_semantics.go
    title: "Delivered at commit, dropped on rollback, folded when identical"
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "15-27"
        note: >
          WaitForNotification blocks until a notification arrives or the
          context ends. pgx reports a context timeout as a timeout error and
          keeps the connection open, so a short deadline is a safe way to poll
          "anything new?".
      - lines: "37-42"
        note: >
          Two plain connections, not a pool: LISTEN belongs to one session,
          and a pooled connection would go back to the pool, still listening,
          for some other request to use.
      - lines: "44-48"
        note: >
          Channels are database-wide, shared by every schema and every app on
          the database, so this run uses a name derived from its own schema.
          LISTEN takes an identifier: quoting it keeps the exact spelling,
          while an unquoted name is folded to lower case.
      - lines: "49-54"
        note: >
          pg_notify(channel, payload) is a function, so both arguments can be
          $n parameters. The NOTIFY command only accepts a string literal as
          payload, which invites string concatenation.
      - lines: "56-61"
        note: >
          Nothing arrives while the sending transaction is open; the
          notification is delivered when it commits. Listeners therefore never
          hear about data they can't see yet.
      - lines: "63-67"
        note: >
          A rolled-back transaction's notifications are never delivered,
          exactly like its writes.
      - lines: "69-73"
        note: >
          Within one transaction, identical channel-and-payload pairs are
          delivered once, and the rest arrive in the order they were sent.
          Transactions arrive in commit order. Different transactions are never
          folded together.
  - id: wake-up
    file: examples/02_wake_up.go
    title: "A table of events, a trigger that rings, a consumer that catches up"
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "13-21"
        note: >
          The events table is the source of truth. The statement-level trigger
          sends one notification per INSERT statement, whatever the row count,
          and an empty payload: the message only says "look again".
      - lines: "25-29"
        note: >
          The consumer owns one connection for its whole life. When consume
          returns an error, the caller calls it again (with backoff): a new
          connection, LISTEN, catch up, wait.
      - lines: "34-36"
        note: >
          LISTEN before the first catch-up query. In the other order, an event
          committed between the query and the LISTEN would be neither read nor
          announced.
      - lines: "37-49"
        note: >
          Each wake-up reads everything after the last ID handled, so a lost
          or folded notification costs nothing. "id > last" is safe here
          because there is one writer; see the pitfall about concurrent
          writers. Competing consumers claim rows with FOR UPDATE SKIP LOCKED
          instead (sql.transactions.skip-locked).
      - lines: "50-52"
        note: >
          A notification sent while this session was disconnected is gone for
          good; NOTIFY has no persistence. The fix is the loop itself:
          reconnect, LISTEN, catch up, wait.
      - lines: "71"
        note: >
          This row is committed before anyone listens. Plain NOTIFY would lose
          it; the catch-up query at start finds it.
      - lines: "76-77"
        note: >
          One INSERT of three rows rings once; the catch-up query after the
          wake-up reads all three.
      - lines: "78-79"
        note: >
          Cancelling the context ends WaitForNotification with
          context.Canceled, which is how a server stops its listener on
          shutdown.
  - id: payload-limit
    file: examples/03_payload_limit.sql
    title: "The payload limit is counted in bytes"
    runtime: sql
    check: exact
    annotations:
      - lines: "2"
        note: >
          length counts characters, octet_length bytes: 4000 Turkish ş are
          8000 bytes in UTF-8.
      - lines: "5-6"
        note: >
          The default limit is "shorter than 8000 bytes", so 3999 ş fit and
          4000 don't. Send an ID, not the record.
use_cases:
  - "Pushing changes to browsers in real time: the Go server LISTENs and fans out over SSE or WebSockets (fullstack.tracelite.step-09-realtime)"
  - "Waking a job worker as soon as a job row is inserted, instead of polling every few seconds"
  - "Invalidating in-process caches on every app instance when configuration rows change"
strengths:
  - "No extra infrastructure: if you run PostgreSQL, you already have it"
  - "Transactional: a notification is sent if and only if the writing transaction commits, which a separate broker can't promise without an outbox"
  - "Low latency compared with polling, and no load while nothing happens"
  - "Works from triggers, so every writer, including psql and migrations, announces changes"
weaknesses:
  - "No persistence and no acknowledgement: a disconnected listener misses notifications for good"
  - "Payloads are short (under 8000 bytes by default) and plain text"
  - "Each listener holds a dedicated connection, which doesn't fit transaction-mode connection poolers"
  - "All notifications go through one queue in the cluster; a listener stuck in a long transaction stops its cleanup"
pitfalls:
  - title: "Treating notifications as the data"
    detail: "Listeners that are offline, restarting or reconnecting miss notifications. Keep the events in a table, send IDs or nothing, and catch up from the table on every (re)connect."
  - title: "Listening on a pooled connection"
    detail: "LISTEN is session state. With pgxpool, acquire a connection for the listener's whole lifetime (or use pgx.Connect); behind PgBouncer in transaction mode, connect to PostgreSQL directly for listening."
  - title: "Case-folded channel names"
    detail: "LISTEN Orders listens on orders (unquoted identifiers fold to lower case), but pg_notify('Orders', ...) uses the exact string, so nothing arrives. Use lower-case names, or quote consistently with pgx.Identifier."
    bad: |
      conn.Exec(ctx, "LISTEN Orders")
      conn.Exec(ctx, "SELECT pg_notify('Orders', 'x')") // never delivered
    good: |
      conn.Exec(ctx, "LISTEN "+pgx.Identifier{"orders"}.Sanitize())
      conn.Exec(ctx, "SELECT pg_notify($1, $2)", "orders", "x")
  - title: "Listening inside a long transaction"
    detail: "A listening session only receives notifications between its transactions, and while it sits in one, the notification queue can't be cleaned up. Keep the listener connection idle except for short catch-up queries."
  - title: "Big payloads"
    detail: "The limit is in bytes, and Turkish text takes up to two bytes per letter. Send a key and let the listener read the row."
  - title: "Assuming IDs commit in order"
    detail: "With concurrent writers, a transaction that took ID 7 can commit before the one that took ID 6. A consumer that already moved its cursor to 7 never reads 6. With more than one writer, track what was processed per row (a processed_at column, or a claim with FOR UPDATE SKIP LOCKED) instead of a high-water mark. IDs also have gaps: a rolled-back insert still uses up its identity value, because sequence increments are never rolled back, so \"last ID + 1\" is never a safe assumption."
quiz:
  - q: "A transaction runs pg_notify('jobs', '42') and then fails with a constraint violation. What do listeners receive?"
    options: ["'42' immediately", "'42' when the transaction ends", "Nothing", "An error notification"]
    answer: 2
    why: "Notifications are delivered only if the transaction commits; a rolled-back transaction has no effects at all."
  - q: "One transaction sends pg_notify('c', 'x') three times. How many notifications arrive?"
    options: ["Three", "One: identical channel and payload in one transaction are folded", "None", "Two"]
    answer: 1
    why: "The documentation guarantees folding only within a transaction; separate transactions are never folded."
  - q: "Your listener reconnects after a network blip. Why does the design in example 2 lose nothing?"
    options: ["PostgreSQL replays missed notifications", "pgx buffers them", "The consumer re-runs its catch-up query, which reads every event after the last ID it handled", "It doesn't; events are lost"]
    answer: 2
    why: "Notifications are not persisted. The table is, and the catch-up query after LISTEN closes the gap."
further_reading:
  - title: "PostgreSQL: NOTIFY"
    url: "https://www.postgresql.org/docs/current/sql-notify.html"
  - title: "PostgreSQL: LISTEN"
    url: "https://www.postgresql.org/docs/current/sql-listen.html"
  - title: "PostgreSQL: pg_notification_queue_usage and other system information functions"
    url: "https://www.postgresql.org/docs/current/functions-info.html"
  - title: "package pgx: Conn.WaitForNotification"
    url: "https://pkg.go.dev/github.com/jackc/pgx/v5#Conn.WaitForNotification"
---

## Concept

PostgreSQL has a small publish/subscribe system built in. A session runs
`LISTEN channel`; any session runs `NOTIFY channel, 'payload'` or `SELECT
pg_notify('channel', 'payload')`; every listening session receives the payload. The
rules that matter, all from the NOTIFY documentation:

- **Delivery happens at commit.** Notifications sent inside a transaction arrive when it
  commits, never if it rolls back.
- **Order is kept**: within a transaction in the order sent, across transactions in
  commit order.
- **Identical notifications fold**: the same channel and payload sent several times in
  one transaction arrive once.
- **Payloads are short**: under 8000 bytes in the default configuration.
- **Nothing is stored for absent listeners.** A session that isn't connected and
  listening at the moment of commit never sees the notification.

The last rule decides the design: NOTIFY is a *doorbell*, not a mailbox. Store events in
a table, send a notification (an ID, or nothing) when they change, and have listeners
read the table: once at start, and again after every wake-up.

In pgx, a listener is one dedicated `*pgx.Conn`: `Exec("LISTEN ...")`, then a loop
around `WaitForNotification(ctx)`, which returns a `*pgconn.Notification` with the
channel, the payload and the sender's backend PID.

## Coming from C/C++/Java/Python/JS

- **C (libpq)**: `PQconsumeInput` and `PQnotifies` after `select()` on the socket; pgx's
  `WaitForNotification` is that loop with a context.
- **Java**: pgJDBC exposes `PGConnection.getNotifications(timeout)` on an unwrapped
  connection; the dedicated-connection rule is the same.
- **Python**: psycopg 3's `conn.notifies()` generator, on a connection in autocommit mode.
- **Node**: node-postgres emits a `notification` event on a `Client` (not a `Pool`).
  Each language arrives at the same advice: one long-lived connection per listener.

## How it works

`NOTIFY` writes into a single queue shared by the whole cluster (8 GB in a standard
installation) when the transaction commits. Each listening backend reads the entries for
its channels and forwards them to its client, but only between transactions. Entries are
removed once every listener has read past them, which is why one listener stuck inside
a long transaction stops the cleanup for everyone; `pg_notification_queue_usage()` shows
the fill level, and if the queue ever fills up, transactions that NOTIFY fail at commit.

## Design notes

Use LISTEN/NOTIFY when the database is where the change happens and latency matters:
live updates, waking a worker, cache invalidation across a few app instances. It is not
a broker: no persistence, no acknowledgements, no consumer groups, no replay. When you
need those, use a jobs table with `FOR UPDATE SKIP LOCKED` (still PostgreSQL), or a real
broker (lesson 7).

In a Go server, run one listener goroutine per process that owns a dedicated connection,
reconnects with backoff (lesson 2), LISTENs, catches up, and then fans notifications out
to the rest of the process through channels. Give it the server's root context, so
shutdown cancels `WaitForNotification`. Your own NOTIFYs come back to you as well; compare
the notification's PID with your connection's `PgConn().PID()` if you need to ignore them.
