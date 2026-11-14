---
id: go.services.client
title: "http.Client without timeouts is a bug; per-request context"
track: go
module: services
order: 1
level: intermediate
minutes: 12
since: go1.26
prerequisites: [go.concurrency.context, go.http.handlers]
compare_with: [ts.services.fetch, ts.async.cancellation, design.architecture.reliability]
tags: [http, client, timeouts, context, connection-pool]
summary: >
  http.Get and the zero http.Client wait forever. Give every client a Timeout as a
  backstop, give every call a context derived from the work that asked for it, and
  close (ideally drain) every response body so connections are reused.
examples:
  - id: give-up
    file: examples/01_give_up.go
    title: "Client.Timeout and a per-call deadline: two ways a call gives up"
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "28"
        note: >
          The server cancels r.Context() when the client's connection closes.
          That is how /hang ends once the client gives up, and it's the signal
          your own handlers should watch. A handler that never returns would
          also block srv.Close.
      - lines: "33"
        note: >
          For an outgoing request the context covers the whole exchange:
          getting a connection, sending, the headers and reading the body.
          http.Get and http.Post take no context, one more reason to keep them
          out of service code.
      - lines: "37-41"
        note: >
          Do returns an error only when no response arrived. When err is nil
          you own resp.Body and must close it. The defer goes after the error
          check, because resp is nil on error.
      - lines: "42-44"
        note: >
          A 404 or a 503 is a successful round trip as far as Do is concerned.
          Turning statuses into errors is your job, and belongs in one helper.
      - lines: "45"
        note: >
          io.LimitReader is the client-side MaxBytesReader: a misbehaving
          upstream can't make you buffer more than you planned for.
      - lines: "55"
        note: >
          http.Get uses DefaultClient, whose Timeout is zero: no limit.
          DefaultTransport only bounds dialing (30 s) and the TLS handshake
          (10 s); a server that accepts the connection and never answers holds
          your goroutine for good.
      - lines: "57"
        note: >
          Client.Timeout covers connecting, redirects, headers and body, and is
          the backstop for every call made with this client. /hang fails after
          one second with "Client.Timeout exceeded while awaiting headers".
      - lines: "60-62"
        note: >
          Every Do error is a *url.Error carrying the method and URL, which is
          what you want in logs. Here the port is random, so only the cause is
          printed. errors.AsType is Go 1.26. errors.Is sees through the wrapper
          either way.
      - lines: "70-72"
        note: >
          The per-call budget. /stall sends its headers, so Do returns 200, and
          the deadline then interrupts the body read. Whichever ends first,
          the context or Client.Timeout, wins.
      - lines: "71"
        note: >
          cancel releases the context's timer even when the call ends early.
          go vet's lostcancel check reports a cancel function that is never
          called.
  - id: clean-up
    file: examples/02_clean_up.go
    title: "Closing and draining bodies decides whether connections are reused"
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "24-29"
        note: >
          The server's view: StateNew fires once per accepted TCP connection,
          so the counter is exactly how many connections the client opened.
      - lines: "33"
        note: >
          A Client's Transport is its connection pool. One client for the
          program means one pool; a new Transport per request would open a
          connection (and do a TLS handshake) for every call.
      - lines: "46"
        note: >
          An unclosed body keeps its connection busy and a transport goroutine
          alive, so every call dials again. Under load this ends in "too many
          open files" or exhausted ephemeral ports.
      - lines: "47-48"
        note: >
          Closing an unread body says the rest isn't wanted. Rather than read
          1 MiB nobody needs, the Transport closes the connection, and the next
          request dials a new one. For large bodies that is the right trade.
      - lines: "49-51"
        note: >
          Read to EOF, then close: the connection goes back to the idle pool
          and the next two requests reuse it. Since Go 1.27 Close also drains
          unread bodies "up to a conservative limit" (release notes); reading
          to EOF yourself doesn't depend on where that limit lies.
  - id: propagate
    file: examples/03_propagate.go
    title: "A handler's outgoing call: budget, status mapping, cancellation"
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "21-22"
        note: >
          The upstream only sees its connection close, so its context reports
          Canceled in both slow cases. A deadline doesn't travel over plain
          HTTP; gRPC sends it along as a header (lesson 7).
      - lines: "29-30"
        note: >
          The outgoing call's context comes from r.Context(): it ends after
          300 ms, or earlier if our own caller disconnects. Using
          context.Background() here would let the work outlive the request
          that asked for it.
      - lines: "33-39"
        note: >
          Our deadline becomes 504 Gateway Timeout, any other transport
          failure 502 Bad Gateway. The caller learns which hop failed, and the
          message reveals nothing about the upstream.
      - lines: "62-65"
        note: >
          The caller leaves after 100 ms, well inside the gateway's budget.
          Cancellation travels the whole chain: caller to gateway (its
          r.Context()), gateway to upstream (the closed connection), and every
          hop stops working. The caller's error is context.Canceled, not
          DeadlineExceeded, so code can tell "not wanted" from "too slow".
use_cases:
  - "Every call from a service to another HTTP API: payment providers, internal services, object storage, webhooks you send"
  - "Handlers that fan out to upstreams and must answer within their own deadline (a gateway or BFF)"
  - "CLIs and jobs that call APIs and must fail with a clear error instead of hanging in CI"
strengths:
  - "One context carries the deadline and the caller's cancellation through every layer (HTTP calls, SQL queries, your own functions), where Java and Python configure timeouts per library, each with its own API"
  - "The Transport pools and reuses connections for you, and the same Client is safe to share between goroutines"
  - "Errors keep their cause: errors.Is(err, context.DeadlineExceeded) works through *url.Error and your own %w wrapping"
  - "httptest makes slow, hanging and broken upstreams a few lines of code, so timeout paths get tested"
weaknesses:
  - "The defaults are unsafe: http.Get, http.Post and the zero Client never time out"
  - "Statuses are not errors, so every call site (or one helper) must check StatusCode"
  - "Connection reuse depends on how you treat the body, which is invisible until a load test or a file-descriptor limit finds it"
  - "Client.Timeout includes reading the body, which is wrong for long downloads; those need a context and a progress check instead"
pitfalls:
  - title: "Calling with http.Get in service code"
    detail: "It uses DefaultClient (no timeout) and takes no context. Build a client once, with a Timeout, and make requests with NewRequestWithContext."
    bad: |
      resp, err := http.Get(url) // waits forever
    good: |
      client := &http.Client{Timeout: 10 * time.Second} // once, at startup

      ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
      defer cancel()
      req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
      if err != nil {
          return err
      }
      resp, err := client.Do(req)
  - title: "Deferring Close before checking the error"
    detail: "On error resp is nil, and the deferred resp.Body.Close() panics when the function returns."
    bad: |
      resp, err := client.Do(req)
      defer resp.Body.Close()
      if err != nil {
          return err
      }
    good: |
      resp, err := client.Do(req)
      if err != nil {
          return err
      }
      defer resp.Body.Close()
  - title: "A new Client or Transport per request"
    detail: "Each Transport has its own pool, so nothing is reused and idle connections pile up. Share one client per upstream. If you fan out to one host from many goroutines, raise Transport.MaxIdleConnsPerHost: the default is 2."
  - title: "defer cancel() inside a loop"
    detail: "Deferred calls run when the function returns, not at the end of the iteration, so every iteration's timer lives until the loop is done. Put the body of the loop in its own function, or call cancel() explicitly."
  - title: "Draining without a limit"
    detail: "io.Copy(io.Discard, resp.Body) on an error response from a broken upstream can read gigabytes. Drain through io.LimitReader and accept losing the connection beyond that."
quiz:
  - q: "client.Do(req) gets back a 503 Service Unavailable. What is err?"
    options: ["An *url.Error with status 503", "nil: the round trip worked", "http.ErrServerClosed", "context.DeadlineExceeded"]
    answer: 1
    why: "Do returns an error only for transport and policy failures. Any status, including 4xx and 5xx, is a response with a nil error."
  - q: "Client.Timeout is 2 s. Do returns after 1.5 s with the headers; the body needs 3 s more. What happens?"
    options: ["The body is read in full: the timeout already passed its check", "Reading the body fails at the 2 s mark", "Do waits for the whole body before returning"]
    answer: 1
    why: "The timer keeps running after Do returns and interrupts the body read. That's why Client.Timeout is wrong for long downloads."
  - q: "Inside an HTTP handler, which context should an outgoing call use?"
    options: ["context.Background(), so a client disconnect can't break the call", "context.WithTimeout(r.Context(), budget)", "context.TODO()", "None: Client.Timeout is enough"]
    answer: 1
    why: "Deriving from r.Context() stops the call when the caller leaves, and the timeout keeps it inside the handler's own budget."
  - q: "You read 100 bytes of a 1 MiB response and call Close. What happens to the connection?"
    options: ["It goes back to the idle pool", "The Transport reads the remaining bytes, then reuses it", "It is closed; the next request dials a new one", "It leaks"]
    answer: 2
    why: "Unread data that is not small is not drained: the Transport closes the connection. Go 1.27 drains only small bodies, up to a conservative limit."
further_reading:
  - title: "package net/http: Client, Client.Do, Transport"
    url: "https://pkg.go.dev/net/http#Client"
  - title: "package net/http: NewRequestWithContext"
    url: "https://pkg.go.dev/net/http#NewRequestWithContext"
  - title: "package net/http/httptrace"
    url: "https://pkg.go.dev/net/http/httptrace"
  - title: "Go 1.27 release notes: Response.Body drains on close"
    url: "https://go.dev/doc/go1.27"
  - title: "MDN: 504 Gateway Timeout"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Status/504"
---

## Concept

Calling another service comes with three obligations: **bound the wait**, **stop when
nobody needs the answer any more**, and **release what you opened**.

`http.Get`, `http.Post` and `http.DefaultClient` do none of the first two. A
`Client` with a zero `Timeout` has no limit, and `DefaultTransport` only bounds dialing
(30 s) and the TLS handshake (10 s). An upstream that accepts the connection and then
never answers keeps your goroutine, and whatever it holds (a request slot, a database
transaction), forever. That is why the title calls it a bug rather than a style issue.

Two layers bound a call, and whichever ends first wins:

- **`Client.Timeout`**: a ceiling for everything a call does, including redirects and
  reading the body. Set it once per client, as a backstop.
- **The request's context**: `http.NewRequestWithContext(ctx, ...)`. It carries this
  call's budget (`context.WithTimeout`) and the caller's cancellation. In a handler it is
  derived from `r.Context()`, so the call stops when the caller disconnects.

Releasing what you opened means: when `Do` returns a nil error, close `resp.Body`. Read
it to EOF if you want the connection back in the pool. And use one client per upstream
for the life of the program, because its `Transport` *is* the connection pool.

## Coming from C/C++/Java/Python/JS

- **Python**: requests doesn't time out by default either; you pass `timeout=` on every
  call, and its read timeout is per gap between bytes, not a total. Go's context
  replaces that argument, is a total, and also carries cancellation, which requests
  has no API for.
- **Java**: `java.net.http.HttpClient` blocks forever unless the request sets
  `HttpRequest.Builder.timeout(...)`; `connectTimeout` is a separate builder setting.
  In Go both the deadline and cancellation travel in the one context.
- **JS**: `fetch` has no timeout option: you pass `AbortSignal.timeout(ms)` (the
  equivalent of `WithTimeout`) or an `AbortController` signal (`WithCancel`). Like Go's
  `Do`, `fetch` doesn't fail on a 404 either (lesson ts.services.fetch).
- **C**: libcurl's `CURLOPT_TIMEOUT` defaults to 0, meaning no limit, the same trap
  with a different spelling.

## How it works

What each knob bounds, from the bottom up:

- `net.Dialer.Timeout` (in `Transport.DialContext`): the TCP connect.
- `Transport.TLSHandshakeTimeout`: the TLS handshake.
- `Transport.ResponseHeaderTimeout`: from the request being written to the response
  headers. It does not include the body.
- `Client.Timeout` and the request's context: the whole call, body included. The timer
  keeps running after `Do` returns, which is how `/stall` in example 1 fails during
  `io.ReadAll`.

When the context ends, the Transport closes the connection (on HTTP/1.1) and returns an
error that wraps `context.DeadlineExceeded` or `context.Canceled`. The server on the
other end notices the closed connection and cancels its own `r.Context()`, which is how
cancellation crosses a process boundary without any protocol support.

## Design notes

Build one client per upstream at startup and pass it in through a constructor, wrapped
in a small type with real methods (`GetUser(ctx, id)`): status checks, size limits,
decoding and error mapping then live in one place instead of at every call site. For
custom pool settings, start from `http.DefaultTransport.(*http.Transport).Clone()` and
change what you need; a hand-written `&http.Transport{}` loses the proxy, dial and
HTTP/2 defaults.

Think of deadlines as a budget that shrinks on the way down. A handler with 2 s to
answer should give its upstream call less than that, so it still has time to answer
504 instead of being cut off itself. Plain HTTP doesn't transmit deadlines, so each hop
sets its own; gRPC does transmit them (lesson 7).

`Client.Timeout` is wrong for large downloads, because it caps the body as well. Use a
context there, and cancel it when no progress is made for a while (lesson 3). Since
Go 1.27, closing an unread `Response.Body` drains small remainders for you, so a plain
`defer resp.Body.Close()` usually keeps connections reusable; draining explicitly, with a
limit, makes it deterministic.
