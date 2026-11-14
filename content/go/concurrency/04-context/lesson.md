---
id: go.concurrency.context
title: "context: cancellation, deadlines, values used sparingly"
track: go
module: concurrency
order: 4
level: intermediate
minutes: 13
since: go1.21
prerequisites: [go.concurrency.select]
compare_with: [ts.async.cancellation, go.services.client, go.postgres.bulk]
tags: [concurrency, context, cancellation, timeouts, deadlines]
summary: >
  A context.Context carries a cancellation signal, an optional deadline and a
  few request-scoped values down the call chain. Every function that does I/O
  takes it as its first parameter and stops when ctx.Done() is closed.
examples:
  - id: timeout-cancel
    file: examples/01_timeout_cancel.go
    title: Deadlines, explicit cancellation, and causes
    runtime: go
    check: exact
    annotations:
      - lines: "12-18"
        note: >
          This select is the whole contract: do the work, but stop waiting
          once ctx.Done() is closed. net/http, database/sql and pgx do the same
          inside every call that takes a ctx.
      - lines: "17"
        note: >
          Wrap ctx.Err() with %w, so callers can still ask errors.Is(err,
          context.DeadlineExceeded) and tell a timeout from a real failure.
      - lines: "23-24"
        note: >
          WithTimeoutCause (Go 1.21) attaches a reason. ctx.Err() still reports
          DeadlineExceeded, so existing checks keep working; context.Cause
          returns the reason for logs and error messages.
      - lines: "25"
        note: >
          Call cancel on every path, even after success: it stops the timer and
          detaches the child from its parent. Deferring it on the next line is
          the idiom, and go vet's lostcancel check reports a dropped CancelFunc.
      - lines: "32-36"
        note: >
          Cancellation is a request, not a kill. query stops only because it
          selects on ctx.Done(); a function that ignores its ctx runs to the end
          regardless. Calling a CancelFunc twice, or from several goroutines, is safe.
      - lines: "39-41"
        note: >
          When the work beats the deadline, the context never fires and the
          result comes back as usual.
  - id: tree
    file: examples/02_tree.go
    title: Cancellation flows down the tree; WithoutCancel and AfterFunc
    runtime: go
    check: exact
    annotations:
      - lines: "13-14"
        note: >
          Every With... call wraps a parent, so contexts form a tree. In a
          server the root is r.Context(), which net/http cancels when the
          client disconnects or the handler returns.
      - lines: "18-22"
        note: >
          Canceling cacheCall leaves request and dbCall alone. Cancellation
          flows from parent to children, never up or sideways.
      - lines: "26"
        note: >
          WithoutCancel (Go 1.21) keeps the parent's values but drops its
          cancellation and deadline. Use it for work that must finish after the
          request is gone, and give that work a timeout of its own.
      - lines: "29-36"
        note: >
          AfterFunc (Go 1.21) runs the callback in a new goroutine once the
          context is canceled. It is how you connect a blocking call that takes
          no ctx, for example closing a connection to unblock a read. The
          closed channel only fixes the output order.
      - lines: "38-39"
        note: >
          One call reached every descendant: the hour-long dbCall is canceled
          too. The audit context isn't, and still carries the user value.
  - id: values
    file: examples/03_values.go
    title: Values with unexported key types and typed accessors
    runtime: go
    check: exact
    annotations:
      - lines: "10-12"
        note: >
          The key's type is unexported, so only this package can construct the
          key. Another package's key with the same underlying 0 has a different
          type and can't collide with it.
      - lines: "14-22"
        note: >
          Export typed helpers, not the key, as the context docs recommend.
          Callers never see the any, and a missing value yields the zero value
          rather than a failed type assertion.
      - lines: "24-28"
        note: >
          Values are for request-scoped data that crosses API boundaries: a
          request ID, a trace span, the authenticated user. What a function
          needs to do its job belongs in its parameters, where the compiler
          checks it.
      - lines: "37-42"
        note: >
          A key matches only if both type and value are equal, so
          authKey("user"), logKey("user") and the plain string "user" are three
          different keys. That is why the docs warn against built-in key types.
use_cases:
  - "Per-request deadlines in HTTP handlers: derive from r.Context() so a disconnected client stops the database query it started"
  - "Outgoing calls with a budget: context.WithTimeout around an HTTP request, a pgx query or a gRPC call"
  - "Graceful shutdown: signal.NotifyContext cancels a root context on Ctrl+C or SIGTERM, and everything derived from it winds down"
  - "Request-scoped metadata that middleware adds and logging reads: request ID, trace span, authenticated user"
strengths:
  - "One convention across the standard library and the ecosystem: net/http, database/sql, pgx and gRPC all take ctx first, unlike Java, where each API has its own timeout and interrupt story"
  - "Cancellation propagates through the whole call tree from one cancel() call, including into goroutines started deeper down"
  - "Deadlines compose: a child can only shorten its parent's deadline, so a 100ms handler budget can't be exceeded by a callee asking for 5s"
weaknesses:
  - "Cooperative only: code that never checks ctx.Done() or passes ctx on can't be cancelled, and nothing in the type system tells you"
  - "ctx is threaded by hand through every signature, which is noisier than the implicit propagation of Python's contextvars or Java's ThreadLocal"
  - "Values are untyped (any) and invisible in signatures, so they turn into hidden parameters when overused"
pitfalls:
  - title: "Storing a Context in a struct"
    detail: "The context docs say to pass it explicitly to each function instead. A stored ctx outlives the request it belonged to, and callers can't give individual calls their own deadline."
    bad: "type Repo struct { ctx context.Context; db *pgxpool.Pool }"
    good: "func (r *Repo) FindUser(ctx context.Context, id int64) (User, error)"
  - title: "Using context.Background() deep in the call chain"
    detail: "It cuts the chain: the request's deadline and cancellation no longer reach the query. Background belongs in main, tests and top-level setup; elsewhere pass the ctx you were given, or context.TODO() while refactoring."
  - title: "Optional parameters in context values"
    detail: "A page size or a feature flag hidden in the context is a parameter the compiler can't check. Use a function argument or an options struct."
  - title: "Forgetting cancel"
    detail: "Without cancel, a WithTimeout child and its timer stay alive until the timeout fires or the parent is canceled; with WithCancel, until the parent is canceled. go vet's lostcancel check reports it."
    bad: "ctx, _ = context.WithTimeout(ctx, time.Second)"
    good: "ctx, cancel := context.WithTimeout(ctx, time.Second)\ndefer cancel()"
quiz:
  - q: "A child context is canceled. What happens to its parent?"
    options: ["Nothing: cancellation only flows to descendants", "It is canceled too", "It gets a deadline"]
    answer: 0
    why: "Canceling a context cancels it and its children; parents and siblings are untouched."
  - q: "ctx was created with WithTimeoutCause(parent, d, myErr) and the deadline passed. What does ctx.Err() return?"
    options: ["myErr", "context.DeadlineExceeded", "nil"]
    answer: 1
    why: "Err keeps reporting DeadlineExceeded so existing checks still work; context.Cause(ctx) returns myErr."
  - q: "A function receives ctx but never selects on ctx.Done() and never passes ctx on. What does canceling ctx do to it?"
    options: ["It panics", "The runtime stops its goroutine", "Nothing: cancellation is cooperative"]
    answer: 2
    why: "A context only signals. Work stops when code observes Done or hands ctx to an API that does."
further_reading:
  - title: "package context"
    url: "https://pkg.go.dev/context"
  - title: "Go Concurrency Patterns: Context"
    url: "https://go.dev/blog/context"
  - title: "Contexts and structs"
    url: "https://go.dev/blog/context-and-structs"
  - title: "package os/signal: NotifyContext"
    url: "https://pkg.go.dev/os/signal#NotifyContext"
---

## Concept

A `context.Context` is an immutable value that carries three things down a call chain:
a **cancellation signal** (`Done()` returns a channel that is closed on cancel), an
optional **deadline**, and a few **request-scoped values**. You never modify a context;
you derive a child from it:

- `context.WithCancel(parent)` returns a child and a `cancel` function.
- `context.WithTimeout(parent, d)` / `WithDeadline(parent, t)` cancel automatically.
- `WithCancelCause` (Go 1.20) and `WithTimeoutCause` / `WithDeadlineCause` (Go 1.21)
  record why; `context.Cause(ctx)` reads it back.
- `context.WithValue(parent, key, val)` attaches one value.

When a context is canceled, every context derived from it is canceled too. After that,
`ctx.Err()` returns `context.Canceled` or `context.DeadlineExceeded`.

The convention is strict: anything that does I/O or may block takes
`ctx context.Context` as its **first parameter** and passes it on.

## Coming from C/C++/Java/Python/JS

- **JS**: `AbortController`/`AbortSignal` is the closest match. `signal` ≈ `ctx`,
  `controller.abort(reason)` ≈ `cancel(cause)`, `AbortSignal.timeout(ms)` ≈
  `WithTimeout`, and `fetch(url, { signal })` ≈ passing `ctx` to `http.NewRequestWithContext`.
- **Java**: there is no single equivalent. `Thread.interrupt()` is per thread,
  `Future.cancel(true)` per task, and every API has its own timeout parameters.
  `context` is one mechanism used everywhere, and it's explicit rather than a flag
  on the thread.
- **Python**: `asyncio` cancels a task by raising `CancelledError` inside it. Go has no
  exceptions to inject: the callee has to check `ctx.Done()` or `ctx.Err()`.
  `contextvars` covers the values part.
- **C/C++**: closest to passing a cancellation token around by hand, as in C++20's
  `std::stop_token`, with deadlines built in.

## How it works

`Done()` returns a channel that is **closed**, never sent on, when the context is
canceled. A closed channel is ready for every receiver, so one `cancel()` wakes any
number of goroutines selecting on `<-ctx.Done()`. That's why the cancellation check is
a `select` case, the pattern from the previous lesson.

Deadlines only get shorter. `WithTimeout(parent, 5*time.Second)` under a parent that
expires in 100ms still expires in 100ms. A handler's budget therefore holds no matter
what its callees ask for.

For incoming HTTP requests, `r.Context()` is canceled when the client's connection
closes, when an HTTP/2 request is canceled, or when `ServeHTTP` returns. Deriving every
query's context from it means a disconnected client stops the database work it started.

## Design notes

Context is plumbing, so keep it boring:

- First parameter, named `ctx`, never stored in a struct, never `nil` (use
  `context.TODO()` if you're unsure which context to use).
- `context.Background()` belongs at the top: in `main`, in tests (or `t.Context()`,
  Go 1.24) and in long-lived background loops.
- Call `cancel` on every path, usually with `defer` on the next line.
- Values only for request-scoped data that crosses API boundaries, behind an unexported
  key type with typed accessors. If a function needs a value to do its job, it's a
  parameter.
- Pick the timeout where the budget is known, usually at the edge (the handler or the
  job runner), and let deeper layers inherit it rather than invent their own.

`errgroup` (lesson 6) builds on this: its context is canceled as soon as one goroutine in
the group fails, so its siblings stop early.
