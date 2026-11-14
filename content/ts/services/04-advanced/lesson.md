---
id: ts.services.advanced
title: "Retries with backoff, timeouts, pagination, uploads with progress"
track: ts
module: services
order: 4
level: advanced
minutes: 15
since: es2022
prerequisites: [ts.services.fetch, ts.async.cancellation]
compare_with: [go.services.retries, go.services.pagination-streaming, sql.indexes.stats-pagination, sql.modifying.returning-upsert]
tags: [retry, backoff, jitter, idempotency, pagination, cursor, progress, streams, upload]
summary: >
  A production client retries only what is safe to repeat, with exponential
  backoff, jitter and the server's Retry-After; pages with server-issued cursors
  instead of offsets; and reports progress by counting streamed bytes, or by
  uploading in parts, because fetch has no upload progress events.
examples:
  - id: retry
    file: examples/01_retry.ts
    title: "Retries: what to retry, how long to wait, and the Idempotency-Key"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-8"
        note: >
          sleep and random are injected, so the example runs instantly and
          prints the same delays every time. In production they are a
          setTimeout promise and Math.random; in tests, no fake timers needed.
      - lines: "9-10"
        note: >
          Retry what may go away: timeouts (408), rate limits (429) and
          server-side trouble (500, 502, 503, 504). A 400, 401, 404 or 422
          will say the same thing next time. The idempotent methods are the
          ones HTTP defines as safe to repeat; POST and PATCH are not.
      - lines: "13-14"
        note: >
          A POST is repeatable only with an Idempotency-Key: the server stores
          the result under the key and answers a repeat with the stored result
          instead of charging twice. The header is an IETF draft and a
          widespread convention; the server has to implement it.
      - lines: "18-19"
        note: >
          A deadline per attempt, so one hung connection doesn't use up the
          whole budget. In real code, merge it with the caller's signal through
          AbortSignal.any() so a cancel stops the retries too. clone(), because
          sending a request consumes its body: the same Request can't be sent
          twice.
      - lines: "21-24"
        note: >
          Only "no response" is retried here: a network TypeError or this
          attempt's TimeoutError. A caller's abort or a bug in your own code is
          not a reason to try again.
      - lines: "25-28"
        note: >
          A POST without a key that got no response has an unknown outcome: the
          server may have committed it. Saying so is the honest answer;
          retrying blindly is how users get charged twice.
      - lines: "29-31"
        note: >
          Exponential backoff with full jitter: a random wait below a ceiling
          that doubles per attempt, so clients that failed together don't retry
          together. A Retry-After header wins. It can also be an HTTP date,
          which needs Date.parse and an injected clock; this sketch handles the
          seconds form only.
      - lines: "70-73"
        note: >
          The key is created once per user intent (one click on Pay) and reused
          by every retry of that intent. A new key per attempt would defeat it.
  - id: pagination
    file: examples/02_pagination.ts
    title: "Offset vs cursor pagination, and following the server's next link"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "15-19"
        note: >
          Offset paging counts rows: "skip 2". When a row is inserted in front,
          everything shifts by one, so page 2 repeats order 4. A deleted row
          shifts the other way and one is skipped. PostgreSQL also has to read
          and throw away every skipped row.
      - lines: "20-26"
        note: >
          Cursor (keyset) paging remembers a position: "older than order 4".
          Inserts and deletes elsewhere don't move it, and an index on id finds
          the start directly (sql.indexes.stats-pagination). The cursor is
          opaque: the client treats it as a token, so the server may change its
          contents later.
      - lines: "30-40"
        note: >
          The client follows the next URL the server returns until it is null,
          and never builds page URLs itself. An async generator fetches a page
          only when the loop asks for it, and break stops it.
      - lines: "46-48"
        note: >
          The same insert in both runs. Offset shows 4 twice; cursor shows every
          order exactly once (the new one appears at the top on the next
          refresh).
  - id: progress
    file: examples/03_progress.ts
    title: "Download progress from the body stream; upload progress in parts"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-6"
        note: >
          Progress is measured in bytes. Each of these Turkish letters is 2
          bytes in UTF-8, so 30000 characters are 60000 bytes: compute progress
          from file.size, not from string length.
      - lines: "25-37"
        note: >
          Reading the body yourself instead of calling json() or blob(): each
          read() returns the next chunk the network delivered, so the count of
          bytes so far is the progress. Chunk sizes are up to the browser; here
          the fake server fixes them at 16 KiB.
      - lines: "39-44"
        note: >
          Content-Length can be missing (a chunked response) or count the
          compressed bytes when Content-Encoding is gzip, while the stream
          yields decompressed ones. Treat it as a hint and fall back to an
          indeterminate progress bar.
      - lines: "46-49"
        note: >
          fetch exposes no upload progress at all. Sending a file in parts gives
          progress that means "the server has this much", and a failed part is
          retried alone. S3 multipart uploads and the tus protocol work this way.
      - lines: "57"
        note: >
          Blob.slice cuts at byte offsets and can split a multi-byte character.
          That is fine for upload parts, which the server reassembles before
          anyone decodes them; never decode a part on its own.
use_cases:
  - "Mobile and flaky networks: retry idempotent requests with backoff, give non-idempotent ones a key"
  - "Rate-limited third-party APIs: honour 429 and Retry-After instead of hammering"
  - "Infinite scroll and exports over live data: cursor pagination with server-issued next links"
  - "Large file uploads (videos, CSV imports) with a progress bar and resume after a dropped connection"
strengths:
  - "Composable platform pieces: AbortSignal for deadlines, Request.clone() for re-sending, async generators for pages, streams for progress"
  - "Injected sleep and randomness make retry logic testable in microseconds, without fake timers"
  - "Server-issued cursors let the server change its paging scheme without a client release"
  - "Uploading in parts survives flaky connections: only the failed part is sent again"
weaknesses:
  - "fetch has no built-in retry, no upload progress and no pagination helper; each is code you own and test"
  - "Retries multiply across layers and can turn a short outage into an overload the clients caused themselves"
  - "Cursor pagination can't jump to page 7 or show a total cheaply"
  - "Download progress depends on Content-Length, which compression and chunked responses make unreliable"
pitfalls:
  - title: "Retrying requests that aren't safe to repeat"
    detail: "A POST that timed out may have succeeded. Retry it only with an Idempotency-Key the server honours, created once per user action."
    bad: "for (let i = 0; i < 3; i++) { try { return await fetch(url, { method: \"POST\", body }); } catch {} }"
    good: "const headers = { \"Idempotency-Key\": crypto.randomUUID() }; // once per click\nawait send(doFetch, new Request(url, { method: \"POST\", body, headers }), policy);"
  - title: "Retrying 4xx responses"
    detail: "400, 401, 403, 404 and 422 won't change on a retry. Only 408 and 429 among the 4xx codes mean \"try again later\"."
  - title: "Fixed delays, or no cap"
    detail: "Every client that failed at the same moment retries at the same moment. Use exponential backoff with jitter, a maximum number of attempts, and the caller's deadline."
  - title: "Retrying at every layer"
    detail: "Three attempts each in the browser, in a backend-for-frontend and in the service behind it make up to 27 requests to the last service. TanStack Query already retries failed queries three times by default; don't add a retrying fetch wrapper under it."
  - title: "Offset pagination over data that changes"
    detail: "Inserts duplicate items across pages and deletes skip them. Use a keyset cursor for feeds, exports and infinite scroll."
quiz:
  - q: "POST /payments timed out with no response. What does the client know?"
    options: ["The payment failed; retry it", "The payment succeeded", "Nothing: the server may or may not have committed it"]
    answer: 2
    why: "A lost response says nothing about the server side. Without an Idempotency-Key the only safe move is to report the unknown outcome, or check the order's status."
  - q: "Why add jitter to exponential backoff?"
    options: ["To make each retry faster", "So clients that failed at the same moment don't all retry at the same moment", "Because Retry-After requires it"]
    answer: 1
    why: "Without jitter, synchronized clients form waves that hit the recovering server together. A random wait spreads them out."
  - q: "A list pages with ?offset=. A new item is inserted at the top between page 1 and page 2. What does the user see?"
    options: ["Nothing unusual", "The last item of page 1 again at the top of page 2", "One item skipped"]
    answer: 1
    why: "Everything shifts down by one, so offset 2 now starts at the item that was at offset 1. A delete does the opposite and skips one."
  - q: "How can a browser app show upload progress for a 500 MB file?"
    options: ["fetch's progress event", "XMLHttpRequest's upload progress events, or uploading in parts with fetch", "Reading response.body while the upload runs"]
    answer: 1
    why: "fetch has no upload progress events. xhr.upload fires progress events; parts give progress that the server has confirmed."
further_reading:
  - title: "MDN: Idempotent (glossary)"
    url: "https://developer.mozilla.org/en-US/docs/Glossary/Idempotent"
  - title: "MDN: Retry-After"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Retry-After"
  - title: "MDN: Request.clone()"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Request/clone"
  - title: "MDN: Using readable streams"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Streams_API/Using_readable_streams"
  - title: "MDN: XMLHttpRequestUpload"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/XMLHttpRequestUpload"
  - title: "MDN: Link header"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Link"
---

## Concept

The plain client from the first lesson becomes production-grade by adding four
behaviours, each a small piece of code you own.

**Retries.** Retry a request only when the failure may go away *and* repeating it is
safe. "May go away": no response at all, a timed-out attempt, 408, 429, 500, 502, 503 and
504. "Safe": the method is idempotent (GET, HEAD, OPTIONS, PUT, DELETE), or the request
carries an `Idempotency-Key` that the server uses to recognise a repeat. Wait between
attempts with exponential backoff and jitter, honour `Retry-After` (seconds or an HTTP
date), and stop at a maximum number of attempts or at the caller's deadline.

**Timeouts.** Two of them: one per attempt (`AbortSignal.timeout`), and one for the whole
operation, which is usually the caller's signal. Combine them with `AbortSignal.any()`.

**Pagination.** Offset paging (`?offset=40`) is simple and lets users jump to a page, but
it shifts when rows are inserted or deleted, and it gets slower with depth. Cursor paging
(`?cursor=…`, keyset in SQL) is stable and fast, and the server should hand out the next
URL or cursor so the client never constructs one. Some APIs put that URL in a `Link`
header with `rel="next"`; reading it cross-origin requires the server to list `Link` in
`Access-Control-Expose-Headers`.

**Progress.** Download progress comes from reading `response.body` chunk by chunk and
counting bytes. Uploads are the gap: fetch has no upload progress events.
`XMLHttpRequest` still has them, and uploading in parts gives progress that the server
has confirmed:

```ts
const xhr = new XMLHttpRequest();
xhr.upload.addEventListener("progress", (e) => {
  if (e.lengthComputable) bar.value = e.loaded / e.total;
});
xhr.open("POST", "/api/uploads");
xhr.send(file);
```

A listener on `xhr.upload` makes a cross-origin request preflighted (next lesson). A
`ReadableStream` request body (with `duplex: "half"`) lets you count bytes as fetch reads
them, but that measures what the browser took, not what the server received, and support
differs between browsers.

## Coming from C/C++/Java/Python/JS

- **Python**: urllib3's `Retry(total=..., backoff_factor=..., status_forcelist=...)`
  retries only `allowed_methods`, which by default excludes POST and PATCH, and respects
  `Retry-After`: the same policy as `send()` here.
- **C**: the curl tool's `--retry` treats a timeout and HTTP 408, 429, 500, 502, 503, 504
  (plus Cloudflare's 522 and 524) as transient, doubles its wait between attempts, and
  lets `Retry-After` override the wait.
- **Go**: the same loop, with `context` instead of `AbortSignal` (go.services.retries). A
  Go request body is also read once; `http.Request.GetBody` returns a fresh copy, the job
  `Request.clone()` does here. `http.Transport` itself replays a request after a network
  error on a reused connection only if it counts as idempotent: GET, HEAD, OPTIONS, TRACE,
  or any request with an `Idempotency-Key` header.
- **Java**: teams reach for Resilience4j or Spring Retry. In the browser there is no
  standard library for this: write the loop, or let TanStack Query retry queries.
- **JS**: TanStack Query retries failed queries three times with exponential backoff by
  default and doesn't retry mutations; keep fetch-level retries out of code that runs under
  it.

## Design notes

Put the policy in one place, the API client, as data: attempts, base delay, which
statuses, which methods. Callers shouldn't sprinkle retry loops. Inject the clock, the
sleep and the random source; deterministic tests of retry logic are otherwise slow and
flaky.

Retry at one layer only, and give the whole operation a budget: a deadline signal from the
caller, merged with the per-attempt timeout. When the user navigates away, the cancel
should stop the waiting as well as the request.

Idempotency is a contract with the server. The client creates the key per user intent and
keeps it for that intent's retries (store it with the pending form if a reload must not
duplicate the order); the server stores the outcome under the key, typically with a unique
constraint and an upsert (sql.modifying.returning-upsert), and replays it.

For pagination, let the server own the cursor format and return the next link; for
progress, report what you can measure honestly: bytes read, parts acknowledged, or an
indeterminate spinner.
