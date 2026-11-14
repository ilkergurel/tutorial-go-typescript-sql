---
id: go.errors.panic-strategy
title: "panic/recover and an error strategy across layers"
track: go
module: errors
order: 3
level: intermediate
minutes: 15
since: go1.26
prerequisites: [go.basics.defer, go.errors.is-as-join]
compare_with: [ts.errors.result, design.paradigms.errors, go.http.middleware, go.concurrency.goroutines-channels]
tags: [panic, recover, errors, goroutines, http, boundaries]
summary: >
  Errors are for failures a correct program expects; panics are for bugs.
  Recover only at boundaries you own, the top of a goroutine and the top of a
  request, and let every other layer return errors upward.
examples:
  - id: bugs-vs-errors
    file: examples/01_bugs_vs_errors.go
    title: Errors for input, panics for bugs, and what recover hands you
    runtime: go
    check: exact
    annotations:
      - lines: "9-11"
        note: >
          Must functions panic by convention (regexp.MustCompile,
          template.Must). Use them only on inputs fixed in the source: a typo
          then fails the first test run or program start, never a request.
      - lines: "16-23"
        note: >
          recover returns any, so a type switch sorts it. A runtime.Error is a
          failure the runtime detected, such as a nil map write or an index
          out of range: always a bug, never a condition to handle.
      - lines: "32-36"
        note: >
          One mistake, two APIs. Compile returns an error because its input may
          come from a user; MustCompile panics because its input is code. Its
          panic value is a plain string, so it lands in the default case.
      - lines: "39-42"
        note: >
          Nothing at this level can handle these sensibly. The right response
          is a 500 at the boundary, a log entry with the stack, and a fix.
      - lines: "43"
        note: >
          Before Go 1.21, panic(nil) made recover return nil, so a boundary
          checking r != nil let the panic vanish. Now it raises
          *runtime.PanicNilError; GODEBUG=panicnil=1 restores the old behavior.
  - id: goroutine-boundary
    file: examples/02_goroutine_boundary.go
    title: Recover at the top of each goroutine and report a typed error
    runtime: go
    check: exact
    annotations:
      - lines: "10-16"
        note: >
          A distinct type lets callers tell bugs from failures with
          errors.AsType. A failure can be retried or shown to the user; a bug
          needs its stack logged and someone told.
      - lines: "18-22"
        note: >
          The deferred recover must run in the goroutine that panics. A recover
          in main, or in the function that started the goroutine, never sees
          this panic, and an unrecovered panic in any goroutine ends the whole
          process.
      - lines: "23"
        note: >
          debug.Stack() still includes the panicking line here, because the
          deferred function runs on top of the panicking frames. Once it
          returns they are gone, so capture the stack now, not later.
      - lines: "37-42"
        note: >
          sync.WaitGroup.Go (Go 1.25) bundles Add, go and Done, and its
          documentation says f must not panic: hence safely. Each goroutine
          writes only its own slot, and Wait makes the writes visible.
      - lines: "44-51"
        note: >
          Three outcomes: success, an ordinary failure, and a bug. Without
          safely, job 2 would have crashed the program and taken the other
          results with it.
  - id: http-boundary
    file: examples/03_http_boundary.go
    title: "One place for status codes, logging and recovery in an HTTP server"
    runtime: go-server
    check: exact
    timeout_s: 10
    annotations:
      - lines: "14-16"
        note: >
          Handlers return an error, as in the Go blog's appHandler pattern.
          They don't write error responses themselves; ServeHTTP does, once,
          for every route.
      - lines: "19-27"
        note: >
          The request boundary. http.ErrAbortHandler means "abort this
          response quietly", so it is re-panicked for net/http to handle;
          swallowing it would turn a deliberate abort into a logged 500.
      - lines: "28-35"
        note: >
          The single place errors become status codes. errors.Is finds
          ErrNotFound however deeply it was wrapped. Unexpected errors are
          logged here, once, and the client gets a 500 without internals.
      - lines: "49"
        note: >
          Without a boundary of your own, net/http recovers the panic, logs a
          stack trace to the server's error log (see stderr), and closes the
          connection. The client gets no status code at all.
      - lines: "54-63"
        note: >
          /raw: a dropped connection. /users/7: a mapped 404, not logged,
          because it isn't a server problem. /users/0: a recovered 500. stderr
          shows both logs; it is never compared, which is just as well, since
          it holds a port number and timestamps.
use_cases:
  - "Must helpers for constants: regexp.MustCompile, template.Must, parsing embedded config at startup"
  - "Panicking on a violated invariant: the default branch of a switch over your own enum, an impossible state in a state machine"
  - "Recovering at the top of worker goroutines, consumers and scheduled jobs so one bad item doesn't stop the process"
  - "Recovery middleware in HTTP and gRPC servers that logs the stack and answers 500"
strengths:
  - "Expected failures stay ordinary values, so control flow is visible in the code, unlike exceptions that can come out of any call"
  - "A panic is loud: it carries a stack trace and, unrecovered, stops the process instead of leaving it running in a corrupted state"
  - "Boundaries are explicit: you can point at the two or three places in a codebase where recover is allowed"
weaknesses:
  - "An unrecovered panic in any goroutine kills the whole process, unlike Java or Python, where an uncaught exception ends only its thread"
  - "Each goroutine needs its own recover boundary and nothing adds one for you: even sync.WaitGroup.Go only documents that f must not panic"
  - "recover works only when called directly by a deferred function, a rule that is easy to break with a helper"
pitfalls:
  - title: "Expecting main's recover to catch a goroutine's panic"
    detail: "recover only sees panics in its own goroutine. Put the deferred recover inside the function the goroutine runs."
    bad: "defer func() { recover() }(); go work()"
    good: "go func() { defer logPanic(); work() }()"
  - title: "Calling a recover helper from a deferred closure"
    detail: "defer logPanic() works when logPanic calls recover. defer func() { logPanic() }() does not: recover is then called by logPanic, not directly by the deferred function, and returns nil."
  - title: "Recovering deep inside the code"
    detail: "A recover in the middle of a call chain hides the bug and leaves state half-updated: a mutex held, a transaction open, a slice half-written. Recover at boundaries, fail that unit of work, log the stack."
  - title: "Panicking for expected failures in a library"
    detail: "Callers can only handle it with recover. \"Not found\", invalid input and timeouts are errors; the Code Review Comments say: don't use panic for normal error handling."
  - title: "Swallowing http.ErrAbortHandler"
    detail: "httputil.ReverseProxy and others panic with it to abort a response. A recovery middleware should re-panic it, not log a 500."
quiz:
  - q: "A goroutine started by main panics. main has defer func() { recover() }(). What happens?"
    options: ["main recovers it", "Only that goroutine stops", "The whole program crashes", "The panic is ignored"]
    answer: 2
    why: "recover only stops a panic in its own goroutine; an unrecovered panic in any goroutine terminates the program."
  - q: "Which situation calls for a panic rather than an error?"
    options: ["A user submits an invalid email", "A row is not found", "A switch over your own enum reaches a value that can't exist", "An upstream API times out"]
    answer: 2
    why: "Only the impossible enum value is a bug in the program; the rest are failures a correct program must expect."
  - q: "An HTTP handler panics and no middleware recovers it. What does the client get?"
    options: ["A 500 response", "A 502 response", "No response: the connection is closed", "The server process exits"]
    answer: 2
    why: "net/http recovers per request, logs the stack and closes the connection (or resets the HTTP/2 stream); it sends no status."
  - q: "Where should HTTP status codes be chosen?"
    options: ["In each repository function", "In the service layer", "In one place at the HTTP boundary, from errors.Is and errors.AsType", "Wherever the error is created"]
    answer: 2
    why: "Lower layers return domain errors; one boundary maps them to status codes, so the mapping stays consistent and the layers stay independent of HTTP."
further_reading:
  - title: "The Go Blog: Defer, Panic, and Recover"
    url: "https://go.dev/blog/defer-panic-and-recover"
  - title: "The Go Blog: Error handling and Go (the appHandler pattern)"
    url: "https://go.dev/blog/error-handling-and-go"
  - title: "Effective Go: Panic and Recover"
    url: "https://go.dev/doc/effective_go#panic"
  - title: "The Go Programming Language Specification: Handling panics"
    url: "https://go.dev/ref/spec#Handling_panics"
  - title: "package net/http: Handler (panic behavior) and ErrAbortHandler"
    url: "https://pkg.go.dev/net/http#Handler"
  - title: "Go Code Review Comments: Don't Panic"
    url: "https://go.dev/wiki/CodeReviewComments#dont-panic"
---

## Concept

Go has two failure channels. Errors carry failures a correct program must expect: bad
input, a missing row, a timeout, an upstream returning 503. Panics signal bugs: a nil map
write, an index out of range, a broken invariant, a constant regular expression that
doesn't compile. A panic unwinds its goroutine, running deferred calls on the way; if
nothing recovers it, the program prints the panic value and a stack trace and exits with
status 2.

`recover` belongs at boundaries you own, where one unit of work may fail without taking
the rest down: the top of each goroutine you start, and the top of each request. There
the panic becomes a logged bug report plus an ordinary error or a 500. Everywhere else,
let it propagate. The defer lesson showed the mechanics of `recover`; this lesson is
about where it goes.

Errors travel the other way. Each layer returns them upward with context, translates
the ones that cross a package boundary, and exactly one place, the HTTP boundary or
`main`, decides what the outside world sees and logs the failure once.

## Coming from C/C++/Java/Python/JS

- Java/Python: a panic is like an unchecked exception that nobody is meant to catch, and
  `recover` is the catch at the thread or request boundary. The difference that bites:
  an uncaught exception in a Java or Python thread ends only that thread, while an
  unrecovered panic in any goroutine ends the whole process.
- Node: closer to Go. An uncaught exception crashes the process, and so does an unhandled
  promise rejection by default since Node 15.
- React: an error boundary is the UI-side version of the HTTP recover boundary: it stops
  one broken subtree from blanking the whole page (lesson ts.errors.result).
- C++: an unrecovered panic is like `std::terminate` after an uncaught exception, except
  that Go runs the panicking goroutine's deferred calls first.

## How it works

```text
layer            returns                               decides
repository       domain errors with context            driver errors -> ErrEmailTaken (%w) + text (%v)
service          domain errors with more context       business rules; knows nothing about HTTP
HTTP boundary    status code and body                  errors.Is/AsType -> 4xx/5xx; logs 5xx once;
                                                       recover -> log stack, 500
goroutine top    an error in a channel or slot         recover -> *PanicError with the stack
main             exit code                             logs once; os.Exit(1) after cleanup
```

Only the last three rows contain `recover` or `log`. A request handler that starts its
own goroutines needs a goroutine boundary inside them as well: the HTTP boundary only
covers the handler's own goroutine.

## Design notes

Panic for programmer errors only: violated invariants, unreachable branches of your own
switches, and `Must` calls on constants. Library code should never panic on bad input,
because its callers can only handle that with `recover`. At each boundary, log the panic
value together with `debug.Stack()` from inside the deferred function, since the
panicking frames are gone once that function returns. Fail that unit of work and keep
the process running. A recovered handler can only send a 500 if it hasn't written the
response headers yet; after that, logging and closing is all that's left. Keep
`log.Fatal` and `os.Exit` in `main`, after cleanup, as the defer lesson explained. And
map errors to status codes in one place, so a new domain error needs one new case, not a
hunt through every handler.
