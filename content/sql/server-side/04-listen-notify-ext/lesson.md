---
id: sql.server-side.listen-notify-ext
title: "LISTEN/NOTIFY; extensions"
track: sql
module: server-side
order: 4
level: advanced
minutes: 15
since: pg13
prerequisites: [sql.server-side.triggers, sql.transactions.acid-mvcc]
compare_with: [go.services.listen-notify, sql.transactions.skip-locked, fullstack.tracelite.step-09-realtime, sql.search.trigram]
tags: [listen-notify, pg_notify, events, extensions, trusted-extensions, tcn, pg_trgm, unaccent]
summary: >
  NOTIFY sends a short message to every session LISTENing on a channel, only
  when the sending transaction commits and with no persistence; with a trigger
  (or the tcn extension) every committed change announces itself. Extensions
  add types, functions and index support per database; trusted ones need no
  superuser.
examples:
  - id: listen-notify
    file: examples/01_listen_notify.sql
    title: "LISTEN and UNLISTEN take effect at commit; NOTIFY vs pg_notify; the byte limit"
    runtime: sql-nontx
    check: exact
    annotations:
      - lines: "3-4"
        note: >
          In autocommit, LISTEN commits at once and the session is registered
          immediately. pg_listening_channels() shows this session's channels.
      - lines: "7-11"
        note: >
          Inside a transaction, LISTEN is only queued: test_runs is still the
          only channel until COMMIT. (In the sql runtime, which rolls the whole
          script back, a LISTEN never takes effect at all.)
      - lines: "13-16"
        note: >
          UNLISTEN * was rolled back, so both channels are still registered.
          Listen registrations follow the same all-or-nothing rule as data.
      - lines: "21"
        note: >
          NOTIFY's payload must be a string literal: fine for a constant,
          useless for a computed value or a bind parameter.
      - lines: "22"
        note: >
          pg_notify(channel, payload) is a function, so both arguments can be
          expressions or $n parameters. It returns void, shown as an empty
          value. Keep JSON payloads to keys, not whole records.
      - lines: "25"
        note: >
          ş takes two bytes in UTF-8: 3999 of them are 7998 bytes, 4000 are
          8000.
      - lines: "26-27"
        note: >
          SQLSTATE 22023. In the default configuration the payload must be shorter than
          8000 bytes, so Turkish text reaches the limit at about half as many
          characters as ASCII. Send an id and let the listener read the row.
  - id: extensions
    file: examples/02_extensions.sql
    title: "Trusted and untrusted extensions; what CREATE EXTENSION installs"
    runtime: sql
    check: exact
    annotations:
      - lines: "3-8"
        note: >
          pg_available_extension_versions lists what is installed on the
          server's disk, with a trusted flag (PG 13+). The list differs between
          installations: the Windows installer ships extras the Docker image
          doesn't, which is why the query names the ones it shows.
      - lines: "12-13"
        note: >
          Run as the non-superuser playground. A trusted extension needs only
          the CREATE privilege on the current database, which a database's
          owner has.
      - lines: "15-23"
        note: >
          The extension object belongs to playground, but its member objects
          (47 for pg_trgm: functions, operators, operator classes and more) are
          owned by the bootstrap superuser, so the installer can drop the
          extension but not alter its parts. pg_depend entries of type 'e' are
          how DROP EXTENSION knows what to remove.
      - lines: "31-34"
        note: >
          unaccent strips the Turkish diacritics, dotless ı included, and
          similarity compares sets of trigrams. Together they let
          "sifre sifirlama", typed on a keyboard without Turkish letters, find
          "Şifre sıfırlama". The search module goes deeper.
      - lines: "37-39"
        note: >
          gin_trgm_ops is an operator class the extension added: a GIN index
          that answers LIKE and ILIKE with a leading wildcard. enable_seqscan =
          off only makes the planner show the index on a three-row table; it
          isn't a production setting.
      - lines: "42"
        note: >
          SQLSTATE 42501. file_fdw reads files on the database server, so it
          isn't trusted: only a superuser may create it.
  - id: table-events
    file: examples/03_table_events.go
    title: "The database as an event source: the tcn extension and a pgx listener"
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "48-50"
        note: >
          Channel names are shared by everything connected to the database.
          Deriving one from this run's schema keeps concurrent runs apart; an
          application uses a fixed name per purpose.
      - lines: "51"
        note: >
          tcn is a trusted contrib extension: CREATE on the database is enough.
          DATABASE_URL puts this run's own schema first on search_path, so the
          extension's function lands there.
      - lines: "53-54"
        note: >
          triggered_change_notification must be an AFTER ... FOR EACH ROW
          trigger; its only argument is the channel (default "tcn"). Trigger
          arguments are literals, never bind parameters, so the name is
          concatenated: acceptable only because it was generated, not typed
          by a user.
      - lines: "55"
        note: >
          LISTEN runs on a separate, dedicated connection. pgx.Identifier quotes
          the name, so the channel keeps its exact spelling.
      - lines: "57-61"
        note: >
          The trigger fired for both rows inside the open transaction, yet
          nothing has arrived: notifications are delivered when the sending
          transaction commits.
      - lines: "62-63"
        note: >
          tcn's payload: the quoted table name, I, U or D, and the primary key
          columns with their values. Enough to know which row to re-read,
          never the row itself.
      - lines: "69-70"
        note: >
          A rolled-back transaction delivers nothing, exactly like its writes.
use_cases:
  - "Waking a worker the moment a job row is committed, instead of polling every few seconds (with the job table as the source of truth)"
  - "Pushing changes to browsers: a Go server LISTENs and fans out over SSE (fullstack.tracelite.step-09-realtime)"
  - "Invalidating in-process caches on every app instance when reference data changes"
  - "Extensions: fuzzy and accent-insensitive search (pg_trgm, unaccent), GiST equality for exclusion and temporal constraints (btree_gist), query statistics (pg_stat_statements), geospatial and vector data (PostGIS, pgvector)"
strengths:
  - "Transactional delivery: a notification goes out if and only if its transaction commits, which a separate broker like Redis pub/sub can't promise without an outbox table"
  - "No extra infrastructure, and no load while nothing happens, unlike polling"
  - "A trigger (or tcn) makes every writer announce changes, including psql and migrations"
  - "Extensions add real types, operators and index methods that the planner uses like built-in ones, not a library layered on top"
weaknesses:
  - "No persistence, no acknowledgement, no replay: a listener that is disconnected misses notifications for good"
  - "Payloads are text under 8000 bytes, and each listener holds a dedicated connection that doesn't fit transaction-mode poolers"
  - "Extensions are per database and depend on files on the server: what exists on your laptop may not exist on a managed service or in a Docker image"
  - "An untrusted extension needs a superuser, and native code in an extension runs inside the server process with full access"
pitfalls:
  - title: "Using NOTIFY as a message queue"
    detail: "Messages sent while a listener is down or reconnecting are lost. Keep events in a table, send ids (or nothing), and have the listener catch up from the table after every LISTEN. Competing workers claim rows with FOR UPDATE SKIP LOCKED."
  - title: "Expecting LISTEN to work at once inside a transaction"
    detail: "LISTEN takes effect at commit, and a session inside a transaction receives notifications only after it ends. Listen in autocommit, then read the current state in a new transaction, then rely on notifications, which is the order the LISTEN docs recommend."
  - title: "Listening through a transaction-mode pooler"
    detail: "LISTEN is session state. Behind PgBouncer in transaction mode the next statement may run on another server connection. Give the listener its own direct connection (pgx.Connect, or a connection held from pgxpool for its lifetime)."
  - title: "Creating extensions at application startup"
    detail: "CREATE EXTENSION needs privileges the app role shouldn't have for untrusted ones, and IF NOT EXISTS never upgrades an older version. Put it in the first migration, and run ALTER EXTENSION ... UPDATE deliberately after server upgrades."
    good: "-- migrations/0001_extensions.sql\nCREATE EXTENSION IF NOT EXISTS pg_trgm;\nCREATE EXTENSION IF NOT EXISTS unaccent;"
  - title: "Assuming an extension exists everywhere"
    detail: "The Windows installer, the Docker image and managed services ship different sets, and some (pg_stat_statements) also need shared_preload_libraries, a server setting. Check pg_available_extensions on every target before designing around one."
quiz:
  - q: "In one transaction a session runs `LISTEN jobs; SELECT pg_listening_channels();`. What does the SELECT return?"
    options: ["jobs", "No rows", "An error"]
    answer: 1
    why: "LISTEN takes effect at commit; until then the session isn't registered."
  - q: "Why use `pg_notify('jobs', $1)` rather than `NOTIFY jobs, ...` from Go?"
    options: ["pg_notify is faster", "NOTIFY's payload must be a string literal; pg_notify takes expressions and bind parameters", "pg_notify keeps the message until a listener reads it"]
    answer: 1
    why: "pg_notify is an ordinary function, so channel and payload can be computed or passed as parameters."
  - q: "The non-superuser playground can run `CREATE EXTENSION pg_trgm` but not `CREATE EXTENSION file_fdw`. Why?"
    options: ["pg_trgm is preinstalled in every database", "pg_trgm is marked trusted (no access outside the database); file_fdw reads server files", "file_fdw isn't shipped with PostgreSQL"]
    answer: 1
    why: "Trusted extensions can be created by any role with CREATE on the database; the rest need a superuser."
  - q: "A listener's connection was down for a minute while five notifications were sent. After reconnecting and running LISTEN, what does it receive?"
    options: ["All five", "None of them", "Only the last one"]
    answer: 1
    why: "Notifications are delivered only to sessions listening when they're sent; there is no persistence or replay."
further_reading:
  - title: "PostgreSQL docs: NOTIFY"
    url: "https://www.postgresql.org/docs/current/sql-notify.html"
  - title: "PostgreSQL docs: LISTEN"
    url: "https://www.postgresql.org/docs/current/sql-listen.html"
  - title: "PostgreSQL docs: CREATE EXTENSION"
    url: "https://www.postgresql.org/docs/current/sql-createextension.html"
  - title: "PostgreSQL docs: Additional Supplied Modules and Extensions (trusted list)"
    url: "https://www.postgresql.org/docs/current/contrib.html"
  - title: "PostgreSQL docs: tcn, triggered change notifications"
    url: "https://www.postgresql.org/docs/current/tcn.html"
  - title: "PostgreSQL docs: Packaging Related Objects into an Extension"
    url: "https://www.postgresql.org/docs/current/extend-extensions.html"
---

## Concept

**LISTEN/NOTIFY** is a publish/subscribe channel built into PostgreSQL. A session runs
`LISTEN channel`; any session runs `NOTIFY channel, 'payload'` (or `pg_notify(channel,
payload)`), and every session listening on that channel in the same database receives
the message. Three properties define it:

- **Transactional:** notifications are delivered when the sending transaction commits,
  and never if it rolls back. `LISTEN` and `UNLISTEN` also take effect at commit.
- **Ephemeral:** nothing is stored for absent listeners. A session that isn't listening
  at that moment never sees the message.
- **Small:** the payload is text shorter than 8000 bytes in the default configuration.

Put a trigger behind it and the database becomes an **event source**: every committed
change announces itself, whoever made it. The contrib extension `tcn` packages that
trigger.

**Extensions** package SQL objects (types, functions, operators, index operator classes)
so they install with one `CREATE EXTENSION` per database and drop with one `DROP
EXTENSION`. The files must already be on the server. Since PG 13, extensions marked
**trusted** (they give no access outside the database) can be created by any role with
`CREATE` on the database; the rest need a superuser. About fifty modules ship with
PostgreSQL itself, and a large outside ecosystem, such as PostGIS for geospatial data and
pgvector for vector similarity search, builds on the same mechanism.

## Coming from C/C++/Java/Python/JS

- **Redis pub/sub:** the same fire-and-forget fan-out with no persistence, plus one thing
  Redis can't do: delivery tied to the commit of the transaction that wrote the data.
- **Kafka or RabbitMQ:** durable, acknowledged, replayable. NOTIFY is none of these. Use
  it as a doorbell next to a table, the way an outbox table wakes a relay.
- **Spring's `@TransactionalEventListener`:** its default phase, after commit, is the
  semantics NOTIFY gives you natively, and across processes.
- **npm, pip, Go modules:** extensions are packages, but they are installed into the
  server by whoever controls its files, and enabled per database. `CREATE EXTENSION` is
  closer to loading an installed shared library than to `npm install`.

The Go side, receiving notifications with pgx, catching up after reconnects and folding
of duplicates, is go.services.listen-notify. The SQL runners here don't display
asynchronous messages, so example 3 is a small Go program.

## How it works

Notifications go through one server-wide queue (8 GB in a standard installation). A message
stays queued until every listening session has read it, and sessions read only between
transactions, so a listener stuck in a long transaction stops the queue's cleanup; the
log warns when the queue is half full, and a full queue makes `NOTIFY` fail at commit.
Within one transaction, identical channel-and-payload pairs are delivered once and the
rest in the order sent; transactions are delivered in commit order.

An extension is a control file plus versioned SQL scripts in the server's extension
directory. `CREATE EXTENSION` runs the script for the default (or requested) version,
creating the objects in the extension's schema (by default the first schema on
`search_path`), and records each object as a member in `pg_depend`. `ALTER EXTENSION ...
UPDATE` runs the upgrade scripts. Extensions that need shared memory or hooks, such as
pg_stat_statements, must also be listed in `shared_preload_libraries`, which takes a
server restart.

## Design notes

Treat NOTIFY as a wake-up call, never as the data. The table holds the events; the
payload is an id or empty; the listener, on every (re)connect, runs `LISTEN`, then catches
up from the table, then waits. Use `pg_notify` with bind parameters, one channel per
purpose, and a dedicated connection for listening. When you need durability,
acknowledgements, replay or fan-out across many services, that is a message broker's job.

Choose extensions like dependencies, because they are. Prefer trusted contrib modules,
check that each one exists on every target (local mode, Docker, a managed service's
allow-list), create them in the first migration, and decide which schema they live in.
The CREATE EXTENSION docs advise installing only into schemas where untrusted roles
can't create objects. An extension written in C runs inside the server process: before
adopting a third-party one, trust it as you would trust the server itself.
