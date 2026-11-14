---
id: go.concurrency.synctest
title: "testing/synctest: deterministic time for concurrent tests"
track: go
module: concurrency
order: 7
level: advanced
minutes: 13
since: go1.27
prerequisites: [go.concurrency.context]
compare_with: [ts.testing.vitest, go.testing.table-driven, go.testing.fakes-httptest]
tags: [concurrency, testing, synctest, time, timeouts]
summary: >
  synctest.Test runs a test in a bubble with a fake clock that jumps forward
  whenever every goroutine in it is blocked. Timeouts, backoff and heartbeats
  are tested instantly and exactly, with no clock interface in the production code.
examples:
  - id: retry
    file: examples/01_retry_test.go
    title: Seven seconds of backoff in zero real time
    runtime: go-test
    check: contains
    annotations:
      - lines: "13-27"
        note: >
          Ordinary production code: time.After and a context deadline, nothing
          test-specific. synctest swaps the clock underneath, so there is no
          Clock interface to inject.
      - lines: "30-31"
        note: >
          synctest.Test (Go 1.25) runs the function in a bubble. Goroutines
          started inside belong to it, and the time package inside uses a fake
          clock that starts at midnight UTC on 2000-01-01.
      - lines: "33-39"
        note: >
          retry waits 1s, 2s and 4s. Whenever every goroutine in the bubble is
          blocked, the clock jumps to the next timer, so seven seconds of
          waiting take no real time: the PASS line shows (0.00s).
      - lines: "43-45"
        note: >
          Fake time is exact, so the test asserts equality. With real time it
          could only check "at least 7s, not much more", and every run would
          cost seven seconds.
      - lines: "52-59"
        note: >
          Deadlines are timers too. The context fires at exactly 5s, in the
          middle of the 4s wait that would have ended at 7s, and retry returns
          the cause.
  - id: watchdog
    file: examples/02_watchdog_test.go
    title: Checking state just before and after a timeout with synctest.Sleep
    runtime: go-test
    check: contains
    annotations:
      - lines: "19-32"
        note: >
          The goroutine under test. Its timer belongs to the bubble's clock.
          Since Go 1.23, Reset on an unfired timer needs no Stop-and-drain
          dance: no stale value can arrive afterwards.
      - lines: "28-29"
        note: >
          When the test function returns, t.Context() is canceled and the
          goroutine exits. synctest.Test waits for every goroutine in the
          bubble, and fails the test if they are deadlocked, so a leaked
          goroutine can't go unnoticed.
      - lines: "36"
        note: >
          Beat is an unbuffered send to the watchdog goroutine. The clock can't
          move until that goroutine is blocked again, so its Reset happens at
          exactly 20s.
      - lines: "44-46"
        note: >
          synctest.Sleep (Go 1.27) is time.Sleep followed by synctest.Wait: the
          clock moves, then every other goroutine in the bubble runs until it
          blocks. On Go 1.25 and 1.26, write those two calls.
      - lines: "52-55"
        note: >
          At 50s the test's sleep and the watchdog's timer fire at the same
          fake instant. After a plain time.Sleep, which goroutine runs first is
          unpredictable and the check could see dead == false; the Wait inside
          synctest.Sleep lets the watchdog finish its turn first.
use_cases:
  - "Retry and backoff: assert the exact schedule (1s, 2s, 4s) without waiting seven seconds"
  - "Timeouts, TTL caches, heartbeats and debouncers: check the state just before and just after the boundary"
  - "Code built on context.WithTimeout, time.AfterFunc or tickers whose tests used to need long sleeps or an injected clock"
  - "HTTP timeouts over an in-memory network: net.Pipe, or httptest.NewTestServer (Go 1.27)"
strengths:
  - "No clock abstraction in production code: it keeps calling time.Now and time.After, where Java tests typically need a java.time.Clock passed in"
  - "Time advances by itself when everything is blocked, unlike Vitest or Jest fake timers, where the test calls advanceTimersByTime(ms) and has to know how far"
  - "Assertions are exact (got == 7*time.Second), with no tolerance windows and no flaky sleeps on a loaded CI machine"
  - "A bubble whose goroutines end up deadlocked fails the test instead of hanging until the go test timeout"
weaknesses:
  - "Only blocking inside the bubble counts. Real network I/O, system calls and mutex waits aren't durably blocking, so code under test needs in-memory fakes such as net.Pipe"
  - "Go 1.25 or later (synctest.Sleep: 1.27); projects stuck on older toolchains still need an injected clock"
  - "Bubble rules take some learning: no t.Run or t.Parallel inside, and channels and timers created in a bubble panic when used from outside"
pitfalls:
  - title: "Real network I/O inside the bubble"
    detail: "A goroutine reading from a TCP socket isn't durably blocked, so the clock won't advance past it and Wait keeps waiting. Use net.Pipe or httptest.NewTestServer, whose in-memory network works with synctest."
  - title: "Channels created outside the bubble"
    detail: "Only operations on channels created inside the bubble count as durably blocking. Create the system under test inside the synctest.Test function, not in a package variable or TestMain."
  - title: "time.Sleep when another goroutine wakes at the same instant"
    detail: "Two goroutines woken at the same fake time run in unpredictable order. Follow the sleep with synctest.Wait, or use synctest.Sleep (Go 1.27)."
    bad: "time.Sleep(30 * time.Second)\nif !w.Dead() { t.Fatal(\"alive\") }"
    good: "synctest.Sleep(30 * time.Second)\nif !w.Dead() { t.Fatal(\"alive\") }"
  - title: "Copying examples written for the Go 1.24 experiment"
    detail: "Go 1.24 had synctest.Run behind GOEXPERIMENT=synctest. Go 1.25 replaced it with synctest.Test(t, f); Run is not in the Go 1.27 package."
quiz:
  - q: "When does the fake clock inside a synctest bubble advance?"
    options: ["When every goroutine in the bubble is durably blocked", "Continuously, like real time", "Only when the test calls synctest.Wait"]
    answer: 0
    why: "The clock jumps to the next pending timer once nothing in the bubble can make progress without time passing."
  - q: "Which of these does not count as durably blocking?"
    options: ["Receiving from a channel created in the bubble", "time.Sleep", "Waiting to lock a sync.Mutex", "sync.WaitGroup.Wait after Add was called in the bubble"]
    answer: 2
    why: "A mutex can be released by a goroutine outside the bubble, so a goroutine waiting on one is not durably blocked."
  - q: "What does synctest.Sleep(d) do?"
    options: ["Sleeps for d of real time", "time.Sleep(d), then synctest.Wait()", "Moves the clock by d without blocking"]
    answer: 1
    why: "Its docs define it as exactly those two calls: advance the fake clock, then let the other goroutines settle."
further_reading:
  - title: "package testing/synctest"
    url: "https://pkg.go.dev/testing/synctest"
  - title: "Testing Time (and other asynchronicities)"
    url: "https://go.dev/blog/testing-time"
  - title: "Go 1.25 release notes: testing/synctest"
    url: "https://go.dev/doc/go1.25#new-testingsynctest-package"
  - title: "Go 1.27 release notes: testing/synctest"
    url: "https://go.dev/doc/go1.27#testingsynctestpkgtestingsynctest"
---

## Concept

Testing concurrent code with timeouts used to mean one of two compromises: real sleeps
(slow and flaky) or a clock interface threaded through production code.
`testing/synctest` (Go 1.25) removes both.

`synctest.Test(t, func(t *testing.T) { ... })` runs the function in a **bubble**. Every
goroutine started inside belongs to the bubble, and inside it the `time` package runs on
a **fake clock** that starts at midnight UTC, 2000-01-01. The clock only moves when
every goroutine in the bubble is **durably blocked**, meaning only another goroutine in
the same bubble could wake it. Then it jumps straight to the next timer.

Two helpers control the timing inside a test:

- `synctest.Wait()` blocks until every other goroutine in the bubble is durably blocked,
  so you can assert on state once the system has settled.
- `synctest.Sleep(d)` (Go 1.27) is `time.Sleep(d)` followed by `synctest.Wait()`.

## Coming from C/C++/Java/Python/JS

- **JS**: Vitest's `vi.useFakeTimers()` and Jest's fake timers replace `setTimeout`,
  but the test drives them with `advanceTimersByTime(ms)`. In a bubble the clock
  advances on its own whenever nothing else can happen, across all goroutines.
- **Java**: the usual approach is injecting a `java.time.Clock` or a fake
  `ScheduledExecutorService`, plus Awaitility-style polling. synctest needs neither:
  production code keeps calling `time.Now` and `time.After`.
- **Python**: `freezegun` or `time-machine` freeze `time.time()`, but a thread that is
  sleeping still sleeps in real time. The bubble controls sleeps, timers and deadlines
  together.
- **C++**: comparable to templating code on a clock type, done by the runtime instead.

## How it works

The runtime tracks every goroutine in the bubble. A goroutine is durably blocked when it
is blocked on a send, receive or `select` over channels created in the bubble, on
`time.Sleep`, on `sync.Cond.Wait`, or on `sync.WaitGroup.Wait` when `Add` was called in
the bubble. Locking a `sync.Mutex`, network I/O and system calls are not durably
blocking, because something outside the bubble could end them.

When every goroutine in the bubble is durably blocked:

1. if the test is in `synctest.Wait`, `Wait` returns;
2. otherwise, while the test function is still running, the clock jumps to the next
   time that would wake a goroutine;
3. if there is no such time, the bubble is deadlocked: `synctest.Test` panics and the
   test fails.

`synctest.Test` returns only after every goroutine in the bubble has exited. Inside the
bubble, `t.Context()` has a `Done` channel that belongs to the bubble, and `t.Run`,
`t.Parallel` and `t.Deadline` must not be called.

These examples run with `go test -v`. Real time per test is milliseconds, while the
logged fake time is seconds.

## Design notes

Write production code against the real `time` and `context` APIs, and let tests decide
how time passes. Reserve injected clocks for the cases synctest doesn't cover, such as
code that has to block on real I/O.

A useful test shape: start the system inside the bubble, `synctest.Sleep` to just
before the interesting moment and assert nothing has happened yet, then to just after
and assert that it did. `synctest.Wait()` on its own is the way to assert "this has
not happened": once it returns, nothing else can happen without time moving.

Fake networks keep tests inside the bubble: `net.Pipe` for raw connections, and
`httptest.NewTestServer` (Go 1.27), whose in-memory network suits synctest, for HTTP.
