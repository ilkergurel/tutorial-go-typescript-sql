---
id: design.architecture.reliability
title: "Reliability: timeouts everywhere, retries with idempotency, graceful degradation, observability"
track: design
module: architecture
order: 3
level: advanced
minutes: 15
since: go1.23
prerequisites: [go.services.retries, go.services.client, sql.modifying.returning-upsert, go.http.middleware]
compare_with: [ts.services.advanced, go.postgres.bulk, go.cli.slog, go.http.server-lifecycle, sql.security-ops.monitoring]
tags: [reliability, timeouts, idempotency, degradation, observability, slog, readiness]
summary: >
  Four habits keep a system quiet at night: bound every wait, make retries safe with an
  idempotency key the database enforces, degrade optional parts instead of failing the
  page, and emit one structured event per request so failures explain themselves.
examples:
  - id: idempotency-table
    file: examples/01_idempotency_table.sql
    title: "The server's half of an idempotency key: two attempts race, one does the work"
    runtime: sql-multisession
    setup: |
      CREATE TABLE payments (
          id    bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
          kurus int NOT NULL
      );
      CREATE TABLE idempotency_keys (
          key        text PRIMARY KEY,
          payment_id bigint REFERENCES payments,
          created_at timestamptz NOT NULL DEFAULT now()
      );
    check: exact
    annotations:
      - lines: "4-5"
        note: >
          Claiming the key is the first statement of the transaction that does the work.
          RETURNING gives one row to the winner; ON CONFLICT DO NOTHING gives zero rows,
          not an error, to anyone else.
      - lines: "6-7"
        note: >
          The retry arrives while the first attempt is still running. Its INSERT waits on
          the primary key's uniqueness check: PostgreSQL cannot tell yet whether A's key
          will commit.
      - lines: "8-11"
        note: >
          The effect (the payment) and the record of it (payment_id on the key) are
          written in one transaction, so there is no moment where one exists without
          the other.
      - lines: "12-14"
        note: >
          After A commits, B's INSERT returns zero rows. B charges nothing and reads the
          stored result, which is what the client should get back for every repeat of
          the request.
      - lines: "16-26"
        note: >
          If the first attempt rolls back (a crash, a declined card), its key goes with
          it: B's waiting INSERT now succeeds and B does the work. A key that is claimed
          outside the work's transaction would stay claimed with no result behind it.
  - id: degrade
    file: examples/02_degrade.ts
    title: "Graceful degradation: a deadline per dependency, a fallback per panel, and a visible banner"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-21"
        note: >
          A stand-in for fetch that honours an AbortSignal the way fetch does: one
          endpoint never answers, one returns 503, and /healthz says ok because, like
          most liveness checks, it checks nothing.
      - lines: "23-25"
        note: >
          The result type records where a value came from. A degraded value that looks
          exactly like a live one is how stale data gets trusted.
      - lines: "29-31"
        note: >
          Two separate failure paths: AbortSignal.timeout ends the wait, and the res.ok
          check turns a 503 into a failure, because fetch resolves its promise for any
          HTTP status.
      - lines: "41-47"
        note: >
          Critical and optional dependencies are decided per call, not per app. The
          lesson has no fallback; the two optional panels run in parallel, each with its
          own budget, so the page waits for the slowest budget, not the sum.
      - lines: "54-55"
        note: >
          Degradation must be visible: a banner for the user, a counter for you. A
          silent fallback hides an outage until someone notices wrong data.
      - lines: "57-60"
        note: >
          A liveness probe at start-up says 200 while the API it guards answers 503. A
          one-time choice of "backend or offline mode" based on it would never notice.
  - id: wide-events
    file: examples/03_wide_events.go
    title: "One structured event per request, the route pattern, and liveness vs readiness"
    runtime: go-server
    check: exact
    annotations:
      - lines: "15-22"
        note: >
          Handlers add fields to the request's event instead of logging lines of their
          own. One line per request, with every field on it, can be filtered and
          counted; ten scattered lines per request have to be stitched together.
      - lines: "35-36"
        note: >
          ServeMux sets r.Pattern on the request it receives. This middleware keeps a
          reference to that exact request (r was reassigned), so it can read the pattern
          afterwards; reading it from the original request would give "".
      - lines: "37-41"
        note: >
          route is the pattern ("GET /notes/{id}"), not the path: a few distinct values
          instead of one per note, which is what metrics labels and dashboards need.
      - lines: "46-47"
        note: >
          The clock is injected, so the example's durations are stable and a test can
          assert on them. Real code passes time.Now.
      - lines: "50-55"
        note: >
          Liveness answers "is the process alive?" and checks nothing; failing it gets the
          process restarted. Readiness answers "should I get traffic?" and checks
          dependencies; failing it takes the instance out of rotation without a restart.
      - lines: "76-79"
        note: >
          With the database down, /healthz stays 200 (a restart would not help) while
          /readyz turns 503, and the failed request says which dependency failed and how
          long it waited.
use_cases:
  - "Any call that leaves the process: HTTP clients, database queries, queue publishes; each gets a deadline, and every layer's deadline fits inside its caller's"
  - "Any POST a client may retry (payments, orders, sign-ups): an idempotency key enforced by a unique constraint in the same transaction as the effect"
  - "Pages assembled from several sources: decide per panel what is critical, and give each optional one a budget and a fallback"
  - "Every service: one structured event per request with route, status, duration and dependency outcomes; separate liveness and readiness endpoints"
strengths:
  - "PostgreSQL's unique index makes concurrent duplicate requests safe without any locking code in the app: the second attempt waits, then sees the first one's result"
  - "Go's context carries one deadline through every call that accepts it, the database driver included; TS's AbortSignal does the same for fetch, with AbortSignal.timeout as the one-liner"
  - "log/slog gives structured, levelled events in the standard library, unlike Java's split between SLF4J, Logback and MDC, or Python's add-on structlog"
  - "A degraded mode written as data (source: live, cached, default) is testable like any other state"
weaknesses:
  - "Idempotency keys need storage, an expiry policy and care with requests that reuse a key with a different body"
  - "Every fallback is a second code path that runs rarely, so it is the least tested code in the system"
  - "Timeouts chosen without data are guesses: too short causes failures under normal load, too long lets queues pile up"
  - "Wide events cost log volume; high-cardinality fields (user IDs, paths) belong in logs and traces, not in metric labels"
pitfalls:
  - title: "Retrying in every layer"
    detail: "A client retrying 3 times, calling a service that retries 3 times, calling a database driver that retries 3 times, sends up to 27 attempts at the bottom. Retry in one layer, usually the one closest to the caller who knows the budget (go.services.retries)."
  - title: "Claiming the idempotency key in its own transaction"
    detail: "If the key commits first and the work fails after, the key is burned: every retry sees 'already done' with no result. Claim the key in the same transaction as the effect, or store a pending state that a retry can resume."
  - title: "A readiness check that is also the liveness check"
    detail: "If /healthz pings the database, an outage makes the orchestrator restart every instance in a loop, which fixes nothing and adds cold starts. Liveness checks the process; readiness checks dependencies."
  - title: "Logging the path as a metric label"
    detail: "r.URL.Path makes one time series per lesson ID or user ID. Use r.Pattern (Go 1.23) for labels and keep the concrete path in the log event."
    bad: |
      requests.WithLabelValues(r.URL.Path).Inc() // one series per note ID
    good: |
      requests.WithLabelValues(r.Pattern).Inc() // "GET /notes/{id}"
quiz:
  - q: "A retry with the same idempotency key arrives while the first attempt is still running. What does its INSERT ... ON CONFLICT DO NOTHING do?"
    options: ["Inserts a second row", "Fails with a unique violation", "Waits until the first transaction ends, then returns zero rows if it committed", "Returns zero rows at once"]
    answer: 2
    why: "The uniqueness check can't decide until the first transaction commits or rolls back, so it waits. On commit it returns no row (DO NOTHING); on rollback it inserts."
  - q: "Why does fetch need an explicit res.ok check for graceful degradation?"
    options: ["fetch rejects only on network errors and aborts; a 503 resolves normally", "fetch retries 5xx automatically", "res.ok is false for 304", "AbortSignal.timeout does not work with fetch"]
    answer: 0
    why: "A fetch promise resolves for every HTTP response, including 4xx and 5xx; only network failures and aborts reject it (ts.services.fetch)."
  - q: "The database is down. What should /healthz and /readyz answer?"
    options: ["Both 503", "/healthz 200, /readyz 503", "/healthz 503, /readyz 200", "Both 200"]
    answer: 1
    why: "The process is alive (restarting it won't fix the database), but it should not receive traffic it cannot serve. That is exactly the split between liveness and readiness."
  - q: "A middleware passes r.WithContext(ctx) to the mux and then logs r.Pattern from the original r. What does it log?"
    options: ["The route pattern", "The URL path", "An empty string", "It panics"]
    answer: 2
    why: "ServeMux sets Pattern on the request it was given, which is the copy. Keep a reference to that copy (as the example does by reassigning r) and read it there."
further_reading:
  - title: "package log/slog"
    url: "https://pkg.go.dev/log/slog"
  - title: "net/http: Request.Pattern"
    url: "https://pkg.go.dev/net/http#Request"
  - title: "MDN: AbortSignal.timeout()"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/AbortSignal/timeout_static"
  - title: "PostgreSQL: INSERT ... ON CONFLICT"
    url: "https://www.postgresql.org/docs/current/sql-insert.html#SQL-ON-CONFLICT"
  - title: "PostgreSQL: Client connection defaults (statement_timeout, idle_in_transaction_session_timeout)"
    url: "https://www.postgresql.org/docs/current/runtime-config-client.html"
---

## Concept

Systems that don't page you share four habits. Each one has its own in-depth lesson;
here they are side by side, with the parts those lessons leave to this one.

1. **Timeouts everywhere.** Every wait has an end: `http.Client` and per-call contexts
   in Go (go.services.client), `AbortSignal.timeout` on `fetch` (ts.services.advanced),
   `statement_timeout`, `lock_timeout` and `idle_in_transaction_session_timeout` in
   PostgreSQL (go.postgres.bulk shows how a query ends under each). Deadlines shrink on
   the way down: a handler with 2 s gives its database call less, so it still has time to
   answer.
2. **Retries with idempotency.** Retry only what is transient, with capped, jittered
   backoff, in one layer (go.services.retries). A retried POST is safe only if the
   server recognizes the repeat, and the robust place to recognize it is a **unique
   constraint on the idempotency key, written in the same transaction as the effect**
   (example 1).
3. **Graceful degradation.** Decide per dependency what is critical. When an optional one
   fails or is slow, show a cached or default value, say so, and keep the page up. When a
   critical one fails, fail clearly. Shed load you cannot serve (a fast 503) rather than
   queueing it until everything times out.
4. **Observability.** Emit one structured event per request with the route pattern,
   status, duration and the outcome of each dependency, so that "what failed, where,
   how long did it wait" is a query, not an investigation. Expose liveness and readiness
   separately, so orchestrators restart only what a restart can fix.

## Coming from C/C++/Java/Python/JS

- **Java**: Resilience4j (or Hystrix before it) packages timeouts, retries, circuit
  breakers and fallbacks as decorators. In Go and TS you write the same thing as a few
  functions around a call, which keeps the policy visible. MDC's per-request context
  is what the event in example 3 replaces.
- **Python**: `requests` has no default timeout either; `tenacity` is the usual retry
  library. `structlog` gives the per-request event style that `log/slog` has built in.
- **JS/Node**: `fetch` has no default timeout and does not reject on 5xx; the two lines
  in `load` are the fix for both. Promise-based fallbacks (`Promise.allSettled`, or a
  try/catch per panel as here) are the degradation tool.
- **C/C++**: `select`/`poll` timeouts and errno checks everywhere; Go's context and
  TS's AbortSignal are the same discipline with propagation built in.

## How it works

Example 1 relies on how a unique index treats concurrent inserts. When B inserts a key
that A has inserted but not committed, B cannot know whether the key will exist, so it
waits for A's transaction. If A commits, B's `ON CONFLICT DO NOTHING` skips the row and
`RETURNING` yields nothing; if A rolls back, B's insert proceeds. Because A wrote the
payment in the same transaction as the key, "key exists" and "payment exists" are always
true together. For effects outside the database (a card charge through a provider),
store the key with a `pending` state first, pass your key on to the provider's own
idempotency mechanism, and let a retry resume the pending operation.

Example 2's `AbortSignal.timeout(ms)` aborts with a `DOMException` named `TimeoutError`,
which lets the fallback say *why* it was used. Running the optional loads with
`Promise.all` over functions that never reject (they return a `Loaded` value instead)
gives the page the latest of the budgets, not their sum.

Example 3's middleware builds the event: it creates it, hands it to handlers through the
context, and logs it once after the handler returns. `r.Pattern` (Go 1.23) is the
matched routing pattern, the right key for grouping requests; the clock is a parameter,
so durations can be tested.

## Design notes

StackLab applies the habits unevenly, which makes it a useful review. Timeouts are
thorough: `http.Server` sets `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout` and
`IdleTimeout` in `main.go`; runexec bounds a run at 40 s and a queue wait at 10 s;
`runclient` has a 45 s client timeout; sqlrun sets `statement_timeout` and
`lock_timeout` on every SQL lesson session; the stores put a 3 s context on their queries.
Load shedding is real: at most two runs execute at once, and a third waits up to 10 s
for a slot, then gets `503 server busy` instead of joining an ever-longer queue. Degradation exists in two places: run history
is best effort (a failed insert is logged, the learner still gets the result), and the
frontend falls back to `localStorage` when the backend is unreachable.

That fallback is where the critical eye goes. `pickApi()` decides once, from a
`/healthz` probe with a 1.5 s timeout, and caches the answer for the life of the page. It
probes liveness, which checks nothing, so a backend whose database is down still wins
the probe. Until 2026-11-14 only an unreachable server selected the fallback, because
`fetch` resolves on any HTTP status: with the backend stopped, Vite's dev proxy answered
500 and the app kept the dead backend. A non-OK answer now counts as unreachable too. A backend that goes away after the first probe is never
re-checked, and progress saved to `localStorage` during an outage never syncs back. The
second example's shape (per call, with a budget, with the source recorded and shown) is
the direction to take if that ever matters.

Observability is minimal and adequate for one user: one `slog` line per request with
method, path, status and duration, plus `/healthz` and `/readyz` split correctly. The
first additions for a multi-user deployment would be a request ID in that line, the
route pattern next to the path, and dependency outcomes on the same event.
