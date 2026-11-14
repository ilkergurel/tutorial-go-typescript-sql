---
id: go.concurrency.select
title: "select: timeouts, default, and multiplexing"
track: go
module: concurrency
order: 2
level: intermediate
minutes: 12
since: go1.23
prerequisites: [go.concurrency.goroutines-channels]
compare_with: [ts.async.combinators, ts.async.cancellation, design.paradigms.concurrency]
tags: [concurrency, select, channels, timeouts, time]
summary: >
  select waits on several channel operations at once and runs one that can
  proceed. A timer channel gives you timeouts, default gives you non-blocking
  sends and receives, and a for-select loop merges many inputs.
examples:
  - id: timeout
    file: examples/01_timeout.go
    title: A timeout is just another case
    runtime: go
    check: exact
    annotations:
      - lines: "15"
        note: >
          Capacity 1 is what makes the timeout safe. With an unbuffered channel
          the goroutine would block forever on line 18 once await had given up:
          one leaked goroutine per timed-out call.
      - lines: "25-30"
        note: >
          select blocks until one case can proceed. time.After returns a channel
          that receives once after the duration, so a timeout is an ordinary
          case, not a special language feature.
      - lines: "34-36"
        note: >
          The reply wins here: the goroutine can send at once, and a one-second
          timer is nowhere near firing.
      - lines: "38-44"
        note: >
          Nothing arrives within 50ms, so the timer case runs. The slow call
          isn't stopped by the timeout; closing slow lets it finish, and its
          send lands in the buffer nobody reads. Stopping the work itself is
          what context is for (lesson 4).
  - id: default
    file: examples/02_default.go
    title: default for non-blocking sends and receives, and random choice
    runtime: go
    check: exact
    annotations:
      - lines: "8-13"
        note: >
          default runs when no other case is ready, so this select never
          blocks. On a full buffer the send case isn't ready and the event is
          dropped: load shedding instead of backpressure.
      - lines: "23-30"
        note: >
          A non-blocking receive drains what is buffered and stops. Without
          default the third receive would block forever, and with no other
          goroutine alive the runtime would stop the program with "all
          goroutines are asleep - deadlock!".
      - lines: "34-46"
        note: >
          Both channels are closed, so both cases are always ready. The spec
          requires a uniform pseudo-random choice, so after 1000 rounds both
          counters are positive (missing one has probability 2^-999). Case order
          gives no priority.
  - id: multiplex
    file: examples/03_multiplex.go
    title: for-select over several inputs, nil channels to switch cases off
    runtime: go
    check: exact
    annotations:
      - lines: "25"
        note: >
          time.After is called once, before the loop, so the second covers the
          whole merge. Called inside the select it would create a fresh timer
          on every iteration: an idle timeout that never fires while messages
          keep trickling in.
      - lines: "31-33"
        note: >
          A receive from a closed channel is always ready and yields ok ==
          false. Without the nil assignment this case would win again and
          again, spinning the loop. A nil channel never proceeds, which
          switches the case off.
      - lines: "34"
        note: >
          continue applies to the for loop. break here would only leave the
          select, the same trap as break inside a switch.
      - lines: "48-51"
        note: >
          Values from one channel arrive in the order they were sent. Across
          channels there is no order, so the output is grouped per source
          rather than printed as received.
use_cases:
  - "Timeouts around any channel operation: a worker's reply, a result from a pool, a lock implemented as a channel"
  - "Load shedding: try-send with default to drop metrics or audit events instead of blocking the request path"
  - "Event loops: one goroutine owns some state and reacts to commands, ticker ticks and shutdown arriving on different channels"
  - "Merging streams: fan-in of several producers, with nil channels switching off the finished ones"
strengths:
  - "Waiting on several queues, a timer and a shutdown signal is one statement. Java has BlockingQueue.poll(timeout) for a single queue and nothing built in for many"
  - "The random choice among ready cases prevents starvation by construction; there is no fairness flag to remember, unlike Java's ReentrantLock(true)"
  - "Sends and receives are both cases, so backpressure (block) and load shedding (default) are the same construct with one line changed"
weaknesses:
  - "No priority: when a shutdown channel and a busy work channel are both ready, select may keep picking work. Preferring one needs an explicit check first"
  - "Like Promise.race, select doesn't stop the work behind the losing case; the slow goroutine keeps running unless a context tells it to stop"
  - "The cases are fixed in source code. Waiting on a dynamic number of channels needs reflect.Select or a fan-in goroutine per channel"
pitfalls:
  - title: "time.After inside the loop is an idle timeout, not a deadline"
    detail: "Each iteration creates a new timer, so the timeout only fires after that long without any message. Create the timer (or a context with a deadline) once, before the loop."
    bad: "for { select { case m := <-in: handle(m); case <-time.After(5 * time.Second): return } }"
    good: "deadline := time.After(5 * time.Second)\nfor { select { case m := <-in: handle(m); case <-deadline: return } }"
  - title: "break leaves the select, not the loop"
    detail: "Inside a for-select, break ends the select statement only and the loop goes round again. Use return, or a labeled break."
    bad: "for { select { case <-done: break } }"
    good: "loop:\n\tfor { select { case <-done: break loop } }"
  - title: "for-select with an empty default spins"
    detail: "A default that does nothing turns a blocking wait into a busy loop that burns a CPU core. Drop the default and let select block."
    bad: "for { select { case m := <-in: handle(m); default: } }"
    good: "for m := range in { handle(m) }"
  - title: "Unbuffered result channel plus a timeout leaks the sender"
    detail: "After the timeout nobody receives, so the worker blocks on its send forever. Give the result channel capacity 1, or have the worker select on ctx.Done() too."
    bad: "result := make(chan string)"
    good: "result := make(chan string, 1)"
quiz:
  - q: "Two cases of a select are ready at the same moment. Which one runs?"
    options: ["One chosen uniformly at random", "The first one in source order", "Both, in source order"]
    answer: 0
    why: "The spec requires a uniform pseudo-random choice among the cases that can proceed, which is why case order can't express priority."
  - q: "Inside a select, what does a receive from a nil channel do?"
    options: ["Panics", "Never proceeds, so the case is effectively disabled", "Returns the zero value immediately"]
    answer: 1
    why: "Communication on a nil channel never proceeds. Setting a finished input to nil is the standard way to switch its case off."
  - q: "for { select { case m := <-in: handle(m); case <-time.After(time.Minute): return } }. When does this return?"
    options: ["One minute after the loop started", "Never", "After one minute without any message on in"]
    answer: 2
    why: "time.After runs again on every iteration, so each message restarts the minute. For a deadline, create the timer once before the loop."
further_reading:
  - title: "The Go Programming Language Specification: Select statements"
    url: "https://go.dev/ref/spec#Select_statements"
  - title: "A Tour of Go: Select"
    url: "https://go.dev/tour/concurrency/5"
  - title: "package time: After"
    url: "https://pkg.go.dev/time#After"
  - title: "Go 1.23 release notes: timer changes"
    url: "https://go.dev/doc/go1.23#timer-changes"
---

## Concept

`select` is a `switch` whose cases are channel operations, sends or receives. On entry
it evaluates every channel expression once. If one or more cases can proceed, it picks
one **uniformly at random** and runs it. If none can, it runs `default` if there is one,
and otherwise blocks until some case becomes ready. A `nil` channel never proceeds, and
an empty `select {}` blocks forever.

Three idioms cover nearly every use:

- **Timeout**: add `case <-time.After(d):` next to the operation you are waiting for.
- **Non-blocking**: add `default:` to try a send or receive without waiting.
- **Multiplexing**: a `for { select { ... } }` loop that reacts to whichever input is
  ready, including a shutdown signal.

## Coming from C/C++/Java/Python/JS

- **C**: the name is no accident: it is `select(2)`/`poll`/`epoll` for typed channels.
  One difference: those report readiness and you do the read afterwards, while Go's
  `select` performs the send or receive itself, so the value is already yours when the
  case body runs.
- **Java**: there is no language equivalent. `BlockingQueue.poll(timeout)` covers one
  queue; `CompletableFuture.anyOf` covers futures. `select` mixes queues, timers and
  shutdown signals in one wait.
- **Python**: `asyncio.wait(..., return_when=FIRST_COMPLETED)` and `asyncio.wait_for`
  play this role for awaitables. Python cancels the loser with `task.cancel()`; Go
  uses a `context` for that.
- **JS**: `Promise.race([fetch(url), timeout(5000)])` is a `select` with a `time.After`
  case, with the same catch: the race doesn't abort the slow `fetch`. That takes an
  `AbortController`, which is what `context` is in Go.

## How it works

`time.After(d)` creates a timer and returns its channel, which receives the current time
once `d` has passed. Because it's an ordinary channel, a timeout is an ordinary case.

Go 1.23 changed timers in two ways (your `go.mod` says `go 1.27`, so you have both).
Timers that are no longer referenced are garbage collected even if they haven't fired,
so `time.After` in a loop no longer piles up live timers. Before Go 1.23 the docs told you
to use `time.NewTimer` and `Stop` instead. Timer channels are also unbuffered now, so
after `Stop` or `Reset` returns, no stale value can be received. Go 1.27 removed the
`asynctimerchan` GODEBUG that could switch the old behaviour back on.

Order within one channel is guaranteed: values arrive in the order they were sent.
Across the cases of a `select` there is no order at all.

## Design notes

Put the timeout at the caller, and give the worker a result channel with capacity 1 so
it can always finish its send and exit. A timeout only stops the waiting, not the work:
to stop the work, pass a `context.Context` and select on `ctx.Done()` inside the worker.
That is the next lesson, and in real code `ctx.Done()` replaces most hand-made `quit`
channels.

A goroutine that owns some state and runs a `for`-`select` over command channels is
Go's actor: nothing else touches the state, so it needs no mutex. It is a good fit when
the state changes in response to messages. When many goroutines just read and update a
shared structure, a mutex (lesson 3) is simpler.

If one case must win, such as shutdown over work, don't rely on case order. Check it
first with a non-blocking select, or test `ctx.Err()` at the top of the loop.
