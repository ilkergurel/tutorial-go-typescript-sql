---
id: fullstack.tracelite.step-09-realtime
title: "Step 9: Real-time: Postgres LISTEN/NOTIFY, a Go hub, SSE, EventSource updating the matrix"
track: fullstack
module: tracelite
order: 9
level: advanced
minutes: 15
since: go1.27
prerequisites: [fullstack.tracelite.step-08-export, go.services.listen-notify, go.http.uploads-streams, ts.services.realtime-cors]
compare_with: [sql.server-side.listen-notify-ext, sql.server-side.triggers, go.concurrency.patterns, ts.react.effects-hooks]
tags: [tracelite, realtime, listen-notify, triggers, sse, eventsource, hub, fan-out, tanstack-query, invalidation]
summary: >
  A trigger NOTIFYs at commit with the project and what changed; one Go connection
  LISTENs and a hub fans the hint out to that project's SSE streams; EventSource in the
  browser invalidates the project's queries, so the matrix refetches. Every gap ends
  in a refetch.
examples:
  - id: listen
    file: examples/01_listen.go
    title: "Migration 002 and one LISTEN connection: a commit becomes one event, a reconnect a resync"
    runtime: go
    fixtures: [fixtures/migrations/002_change_events.sql]
    check: exact
    timeout_s: 15
    annotations:
      - lines: "17-23"
        note: >
          Listen runs for the life of the server: whenever the connection
          breaks (a database restart, a network drop), it waits a second and
          connects again. TraceLite passes the hub's methods:
          Listen(ctx, dsn, hub.Publish, hub.Resync).
      - lines: "26-30"
        note: >
          One dedicated connection, not the pool: LISTEN belongs to a session,
          and a pooled connection would go back to the pool still listening.
      - lines: "31-34"
        note: >
          LISTEN first, then resync. A change committed while no connection
          was listening was never heard, so every open stream is told to
          refetch. On the very first connection there are no streams yet and
          it costs nothing.
      - lines: "36-39"
        note: >
          WaitForNotification blocks until a notification arrives, the context
          ends, or the connection fails.
      - lines: "40-47"
        note: >
          The payload is the trigger's JSON: the project and the topic, nothing
          more. One that doesn't decode is a change nobody can name; returning
          makes Listen reconnect, and the reconnect resyncs everyone.
      - lines: "62-63"
        note: >
          The migration is this example's fixture, at the path TraceLite keeps
          it (How it works shows the file). The tables have only the columns
          the trigger reads.
      - lines: "64-67"
        note: >
          A channel stands in for the hub of example 2. main waits for the
          first Resync, which proves LISTEN is active before anything is
          written.
      - lines: "68-69"
        note: >
          One INSERT of three rows fires the row-level trigger three times
          with identical payloads, and PostgreSQL delivers identical
          notifications of one transaction once: one Publish. Step 7's import
          of 500 rows sends one message the same way.
      - lines: "70-71"
        note: >
          test_runs has no project_id; the trigger finds the project through
          the test case, so the event still reaches the right project.
      - lines: "72-73"
        note: >
          The listener's backend is terminated, as a database restart would
          do. listen returns, Listen waits a second, connects, LISTENs and
          resyncs. pg_stat_activity finds the backend by its last query.
  - id: hub
    file: examples/02_hub.go
    title: "The hub: a buffered channel per stream, filtered by project; a stalled stream is dropped, never waited for"
    runtime: go
    check: exact
    annotations:
      - lines: "9-13"
        note: >
          Every open stream with its project. A flat map is enough for the few
          dozen tabs of a TraceLite install; with thousands of streams, a map
          per project would spare Publish the scan.
      - lines: "15-25"
        note: >
          The unsubscribe function closes over the channel, so the SSE handler
          can't remove anybody else's stream.
      - lines: "45-52"
        note: >
          The listener must never wait for a slow browser: while it waits it
          reads no notifications, and every project's streams stall with it. A
          stream with eight unread events is dropped instead; its response
          ends, and EventSource reconnects and refetches.
      - lines: "54-59"
        note: >
          Closing a channel twice panics. Checking membership under the lock
          makes drop safe to call from both the hub and the handler.
      - lines: "67-75"
        note: >
          The stalled tab received eight events, then its channel was closed.
          Raporlama's stream got none of Ödeme's events.
      - lines: "76-77"
        note: >
          Resync goes to every open stream, whatever its project: after the
          listener was away, nobody knows what changed.
      - lines: "78-79"
        note: >
          closeTab is what the SSE handler's deferred unsubscribe does when the
          tab closes (example 3). Raporlama's stream is the one left.
  - id: sse
    file: examples/03_sse.go
    title: "The SSE endpoint: 404 before the stream, then events flushed as they come"
    runtime: go-server
    check: exact
    timeout_s: 10
    annotations:
      - lines: "16"
        note: >
          In TraceLite, exists is the store's project lookup and subscribe is
          the hub's Subscribe method (example 2), passed as a method value.
      - lines: "18-22"
        note: >
          An unknown project is a 404 before the stream starts. EventSource
          treats any status other than 200 as final and never retries, which is
          right for a project that doesn't exist.
      - lines: "25-29"
        note: >
          text/event-stream, no caching, and no write deadline: a server-wide
          WriteTimeout would end every stream after that many seconds. retry
          sets EventSource's reconnection delay; the blank line after it
          dispatches no event, because no data came with it.
      - lines: "33-35"
        note: >
          Flush after every write, or the event waits in net/http's buffer. A
          failed flush means the client is gone.
      - lines: "41"
        note: >
          No id: field. EventSource would send it back as Last-Event-ID on
          reconnect, but TraceLite keeps no log of events to replay from; a
          reconnect refetches instead.
      - lines: "42-43"
        note: >
          A comment line every 25 seconds keeps proxies from closing a stream
          that has been quiet; EventSource ignores lines that start with a
          colon.
      - lines: "44-45"
        note: >
          Closing the tab cancels the request; the handler sees r.Context()
          done, returns, and its deferred unsubscribe removes the channel. main
          waits for exactly that on line 78.
      - lines: "73-76"
        note: >
          The wire format: each event is an event: line and a data: line,
          ended by a blank line.
  - id: live
    file: examples/04_live.tsx
    title: "The browser side: EventSource invalidates the project's queries (simulated stream)"
    runtime: ts-browser
    check: exact
    timeout_s: 15
    annotations:
      - lines: "10-14"
        note: >
          The hook depends on three members of EventSource, so the lesson can
          pass a fake. In the app it gets (url) => new EventSource(url): same
          origin through the Vite proxy, so cookies go along and no CORS is
          involved.
      - lines: "20-33"
        note: >
          One EventSource per project view, opened in an effect and closed in
          its cleanup, so switching projects or unmounting never leaves a
          stream open.
      - lines: "22-23"
        note: >
          Every event becomes an invalidation of the project prefix, the same
          one step 5's mutations use. Events carry no data to merge, so a lost
          or duplicated event can't corrupt the cache; it costs a refetch at
          most.
      - lines: "24-27"
        note: >
          open fires on the first connection and after every automatic
          reconnect. Either way, changes may have happened while there was no
          stream, so the view refetches.
      - lines: "28-30"
        note: >
          readyState tells the two kinds of error apart. After a network error
          it is back to 0 (CONNECTING) and EventSource retries by itself; after
          a non-200 answer it is 2 (CLOSED) for good, and only a reload or new
          code can try again.
      - lines: "36-45"
        note: >
          Harness: the iframe has no network, so the stream is an EventTarget
          driven by the script. emit stands for what the server or the network
          would do, with the readyState EventSource would have.
      - lines: "55-57"
        note: >
          gcTime: Infinity, so finished queries leave no garbage-collection
          timer behind and the run can end.
      - lines: "74-75"
        note: >
          The first fetch at mount, then one more when the stream opens: the
          refetch that closes the gap between loading and subscribing.
      - lines: "77-78"
        note: >
          A drop doesn't refetch; the reconnect's open does, and picks up
          whatever changed while the stream was down.
      - lines: "79"
        note: >
          A refused reconnect (the server said 404 or 503) closes the stream
          for good, and the page can say so instead of showing data as live.
use_cases:
  - "Live views that several people watch: trace matrices, dashboards, test run results as a CI pipeline posts them"
  - "Invalidating client caches when another user, an import or a background job changes data"
  - "Any server-to-browser notification where the browser never needs to send back on the same channel"
  - "Small deployments that want pub/sub without running a broker: PostgreSQL is already there"
strengths:
  - "Every writer notifies, including psql and imports, because the trigger is in the database rather than in one service's code"
  - "Notifications are transactional: sent at commit, dropped on rollback, folded when identical, so the browser never refetches data it can't see yet"
  - "SSE is plain HTTP: it goes through the Vite proxy and ordinary middleware, sends cookies, and EventSource reconnects by itself, unlike a hand-rolled WebSocket"
  - "Events are hints and every gap ends in a refetch, so lost, duplicated or reordered events can't leave the cache wrong"
weaknesses:
  - "Each event refetches whole queries; for very large matrices, a payload with the changed ids and a targeted cache update would move less data"
  - "One SSE stream per tab holds a connection; over HTTP/1.1 browsers allow only six per browser and domain, so many open tabs need HTTP/2 or a shared worker"
  - "NOTIFY is not durable: a listener that is away misses events, which is why resync exists; consumers that must see every change need an outbox table (step 11)"
  - "Notifications go through one queue for the whole database cluster; a listening session that stays in a long transaction keeps it from being cleaned up, and when it is full, transactions that notify fail at commit"
pitfalls:
  - title: "Listening on a pooled connection"
    detail: "LISTEN is per session. A pooled connection returns to the pool still listening, and the next borrower receives its notifications. Use one dedicated pgx.Conn."
  - title: "Sending the changed row in the payload"
    detail: "Payloads must stay under 8000 bytes, and a client that missed one would apply the next on stale data. Send what changed (project, topic) and let the client refetch."
  - title: "Blocking the listener on a slow subscriber"
    detail: "A plain send to a full channel stops the loop that reads notifications for every project. Send with select and default, and drop the subscriber that can't keep up."
    bad: "ch <- topic"
    good: "select { case ch <- topic: default: h.drop(ch) }"
  - title: "Forgetting the gap at (re)connect"
    detail: "Events sent between the first fetch and the stream opening, or during a reconnect, are gone. Refetch on every open, and resync everyone after the listener reconnects."
  - title: "A server WriteTimeout on streaming routes"
    detail: "http.Server.WriteTimeout ends every response after a fixed time, streams included. Clear it per handler with ResponseController.SetWriteDeadline(time.Time{})."
quiz:
  - q: "One transaction inserts 500 trace links into project Ödeme. How many notifications does the listener receive?"
    options: ["500", "1", "None until VACUUM"]
    answer: 1
    why: "Identical channel and payload pairs within one transaction are delivered once, and the payload holds only the project and the topic."
  - q: "The server answers EventSource's reconnect with 503. What does the browser do?"
    options: ["Retries after the retry: delay", "Fails the connection: readyState CLOSED, no more retries", "Falls back to polling"]
    answer: 1
    why: "Any status other than 200, or a Content-Type other than text/event-stream, fails the connection for good. Network errors and a normal end of the response reconnect."
  - q: "Why does the hub drop a subscriber whose channel is full instead of waiting?"
    options: ["Channels can't block", "Waiting would stop the listener from reading notifications for every project", "SSE requires it"]
    answer: 1
    why: "The listener is shared. A dropped stream ends, EventSource reconnects and refetches, which costs one tab a request instead of costing every tab its updates."
  - q: "Why does the browser refetch on every open event, even the first?"
    options: ["EventSource requires an acknowledgement", "Changes between the initial load and the stream opening would otherwise be missed", "To warm the cache"]
    answer: 1
    why: "Nothing replays events from before the subscription, so the view reloads once it is sure to hear about later changes."
further_reading:
  - title: "PostgreSQL docs: NOTIFY (folding, ordering, payload limit, queue)"
    url: "https://www.postgresql.org/docs/current/sql-notify.html"
  - title: "PostgreSQL docs: LISTEN"
    url: "https://www.postgresql.org/docs/current/sql-listen.html"
  - title: "pgx v5: Conn.WaitForNotification"
    url: "https://pkg.go.dev/github.com/jackc/pgx/v5#Conn.WaitForNotification"
  - title: "package net/http: ResponseController (SetWriteDeadline, Flush)"
    url: "https://pkg.go.dev/net/http#ResponseController"
  - title: "MDN: Using server-sent events"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events"
  - title: "MDN: EventSource (readyState, connection limit)"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/EventSource"
---

## Concept

A change reaches every open matrix in five hops:

1. **A trigger** (migration 002) on `requirements`, `trace_links` and `test_runs` calls
   `pg_notify('tracelite_changes', '{"projectId": …, "topic": "trace-links"}')`. PostgreSQL
   delivers it when the transaction commits, and never if it rolls back.
2. **One listener** in the Go server holds a dedicated connection with `LISTEN
   tracelite_changes` and reads notifications as they arrive (example 1).
3. **The hub** forwards each one to the channels of the streams open for that project
   (example 2).
4. **`GET /api/projects/{projectId}/events`** is a server-sent events stream: one
   long HTTP response that writes `event: trace-links` whenever the hub delivers
   (example 3).
5. **EventSource** in the browser turns each event into
   `invalidateQueries(["projects", id])`, and TanStack refetches what is on screen
   (example 4).

Events say *that* something changed, never *what*. The browser always refetches from
the API, so events can be lost, folded or repeated without the cache ever being wrong,
and every gap (page load, network drop, listener restart) ends in a refetch too.

## Coming from C/C++/Java/Python/JS

- **Java/Spring**: `SseEmitter` per client plus an in-memory registry is the same hub;
  Spring's `ApplicationEventPublisher` only sees writes from this one service, where the
  trigger sees every writer.
- **Python/Django Channels**: a channel layer (usually Redis) fans out to WebSocket
  consumers. Here PostgreSQL is the channel layer and SSE the transport; no extra
  server.
- **Node/Socket.IO**: rooms per project are the hub's map. Socket.IO reconnects and
  re-joins rooms, but can't know what was missed either; a refetch on reconnect is the
  same answer.
- **C/C++**: a producer thread and a set of bounded queues with a drop-on-full policy, the
  classic way to keep one slow consumer from stalling the others.

## How it works

The migration, `migrations/002_change_events.sql` (example 1 reads it from its fixture):

```sql
-- +goose Up
-- +goose StatementBegin
CREATE FUNCTION notify_change() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
  project uuid;
BEGIN
  IF TG_TABLE_NAME = 'test_runs' THEN
    SELECT project_id INTO project FROM test_cases
    WHERE id = CASE TG_OP WHEN 'DELETE' THEN OLD.test_case_id ELSE NEW.test_case_id END;
  ELSE
    project := CASE TG_OP WHEN 'DELETE' THEN OLD.project_id ELSE NEW.project_id END;
  END IF;
  PERFORM pg_notify('tracelite_changes',
    json_build_object('projectId', project, 'topic', TG_ARGV[0])::text);
  RETURN NULL;
END $$;
-- +goose StatementEnd
CREATE TRIGGER requirements_notify AFTER INSERT OR UPDATE OR DELETE ON requirements
  FOR EACH ROW EXECUTE FUNCTION notify_change('requirements');
CREATE TRIGGER trace_links_notify AFTER INSERT OR UPDATE OR DELETE ON trace_links
  FOR EACH ROW EXECUTE FUNCTION notify_change('trace-links');
CREATE TRIGGER test_runs_notify AFTER INSERT OR UPDATE OR DELETE ON test_runs
  FOR EACH ROW EXECUTE FUNCTION notify_change('test-runs');
```

- goose's `StatementBegin`/`StatementEnd` keep the function body, which contains
  semicolons, in one statement.
- `NEW` is null in a DELETE trigger and `OLD` in an INSERT, hence the `CASE` on `TG_OP`.
  `test_runs` has no `project_id`, so the function looks the project up through the test
  case.
- The payload names the project and the kind of change, nothing more: it must stay under
  8000 bytes, and the browser refetches anyway.
- The trigger fires for every writer: the API, step 7's importer, a psql session.

NOTIFY's rules, from the PostgreSQL documentation and shown in go.services.listen-notify:
a notification is delivered when its transaction commits and never if it rolls back;
identical channel-and-payload pairs within one transaction are delivered once; within a
transaction, notifications arrive in the order they were sent, and transactions arrive in
commit order.

The new endpoint, added next to the others in `openapi.yaml`:

```yaml
  /api/projects/{projectId}/events:
    parameters: [{ $ref: "#/components/parameters/ProjectId" }]
    get:
      operationId: streamProjectEvents
      description: >
        Server-sent events. Event names: requirements, trace-links, test-runs, resync.
        Data: {"topic": <the event name>}. No ids; reconnecting clients refetch.
      responses:
        "200": { description: The stream, content: { text/event-stream: { schema: { type: string } } } }
        "404": { $ref: "#/components/responses/Problem" }
```

```mermaid
sequenceDiagram
  participant W as Writer (API, import, psql)
  participant P as PostgreSQL
  participant L as Go listener
  participant H as Hub
  participant B as Browser (EventSource)
  W->>P: INSERT INTO trace_links ... ; COMMIT
  P-->>L: NOTIFY tracelite_changes {"projectId", "topic": "trace-links"}
  L->>H: Publish(projectId, trace-links)
  H-->>B: event: trace-links (SSE, flushed)
  B->>B: invalidateQueries(["projects", id])
  B->>P: GET trace-links, coverage (through the API)
```

What each event refetches, by key prefix (step 5): all of them invalidate
`["projects", projectId]`, which covers the tree, the lists, the matrix's links and the
coverage. Topic-specific invalidation (`trace-links` only touching links and coverage) is
a later optimization; the topic is in the event so that it can be added without a
server change.

Where the gaps are, and what closes each:

| Gap | Closed by |
|---|---|
| between the first fetch and the stream opening | refetch on `open` |
| a network drop; EventSource reconnects by itself | refetch on `open` |
| a slow tab dropped by the hub; its response ends and EventSource reconnects | refetch on `open` |
| the listener's connection broke (database restart) | `resync` to every stream after `LISTEN` is active again |
| the server answered the reconnect with 404 or 503 | nothing automatic: the page shows that it is no longer live |

## Design notes

**Trigger, not application events.** The service could publish after each commit, but
then the importer, a psql fix and any future writer would each have to remember to. The
trigger makes "committed" and "notified" the same event. Step 13 weighs the same choice
for the audit history, where the trade-offs come out differently because the record
must survive.

**SSE, not WebSockets.** The browser never sends anything on this channel; mutations
already go through the REST API. SSE is a normal HTTP response, so it works through the
Vite proxy, carries the session cookie, and reconnects without code. A WebSocket would
add a second protocol and hand-written reconnection for no message the browser needs to
send.

Vite's [server.proxy docs](https://vite.dev/config/server-options#server-proxy) don't
mention SSE; they say the proxy "Extends http-proxy-3". http-proxy-3's response pass
pipes the backend's response straight into the browser's (`proxyRes.pipe(res)` in
`passes/web-incoming.js`, bundled into Vite 7.3.7's `dist/node/chunks/config.js`), so
each event goes through as the backend flushes it. A run on 2026-11-14 (Vite 7.3.7,
Node 24.22.0) showed it: a `text/event-stream` backend writing every 500 ms, read
through a Vite dev proxy, delivered its events at 1, 507, 1014 and 1520 ms.

**Hints, not data.** A payload with the new link would let the browser patch its cache
without a request, but then a missed or reordered event leaves the cache wrong until
the next full load. Refetching costs a few small requests per change and is always
right.

**Dedicated listener, bounded fan-out.** One connection serves every tab, and the hub
never blocks it; the policy for a slow tab is "drop and let it reconnect", which
EventSource implements on the client for free. The listener resyncs after every
`LISTEN`, not only after a reconnect: on a fresh server there are no streams to tell,
and if the first connection failed while tabs were already open, they need the refetch
too.
