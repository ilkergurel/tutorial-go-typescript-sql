---
id: go.services.retries
title: "Retries with backoff and jitter; 429 and Retry-After; idempotency"
track: go
module: services
order: 2
level: advanced
minutes: 15
since: go1.27
prerequisites: [go.services.client]
compare_with: [ts.services.advanced, design.architecture.reliability, sql.modifying.returning-upsert]
tags: [http, retries, backoff, jitter, idempotency, rate-limiting]
summary: >
  Retry only failures that may go away, only requests that are safe to repeat, and
  only inside the caller's deadline: exponential backoff with full jitter, the
  server's Retry-After when it sends one, and an Idempotency-Key for POSTs.
examples:
  - id: jitter
    file: examples/01_jitter.go
    title: "Exponential backoff, with and without jitter"
    runtime: go
    check: exact
    annotations:
      - lines: "16-18"
        note: >
          The ceiling doubles per attempt and is capped, so a long outage
          doesn't turn into hour-long sleeps. Clamping attempt before the
          shift matters too: see the last two lines of output.
      - lines: "29-31"
        note: >
          A seeded PCG source makes this output repeatable. Production code
          uses the top-level rand.N, which is randomly seeded per process, so
          two replicas never share a sequence.
      - lines: "33-36"
        note: >
          Without jitter every client that failed together retries together:
          an overloaded service gets the same spike again at 100 ms, 200 ms,
          400 ms. That synchronized wave is what keeps it down.
      - lines: "39"
        note: >
          Full jitter: a uniformly random wait below the ceiling. Rand.N is a
          generic method, new in Go 1.27 together with generic methods
          themselves; it accepts time.Duration directly, so there is no int64
          conversion.
      - lines: "41-43"
        note: >
          100ms << 40 overflows int64 into a negative Duration. time.Sleep
          returns at once for it, turning backoff into a hot loop, and Rand.N
          panics on a non-positive argument.
  - id: parse-retry-after
    file: examples/02_parse_retry_after.go
    title: "Retry-After: delay-seconds or an HTTP-date"
    runtime: go
    check: exact
    annotations:
      - lines: "14"
        note: >
          strconv.Atoi accepts a sign, and the header's grammar doesn't: a
          negative value is treated as no hint rather than as "retry
          immediately".
      - lines: "17"
        note: >
          There is no Retry-After parser in net/http, but http.ParseTime reads
          all three date formats HTTP allows: the preferred IMF-fixdate and
          the obsolete RFC 850 and asctime formats.
      - lines: "21-25"
        note: >
          Measuring the date against the response's own Date header cancels
          out clock skew between the two machines; the local clock is only
          the fallback. A date in the past means "retry now", not a negative
          wait.
  - id: retry-after
    file: examples/03_retry_loop.go
    title: "A retry loop that honours 429/503 and Retry-After within a budget"
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "21-24"
        note: >
          This example gives up on network errors. A GET could be retried after
          one too; a request that isn't idempotent can't, because the server
          may have acted on it before the connection died (example 4).
      - lines: "25"
        note: >
          Close the body before waiting, so the connection is back in the pool
          during the sleep instead of held by a goroutine that's doing nothing.
      - lines: "26-29"
        note: >
          Retry statuses that mean "try later": 429 and 503 here, 502 and 504
          as well for idempotent requests. A 400 or 404 will be just as wrong
          the second time.
      - lines: "30-33"
        note: >
          The server's hint wins over our own schedule. Without one, back off
          with full jitter as in example 1.
      - lines: "34-36"
        note: >
          When the server asks for longer than our budget allows, fail now and
          say why. Sleeping until the deadline would fail anyway, later, with
          a less useful error.
      - lines: "38-42"
        note: >
          The wait itself watches ctx, so a caller that gives up isn't kept
          waiting by a sleep. time.Sleep can't be interrupted; a select on a
          timer and ctx.Done() can.
  - id: idempotency-key
    file: examples/04_idempotency_key.go
    title: "A lost response, a retried POST, and the Idempotency-Key that saves it"
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "28-34"
        note: >
          The server's half of the contract: the first request with a key does
          the work and stores the result under the key; every repeat gets the
          stored result back. In a real service the key row and the charge
          commit in one transaction, with a unique constraint on the key.
      - lines: "35-37"
        note: >
          The charge is committed, then the connection dies:
          http.ErrAbortHandler makes the server drop the connection without
          answering. To the client this looks exactly like a request that
          never arrived; nothing on its side can tell the two apart.
      - lines: "44-45"
        note: >
          A new request per attempt: the first send consumes the body. For a
          bytes.Reader, NewRequest also sets GetBody, which the Transport uses
          when it replays a request itself.
      - lines: "63"
        note: >
          crypto/rand.Text (Go 1.24) returns at least 128 random bits as text.
          The IETF draft recommends a UUID or a similar random identifier.
      - lines: "66"
        note: >
          A new server per run also means a fresh connection. On a reused
          connection, net/http's Transport would itself retry this POST once:
          it treats any request with an Idempotency-Key header as idempotent.
      - lines: "70"
        note: >
          One key per logical operation, sent unchanged with every attempt of
          that operation. A fresh key per attempt would defeat the point.
use_cases:
  - "Calls to rate-limited third-party APIs (payment, SMS, e-mail providers) that answer 429 with Retry-After"
  - "Internal calls during deploys and failovers, when a few 502/503 responses are expected"
  - "Payment, order and booking endpoints, where a client must be able to retry a POST without doing it twice"
  - "Background jobs that sync with an external API and must survive transient outages"
strengths:
  - "The whole policy is a short loop you can read, with no annotation magic like Spring's @Retryable"
  - "Context makes the overall budget explicit: the loop stops when the caller's deadline arrives, not after a fixed attempt count"
  - "math/rand/v2's seeded PCG and Rand.N make jittered schedules testable and repeatable"
  - "Idempotency keys turn at-least-once delivery into exactly-once effects, which no amount of client-side cleverness can"
weaknesses:
  - "No retry policy in the standard library; the Transport's built-in retry covers only one narrow case"
  - "Retries add load exactly when the upstream is struggling; without jitter, caps and a budget they prolong outages"
  - "Idempotency needs the server's cooperation (key storage, replay, expiry); the client alone can't make a POST safe"
  - "Retry-After has two formats and no parser in net/http"
pitfalls:
  - title: "Retrying everything"
    detail: "A POST that timed out may have succeeded. Retry automatically only GET, HEAD, OPTIONS, PUT and DELETE (idempotent by RFC 9110), or requests carrying an idempotency key the server honours. Never retry 4xx other than 408 and 429."
  - title: "Retrying at every layer"
    detail: "A handler retries 3 times a service call that retries 3 times a database call that retries 3 times: 27 attempts per user request during an outage. Retry in one layer, usually the one closest to the failing call, and make the rest fail fast."
  - title: "Sleeping with time.Sleep"
    detail: "It ignores cancellation and keeps the goroutine (and whatever it holds) busy after the caller left."
    bad: |
      time.Sleep(wait)
    good: |
      t := time.NewTimer(wait)
      defer t.Stop()
      select {
      case <-t.C:
      case <-ctx.Done():
          return ctx.Err()
      }
  - title: "A new idempotency key per attempt"
    detail: "Generate the key when the user's action starts (and persist it with the pending operation if the client can crash), not inside the retry loop."
  - title: "Unclamped exponentials"
    detail: "base << attempt overflows for large attempts and produces a negative or zero Duration. Clamp the exponent and cap the result."
quiz:
  - q: "Your POST /payments timed out after the request was fully sent. What do you know about the payment?"
    options: ["It failed: no response means no effect", "It succeeded: the request was sent", "Nothing: it may or may not have happened", "It will be rolled back by TCP"]
    answer: 2
    why: "The server may have committed and lost the response, as in example 4. Only an idempotency key makes a retry safe."
  - q: "Why add jitter to exponential backoff?"
    options: ["To make the average wait shorter", "To spread out clients that failed at the same moment, so their retries don't arrive together", "Because Go's timers are imprecise", "To satisfy RFC 9110"]
    answer: 1
    why: "Plain exponential backoff keeps synchronized clients synchronized; random waits spread the load over the whole interval."
  - q: "A 429 says Retry-After: 120 and your context has 30 s left. What should the client do?"
    options: ["Sleep 120 s anyway", "Retry immediately", "Fail now with an error that mentions the server's requested wait", "Sleep 30 s and retry once"]
    answer: 2
    why: "No attempt within the budget can succeed. Failing at once frees the caller and reports the real cause."
  - q: "Which request does net/http's Transport retry by itself after a network error on a reused connection?"
    options: ["Any request", "GET, HEAD, OPTIONS, TRACE, or any request with an Idempotency-Key header, if its body can be replayed", "Only GET", "None: Go never retries"]
    answer: 1
    why: "The Transport documentation lists exactly these conditions. It's one retry for a stale keep-alive connection, not a retry policy."
further_reading:
  - title: "package math/rand/v2: Rand.N, NewPCG"
    url: "https://pkg.go.dev/math/rand/v2#Rand.N"
  - title: "package net/http: Transport (when it retries by itself)"
    url: "https://pkg.go.dev/net/http#Transport"
  - title: "MDN: Retry-After"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Retry-After"
  - title: "MDN: 429 Too Many Requests"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Status/429"
  - title: "MDN: Idempotent"
    url: "https://developer.mozilla.org/en-US/docs/Glossary/Idempotent"
---

## Concept

A retry turns a transient failure into a success, and a permanent one into extra load.
Four questions decide whether to retry:

1. **Is the failure transient?** 429 Too Many Requests and 503 Service Unavailable say
   "later" outright; 502 and 504 usually mean a proxy lost its upstream for a moment;
   connection errors often mean a restart. Other 4xx statuses won't change on a retry.
2. **Is the request safe to repeat?** An idempotent request has the same effect whether
   it runs once or five times: GET, HEAD, OPTIONS, PUT and DELETE are defined that way by
   RFC 9110. A POST isn't, unless it carries an **idempotency key** that the server uses
   to recognize the repeat and answer with the original result.
3. **How long to wait?** Exponential backoff, capped, with **jitter**, so that clients
   that failed together don't retry together. If the server sends `Retry-After`, that
   wins.
4. **When to stop?** After a few attempts, and always when the caller's context ends.
   Retries happen inside the deadline, never in addition to it.

## Coming from C/C++/Java/Python/JS

- **Java**: Spring Retry's `@Retryable` and Resilience4j's `Retry` configure the same
  knobs (attempts, backoff, which exceptions) declaratively. In Go it's a loop you write
  once in your client type, and the context replaces the separate total-timeout setting.
- **Python**: urllib3's `Retry`, mounted into a requests `Session` through `HTTPAdapter`,
  has `total`, `backoff_factor`, `status_forcelist` and `allowed_methods`. Those names
  map directly onto the loop in example 3.
- **JS**: `fetch` never retries. The usual hand-written loop with `await new
  Promise(r => setTimeout(r, ms))` is example 3 with an `AbortSignal` in place of the
  context (lesson ts.services.advanced).
- **C**: nothing built in; libcurl leaves retries to the application as well.

## How it works

net/http's `Transport` does retry by itself, in one narrow case: a network error on a
**reused** keep-alive connection, for a request that is idempotent and whose body can
be replayed. Its documentation counts GET, HEAD, OPTIONS and TRACE as idempotent, and
also any request whose header map contains `Idempotency-Key` or `X-Idempotency-Key`.
That covers a server closing an idle connection just as you reuse it; it is not a policy
for 503s.

The `Idempotency-Key` header is specified in an IETF draft
(draft-ietf-httpapi-idempotency-key-header, not yet an RFC). It recommends a UUID or a
similar random value per operation, and says the server should answer 409 Conflict while
a request with the same key is still being processed, 422 when a key is reused with a
different payload, and 400 when a required key is missing. Payment providers have used
the same header for years.

## Design notes

Retry in one layer only, usually the client type that wraps the upstream; everything
above it should fail fast. Bound the extra load: a few attempts, a capped backoff, and in
busy systems a retry budget (for example, retries may add at most 10% to the request
rate) or a circuit breaker that stops calling an upstream that keeps failing.

The idempotency key is the client's half of a contract. The server's half is a table
keyed by the idempotency key, written in the same transaction as the effect, with a unique
constraint so two concurrent attempts can't both win (`INSERT ... ON CONFLICT`, lesson
sql.modifying.returning-upsert), and an expiry after which keys are forgotten. Generate
the key when the operation starts, not per attempt, and store it with the pending
operation if the client can crash between attempts.

Keep the policy testable: inject the random source (and, if you like, the sleep
function), as the examples do with a seeded `rand.Rand`, and a test can assert the
exact schedule without waiting for it.
