---
id: inside.codebase.wiring
title: "How main.go wires dependencies"
track: inside
module: codebase
order: 1
level: intermediate
minutes: 12
since: go1.24
prerequisites: [go.design.constructors, go.interfaces.implicit]
compare_with: [go.design.packages, go.http.server-lifecycle, go.postgres.pgx, design.patterns.constructors]
tags: [dependency-injection, constructors, interfaces, main, lifecycle, signals]
summary: >
  StackLab's backend has no DI framework: server/cmd/stacklab/main.go calls
  constructors in order and passes each result to the next. One consumer-declared
  interface, httpapi.Runner, lets one if switch between running lessons in-process
  and in the Docker runner.
examples:
  - id: wiring
    file: examples/01_wiring.go
    title: "The dependency graph in miniature: config, logger, runner, router"
    runtime: go
    check: exact
    annotations:
      - lines: "15-19"
        note: >
          A plain struct and a Load function, no config library. The real
          `config.Load` reads `os.Getenv` directly; taking `getenv` as a
          parameter here lets one program show both modes, and is the shape
          you'd want if config ever got tests (it has none today).
      - lines: "21-24"
        note: >
          The interface lives with its consumer, `internal/httpapi/run.go`.
          `runexec` and `runclient` never import httpapi and never name
          Runner: they satisfy it because their Run methods have the right
          signature. A Java-style `implements` would make both
          implementations depend on the HTTP package.
      - lines: "26-38"
        note: >
          Constructors return concrete types (`*Executor`, `*Client`), not
          Runner. `cmd/runner` uses `*runexec.Executor` directly, with no
          interface at all; only the backend, which has two choices, needs one.
      - lines: "40"
        note: >
          The parameter list is the dependency list. The real signature is
          `NewRouter(logger, pool *platform.DB, runner Runner)`, and it builds
          the run-history and learning stores from `pool.Pool()` itself, so
          those two are not injectable from outside. Only the runner is.
      - lines: "58-61"
        note: >
          The entire run-mode switch (ADR-0008) is one if. The real main also
          builds `runexec.New(...)` first and replaces it in Docker mode, which
          is cheap: New only allocates a two-slot channel.
      - lines: "66"
        note: >
          `%T` prints the dynamic type behind the interface value. The handler
          code was identical in both runs; only the value handed to NewRouter
          changed.
  - id: lifecycle
    file: examples/02_lifecycle.go
    title: "run() error, defers in reverse order, and the shutdown context"
    runtime: go
    check: exact
    annotations:
      - lines: "27-28"
        note: >
          `main` only calls `run` and turns its error into `os.Exit(1)`.
          `os.Exit` and `log.Fatal` skip deferred calls, so calling them inside
          run, after the pool opened, would leave the pool unclosed.
      - lines: "29-30"
        note: >
          The real line is `signal.NotifyContext(ctx, os.Interrupt,
          syscall.SIGTERM)`. One context then bounds the pool's first connect,
          the migrations, and the wait below. On Windows, Ctrl+C arrives as
          `os.Interrupt`, and closing the console window arrives as SIGTERM.
      - lines: "32-36"
        note: >
          Defer right after each successful acquire. The first start fails in
          Migrate and still prints "pool: closed": every later return path
          releases what was acquired before it.
      - lines: "41-49"
        note: >
          In main.go, `ListenAndServe` runs in a goroutine and sends a startup
          failure (port 8080 taken) on this channel. The buffer of 1 lets that
          goroutine finish even if nobody reads. select waits for whichever
          comes first: a broken server or a signal.
      - lines: "51-56"
        note: >
          The drain budget must start from `context.Background()`. ctx is
          already cancelled by the signal, so anything derived from it is born
          cancelled and `srv.Shutdown` would give up at once, cutting off
          in-flight requests. main.go gets this right.
      - lines: "57-58"
        note: >
          When run returns here, its deferred calls run last-in, first-out:
          the two cancels, then `pool.Close`, then stop. The output shows
          "server drained" before "pool: closed", so a request still finishing
          during the drain can use the database.
use_cases:
  - "Reading a Go service for the first time: open `cmd/<name>/main.go` and read the dependency graph top to bottom before any package"
  - "Swapping an implementation per environment (in-process runner locally, HTTP client in Docker) with one interface and one if"
  - "Building handlers in tests with fakes instead of a database or a runner process"
strengths:
  - "The whole graph is ordinary code: Go to Definition works on every edge, and a missing dependency is a compile error, not a startup failure in Spring's or Guice's container"
  - "Structural interfaces let the consumer define the seam after the fact: `runexec` and `runclient` were written without knowing `httpapi.Runner` exists"
  - "Startup order is visible: config, logger, signal context, pool, migrations, runner, router, server. Teardown order follows from the defers"
weaknesses:
  - "Manual wiring grows by hand. Every new store or service is another constructor call and parameter; there is no autowiring"
  - "Only half-injected: `NewRouter` builds `runstore` and `learningstore` from the pool itself, and handlers hold concrete `*platform.DB` and store types. A test can build the real router without a database only by passing `platform.NewDB(nil)` and sending requests that never reach a store, which is what the one router test does"
  - "`slog.SetDefault(logger)` sets process-wide state next to the explicitly passed logger. Code that calls `slog.Info` directly uses the global and bypasses injection"
  - "`config.Load` reads `os.Getenv` directly and only validates integers: an empty or wrong database URL fails later, at the pool's ping, not at load"
pitfalls:
  - title: "Deriving the shutdown context from the signal context"
    detail: "The signal already cancelled it, so `Shutdown` returns immediately and drops in-flight requests. Start the drain budget from `context.Background()`."
    bad: "shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)"
    good: "shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)"
  - title: "log.Fatal deep inside startup"
    detail: "It calls `os.Exit`, which skips every deferred `Close`. Return the error to `main`, which is the only place that exits."
  - title: "Declaring the interface next to its implementation"
    detail: "A `runexec.Runner` interface would make httpapi import runexec, and runclient would have to import runexec to say it implements it. Declare small interfaces where they are used."
  - title: "Handing go:embed's FS straight to goose"
    detail: "go:embed keeps the `migrations/` prefix and goose only reads the FS root, so the backend died with \"no migrations found\". main.go wraps it in `fs.Sub`, and `main_test.go` guards that contract."
quiz:
  - q: "Where is `httpapi.Runner` declared, and why there?"
    options:
      - "In runexec, next to the main implementation"
      - "In httpapi, the package that calls Run, so neither implementation depends on HTTP code"
      - "In runapi, the shared wire-format package"
      - "In main.go, where the choice is made"
    answer: 1
    why: "Go interfaces are satisfied implicitly, so the consumer can declare the smallest interface it needs. runexec and runclient compile without knowing it exists."
  - q: "main.go derives the 10 s drain budget from `context.Background()`, not from the signal context. What would deriving it from the signal context do?"
    options:
      - "Nothing different"
      - "Give Shutdown an already-cancelled context, so it stops waiting at once and in-flight requests are cut off"
      - "Make Shutdown wait forever"
    answer: 1
    why: "A context derived from a cancelled parent is cancelled from birth. The example prints exactly that: context canceled versus <nil>."
  - q: "Which of these does `NewRouter` construct itself instead of receiving it from main?"
    options: ["The logger", "The Runner", "The run-history store", "The *platform.DB"]
    answer: 2
    why: "router.go calls `runstore.NewStore(pool.Pool())` and `learningstore.NewStore(pool.Pool())`. That is why a test can fake the runner but not the stores."
further_reading:
  - title: "Go Code Review Comments: Interfaces"
    url: "https://go.dev/wiki/CodeReviewComments#interfaces"
  - title: "package os/signal (NotifyContext, Windows behaviour)"
    url: "https://pkg.go.dev/os/signal"
  - title: "net/http: Server.Shutdown"
    url: "https://pkg.go.dev/net/http#Server.Shutdown"
  - title: "Effective Go: Interfaces and other types"
    url: "https://go.dev/doc/effective_go#interfaces_and_types"
---

## Concept

`server/cmd/stacklab/main.go` is about 140 lines, and its package comment states the
rule: main only wires dependencies, "config -> logger -> pool -> router -> server". Read
`run()` top to bottom and you have the backend's whole object graph:

1. `config.Load()` builds a `Config` from environment variables, with defaults for local
   mode, so `npm start` needs no environment at all.
2. `platform.NewLogger(...)` builds an `*slog.Logger`, also installed with `slog.SetDefault`.
3. `signal.NotifyContext` gives the context that bounds startup I/O and the shutdown wait.
4. `platform.NewPool(ctx, ...)` opens the pgx pool for the `stacklab` database, wrapped in
   `platform.NewDB(pool)`.
5. `platform.Migrate` applies the embedded goose migrations before anything is served.
6. The runner: `runexec.New(...)` (in-process, local mode) unless `STACKLAB_RUNNER_URL` is
   set, then `runclient.New(url)` (the Docker runner container).
7. An `http.Server` with explicit timeouts whose handler is
   `httpapi.NewRouter(logger, db, runner)`.
8. Serve in a goroutine, wait for a server error or a signal, then `Shutdown` with 10 s.

That is constructor injection with nothing else: no container, no reflection, no
annotations. Each constructor takes what it needs as parameters and returns a concrete
type. The only interface on the graph is `httpapi.Runner`, declared in
`internal/httpapi/run.go` by the code that calls it, because it is the only dependency
with two implementations.

## Coming from C/C++/Java/Python/JS

- **Java**: Spring and Guice build this graph by reflection from annotations; Dagger
  generates it at compile time. main.go is roughly what Dagger would generate, written
  and read by hand. There is no `implements`: `*runexec.Executor` satisfies
  `httpapi.Runner` because it has a matching `Run` method.
- **Python**: FastAPI's `Depends` or module-level singletons. The Go habit is to avoid
  package-level state and pass things explicitly, which is also why `slog.SetDefault`
  here stands out.
- **JS/TS**: NestJS has a decorator-driven container; plain Node code often uses module
  singletons (`import { db } from './db'`). Go's version is closer to a hand-written
  composition root in an `index.ts`.
- **C++**: the same constructor injection you would write by hand, minus abstract base
  classes. The interface can be declared after the implementations exist, by the code
  that needs it.

## How it works

The runner choice is the one place the graph branches:

```go
var runner httpapi.Runner = runexec.New(cfg.PlaygroundDatabaseURL, cfg.NodeProjectDir)
if cfg.RunnerBaseURL != "" {
	runner = runclient.New(cfg.RunnerBaseURL)
}
```

Both `runexec.Executor.Run` and `runclient.Client.Run` return
`(*runapi.Result, *runapi.Error, error)`, so the handler code in `run.go` never knows
which one it has. `cmd/runner/main.go`, the Docker runner's own main, wires the same
`runexec` package behind a different HTTP shell, which is how ADR-0008 keeps the two
modes from drifting.

Shutdown is the graph in reverse. `run()` defers `stop()`, then `pool.Close()`, then the
drain budget's `cancel()`. When the signal arrives, `srv.Shutdown` drains first, and only
when `run` returns do the defers close the pool, so requests finishing during the drain
still have a database.

## Design notes

What the file gets right: one readable composition root, a `run() error` that keeps
`os.Exit` out of reach of the defers, explicit `http.Server` timeouts (Go's zero values
mean no deadline), and a self-migrating start, so a fresh database comes up correct.

What it doesn't, read honestly:

- **The injection stops at the router.** `NewRouter` receives `*platform.DB` and builds
  `runstore.NewStore(pool.Pool())` and `learningstore.NewStore(pool.Pool())` inside, and
  the handlers hold concrete store types. The doc comment on `platform.DB` used to say it
  exists so handlers depend on "an interface declared by the consumer", but no such
  interface exists: `handleReady` takes `*platform.DB`. The comment now says so. The
  `Runner` seam is real; the database seam is aspirational. The one test that builds the
  router (`TestRouterAnswersUnmatchedRequestsWithProblems` in `middleware_test.go`)
  passes `platform.NewDB(nil)` and sends only requests that never reach a store
  (unmatched paths and `/healthz`).
- **The next step** would be small consumer-declared interfaces in httpapi (`pinger`,
  `runHistory`, `learningStore`) and a `NewRouter` that receives the stores built in
  main. That is a few lines in each file and buys handler tests without a database.
- **No framework is the right call at this size.** With fewer than ten constructors,
  explicit wiring costs a few lines per dependency and is never wrong silently. A
  generated-wiring tool starts paying off when the graph has dozens of nodes and several
  binaries share it.

Compare `go.design.constructors` for the pattern itself and `go.http.server-lifecycle`
for the timeout and shutdown details.
