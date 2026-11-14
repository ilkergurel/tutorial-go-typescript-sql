---
id: fullstack.tracelite.step-10-external
title: "Step 10: External service: standards lookup with timeouts, retries, caching; outbound webhooks"
track: fullstack
module: tracelite
order: 10
level: advanced
minutes: 15
since: go1.27
prerequisites: [fullstack.tracelite.step-09-realtime, go.services.client, go.services.retries, go.services.webhooks-hmac]
compare_with: [design.architecture.reliability, design.architecture.security, go.concurrency.context, ts.services.advanced]
tags: [tracelite, http-client, timeouts, retries, backoff, jitter, retry-after, cache, webhooks, hmac, idempotency]
summary: >
  TraceLite calls a standards catalogue with a deadline per attempt, retries only what
  may go away, honors Retry-After within its own budget and serves a stale title rather
  than an error. It calls subscribers' webhooks the same way, signed with HMAC and
  carrying an event id that makes redelivery harmless.
examples:
  - id: lookup
    file: examples/01_lookup.go
    title: "One attempt at the catalogue: its own timeout inside the caller's deadline, and what each answer means"
    runtime: go-server
    check: exact
    timeout_s: 15
    annotations:
      - lines: "16-19"
        note: >
          Two errors callers test with errors.Is. ErrUnknownClause is an answer,
          which example 3 caches. errTransient marks what may go away; wrapping
          it with %w next to the cause keeps both visible.
      - lines: "23"
        note: >
          context.WithTimeout derives from the caller's context, so an attempt
          ends at whichever comes first: its own 300 ms or the API request's
          deadline. TraceLite uses 2 s per attempt.
      - lines: "29-32"
        note: >
          No answer at all (a refused connection, a reset, this attempt's
          timeout) is transient: the next attempt may well get through.
      - lines: "34-45"
        note: >
          The status decides. 200 is the title, 404 an answer, 429 and 5xx may
          go away, and anything else is TraceLite's mistake, which no retry
          fixes.
      - lines: "36"
        note: >
          A size limit on what is read back, so a broken catalogue can't make
          TraceLite read without end. The real catalogue answers JSON, and
          TraceLite decodes it without RejectUnknownMembers, unlike its own API:
          a client of someone else's API accepts members it doesn't know yet.
      - lines: "41-42"
        note: >
          Retry-After is either a number of seconds or an HTTP date. This reads
          the seconds form; a date, or no header, counts as no wish at all.
      - lines: "53-56"
        note: >
          The slow answer: the handler waits until the client gives up. The
          attempt fails after 300 ms whatever the catalogue does, which keeps
          the run deterministic.
      - lines: "65"
        note: >
          The client's Timeout is only a backstop for code that forgets a
          context; http.DefaultClient has none. The client is built once and
          shared, so connections are reused.
      - lines: "66"
        note: >
          The API request's budget: 2 s in all, shared by every attempt. The
          slow attempt spends 300 ms of it.
  - id: retries
    file: examples/02_retries.go
    title: "Retrying what may go away: full jitter, Retry-After, and never a wait past the caller's deadline"
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "15-32"
        note: >
          The retry loop, as in go.services.retries: a bounded number of
          attempts, and a wait between them that the caller can cancel.
      - lines: "17-19"
        note: >
          Only errTransient is retried. A 404 or a 400 fails the same way every
          time, so it returns at once.
      - lines: "23"
        note: >
          Full jitter: a random wait between zero and a ceiling that doubles
          (200 ms, then 400 ms), so that many clients failing together don't
          retry together. Retry-After wins when it is longer; a server under
          load knows better.
      - lines: "24-26"
        note: >
          A wait longer than the time left can only end in a timeout, so the
          loop gives up at once and the caller can fall back to its cache
          (example 3).
      - lines: "53"
        note: >
          Three lookups: 503, a timeout, then 200 (three attempts, one answer);
          a 429 asking for 120 s against a 2 s budget; three 502s, which fail
          after the last attempt. The API answers 503 for the last two, unless
          the cache has an older title.
  - id: cache
    file: examples/03_cache.go
    title: "The cache: answers for 24 hours, 404s for 10 minutes, and a stale title when the catalogue is down"
    runtime: go
    check: exact
    annotations:
      - lines: "20-24"
        note: >
          One entry per reference. err holds ErrUnknownClause for a remembered
          404, so a typo doesn't cost a call per keystroke.
      - lines: "33-39"
        note: >
          Cache first. A fresh entry, including a remembered 404, answers
          without any call.
      - lines: "40"
        note: >
          The call runs outside the lock: it may take the request's whole
          budget, and other lookups shouldn't wait for it. Two concurrent
          misses for one reference both call the catalogue (see Weaknesses).
      - lines: "42-45"
        note: >
          An answer is kept for 24 hours, a 404 for 10 minutes, so a newly
          added clause appears soon.
      - lines: "46-48"
        note: >
          The catalogue failed and an older title exists: it comes back
          marked stale. Degraded, not broken, and the UI can say so.
      - lines: "77-79"
        note: >
          25 hours later the entry has expired, the catalogue fails, and the
          lookup returns the old title with stale: true.
  - id: webhooks
    file: examples/04_webhooks.go
    title: "Outbound webhooks: one body signed per attempt; the receiver checks the MAC, the time and the event id"
    runtime: go
    check: exact
    annotations:
      - lines: "17-23"
        note: >
          The envelope. id is minted once per event, when the failed run is
          recorded, and stays the same across every retry to every subscriber:
          it is the key receivers deduplicate by. data is abridged here.
      - lines: "25-30"
        note: >
          go.services.webhooks-hmac's scheme: the timestamp is part of the
          signed bytes, so a captured request can't be replayed with a new
          time. Each webhook has its own secret.
      - lines: "36"
        note: >
          sync.Map's LoadOrStore checks and records an id in one step, safe for
          concurrent requests. A real receiver inserts the id into a table in
          the transaction that does the work, with ON CONFLICT DO NOTHING.
      - lines: "43"
        note: >
          The header is "t=<unix seconds>,v1=<hex>". A malformed one leaves got
          empty, which can't match any MAC.
      - lines: "44-48"
        note: >
          The receiver checks the HMAC over the raw bytes with hmac.Equal, which
          doesn't leak timing information, and refuses timestamps more than
          five minutes away from its own clock.
      - lines: "54-57"
        note: >
          Deduplication by event id is the receiver's half of idempotency:
          TraceLite will deliver some events twice, and this is where twice
          becomes once. The duplicate still gets a 2xx, or the sender would keep
          trying.
      - lines: "64-65"
        note: >
          Marshaled once: every attempt sends the same bytes. Marshaling again
          could even change them: encoding/json/v2 writes map entries in no
          particular order unless the Deterministic option is set.
      - lines: "68"
        note: >
          Signed at each attempt, so the timestamp is fresh and a retry minutes
          later still falls inside the receiver's window.
      - lines: "73-75"
        note: >
          The lost response: attempt 1 was processed, but its answer never
          reached TraceLite, which timed out and sent attempt 2. The receiver
          recognizes the id and does nothing twice.
      - lines: "76-77"
        note: >
          Someone without the secret, and a captured request replayed later:
          both 401. The id check would catch the replay too, but only while
          the receiver still remembers the id; the window means it has to
          remember ids for minutes, not forever.
use_cases:
  - "Enriching your data from a third-party catalogue: standards, postal codes, exchange rates, VAT numbers"
  - "Notifying CI, chat or ticketing systems when a test run fails"
  - "Any server-to-server call where the other side can be slow, rate-limited or down for an hour"
  - "Keeping a page usable while a dependency is down, by showing slightly old data marked as such"
strengths:
  - "Every call is bounded by the caller's deadline, so a slow catalogue slows the detail page by at most its budget and never hangs a handler"
  - "Full jitter spreads retries from many clients, and Retry-After is honored, so a struggling service isn't hammered back down"
  - "Cached answers, including 404s, cut traffic to the catalogue to roughly one call per reference per day"
  - "HMAC signatures with a timestamp and per-event ids give receivers authenticity, replay protection and exactly-once processing on top of at-least-once delivery"
weaknesses:
  - "The in-memory cache is per process and empties on restart; several TraceLite instances would each call the catalogue"
  - "Concurrent misses for the same reference each call the service; golang.org/x/sync/singleflight would merge them into one call"
  - "In-process webhook retries die with the process; durable delivery needs the job table of step 11"
  - "Webhook URLs come from users, so the server makes requests to addresses users chose; a multi-user deployment must refuse private and loopback targets"
pitfalls:
  - title: "http.DefaultClient and no context"
    detail: "DefaultClient has no timeout, and a request without a deadline can hang a handler until the TCP connection dies. Derive a per-attempt context from the caller's."
    bad: "resp, err := http.Get(catalogue + \"/clauses/\" + ref)"
    good: "ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second); defer cancel(); req, _ := http.NewRequestWithContext(ctx, \"GET\", u, nil)"
  - title: "Retrying everything"
    detail: "A 400 or 403 fails the same way every time, and a retried POST without an idempotency key may run twice. Retry timeouts, 429 and 5xx, and only operations that are safe to repeat."
  - title: "Sleeping past the deadline"
    detail: "A Retry-After of 120 s inside a 2 s request budget only produces a timeout two seconds later. Compare the wait with ctx.Deadline() and give up, or serve from the cache, at once."
  - title: "Re-encoding the event per attempt"
    detail: "Building the body again for a retry can change it: a new timestamp, or map entries in another order (encoding/json/v2 doesn't sort them unless the Deterministic option is set). Marshal once, sign per attempt, send the same bytes."
  - title: "Treating a timeout as a failure to deliver"
    detail: "A webhook that timed out may have been processed. Retry it with the same event id, and expect receivers to deduplicate; never mint a new id for a retry."
quiz:
  - q: "The catalogue answers 429 with Retry-After: 120, and the API request has 2 s left. What does the client do?"
    options: ["Waits 120 s, then retries", "Gives up at once, and serves the cached title if there is one", "Retries immediately with jitter"]
    answer: 1
    why: "Waiting longer than the budget can only end in a timeout. The client compares the wait with its deadline, and Lookup falls back to a stale entry."
  - q: "A webhook delivery times out. Was the event processed?"
    options: ["No", "Yes", "Maybe: the receiver may have processed it and the answer been lost"]
    answer: 2
    why: "That is why delivery is at least once, retries reuse the event id, and receivers deduplicate on it."
  - q: "Why is a 404 from the catalogue cached?"
    options: ["HTTP requires it", "So that repeated lookups of a wrong reference don't each call the service; for 10 minutes, not 24 hours", "To return it as stale later"]
    answer: 1
    why: "A negative answer is still an answer. A shorter lifetime lets a newly added clause appear soon."
  - q: "Why is the timestamp inside the HMAC rather than only in a header?"
    options: ["To make the header shorter", "So a captured request can't be replayed later with a fresh timestamp", "Because hmac.Equal needs it"]
    answer: 1
    why: "If the time weren't signed, an attacker could change it freely and pass the receiver's tolerance check forever."
further_reading:
  - title: "package net/http: Client (Timeout) and NewRequestWithContext"
    url: "https://pkg.go.dev/net/http#Client"
  - title: "package context: WithTimeout"
    url: "https://pkg.go.dev/context#WithTimeout"
  - title: "package math/rand/v2: N"
    url: "https://pkg.go.dev/math/rand/v2#N"
  - title: "package crypto/hmac: Equal"
    url: "https://pkg.go.dev/crypto/hmac#Equal"
  - title: "package encoding/json/v2: Deterministic"
    url: "https://pkg.go.dev/encoding/json/v2#Deterministic"
  - title: "MDN: Retry-After"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Retry-After"
  - title: "MDN: 410 Gone"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Status/410"
---

## Concept

TraceLite talks to two kinds of outside services, and is polite to both the same way:

- **A standards catalogue it calls.** Requirements cite clauses of standards
  (`ISO27001:A.5.17`). Step 6's detail view gains a lookup that shows a clause's title,
  fetched through `GET /api/standards/{ref}`, which calls the catalogue. The browser never
  calls it directly: that would need CORS, would expose any API key, and would bypass the
  cache.
- **Webhooks it calls on behalf of users.** A project can register URLs that receive a
  signed POST when a test run fails, so a CI system or a chat channel hears about it.

Politeness has the same four parts in both directions:

1. **Bound every call.** A deadline per attempt, derived from the caller's context, and a
   size limit on what is read back.
2. **Retry only what may go away**, and only operations that are safe to repeat: network
   errors, timeouts, 429 and 5xx. Wait with jittered backoff, or as long as Retry-After
   asks, but never past the caller's deadline.
3. **Remember answers.** A cache with expiry turns a daily-changing catalogue into a
   daily call, and a stale entry beats an error when the service is down.
4. **Make repeats harmless.** A retried webhook carries the same event id; receivers
   deduplicate on it. HMAC signatures prove the request came from TraceLite.

## Coming from C/C++/Java/Python/JS

- **Java**: Resilience4j's Retry, TimeLimiter and a Caffeine cache are these pieces as
  annotations and decorators. Here each is a few visible lines, and the deadline comes
  from the incoming request's `context.Context` instead of a thread-local.
- **Python**: `requests` without `timeout=` waits forever, like `http.DefaultClient`;
  `urllib3.Retry(backoff_factor=..., respect_retry_after_header=True)` is the loop of
  example 2.
- **JS**: `fetch` with `AbortSignal.timeout(2000)` is `context.WithTimeout`;
  `AbortSignal.any([request, timeout])` composes the two deadlines that Go derives by
  passing the context down.
- **C/C++ (libcurl)**: `CURLOPT_TIMEOUT_MS` per transfer plus your own retry loop; the
  shape is the same, the context that ties it to the caller's budget is not.

## How it works

The catalogue behind TraceLite's own endpoint:

```yaml
  /api/standards/{ref}:
    parameters: [{ name: ref, in: path, required: true, schema: { type: string, maxLength: 64 } }]
    get:
      operationId: getStandardClause
      responses:
        "200":
          description: The clause; stale is true when the catalogue was unreachable and the title comes from the cache
          content:
            application/json:
              schema:
                type: object
                required: [ref, title, stale]
                properties: { ref: { type: string }, title: { type: string }, stale: { type: boolean } }
        "404": { $ref: "#/components/responses/Problem" }   # the catalogue has no such clause
        "503": { $ref: "#/components/responses/Problem" }   # unreachable and nothing cached
```

The catalogue's URL and key come from configuration (`TRACELITE_STANDARDS_URL`), not from
the repo; the client is built once in `cmd/tracelite` and shared by every request.

Webhooks are registered per project, in `migrations/003_webhooks.sql`:

```sql
-- +goose Up
CREATE TABLE webhooks (
  id         uuid PRIMARY KEY DEFAULT uuidv7(),
  project_id uuid NOT NULL REFERENCES projects ON DELETE CASCADE,
  url        text NOT NULL CONSTRAINT webhooks_url_valid CHECK (url ~ '^https?://'),
  secret     text NOT NULL, -- the HMAC key; the receiver holds the same one
  events     text[] NOT NULL DEFAULT '{test-run.failed}'
               CONSTRAINT webhooks_events_valid CHECK (events <@ '{test-run.failed}'),
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX webhooks_project ON webhooks (project_id);
```

- The secret is stored in clear because TraceLite must compute HMACs with it; a hash
  would be useless. `events` lists the types a subscriber wants, constrained to the ones
  that exist, so a later type is added by replacing the CHECK. Deleting a project
  cascades to its webhooks.
- One event type so far, `test-run.failed`, published by the service after a failed run
  has been committed. Body: `{"id", "type", "createdAt", "data": {"projectId", "testCase",
  "requirements", "finishedAt"}}`, marshaled once (example 4).
- The dispatcher sends it to every webhook of the project that lists the type
  (`WHERE project_id = $1 AND $2 = ANY (events)`). Each POST is an attempt like example
  1's: its own timeout (10 s) derived from the caller's context, the shared client, and
  the response body read (bounded) and closed, because the Transport reuses a keep-alive
  connection only after the body was read to the end and closed.
- Header `TraceLite-Signature: t=<unix seconds>,v1=<hex HMAC-SHA256 of "<t>.<body>">`,
  computed for each attempt, and `Content-Type: application/json`. Receivers check the
  MAC and a five-minute window, and deduplicate on `id` (example 4).
- Responses: 2xx delivered. 410 Gone is the receiver saying "stop, for good", so the
  webhook is deleted (`DELETE FROM webhooks WHERE id = $1`). 429, 5xx and timeouts are
  retried with a loop like example 2's: delivery is at least once, since a timeout means
  "maybe processed". Any other 4xx stops this delivery: the receiver refused the request,
  and sending it again won't change its mind.

```mermaid
sequenceDiagram
  participant S as TraceLite service
  participant D as Dispatcher
  participant R as Receiver
  S->>D: Publish(project, test-run.failed, id e1)
  D->>R: POST, TraceLite-Signature t=...,v1=...
  R->>R: verify, process e1, remember e1
  R--xD: (the response is lost)
  D->>D: attempt 1 timed out: wait, retry
  D->>R: POST, same body, fresh signature
  R-->>D: 200, e1 seen before, nothing done
```

## Design notes

**Stale beats broken.** The clause title is decoration on the detail page; showing
yesterday's title with a "stale" flag is better than an error box. Data that must be
current (a payment status, a permission) would fail closed instead. The choice is per
call, and the `Stale` field makes it visible to the UI rather than hiding it.

**A circuit breaker, not yet.** When a dependency is down for minutes, every request
still spends its first attempt finding out. A circuit breaker would skip the call for a
while after repeated failures. With a cache that serves most lookups and a 2 s budget,
TraceLite doesn't need one; the retry and cache code is where one would go.

**Webhooks belong in the job table.** Step 10's dispatcher retries in memory for a few
seconds. Real receivers are down for hours, and TraceLite restarts. Step 11 turns each
delivery into a row in a job table, inserted in the same transaction as the failed run
(so a committed run always gets its webhook, an outbox), with `run_after` holding the
next retry time on a schedule of minutes to hours. The signing, the event id and the
status handling stay exactly as here.

**User-chosen URLs.** A webhook URL makes TraceLite's server send requests wherever a user
points it. On a single-user local install that is the user's own business; a shared
deployment must resolve the host and refuse loopback, private and link-local addresses
before connecting, or a webhook becomes a way to reach internal services.
