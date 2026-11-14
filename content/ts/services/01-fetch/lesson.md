---
id: ts.services.fetch
title: "fetch basics and pitfalls: it doesn't throw on 4xx/5xx"
track: ts
module: services
order: 1
level: intermediate
minutes: 14
since: es2022
prerequisites: [ts.async.cancellation, ts.errors.try-catch]
compare_with: [go.services.client, ts.services.advanced, ts.testing.msw-playwright]
tags: [fetch, http, response, timeout, abortsignal, textdecoder]
summary: >
  fetch rejects only when no HTTP response arrives; a 404 or a 500 fulfils the
  promise like a 200. Check response.ok, read the body exactly once, and pass a
  timeout signal, because fetch has no timeout of its own.
examples:
  - id: check-ok
    file: examples/01_check_ok.ts
    title: "A 404 and a 502 fulfil the promise; only a failed request rejects"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "5-16"
        note: >
          The fake server has fetch's exact type and returns real Response
          objects, so the client code can't tell the difference. Response.json()
          builds a JSON body with Content-Type: application/json. Taking the
          fetch function as a dependency is also how tests swap the network out.
      - lines: "25-29"
        note: >
          res.json() is typed Promise<any>, so returning it as Promise<User>
          compiles whatever arrives. The 404's error object becomes a "User"
          whose name is undefined, and the proxy's HTML page fails as a
          SyntaxError that looks like a bug in your code, not an outage.
      - lines: "40-45"
        note: >
          ok is true only for status 200-299. Check it before touching the
          body, read error bodies as text (they may be HTML or empty), and keep
          the status on the error so callers can tell "not found" from "try
          again later".
      - lines: "59-64"
        note: >
          This is what a rejection looks like: no response at all. The sandbox's
          CSP blocks the request, and offline, DNS failures and CORS refusals
          produce the same TypeError. The message is the browser's own ("Failed
          to fetch" in Chromium) and deliberately carries no details.
  - id: body-once
    file: examples/02_body_once.ts
    title: "The body is a one-shot stream: clone(), empty bodies, and the charset"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "5-12"
        note: >
          The body is a ReadableStream, and text(), json(), blob() and
          arrayBuffer() consume it. Afterwards bodyUsed is true and any further
          read rejects with a TypeError.
      - lines: "14-18"
        note: >
          clone() splits the stream so each copy can be read once. Clone before
          the first read. If only one copy is ever read, the other buffers the
          whole body in memory, so don't clone large downloads.
      - lines: "22-26"
        note: >
          A 204 has no body, and the empty string is not JSON, so json() rejects
          with a SyntaxError. Any empty body does the same, whatever the status.
      - lines: "27-45"
        note: >
          Read the text once and parse it yourself: the error can then quote
          what came back. The classic case: the session expired, the API
          redirected to the login page, fetch followed the redirect, and the
          result is a 200 with HTML.
      - lines: "47-55"
        note: >
          text() always decodes as UTF-8 and ignores the charset parameter. In
          windows-1254 (labels iso-8859-9 and latin5 too) the byte 0xDE is Ş;
          as UTF-8 it is invalid and becomes U+FFFD. Read the bytes and decode
          with the declared charset. TextDecoder throws a RangeError for a label
          it doesn't know.
  - id: timeout
    file: examples/03_timeout.ts
    title: "Timeouts with AbortSignal, and one place that classifies every outcome"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "4-16"
        note: >
          A fake with a delay that honours init.signal the way fetch does: an
          already aborted signal rejects at once, and an abort later stops the
          timer and rejects with signal.reason.
      - lines: "18-24"
        note: >
          Every way a request can end, as a discriminated union. Callers switch
          on kind and the compiler checks they handled each case, which a
          catch block holding unknown can't give you.
      - lines: "31-34"
        note: >
          fetch has no timeout option. AbortSignal.timeout() is the deadline,
          and AbortSignal.any() merges it with the caller's own cancel signal:
          whichever fires first decides the reason.
      - lines: "36-38"
        note: >
          The signal stays attached to the response. If it fires after the
          headers arrived but before the body was read, the body read rejects,
          so the deadline covers the whole exchange, as http.Client.Timeout
          does in Go.
      - lines: "39-48"
        note: >
          The DOMException's name says why the signal fired: TimeoutError for
          the deadline, AbortError for abort(). A TypeError means no response.
          Anything else, such as a 200 whose body isn't JSON, is a bug and
          propagates.
      - lines: "59-62"
        note: >
          A cancel is not a failure: nobody wants the answer anymore. Keeping
          it a separate kind lets the UI skip the error message.
use_cases:
  - "Every call to your own API, through one small wrapper that owns the ok check, the timeout and the error type"
  - "Showing the server's error message: check the status first, then read the error body (often application/problem+json)"
  - "Downloading files and legacy exports: arrayBuffer() and TextDecoder with the charset the server declared"
  - "Logging raw responses in development: clone() before the parser reads the body"
strengths:
  - "One API in browsers, workers and Node (built in since Node 18), replacing XMLHttpRequest and Node's http.request"
  - "Response is a constructible value, so fakes and tests return real Response objects without a mocking library"
  - "Cancellation and deadlines use the same AbortSignal as the rest of the platform, the way Go threads context.Context through I/O"
  - "Like Go's http.Get and unlike axios, an HTTP error status is a value: the caller decides which statuses count as failures"
weaknesses:
  - "No timeout and no status check by default: every call site that forgets either is a latent bug, so wrap fetch once"
  - "res.json() returns Promise<any>, which silently satisfies any declared return type"
  - "Offline, DNS, CORS and CSP failures all arrive as one TypeError whose message differs between browsers"
  - "No upload progress events; XMLHttpRequest still has them (ts.services.advanced)"
pitfalls:
  - title: "Treating a fulfilled fetch as success"
    detail: "A 404, a 500 or a proxy's 502 page all fulfil the promise. Check res.ok (or the exact status) before reading the body."
    bad: "const user: User = await (await fetch(url)).json();"
    good: "const res = await fetch(url);\nif (!res.ok) throw new HttpError(res.status, await res.text());\nconst body: unknown = await res.json();"
  - title: "json() on a 204 or an HTML page"
    detail: "An empty body and an HTML error page both reject with a SyntaxError that hides the real cause. Read the text once, return undefined for an empty body, and include the status, the Content-Type and the start of the body in the error."
  - title: "Reading the body twice"
    detail: "The second read rejects with a TypeError. Keep the text you already read, or clone() before the first read."
  - title: "No timeout"
    detail: "Without a signal, a hung server keeps the request, and your spinner, pending for as long as the browser allows."
    bad: "await fetch(url);"
    good: "await fetch(url, { signal: AbortSignal.timeout(10_000) });"
  - title: "Matching on error messages"
    detail: "\"Failed to fetch\" is Chromium's wording; Firefox and Safari say something else. Check instanceof TypeError and the DOMException name instead."
quiz:
  - q: "await fetch(\"/api/orders/9\") reaches the server, which answers 404 with a JSON error body. What happens?"
    options: ["The promise rejects with a TypeError", "The promise fulfils; res.ok is false and res.status is 404", "The promise rejects with an HTTP error carrying the status"]
    answer: 1
    why: "fetch rejects only when no response arrives. Any status, including 404 and 500, fulfils the promise; checking ok is your job."
  - q: "A fetch rejects with a TypeError. Which of these cannot be the cause?"
    options: ["The device is offline", "The server didn't allow the cross-origin read (CORS)", "The server answered 500", "The page's Content Security Policy blocks the URL"]
    answer: 2
    why: "A 500 is a response, so the promise fulfils with ok === false. The other three leave fetch with no response to return."
  - q: "fetch(url, { signal: AbortSignal.timeout(3000) }): the headers arrive after 1 s, the body takes 5 s more. What happens?"
    options: ["fetch fulfils and json() succeeds: the timeout only covers the request", "fetch fulfils, then json() rejects when the signal fires at 3 s", "fetch rejects at 3 s"]
    answer: 1
    why: "fetch fulfils when the headers arrive, but the signal still governs the body: once it fires, reading the body rejects."
  - q: "A logger and the parser both need the response body. What works?"
    options: ["Read it twice; the browser caches the body", "Call res.clone() before the first read and give one copy to the logger", "Call res.json() and then res.text()"]
    answer: 1
    why: "A body is a one-shot stream. clone() before reading gives two independently readable copies; reading the text once and sharing it works too."
further_reading:
  - title: "MDN: Window.fetch()"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Window/fetch"
  - title: "MDN: Using the Fetch API"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch"
  - title: "MDN: Response"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Response"
  - title: "MDN: Response.clone()"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Response/clone"
  - title: "MDN: AbortSignal.timeout()"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/AbortSignal/timeout_static"
  - title: "MDN: Encoding API encodings (TextDecoder labels)"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Encoding_API/Encodings"
  - title: "net/http Client (Go, for comparison)"
    url: "https://pkg.go.dev/net/http#Client"
---

## Concept

`fetch(url, init)` returns a `Promise<Response>` that settles in two phases. The promise
fulfils as soon as the status line and the headers arrive. The body comes afterwards, as a
`ReadableStream` you consume with `text()`, `json()`, `blob()`, `arrayBuffer()` or
`formData()`.

The promise **rejects** only when there is no HTTP response to hand you:

- a `TypeError`: network failure (offline, DNS, connection reset), a cross-origin response
  that CORS doesn't let the page read, a URL the page may not fetch (blocked by its Content
  Security Policy), an invalid URL or invalid options. The browser deliberately reveals
  nothing more.
- a `DOMException` from the request's `signal`: `AbortError` after `abort()`,
  `TimeoutError` from `AbortSignal.timeout()`.

Everything else **fulfils**: a 404, a 500, a proxy's 502 page, a redirect that ends on the
login page. `response.ok` is `true` for status 200-299 only, and checking it is your job.

Three rules for the body:

- **Once.** Reading consumes the stream; `bodyUsed` turns `true` and a second read rejects.
  `clone()` before the first read if two consumers need it.
- **`json()` is `JSON.parse` over the text.** An empty body (204) or an HTML page rejects
  with a `SyntaxError`.
- **`text()` is always UTF-8**, whatever charset the `Content-Type` declares.

And one for time: `fetch` has **no timeout option**. Pass
`signal: AbortSignal.timeout(ms)`; the same signal also aborts reading the body.

## Coming from C/C++/Java/Python/JS

- **Python**: `requests.get()` doesn't raise on a 404 either. `raise_for_status()` is the
  `if (!res.ok) throw` line, and as with `requests` without `timeout=`, nothing stops a
  hung server for you.
- **Java**: `HttpClient.send()` returns an `HttpResponse` for every status and throws only
  for I/O failures (`IOException`), so `if (res.statusCode() >= 400)` is the same check you
  write by hand.
- **Go**: `http.Get` returns `err == nil` for a 404, and you check `resp.StatusCode`: the
  same rule. Go's zero `http.Client` has no timeout either, and `Client.Timeout`, like the
  signal here, also covers reading the body. JavaScript has no `defer resp.Body.Close()`;
  if you don't need a body, `res.body?.cancel()` releases it early instead of leaving it
  to the garbage collector.
- **JS (axios)**: axios rejects on any non-2xx status by default and has a `timeout`
  option. Code ported from axios to fetch silently stops catching HTTP errors.
- **C (libcurl)**: `curl_easy_perform` returns `CURLE_OK` for a 404 unless you set
  `CURLOPT_FAILONERROR`.

## Design notes

Wrap fetch once and never call it raw from components. A small module (`api/http.ts`) owns
the base URL, the default timeout, the `ok` check, an error type that carries the status
and the start of the body, and the `Content-Type` check before parsing. It returns
`unknown`, not `any`: the next lesson validates that `unknown` with zod at exactly this
point.

Take the fetch function as a parameter with the real one as default
(`doFetch: typeof fetch = fetch`). The examples' fake servers use the same seam, and so do
unit tests; MSW, in the testing module, intercepts at the network layer instead, without
the seam.

Choose an error model deliberately. Throwing an `HttpError` suits TanStack Query, which
expects a rejected promise; an `Outcome` union suits code that must handle every case, and
the compiler checks the `switch`. In both, keep "cancelled" distinct and silent, treat
timeouts, 5xx and 429 as retry candidates (ts.services.advanced), and other 4xx as final.
Error bodies from well-behaved APIs are `application/problem+json` (RFC 9457); this app's
Go backend writes every error that way (`problem()` in `server/internal/httpapi`).
