---
id: go.http.middleware
title: "Middleware as function wrappers: logging, recovery, request ID, timeouts"
track: go
module: http
order: 3
level: intermediate
minutes: 13
since: go1.23
prerequisites: [go.http.handlers, go.errors.panic-strategy, go.concurrency.context]
compare_with: [go.cli.slog, inside.codebase.http-layer, design.patterns.classic]
tags: [http, middleware, context, slog, recover, timeouts]
summary: >
  Middleware is a function from http.Handler to http.Handler. Wrapping order is
  the whole design: request ID outside logging, logging outside recovery. Wrapped
  ResponseWriters need Unwrap, and timeouts only help handlers that watch their context.
examples:
  - id: chain-request-id
    file: examples/01_chain_request_id.go
    title: "The middleware type, a chain helper, and a request ID in the context"
    runtime: go-server
    check: exact
    timeout_s: 10
    annotations:
      - lines: "12"
        note: >
          The whole abstraction is one function type. No framework interface,
          no next() callback: a middleware receives the next Handler and
          returns a new one that calls it, or doesn't.
      - lines: "14-21"
        note: >
          chain applies the list back to front (slices.Backward, Go 1.23), so
          the code reads in request order. Without it you write
          a(b(c(app))), which is fine for two layers and unreadable for six.
      - lines: "23-32"
        note: >
          Code before next.ServeHTTP runs on the way in, code after it on the
          way out. The output shows the onion: outer, inner, handler, inner,
          outer.
      - lines: "34"
        note: >
          A key of an unexported type can't collide with another package's
          key, which a plain string like "requestID" can. The context
          documentation says so explicitly.
      - lines: "36-46"
        note: >
          Configuration arrives through the outer function (here the ID
          generator), the classic closure-based constructor. Behind your own
          proxy you may reuse the X-Request-ID it sends instead; accept it
          only from that proxy and cap its length, otherwise clients choose
          what lands in your logs.
      - lines: "43"
        note: >
          A request is never modified in place: WithContext returns a shallow
          copy carrying the new context. Go has no thread-local storage, so
          this is how per-request data travels, explicitly.
      - lines: "57-60"
        note: >
          Handlers run on many goroutines at once, so even a counter needs
          atomic.Int64 or a mutex. In production, crypto/rand.Text (Go 1.24)
          returns an unguessable ID.
      - lines: "63-64"
        note: >
          A ResponseRecorder runs the whole chain without a server: ServeHTTP
          is just a method call, which is also why middleware is easy to test.
  - id: logging-recovery
    file: examples/02_logging_recovery.go
    title: "A status-recording access log, panic recovery, and why wrappers need Unwrap"
    runtime: go-server
    check: exact
    timeout_s: 10
    annotations:
      - lines: "12-22"
        note: >
          A ResponseWriter can't tell you the status afterwards, so the logger
          wraps it. Embedding the interface forwards everything else. The
          default must be 200, because a handler that only calls Write never
          calls WriteHeader.
      - lines: "24-25"
        note: >
          The wrapper hides the real writer's optional methods (Flush,
          SetWriteDeadline, Hijack). http.ResponseController (Go 1.20) follows
          Unwrap to find them. StackLab's own statusRecorder gained this
          method after its absence silently broke the run route's deadline.
      - lines: "27-33"
        note: >
          One log line per request, written after the handler returns, when
          the status is known. Log the duration here too in real code; it is
          left out only to keep the output stable.
      - lines: "35-45"
        note: >
          The request boundary from the panic lesson as a reusable layer. It
          sits inside logging, so the access log records the 500 it writes.
          Placed outside, the panic would unwind through logging and that
          request would never be logged. A production recoverer also
          re-panics http.ErrAbortHandler, the value a handler panics with to
          abort a response on purpose.
      - lines: "62-66"
        note: >
          struct{ http.ResponseWriter }{w} is a wrapper without Unwrap. The
          same Flush call works through statusRecorder and fails with
          http.ErrNotSupported through the bare wrapper. For a streaming
          endpoint that means a response that never streams.
      - lines: "67"
        note: >
          The chain reads outside-in: logging(recoverer(mux)). Middleware that
          must see every request, including 404s and 405s, wraps the mux;
          middleware for one route wraps that route's handler.
  - id: timeouts
    file: examples/03_timeouts.go
    title: "http.TimeoutHandler and handlers that do or don't watch their context"
    runtime: go-server
    check: exact
    timeout_s: 10
    annotations:
      - lines: "13-19"
        note: >
          The client got its 503 long ago, but this goroutine keeps working
          until the sleep ends, and its write fails with ErrHandlerTimeout. A
          timeout layer can't stop code; it can only tell the code to stop.
      - lines: "24-29"
        note: >
          TimeoutHandler gives the handler a context with a deadline. Selecting
          on r.Context().Done(), or passing r.Context() to database and HTTP
          calls, which do the same, stops the work when the client is no
          longer waiting.
      - lines: "35-37"
        note: >
          TimeoutHandler buffers the whole response so it can replace it with
          a 503 at any moment. That is also why it doesn't support Flush:
          never put it in front of a streaming or SSE endpoint.
      - lines: "56-58"
        note: >
          The handlers report through a channel, so main prints their side
          after the client's. Without that, the order of the lines would
          depend on scheduling.
use_cases:
  - "Access logs, metrics and tracing that wrap every route, including the mux's own 404s and 405s"
  - "Request IDs that tie a client's error report to every log line of that request"
  - "Recovering panics per request into a 500 plus a logged stack"
  - "Per-route policies: a body limit or a TimeoutHandler on one endpoint, authentication on a sub-mux"
strengths:
  - "No framework: a middleware is a plain function, testable with httptest and composable with any router that speaks http.Handler"
  - "Order is visible in one expression, unlike annotation- or registration-order-driven filter chains in Java containers"
  - "Context values carry request-scoped data explicitly through every layer and goroutine, where Java's MDC and Python's thread-locals rely on thread affinity"
weaknesses:
  - "Wrapping ResponseWriter hides optional interfaces; every wrapper needs Unwrap, and older code that type-asserts w.(http.Flusher) still breaks behind a wrapper"
  - "context.Value is untyped: a typo in the key type or a wrong assertion yields the zero value, not a compile error"
  - "No error return from handlers: error mapping needs a convention of your own (the appHandler pattern) or panics"
  - "A timeout can't preempt a goroutine; a handler that ignores its context keeps running after the client has its 503"
pitfalls:
  - title: "A ResponseWriter wrapper without Unwrap"
    detail: "Flush, SetWriteDeadline and Hijack then fail with http.ErrNotSupported. Streaming stalls and per-route deadlines silently don't apply."
    bad: "type rec struct {\n\thttp.ResponseWriter\n\tstatus int\n}"
    good: "func (r *rec) Unwrap() http.ResponseWriter { return r.ResponseWriter }"
  - title: "Recovery outside logging"
    detail: "The panic unwinds through the logging middleware's ServeHTTP before anything recovers it, so the log line after next.ServeHTTP never runs. Put recovery inside logging, or log from a defer."
  - title: "Strings as context keys"
    detail: "context.WithValue(ctx, \"user\", u) collides with any other package that uses \"user\". Declare type ctxKey struct{} and expose typed getters."
  - title: "Context values for dependencies"
    detail: "Databases, loggers with configuration and feature flags belong in constructor arguments. The context documentation limits values to request-scoped data that transits processes and APIs."
  - title: "TimeoutHandler on a streaming endpoint"
    detail: "It buffers the response and doesn't support Flush, so an SSE stream or a large download is held back and then cut off with a 503. Use the context deadline or ResponseController.SetWriteDeadline for those routes."
quiz:
  - q: "srv := chain(app, a, b) with the chain helper from example 1. In which order do the layers see the request?"
    options: ["app, b, a", "a, b, app", "b, a, app", "It depends on registration"]
    answer: 1
    why: "chain wraps back to front, so the first listed middleware is outermost: it sees the request first and the response last."
  - q: "Recovery middleware wraps logging middleware (recoverer(logging(mux))). A handler panics. What does the access log show?"
    options: ["The request with status 500", "The request with status 200", "Nothing for that request", "The panic value"]
    answer: 2
    why: "The panic unwinds through logging before recovery catches it, so logging's code after next.ServeHTTP never runs."
  - q: "A middleware wraps w in a struct that embeds http.ResponseWriter and has no other methods. The handler calls http.NewResponseController(w).Flush(). Result?"
    options: ["The response is flushed", "A compile error", "An error matching http.ErrNotSupported", "A panic"]
    answer: 2
    why: "Embedding the interface only promotes Header, Write and WriteHeader. Without Flush or Unwrap on the wrapper, ResponseController can't reach the real writer."
  - q: "TimeoutHandler answered 503. What happens to a handler that called time.Sleep and never looks at r.Context()?"
    options: ["It is killed", "It keeps running; its later writes return http.ErrHandlerTimeout", "It panics", "Its output is appended to the 503"]
    answer: 1
    why: "Go can't kill a goroutine from outside. The deadline is a request to stop that only context-aware code honors."
further_reading:
  - title: "package net/http: TimeoutHandler"
    url: "https://pkg.go.dev/net/http#TimeoutHandler"
  - title: "package net/http: ResponseController"
    url: "https://pkg.go.dev/net/http#ResponseController"
  - title: "package context: WithValue"
    url: "https://pkg.go.dev/context#WithValue"
  - title: "The Go Blog: Go Concurrency Patterns: Context"
    url: "https://go.dev/blog/context"
---

## Concept

A middleware is any function `func(http.Handler) http.Handler`. It returns a handler that
runs code before calling `next.ServeHTTP`, after it, around it, or instead of it. Stacked
middleware form an onion: the outermost layer sees the request first and the response
last.

Three mechanics carry most real middleware:

- **Context values** pass per-request data inward: a request ID, the authenticated user.
  The middleware builds a new request with `r.WithContext(context.WithValue(...))`; handlers
  read it with a typed getter.
- **ResponseWriter wrappers** observe the response: a struct embeds the original writer
  and overrides `WriteHeader` to record the status. Since Go 1.20 such a wrapper should also
  have `Unwrap() http.ResponseWriter`, so `http.ResponseController` can still reach the real
  connection's `Flush`, deadlines and `Hijack`.
- **`defer` + `recover`** turns a panic in one request into a logged 500.

Order is a design decision, not a detail. A sound default from the outside in: request ID,
logging, recovery, then per-route concerns (authentication, body limits, timeouts), then
the handler.

## Coming from C/C++/Java/Python/JS

- **Java**: a Servlet `Filter.doFilter(req, res, chain)` is the same onion, with
  `chain.doFilter` as `next.ServeHTTP` and `HttpServletResponseWrapper` as the status
  recorder. SLF4J's MDC keeps the request ID in a thread-local; Go has no goroutine-local
  storage, so the context is passed explicitly, and it keeps working across goroutines.
- **Express**: `app.use((req, res, next) => ...)` is the closest relative. Express passes
  errors with `next(err)` to an error middleware; Go handlers have no error return, so you
  either adopt the appHandler pattern from the panic lesson or recover panics.
- **Python**: WSGI middleware wraps the application callable, exactly like a Go middleware
  wraps a Handler. Django's `get_response` style is the same idea.
- **C/C++**: think of it as function composition over one callback type, with closures
  holding the configuration.

## Design notes

Keep the chain in one place, next to the routing table, so a reader can see what every
request goes through. StackLab's `NewRouter` ends with
`withLogging(logger)(withRecovery(logger)(withRequestGuards(logger, withProblemFallbacks(mux))))`:
logging outermost, so even 404s and recovered panics are logged with their status, then
recovery, then the request guards around the mux. Its `statusRecorder` has the `Unwrap`
method this lesson explains, added after the run route's extended write deadline silently
failed behind the wrapper; the recoverer's own wrapper has it for the same reason.

Write middleware as constructors that take their dependencies (`requestID(newID)`,
`logging(log, next)`), not as code that reaches for globals. Put request-scoped data in the
context, dependencies in constructor arguments. A small `chain` helper is enough; you don't
need a library for it.

Timeouts come in three layers, and this lesson covers the middle one. Server-wide timeouts
on `http.Server` protect the connection (next lesson). Per-request deadlines, through
`TimeoutHandler` or a middleware that wraps the context, limit how long the work may take.
Per-call deadlines on each database or HTTP call protect against one slow dependency. All
three only work if handlers pass `r.Context()` down to every call that can block.
