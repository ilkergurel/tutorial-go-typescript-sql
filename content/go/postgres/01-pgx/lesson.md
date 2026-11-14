---
id: go.postgres.pgx
title: "database/sql vs native pgx; pgxpool sizing and lifecycle"
track: go
module: postgres
order: 1
level: intermediate
minutes: 14
since: go1.27
prerequisites: [go.errors.is-as-join, go.interfaces.implicit]
compare_with: [sql.security-ops.pooling-config, inside.codebase.wiring]
tags: [postgres, pgx, pgxpool, database-sql, connection-pool]
summary: >
  For a PostgreSQL-only service use pgx's native API with one pgxpool per
  process; use database/sql where a library demands *sql.DB, and bridge the two.
  Size the pool from the server's connection budget, and give every connection back.
examples:
  - id: pool
    file: examples/01_pool.go
    title: A pool of two connections, and what happens when both are taken
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "17-22"
        note: >
          ParseConfig is the only way to get a Config (NewWithConfig panics on a
          hand-built one) because it fills in the defaults: MaxConns is the
          greater of 4 and runtime.NumCPU(), MaxConnLifetime 1 hour,
          MaxConnIdleTime 30 minutes. The same settings can ride in the URL as
          pool_max_conns=10&pool_max_conn_lifetime=30m.
      - lines: "24-31"
        note: >
          NewWithConfig returns without connecting, so a wrong password or host
          would only surface on the first request. Ping at startup makes the
          deploy fail instead. This app's platform.NewPool does exactly that.
      - lines: "34-37"
        note: >
          A Query borrows a connection and keeps it until its rows are closed
          (or Next returns false). The two connections are two server
          processes, which is why each one costs the database real memory.
      - lines: "40-44"
        note: >
          The pool is empty, so the third query waits in Acquire until its
          context gives up. Without a deadline it would wait forever: a leaked
          Rows in a handler looks exactly like this, as a hang, not an error.
      - lines: "46-49"
        note: >
          Closing rows hands the connection back and the queued work proceeds.
          The deferred pool.Close() blocks until every connection is returned,
          so a leak would also hang shutdown; rows2.Close() is not optional.
  - id: database-sql
    file: examples/02_database_sql.go
    title: database/sql over the pgx driver, and the bridges between the two APIs
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "12"
        note: >
          Importing stdlib registers the "pgx" driver with database/sql in its
          init function. Code that needs nothing else from it imports it as _;
          here it is also used by name, for the two bridges below.
      - lines: "19-25"
        note: >
          *sql.DB is a pool despite its name, and its default is no limit on open
          connections (MaxIdleConns defaults to 2). Under a traffic spike it
          opens connections until PostgreSQL refuses them, so always set the
          limits yourself.
      - lines: "27-37"
        note: >
          Same query, two APIs: database/sql adds Context-suffixed methods,
          pgx takes ctx first. Scanning an array into []string through
          database/sql needs Go 1.27 and pgx 5.11+ (driver.RowsColumnScanner);
          older code wraps the target with pgtype.Map.SQLScanner.
      - lines: "39-54"
        note: >
          Raw lends you the *pgx.Conn under a database/sql connection, so
          choosing database/sql never locks you out of COPY, batches or
          LISTEN. The connection is pinned to conn until conn.Close().
      - lines: "59-63"
        note: >
          The other direction: OpenDBFromPool wraps the pgx pool as *sql.DB for
          a library such as goose. It sets MaxIdleConns to 0 so database/sql
          doesn't sit on connections the pgx side needs, and closing it leaves
          the pool open.
use_cases:
  - "pgxpool: HTTP services, workers and CLIs that only talk to PostgreSQL, especially when they need COPY, batches, LISTEN/NOTIFY or arrays, jsonb and ranges as Go values"
  - "database/sql with the pgx driver: libraries that require *sql.DB (goose, sqlx, many ORMs) and code that must also run on another database"
  - "A single *pgx.Conn: one-shot CLIs and scripts, and the dedicated connection a LISTEN loop holds for its whole life"
strengths:
  - "Unlike JDBC plus HikariCP, the pool ships with the driver: one module gives you connections, pooling, health checks and lifetime rotation"
  - "The native API exposes PostgreSQL itself (COPY, batches, notifications, numeric, intervals, ranges) instead of the common subset database/sql must offer across databases"
  - "The bridges work both ways (stdlib.OpenDBFromPool, (*sql.Conn).Raw), so the choice is never all or nothing"
  - "pgxpool's default ceiling is bounded at max(4, NumCPU); database/sql's is unlimited"
weaknesses:
  - "Native pgx code is PostgreSQL-only: switching databases means rewriting the data layer, where database/sql code would mostly need SQL dialect changes"
  - "The two APIs look alike but differ in semantics: database/sql rolls a transaction back when its context is cancelled, pgx's Begin uses the context only for BEGIN itself"
  - "MaxConns is per process; nothing in Go knows how many replicas share the server's max_connections, so the arithmetic is yours to write down"
pitfalls:
  - title: "Leaving rows open"
    detail: "Every open pgx.Rows holds a pool connection. Return early without Close and the pool drains, later queries hang until their deadline, and pool.Close() at shutdown blocks. Defer Close right after the error check, or use pgx.CollectRows and ForEachRow, which close for you."
    bad: |
      rows, err := pool.Query(ctx, "SELECT id, is_admin FROM users")
      if err != nil {
          return 0, err
      }
      for rows.Next() {
          var id int64
          var admin bool
          if err := rows.Scan(&id, &admin); err != nil {
              return 0, err // a failed Scan closes rows itself
          }
          if admin {
              return id, nil // found it, and leaked the connection
          }
      }
      return 0, rows.Err()
    good: |
      rows, err := pool.Query(ctx, "SELECT id, is_admin FROM users")
      if err != nil {
          return 0, err
      }
      defer rows.Close() // every return path now gives the connection back
  - title: "Sharing one *pgx.Conn between goroutines"
    detail: "A *pgx.Conn is not safe for concurrent use; a second caller arriving mid-query typically fails with \"conn busy\". Use pgxpool, or give each goroutine its own connection."
  - title: "Trusting database/sql's pool defaults"
    detail: "MaxOpenConns defaults to unlimited and MaxIdleConns to 2: a burst opens connections until PostgreSQL refuses with SQLSTATE 53300 (too_many_connections), and after it all but two are closed again. Call SetMaxOpenConns, SetMaxIdleConns and SetConnMaxLifetime explicitly."
  - title: "A pool per request or per repository"
    detail: "Each pool opens its own server processes and multiplies your share of max_connections. Build one pool in main, inject it, close it after the HTTP server has drained."
  - title: "PgBouncer in transaction mode"
    detail: "pgx caches named prepared statements per connection. PgBouncer 1.21+ supports that when max_prepared_statements > 0; with an older PgBouncer, set ConnConfig.DefaultQueryExecMode to pgx.QueryExecModeExec."
quiz:
  - q: "On a 2-core laptop, how many connections can a pool from pgxpool.New(ctx, url) open, with no pool_max_conns in the URL?"
    options: ["2", "4", "10", "Unlimited"]
    answer: 1
    why: "ParseConfig sets MaxConns to the greater of 4 and runtime.NumCPU()."
  - q: "max_connections is 100 with the default 3 superuser slots. Three replicas of your service and a reporting job that holds 10 connections share the server. What is the largest MaxConns per replica that cannot exhaust it?"
    options: ["33", "30", "29", "97"]
    answer: 2
    why: "(100 - 3 - 10) / 3 = 29. Every replica's pool draws on the same server budget."
  - q: "A handler calls pool.Query, then returns early on a validation error without closing the rows. What happens as traffic continues?"
    options: ["Nothing: the garbage collector closes the rows", "The pool runs dry and later queries wait until their context expires", "PostgreSQL drops the idle connection after 30 minutes and the pool heals"]
    answer: 1
    why: "pgx has no finalizer on Rows: the connection stays checked out until Close, so each early return leaks one."
  - q: "goose needs a *sql.DB; the rest of the service uses a pgxpool. What keeps a single pool?"
    options: ["Rewrite the service on database/sql", "stdlib.OpenDBFromPool(pool)", "sql.Open(\"pgx\", url) next to the pool"]
    answer: 1
    why: "OpenDBFromPool serves database/sql from the pgx pool's connections; sql.Open would build a second pool with its own connections."
further_reading:
  - title: "package pgx (v5)"
    url: "https://pkg.go.dev/github.com/jackc/pgx/v5"
  - title: "package pgxpool: Config and ParseConfig"
    url: "https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool"
  - title: "package stdlib: pgx as a database/sql driver"
    url: "https://pkg.go.dev/github.com/jackc/pgx/v5/stdlib"
  - title: "package database/sql: DB and its pool settings"
    url: "https://pkg.go.dev/database/sql#DB"
  - title: "PostgreSQL: Connection settings (max_connections)"
    url: "https://www.postgresql.org/docs/current/runtime-config-connection.html"
  - title: "PostgreSQL: Architectural fundamentals (a process per connection)"
    url: "https://www.postgresql.org/docs/current/tutorial-arch.html"
---

## Concept

Go has two ways to talk to PostgreSQL, and pgx serves both. **database/sql** is the
standard library's database-neutral interface: you program against `*sql.DB` and plug
in a driver; `github.com/jackc/pgx/v5/stdlib` registers one called `"pgx"`. **Native
pgx** (`pgx.Conn`, `pgxpool.Pool`) skips that layer. It only speaks PostgreSQL, and in
exchange gives you COPY, batches, LISTEN/NOTIFY, every PostgreSQL type through `pgtype`,
and generic helpers such as `pgx.CollectRows`.

The choice is simple:

- A service that only uses PostgreSQL: native pgx with one `pgxpool.Pool`.
- A library demands `*sql.DB` (goose, sqlx, most ORMs), or you must support other
  databases: database/sql with the pgx driver.
- Both: keep the pgx pool and lend it out with `stdlib.OpenDBFromPool`.

A `*pgx.Conn` is one connection, which on the server is one process, and it is not safe
for concurrent use. `pgxpool.Pool` is: create it once in `main`, pass it to whatever
needs it, and close it at shutdown. Every query borrows a connection until its rows are
closed.

**Sizing.** `MaxConns` is a per-process ceiling. The real budget is the server's
`max_connections` (default 100) minus `superuser_reserved_connections` (default 3) and
`reserved_connections` (default 0), and every replica, migration job, cron script and
`psql` session draws on it: replicas × `MaxConns` + everything else must fit. Once the
server's CPUs and disks are busy, more connections don't add throughput: queries queue
on the server instead of in your pool, and each extra connection is another process
holding memory. Start from the default, watch `pool.Stat().EmptyAcquireCount()`
and `EmptyAcquireWaitTime()` (acquires that waited because the pool was empty), and grow
it only while the database still has headroom.

## Coming from C/C++/Java/Python/JS

- Java: `pgxpool.Pool` is HikariCP's `DataSource` and the driver in one. `*sql.DB` is
  JDBC with the pool built in: it's a `DataSource`, not a `Connection`, despite the name.
  There's no try-with-resources; closing rows is your job, or `CollectRows`'s.
- Python: native pgx is closer to asyncpg (PostgreSQL-only API) and database/sql to
  DB-API drivers such as psycopg; `pgxpool` plays `psycopg_pool`'s role. Placeholders are
  `$1, $2`, never `%s` or `?`.
- Node: `pg.Pool` is the same idea. The difference is that Go code calling the pool is
  ordinary blocking code; many goroutines share it with no `async` anywhere.
- C: libpq gives you one `PGconn` and no pool. pgx's own libpq-level layer is `pgconn`,
  reachable through `conn.PgConn()`.

## How it works

The pool's lifecycle settings, all on `pgxpool.Config` (defaults from `ParseConfig`):

- `MaxConnLifetime` (1 h) and `MaxConnLifetimeJitter` (0): connections are retired by
  age, so a long-running service eventually reconnects, for example to a new primary
  after a DNS change; jitter keeps them from all expiring at once.
- `MaxConnIdleTime` (30 min) and `HealthCheckPeriod` (1 min): the background check closes
  idle connections that are too old or too quiet.
- `MinConns` and `MinIdleConns` (0): keep warm connections ready so a request doesn't pay
  for the TCP and authentication handshake. `MinIdleConns` arrived in pgx 5.7.3.
- On `Acquire`, a connection that sat idle for at least a second is pinged first, so a
  connection the server dropped is replaced instead of failing your query.

The connection string also carries session settings: parameters pgx doesn't recognise
become `RuntimeParams`, sent at connect time. That is how this runner gives your program
its own schema: its `DATABASE_URL` carries `search_path=run_...`. `application_name` (to
find your service in `pg_stat_activity`) and `statement_timeout` work the same way.

## Design notes

One pool per database per process, built at the edge and injected: this app's
`server/internal/platform/platform.go` parses the URL, applies `MaxConns`, `MinConns` and
`MaxConnLifetime` from configuration, pings, and returns the pool; `main.go` defers
`pool.Close()`, which runs after the HTTP server has drained. That order matters, because
`Close` waits for every borrowed connection.

The app's migrations show the database/sql bridge in practice:
`server/internal/platform/migrate.go` gives goose a short-lived `*sql.DB` built with
`sql.OpenDB(stdlib.GetConnector(...))` from the pool's connection config, and closes it
when the migrations are done. `stdlib.OpenDBFromPool` would have shared the pool's
connections instead. Both are fine; what you avoid is a second long-lived pool nobody
counted against `max_connections`.
