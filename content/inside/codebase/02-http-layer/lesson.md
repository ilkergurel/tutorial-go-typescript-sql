---
id: inside.codebase.http-layer
title: "The HTTP layer and middleware chain"
track: inside
module: codebase
order: 2
level: intermediate
minutes: 15
since: go1.25
prerequisites: [go.http.handlers, go.http.middleware, go.http.json-validation]
compare_with: [go.design.layers, go.errors.is-as-join, go.http.server-lifecycle, ts.services.zod]
tags: [http, servemux, middleware, csrf, problem-details, rfc9457, error-mapping, responsecontroller]
summary: >
  internal/httpapi is a standard-library ServeMux with fifteen routes, wrapped in logging,
  panic recovery and three request guards; one problem() function writes every error,
  the mux's own 404 and 405 included, as RFC 9457 JSON. The run route answers 200, 400,
  502 or the runner's status; a failing program is a 200.
examples:
  - id: router-chain
    file: examples/01_router_chain.go
    title: "router.go and middleware.go in small: patterns, logging, problem details"
    runtime: go-server
    check: exact
    annotations:
      - lines: "15-21"
        note: >
          The real `statusRecorder`. The default is 200 because a handler that
          only calls Write never calls WriteHeader. Unwrap lets
          http.ResponseController reach the real connection: embedding the
          ResponseWriter interface forwards Header, Write and WriteHeader and
          hides everything else (How it works tells the bug its absence caused).
      - lines: "23-31"
        note: >
          `withLogging(logger)` returns a `func(http.Handler) http.Handler`, so
          the logger is captured by a closure and the result composes with any
          other wrapper. The real one also logs the duration.
      - lines: "33-39"
        note: >
          Every error a handler writes goes through this one function: one
          shape, `application/problem+json`, with `type: about:blank`, which
          RFC 9457 defines as "the status code says it all". Changing the
          error format means changing one function.
      - lines: "43-47"
        note: >
          Go 1.22 patterns put the method and a `{lessonId}` wildcard in the
          pattern string, read back with `r.PathValue`. ADR-0001 chose this
          over chi or gin: the router is itself lesson material.
      - lines: "49"
        note: >
          Each handler caps its own body: 64 KiB for the learning routes, 2 MiB
          for the run route. There is no global body-size middleware.
      - lines: "55"
        note: >
          The middleware wraps the mux, not individual routes, so it also sees
          the requests the mux rejects itself: the 405 and 404 lines are logged.
          The real router puts recovery (example 4) and the request guards
          (example 2) between the two, and one more wrapper around the mux
          (example 5).
      - lines: "69-70"
        note: >
          What a bare mux does: it writes 405 (with `Allow: GET, HEAD, PUT`,
          since a GET pattern also serves HEAD) and 404 itself, as plain text,
          never reaching problem(). The real router had this gap in "one error
          shape" until withProblemFallbacks (example 5) took both over.
          `/api/lessons/{id}` is the route docs/ARCHITECTURE.md describes; it
          does not exist.
  - id: request-guards
    file: examples/02_request_guards.go
    title: "security.go in small: three guards in front of the code-running API"
    runtime: go-server
    check: exact
    annotations:
      - lines: "16-22"
        note: >
          `http.CrossOriginProtection` (Go 1.25) is the outer guard; the real
          deny handler also logs and answers through problem(). The order of
          the wrappers is the order a request meets them.
      - lines: "24-32"
        note: >
          Against DNS rebinding: a hostile name re-pointed at 127.0.0.1 makes
          the browser treat the attacker's page as same-origin, but the Host
          header still carries that name. Only /api/ is checked, because
          Compose probes /healthz by container name.
      - lines: "35-42"
        note: >
          Any port is fine: the Vite proxy forwards the browser's Host, which
          names port 5173, not 8080.
      - lines: "44-57"
        note: >
          A cross-site page can't send application/json without a CORS
          preflight, which this server never approves. Bodyless POSTs
          (progress, bookmarks) have Content-Length 0 and pass.
      - lines: "75"
        note: >
          A form or fetch from another site: the browser labels it with
          Sec-Fetch-Site: cross-site, and CrossOriginProtection answers 403
          before any handler runs.
      - lines: "76"
        note: >
          Without Sec-Fetch-Site and Origin, CrossOriginProtection assumes a
          same-origin or non-browser request and lets it through; the JSON-only
          check is what stops a text/plain body here.
      - lines: "77"
        note: >
          The browser believes it is same-origin, so the first two guards pass;
          the Host check doesn't.
  - id: error-mapping
    file: examples/03_error_mapping.go
    title: "POST /api/run: one handler, every outcome mapped to a status"
    runtime: go-server
    check: exact
    annotations:
      - lines: "24-26"
        note: >
          `Run` returns three things. `*RunError` is "the runner said no"
          (busy, runtime unsupported): pass its status through. `error` is
          "the run machinery broke". Two channels instead of one error and an
          `errors.As`, at the cost that forgetting one check still compiles.
      - lines: "28-38"
        note: >
          A fake behind the consumer-declared interface: three cases, no
          mocking library. The last line is the important one: a program that
          fails to compile is a successful run, a 200 with exitCode 1.
      - lines: "44-47"
        note: >
          400 for what the client can fix: a body that isn't JSON, or one that
          fails `runapi.Request.Validate` (runtime required, at most 32 files,
          names of at most 128 bytes, 1 MiB of sources, timeout 0 to 15 s).
      - lines: "50-52"
        note: >
          502 with a generic detail: the real cause (a pgx error naming host and
          port) goes to the log only. The learner learns which side failed, not
          the infrastructure's internals.
      - lines: "53-54"
        note: >
          A refusal keeps its status: 503 "server busy" when both run slots are
          taken, 501 for a ts-test lesson in Docker mode.
      - lines: "55-57"
        note: >
          The real handler records run history first, unless the request
          carries `X-Stacklab-No-History` (verify-lessons sets it). A failed
          insert is logged as a warning and the result is still returned.
  - id: recovery
    file: examples/04_recovery.go
    title: "withRecovery in small: a panic becomes a 500, unless the reply has started"
    runtime: go-server
    check: exact
    annotations:
      - lines: "20-26"
        note: >
          The recovery needs one fact the ResponseWriter doesn't expose: has
          the handler written anything? The real writeTracker also has Unwrap,
          so the run route's write deadline still works through it, and
          FlushError, because a flush sends the status line too.
      - lines: "31-35"
        note: >
          The deferred function runs while the panic unwinds, so recover()
          stops it here. http.ErrAbortHandler is how a handler asks net/http to
          drop the connection without logging; recovering it would turn a
          deliberate abort into a 500, so it goes back up untouched.
      - lines: "37"
        note: >
          The real one logs through slog with method, path, the panic value and
          `debug.Stack()`. Called inside the deferred function, it still shows
          the frames that panicked, because the stack hasn't unwound yet.
      - lines: "38-40"
        note: >
          After a write the status line may already be on the wire, and a 500
          can't replace it. Returning normally would end a half-written body as
          if it were complete; re-panicking with ErrAbortHandler makes net/http
          drop the connection, so the client sees a broken reply instead.
      - lines: "41"
        note: >
          Nothing written yet: the client gets the same problem JSON as every
          other error. In NewRouter this sits just inside withLogging, so the
          access log records the 500 too.
      - lines: "65-69"
        note: >
          This recover stands in for net/http's own, one level up in the
          connection's goroutine. With ErrAbortHandler it closes the connection
          silently; with any other value it logs `http: panic serving ...`.
  - id: mux-problems
    file: examples/05_mux_problems.go
    title: "withProblemFallbacks in small: the mux's own 404 and 405 as problem JSON"
    runtime: go-server
    check: exact
    annotations:
      - lines: "25-36"
        note: >
          The mux answers with http.Error, which sets text/plain and calls
          WriteHeader before writing its text. This writer swaps that
          WriteHeader for problem(). For a 405 the mux has already set Allow,
          so the header survives into the problem reply.
      - lines: "38-43"
        note: >
          http.Error writes "404 page not found" after WriteHeader; once
          problem() has answered, that text is dropped instead of being
          appended to the JSON.
      - lines: "47-49"
        note: >
          ServeMux.Handler returns an empty pattern when no route will handle
          the request, so only the mux's own replies (404, 405, or a
          path-cleaning redirect, which passes through) get the rewriting
          writer. A catch-all "/" route can't do this: it matches every method,
          so DELETE /api/notes/x would get its 404 instead of a 405.
      - lines: "56-60"
        note: >
          A matched route's reply is never touched, even a 404 in another
          format. StackLab's handlers have no 404 of their own; this one is here
          to show where the wrapper stops.
use_cases:
  - "Reading any Go service's HTTP layer: find the mux, the middleware wrapped around it, and the one function errors go through"
  - "Long-running endpoints (runs, exports, reports) that need a longer write deadline than the server default"
  - "Deciding which failures are 4xx, which are 5xx, and which are a 200 carrying a failed result"
  - "Protecting a localhost service that runs code or holds private data from the other websites open in the same browser"
strengths:
  - "Fifteen routes in one flat list in `router.go`: method, path and handler, greppable, with no annotations or route groups to chase as in Spring MVC or NestJS"
  - "One error shape for every error the API writes (handlers, guards, recovered panics, the mux's own 404 and 405): RFC 9457 problem details from one function"
  - "A compile error or failing test is a 200 result, not an HTTP error, so the UI never confuses \"your code failed\" with \"the server failed\""
  - "Infrastructure errors are logged in full and summarized to the client, so pgx messages with hosts and ports never reach the page"
  - "Cross-site protection is the standard library's `http.CrossOriginProtection` plus two short wrappers: no CSRF tokens, sessions or token endpoint"
weaknesses:
  - "Recovery can only answer a panic that happens before the handler writes. After that it aborts the connection, and the client sees a broken reply rather than a 500"
  - "The 404/405 rewrite leans on two ServeMux behaviours: an empty pattern from `Handler` when nothing matched, and `http.Error` calling `WriteHeader` before it writes. If either changed, text/plain would come back quietly; a router test (`middleware_test.go`) is what would notice"
  - "Status mapping happens per handler, not in one function over typed errors: every store failure becomes a 500 with a hand-written message"
  - "No request IDs: the log line for a failed run and the problem JSON the browser saw can only be matched by time and path"
pitfalls:
  - title: "A ResponseWriter wrapper without Unwrap"
    detail: "SetWriteDeadline, Flush and Hijack then fail with `http.ErrNotSupported` through `http.ResponseController`. Ignoring that error is how the run route silently kept the 15 s deadline."
    bad: "_ = http.NewResponseController(w).SetWriteDeadline(deadline)"
    good: "if err := http.NewResponseController(w).SetWriteDeadline(deadline); err != nil {\n\tlogger.Warn(\"cannot extend write deadline\", slog.Any(\"err\", err))\n}"
  - title: "Treating 127.0.0.1 as private"
    detail: "A loopback listener keeps other machines out, not other websites: any page open in the browser can send requests to it, and a text/plain or form POST needs no CORS preflight. An endpoint that acts on a request (runs code, writes data) needs cross-origin protection even on localhost."
  - title: "Returning 4xx/5xx for failing user code"
    detail: "The program failing is the expected outcome of a lesson about errors. Reserve HTTP errors for requests the server could not process."
  - title: "Assuming middleware sees panics"
    detail: "A panic unwinds through every middleware outside the recover, so the code after `next.ServeHTTP` never runs there. Without `withRecovery`, `withLogging` would never write that request's line. A recovery middleware belongs inside the logging one, which is where NewRouter puts it."
  - title: "Answering a panic with 500 after the handler wrote"
    detail: "The status line may already be on the wire. A second WriteHeader only logs \"superfluous response.WriteHeader call\", and returning normally ends a half-written body as if it were complete. Re-panic with `http.ErrAbortHandler` so net/http drops the connection."
    bad: "defer func() { if v := recover(); v != nil { problem(w, 500, \"internal server error\") } }()"
    good: "if tw.wrote {\n\tpanic(http.ErrAbortHandler)\n}\nproblem(w, http.StatusInternalServerError, \"internal server error\")"
  - title: "A catch-all \"/\" route for problem 404s"
    detail: "It matches every method and every path, so the mux never gets to answer 405: GET /api/run would become a 404. Wrap the mux and rewrite only the replies it writes when no pattern matched."
quiz:
  - q: "With the app running, what does `GET /api/run` return?"
    options:
      - "400 problem+json: invalid request body"
      - "405 text/plain with `Allow: POST`, written by the ServeMux"
      - "405 problem+json with `Allow: POST`"
      - "404 problem+json"
    answer: 2
    why: "Only `POST /api/run` is registered, so the mux decides on 405 and sets `Allow: POST` itself. No handler runs; `withProblemFallbacks` sees that no pattern matched and swaps the mux's plain-text body for problem()."
  - q: "A website the owner has open sends a text/plain POST to http://127.0.0.1:8080/api/run. In a current browser, what stops it?"
    options:
      - "Nothing needs to: other sites can't reach 127.0.0.1"
      - "CrossOriginProtection: the browser marks it Sec-Fetch-Site: cross-site, and the guard answers 403"
      - "CORS: the browser refuses to send it"
      - "The run handler, with 400"
    answer: 1
    why: "Loopback keeps other machines out, not other pages in the same browser, and a text/plain POST needs no preflight, so it is sent. The browser labels it cross-site and the outer guard rejects it; in a browser that sends no such header, the JSON-only guard would answer 415."
  - q: "Why did the run route need `statusRecorder.Unwrap`?"
    options:
      - "So the access log could read the status code"
      - "So http.ResponseController could reach the real connection and extend its write deadline"
      - "So the JSON encoder could flush"
    answer: 1
    why: "Without Unwrap, SetWriteDeadline returned ErrNotSupported, the 15 s WriteTimeout stayed in force, and slow runs lost their response."
  - q: "A handler panics before writing anything. What does the browser get?"
    options:
      - "500 problem+json from problem()"
      - "500 text/plain from net/http"
      - "No response: net/http logs the panic and closes the connection"
    answer: 0
    why: "`withRecovery` recovers it, logs the panic with its stack, and answers through problem(). Without it, net/http would recover the panic itself, log `http: panic serving ...` and close the connection, and the client would read EOF. That is still what happens when the handler had already started writing."
further_reading:
  - title: "net/http: ServeMux (patterns and precedence)"
    url: "https://pkg.go.dev/net/http#ServeMux"
  - title: "net/http: ResponseController"
    url: "https://pkg.go.dev/net/http#ResponseController"
  - title: "Routing enhancements for Go 1.22"
    url: "https://go.dev/blog/routing-enhancements"
  - title: "MDN: HTTP response status codes"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Status"
---

## Concept

`internal/httpapi` is the backend's whole HTTP surface, in five files:

- `router.go`: `NewRouter` registers fifteen routes on a standard `http.ServeMux` (health,
  readiness, `POST /api/run`, and twelve progress, notes, bookmarks, quiz and history
  routes), wraps the mux, and defines `problem()`.
- `middleware.go`: `statusRecorder` and `withLogging`, `withRecovery` (panics to 500), and
  `withProblemFallbacks` (the mux's own 404 and 405 to problem JSON).
- `security.go`: `withRequestGuards`, three checks that keep other websites away from
  the API (ADR-0011).
- `run.go`: the `Runner` interface and `handleRun`.
- `learning.go`: one `learningHandlers` type for the per-lesson state routes.

The whole chain is the last line of `NewRouter`:
`withLogging(logger)(withRecovery(logger)(withRequestGuards(logger, withProblemFallbacks(mux))))`.
A request meets `withLogging` → `withRecovery` → `http.CrossOriginProtection` →
`requireLoopbackHost` → `requireJSONBody` → `withProblemFallbacks` → `ServeMux` (method +
path match, or its own 404/405, rewritten as problem JSON) → handler → decode with
`http.MaxBytesReader` → call a store or the runner → write JSON with `writeJSON`, or an
error with `problem`. Logging sits outermost so that rejected and panicking requests show
up in the log too; recovery sits just inside it, so the log line says 500.

"The lessons applied" is accurate for the router (ADR-0001's Go 1.22 patterns), for
middleware as a function wrapper with an `Unwrap`-capable recorder, for recovery and for
problem details. It is partial elsewhere: `go.http.middleware` also teaches request IDs
and timeouts, and this app has neither as middleware (its timeouts are the
`http.Server` fields).

## Coming from C/C++/Java/Python/JS

- **Java (Spring MVC)**: routes are annotations spread across controllers, and errors map
  through `@ControllerAdvice`. Here routes are one list of `mux.HandleFunc` calls, and
  each handler calls `problem()` itself. Spring 6's `ProblemDetail` writes the same wire
  format (RFC 7807, which RFC 9457 replaced).
- **Python (Flask/FastAPI)**: an unhandled exception becomes a 500 by default. In Go a
  panic is not that: without recovery middleware, net/http drops the connection, which
  is why `NewRouter` has `withRecovery`.
- **JS (Express)**: `app.use(logger)` is `withLogging(logger)(mux)`. Express has
  `next(err)` and error-handling middleware; Go handlers return nothing, so error mapping
  is a convention you choose (here: call `problem` and return).
- **C++**: no standard HTTP server to compare with; the closest analogue is a request
  dispatcher wrapped in a decorator that records the response code.

## How it works

**The guards.** In local mode `POST /api/run` runs lesson code as the owner's Windows
user. Listening on 127.0.0.1 keeps other machines out, but not other websites: any page
open in the owner's browser can send requests to 127.0.0.1, and a `text/plain` POST needs
no CORS preflight. Each guard closes one door, and each answers through `problem()`:

- 403 for an unsafe request the browser marks as cross-site, through `Sec-Fetch-Site`
  or an `Origin` that doesn't match `Host`. GET, HEAD and OPTIONS always pass.
- 415 for an unsafe request whose body isn't `application/json`, which a cross-site page
  can't send without a preflight. This catches the same attack in a browser that sends
  neither header.
- 403 for an `/api/` request addressed to anything but `localhost` or a loopback IP,
  which stops DNS rebinding.

CORS is not one of them: it controls who may read a response, not who may send a simple
request. ADR-0011 also rejected CSRF tokens, which need a session and a token endpoint.

**The run route** is where the status mapping is most deliberate. `handleRun` produces
one of:

1. **400**: the body isn't JSON, exceeds 2 MiB, or fails `req.Validate()`. (A body with
   any other `Content-Type` never gets this far: `requireJSONBody` answers 415.)
2. **502**: `Run` returned an `error`. The runner container (Docker) or the playground
   database (local) failed. Logged in full, summarized to the client.
3. **The refusal's status**: `Run` returned a `*runapi.Error`, which runexec uses for 503
   "server busy" and 501 (ts-test in Docker mode, or a runtime with no case in its switch).
   In Docker mode `runclient` passes the runner's refusal through with its detail, and
   turns a runner 500 into an `error`, so that case is a 502 as in local mode.
4. **200**: everything else, including programs that don't compile, panic, exit non-zero
   or time out. Those facts are fields of the result (`exitCode`, `timedOut`,
   `diagnostics`), which the UI's `judge.ts` interprets.

**The write deadline.** The server's `WriteTimeout` of 15 s counts from the request, and
a run can take much longer: up to 10 s waiting for one of runexec's two run slots, then
up to its 40 s run budget (20 s to compile plus 15 s to run fits inside). So before
calling the runner the handler lifts its own write deadline
(`SetWriteDeadline(time.Time{})`), and once the run is over it sets a fresh one for the
reply, 10 s (`replyWriteBudget`). Nothing is written in between, and the run's length is
the Runner's business. An earlier version set 45 s up front (`runWriteBudget`, matching
runclient's HTTP timeout), which forgot the wait for a slot: a run that queued and then
used its whole budget lost its reply. `http.ResponseController` (Go 1.20) finds
`SetWriteDeadline` by following `Unwrap` through wrappers, so the deadline calls work
through `withLogging` and `withRecovery` only because `statusRecorder` and
`writeTracker` have that method. When `statusRecorder` first shipped it didn't: the call
returned `http.ErrNotSupported`, the error was ignored, the 15 s deadline stayed, and on
slow runs the server dropped the connection mid-reply, which the browser reported as
"socket hang up". Today `run.go` logs a warning when a deadline call fails,
`middleware_test.go` pins the `Unwrap` behaviour, and `run_test.go` checks that a run
longer than the server's `WriteTimeout` still gets its reply.

What is not here, despite older docs: no `/api/lessons` or `/api/source` routes (content
never touches the backend, see `inside.codebase.content-pipeline`), no `/mock/*` proxy, no
CORS (the Vite dev server proxies `/api` so the browser stays same-origin), and no
embedded frontend.

## Design notes

- **Errors are mapped per handler, not centrally.** CLAUDE.md asks to "map to HTTP status
  in one place (httpapi)". The package is the one place, but there is no single function
  from error to status: each learning handler logs and writes a 500 with its own message.
  That is fine while every store error really is a 500. When a store grows a meaningful
  error (not found, conflict), a sentinel such as `learningstore.ErrNotFound` plus one
  `writeError(w, err)` doing `errors.Is` would keep the mapping in one function.
- **Recovery inside logging.** `withRecovery` sits just inside `withLogging` and turns a
  panic into a logged 500 problem response instead of a dropped connection (example 4).
  It answers only when the handler hasn't written yet; otherwise it re-panics with
  `http.ErrAbortHandler`, which also passes through untouched when a handler raises it.
  Its `writeTracker` needs `Unwrap` for the same reason `statusRecorder` does.
- **The mux's own 404 and 405 are problem JSON.** A catch-all `"/"` route is not the fix:
  it matches every method, so it would also swallow the requests that should get a 405.
  `withProblemFallbacks` asks `mux.Handler` first; an empty pattern means no route
  matched, and only then does a writer rewrite 404 and 405 through `problem()` (example
  5). Before it, a client had to handle two error formats.
- **The three-value `Run` signature** keeps refusals and failures apart without
  `errors.As`, and both runners implement it. The cost is a convention the compiler
  can't enforce: check `err` first, then the refusal, then use the result.
