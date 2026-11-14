---
id: go.concurrency.errgroup
title: "errgroup with limits, data races, -race, goroutine leaks"
track: go
module: concurrency
order: 6
level: advanced
minutes: 15
since: go1.27
prerequisites: [go.concurrency.patterns]
compare_with: [ts.async.combinators, go.performance.pprof, design.paradigms.concurrency]
tags: [concurrency, errgroup, structured-concurrency, data-races, race-detector, goroutine-leaks]
summary: >
  errgroup runs a group of goroutines, cancels the rest when one fails, bounds
  how many run at once, and returns the first error. The race detector finds
  unsynchronised memory access; the goroutineleak profile (Go 1.27) finds
  goroutines that can never finish.
examples:
  - id: errgroup
    file: examples/01_errgroup.go
    title: "errgroup.WithContext and SetLimit: first error wins, the rest are canceled"
    runtime: go
    check: exact
    annotations:
      - lines: "27"
        note: >
          WithContext returns the group and a derived context. It is canceled
          when the first task returns an error, or when Wait returns, so it
          must not be used after Wait.
      - lines: "28"
        note: >
          With a limit, g.Go blocks until fewer than two tasks are running:
          the semaphore from the previous lesson, built in. Set it before the
          first Go.
      - lines: "30-47"
        note: >
          Each task writes only its own status[i], so collecting results needs
          no mutex. i and page are fresh in every iteration (Go 1.22), so each
          closure sees its own values.
      - lines: "33-34"
        note: >
          Page 2 fails at once and cancels ctx; page 1 can only end through
          that cancellation. Pages 3 to 5 get a slot only after one of the
          first two has returned, which is after the cancel, so they start on
          a canceled context. That ordering, not timing, makes the output the
          same on every run.
      - lines: "43-45"
        note: >
          The group keeps only the first non-nil error. Page 1's
          context.Canceled comes after page 2's failure, so Wait reports the
          real cause rather than the cancellation it triggered.
      - lines: "50-55"
        note: >
          Wait returns only after every task has returned, failure or not, so
          no goroutine outlives the call. x/sync v0.25.0 cancels the context
          with the error as its cause, which context.Cause reads back.
  - id: leak
    file: examples/02_leak.go
    title: Finding a goroutine leak with the goroutineleak profile
    runtime: go
    check: exact
    annotations:
      - lines: "14-17"
        note: >
          The fix is room for every sender: each goroutine completes its send
          and exits, and the unread answers are garbage collected along with
          the channel. Having each sender also select on ctx.Done() works too.
      - lines: "18-22"
        note: >
          first returns after one receive. On an unbuffered channel, each of
          the other replicas waits on its send for a receiver that will never
          come, and keeps its stack and everything it references alive.
      - lines: "27-31"
        note: >
          The goroutineleak profile (an experiment in Go 1.26, always available
          since Go 1.27) runs a special GC cycle. A goroutine blocked on a
          channel or lock that no runnable goroutine can reach can never wake,
          so it is reported. WriteTo with debug=1 prints the leaked stacks.
      - lines: "35-41"
        note: >
          The losing replicas need a moment to reach their send, so this polls
          with a bound instead of trusting a fixed sleep. A goroutine that a
          live goroutine could still unblock is never counted.
      - lines: "49-50"
        note: >
          Two leaked goroutines per call. In a server that calls first on
          every request, the goroutine count only ever grows.
use_cases:
  - "Fetch several independent resources for one request (user, orders, recommendations) and fail fast if one fails"
  - "Process a batch with bounded parallelism: g.SetLimit(8) instead of a hand-written semaphore"
  - "Run go test -race in CI on Linux for every package that starts goroutines"
  - "Expose /debug/pprof/goroutineleak (net/http/pprof, Go 1.27) next to /debug/pprof/goroutine in services, or check the profile in tests"
strengths:
  - "Structured concurrency in a few lines: no goroutine outlives Wait, and one failure cancels its siblings, which Promise.all does not do: the other promises keep running"
  - "SetLimit and TryGo cover bounded parallelism without a separate semaphore"
  - "The race detector reports races with both stacks and the goroutine creation site, where a C++ or Java data race just yields wrong numbers now and then"
  - "Leak detection uses the garbage collector's reachability, so it finds leaks no test asserted on, without changing the code"
weaknesses:
  - "errgroup is in golang.org/x/sync, not the standard library: one go get away, but a dependency"
  - "Only the first error is returned; to report every failure, collect errors yourself and errors.Join them"
  - "-race needs cgo and a C compiler, costs 5-10x memory and 2-20x time per the docs, and only finds races on code paths that actually ran"
  - "The leak profile misses goroutines blocked on primitives reachable from globals or from running goroutines; a clean profile is not proof"
pitfalls:
  - title: "Using the group's context after Wait"
    detail: "WithContext's context is canceled when Wait returns, even on success. Any later call made with it fails with context.Canceled. Derive follow-up work from the parent context."
  - title: "Expecting errgroup to recover panics"
    detail: "A panic in a task crashes the program like any goroutine panic. The x/sync v0.25.0 source explains why it doesn't propagate panics to Wait. Recover inside the task if it calls code that may panic."
  - title: "Ignoring ctx inside the tasks"
    detail: "Cancellation only helps if tasks pass ctx to their I/O calls or select on ctx.Done(). A task that ignores ctx keeps the whole Wait waiting."
    bad: "g.Go(func() error { return fetch(context.Background(), url) })"
    good: "g.Go(func() error { return fetch(ctx, url) })"
  - title: "Treating a quiet program as race-free"
    detail: "A racy counter can print the right total a thousand times in a row and lose updates under load. Races on strings, slices or maps can even corrupt memory. Only -race, or reasoning about happens-before, tells you the code is correct."
quiz:
  - q: "errgroup.WithContext: task A returns an error while task B is waiting on <-ctx.Done(). What happens?"
    options: ["Wait returns immediately with A's error", "B keeps running; Wait returns when B finishes on its own", "ctx is canceled, B returns, and then Wait returns A's error"]
    answer: 2
    why: "The first error cancels the group's context; Wait still waits for every task, then returns the first error."
  - q: "go test ./... passes 100 times in a row. Is the code race-free?"
    options: ["Yes", "Not necessarily: run go test -race, and even then only executed paths are checked", "Yes, if GOMAXPROCS > 1"]
    answer: 1
    why: "Races often produce correct-looking output. The race detector finds them only on code paths the tests actually run."
  - q: "Which goroutine does the goroutineleak profile report?"
    options: ["A goroutine blocked on a channel or lock that no runnable goroutine can reach", "Any goroutine blocked for more than a minute", "Every goroutine still running when main returns"]
    answer: 0
    why: "Reachability is the test: if nothing that can run can ever reach the primitive, the blocked goroutine can never wake."
further_reading:
  - title: "package golang.org/x/sync/errgroup"
    url: "https://pkg.go.dev/golang.org/x/sync/errgroup"
  - title: "Data Race Detector"
    url: "https://go.dev/doc/articles/race_detector"
  - title: "Go 1.27 release notes: goroutine leak profile"
    url: "https://go.dev/doc/go1.27#goroutineleak-profiles"
  - title: "package runtime/pprof"
    url: "https://pkg.go.dev/runtime/pprof"
  - title: "The Go Memory Model"
    url: "https://go.dev/ref/mem"
---

## Concept

**errgroup** (`golang.org/x/sync/errgroup`) is a `WaitGroup` that understands errors:

- `g, ctx := errgroup.WithContext(parent)` creates a group and a context.
- `g.Go(func() error {...})` starts a task; `g.SetLimit(n)` caps how many run at once,
  and `g.TryGo` starts one only if a slot is free.
- `g.Wait()` waits for all tasks and returns the first non-nil error. The first error
  also cancels `ctx`, so well-behaved siblings stop early.

That's **structured concurrency**: goroutines are started inside a scope and none
outlive it. Two tools find the bugs that slip through anyway:

- A **data race** is two goroutines accessing the same memory, at least one writing,
  with no happens-before ordering between them. `go test -race` and `go run -race`
  detect races while the code runs.
- A **goroutine leak** is a goroutine blocked forever. Go 1.27's `goroutineleak` profile
  reports the ones that provably can never wake up.

## Coming from C/C++/Java/Python/JS

- **JS**: `await Promise.all(tasks)` rejects on the first failure, but the other
  promises keep running. errgroup also cancels them, the way an `AbortController`
  shared by all the fetches would.
- **Java**: errgroup is close to `StructuredTaskScope` (still a preview API) set to shut
  down on the first failure: fork tasks, join, get the first exception, siblings cancelled.
- **Python**: `asyncio.TaskGroup` (3.11) is the same idea, cancelling the rest when one
  task fails. It reports all failures in an `ExceptionGroup`; errgroup keeps only the
  first error.
- **C/C++**: ThreadSanitizer (`-fsanitize=thread`) is the engine behind Go's `-race`.
  The memory model is milder than C++'s: a racy read of a single word sees some value
  actually written there, as in Java. But races on multiword values (strings, slices,
  interfaces, maps) can corrupt memory, so a race report is always a bug, never noise.

## How it works

`-race` instruments every memory access at compile time and checks happens-before at
run time. Take a goroutine and `main` that both increment a counter without a lock:

```go
package main

import "fmt"

func main() {
	hits := 0
	done := make(chan bool)
	go func() {
		hits++ // line 9
		done <- true
	}()
	hits++ // line 12
	<-done
	fmt.Println(hits)
}
```

Built normally, it prints `2` and looks fine. `go run -race main.go` still prints `2`,
but also writes a report to stderr that names both accesses and where the goroutine was
started, and the program exits with status 66 (addresses, paths, and which access is
the "Previous" one vary by run):

```
==================
WARNING: DATA RACE
Read at 0x00c000012128 by goroutine 7:
  main.main.func1()
      /home/you/race/main.go:9 +0x2e

Previous write at 0x00c000012128 by main goroutine:
  main.main()
      /home/you/race/main.go:12 +0x8c

Goroutine 7 (running) created at:
  main.main()
      /home/you/race/main.go:8 +0x7a
==================
Found 1 data race(s)
exit status 66
```

This report was not captured on the owner's machine (no C compiler, so no `-race`); it
shows the documented format. The [race detector article](https://go.dev/doc/articles/race_detector)
says a report holds "stack traces for conflicting accesses, as well as stacks where the
involved goroutines were created", and its `exitcode` option defaults to 66, which
`go run` prints as `exit status 66`. The exact lines (`Read at 0x... by goroutine N:`,
`Previous write at 0x... by main goroutine:`, `Goroutine N (running) created at:`,
`Found 1 data race(s)`) are the ones Go 1.27.4's own tests expect, in
`src/runtime/race/output_test.go` (cases `simple` and `midstack_inlining_traceback`).
The goroutine number, addresses, paths, frame names and line numbers above are
illustrative, not observed.

Under `go test -race`, the failing test also reports `race detected during execution
of test`. The race detector needs cgo, and on Windows a MinGW-w64 C compiler (runtime
version 8 or later). Without one, `go run -race` stops with `go: -race requires cgo;
enable cgo by setting CGO_ENABLED=1`, and with `CGO_ENABLED=1` cgo reports that `gcc`
was not found. The lesson runner doesn't build with `-race`, which is why the runnable
examples show a leak rather than a race; run `-race` in CI on Linux.

The leak profile works through the garbage collector. If goroutine G is blocked on a
channel or mutex that no runnable goroutine can reach, nothing can ever unblock G. It
was an experiment in Go 1.26 (`GOEXPERIMENT=goroutineleakprofile`) and is always
available from Go 1.27, both as `pprof.Lookup("goroutineleak")` and as
`/debug/pprof/goroutineleak` from `net/http/pprof`.

## Design notes

Reach for errgroup whenever you'd write a `WaitGroup` plus an error variable plus a
mutex. Pass the group's `ctx` into every task, write results into a pre-sized slice by
index, and return wrapped errors so `Wait` tells you which task failed. Add it to a
module with `go get golang.org/x/sync`.

Make `go test -race ./...` part of CI for any package that starts goroutines, and
write tests that actually exercise the concurrent paths; the detector can't see code
that didn't run. Leaks are cheapest to find in tests: check `runtime.NumGoroutine()` or
the `goroutineleak` profile at the end, or run the test under `testing/synctest`
(next lesson), which fails a test whose goroutines are deadlocked. In production,
watch the goroutine count over time; a line that only goes up is a leak.
