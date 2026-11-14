---
id: ts.async.event-loop
title: "The event loop, compared with goroutines"
track: ts
module: async
order: 1
level: basic
minutes: 12
since: es2017
prerequisites: [ts.functions.higher-order]
compare_with: [go.concurrency.goroutines-channels, go.concurrency.sync, design.paradigms.concurrency]
tags: [async, event-loop, microtasks, concurrency, run-to-completion]
summary: >
  A page runs your code on one thread: one task at a time, each to completion,
  then every queued microtask. So output order is predictable, a long loop
  freezes the page, and shared variables only change at await points.
examples:
  - id: order
    file: examples/01_order.ts
    title: "Sync code, then microtasks, then the next task"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "5"
        note: >
          The module body is itself one task. Nothing else starts until it has
          run to its last line, whatever was scheduled along the way.
      - lines: "7"
        note: >
          setTimeout(fn, 0) does not mean "now": it queues a new task, which waits
          for the current task and for every microtask. It prints last.
      - lines: "9-13"
        note: >
          then() callbacks and queueMicrotask() go to the microtask queue. After
          each task the loop drains that queue completely, including microtasks
          queued during the drain (line 11), so "7" still beats the timer.
      - lines: "15-20"
        note: >
          An async function runs synchronously up to its first await; the rest
          is a continuation queued as a microtask. void marks the returned
          promise as deliberately ignored, which lint rules about floating
          promises look for.
  - id: blocking
    file: examples/02_blocking.ts
    title: "A busy loop blocks the timer; only a task-sized pause lets it in"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-8"
        note: >
          CPU-bound work as the event loop sees it. Nothing preempts it: timers,
          input events and rendering all wait. A Go goroutine in the same loop
          gets preempted (asynchronously since Go 1.14), and the other goroutines
          keep running on other threads anyway.
      - lines: "16-19"
        note: >
          The timer is due at once. It stands for anything that arrives while
          the work runs: a click, a network response, a message.
      - lines: "11"
        note: >
          Awaiting a resolved promise is not a pause. It queues a microtask, and
          microtasks drain before the next task, so the timer still waits for
          all three chunks.
      - lines: "12"
        note: >
          Waiting for setTimeout(resolve, 0) lets the current task end; the work
          resumes in a later task. The timer that was queued first runs in
          between: after chunk 1. scheduler.yield() is
          the newer, priority-aware form, not yet in every browser.
  - id: interleaving
    file: examples/03_interleaving.ts
    title: "No data races, but every await is an interleaving point"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "6-15"
        note: >
          Run-to-completion: between two awaits nothing else runs, so 1000
          concurrent increments lose nothing. The same counter shared by 1000
          goroutines is a data race in Go; it needs sync/atomic or a Mutex, and
          go test -race would flag it.
      - lines: "19-23"
        note: >
          Read, await, write: both calls read 100 before either writes, and the
          second write wins. This is a lost update, deterministic rather than a
          memory race, and the same shape as two SQL transactions doing
          SELECT-then-UPDATE without FOR UPDATE.
      - lines: "30-35"
        note: >
          A promise chain is an async mutex: each call starts after the previous
          one settled. catch() keeps one failure from blocking every later call.
          When the code allows it, the simpler fix is to read after the await.
use_cases:
  - "Explaining a frozen page: any synchronous work longer than a frame (about 16 ms at 60 Hz) delays input and painting"
  - "Splitting long CPU work into task-sized chunks, or moving it to a Web Worker"
  - "Predicting log and test output order: synchronous code, then microtasks, then tasks"
  - "Finding the interleaving points in async code that shares state: every await"
strengths:
  - "No data races on ordinary variables: unlike Go or Java threads, code between two awaits can't be interrupted, so most shared state needs no lock"
  - "Thousands of pending I/O operations cost closures and promises on one thread, where Java's classic thread-per-request model needed a thread each"
  - "Ordering is specified by the HTML standard (task, then all microtasks), not left to a scheduler, so it is reproducible"
weaknesses:
  - "One thread for everything: a 200 ms loop freezes input, timers and rendering for 200 ms, where Go would run other goroutines on other cores"
  - "No preemption: a microtask that keeps queueing microtasks starves the page exactly like a busy loop"
  - "CPU parallelism means Web Workers with message passing, not goroutines sharing memory"
pitfalls:
  - title: "Expecting setTimeout(fn, 0) to run immediately"
    detail: "It queues a task that runs after the current task and all microtasks. Browsers also clamp deeply nested timers to at least 4 ms and throttle timers in background tabs (Chrome runs them once per second, and once per minute for chained timers after five minutes hidden)."
  - title: "Yielding with a microtask"
    detail: "await Promise.resolve() (or await on anything already settled) never lets a task run. To let input and rendering in, end the task."
    bad: "for (const c of chunks) { work(c); await Promise.resolve(); }"
    good: "for (const c of chunks) { work(c); await new Promise((r) => setTimeout(r, 0)); }"
  - title: "Read before an await, write after it"
    detail: "Another call can change the value while this one is suspended. Do the read-modify-write in one synchronous step after the await, or serialize the calls."
    bad: "const current = balance; await save(); balance = current - amount;"
    good: "await save(); balance -= amount;"
  - title: "Depending on order across task sources"
    detail: "Timers created in the same task with the same delay run in creation order, and microtasks are FIFO. Between a timer, a postMessage and a click, the browser picks the queue: don't rely on that order."
quiz:
  - q: "setTimeout(() => log(\"A\"), 0); Promise.resolve().then(() => log(\"B\")); log(\"C\"); What is printed?"
    options: ["A B C", "C B A", "C A B", "B C A"]
    answer: 1
    why: "C is synchronous; B is a microtask, drained when the current task ends; A is a new task and runs after that."
  - q: "A click handler runs a 2-second computation. What does the user see?"
    options: ["Other handlers keep running in parallel", "The page freezes for 2 s: no input, no timers, no repaint", "The browser preempts the loop every few milliseconds"]
    answer: 1
    why: "One thread, run-to-completion: nothing else runs until the handler returns. Chunk the work or move it to a Worker."
  - q: "Inside a long loop, does await Promise.resolve() let a pending setTimeout callback run?"
    options: ["Yes, every await yields to the event loop", "No: it only queues a microtask, and microtasks drain before the next task"]
    answer: 1
    why: "The continuation is a microtask; the timer is a task. The timer waits until the loop finishes or a real task boundary appears."
  - q: "Two calls start together; each does const v = x; await io(); x = v + 1; on the same x, which starts at 0. What is x afterwards?"
    options: ["2", "1", "Unpredictable memory corruption, like a Go data race"]
    answer: 1
    why: "Both read 0 before either writes, then each writes 1. A lost update, but a deterministic one: there is no parallel memory access."
further_reading:
  - title: "MDN: JavaScript execution model"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Execution_model"
  - title: "MDN: In depth: Microtasks and the JavaScript runtime environment"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/HTML_DOM_API/Microtask_guide/In_depth"
  - title: "MDN: setTimeout() and its delays"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Window/setTimeout"
  - title: "The Go Memory Model (data races, for comparison)"
    url: "https://go.dev/ref/mem"
---

## Concept

A browser tab runs your JavaScript on **one thread** with an **event loop**. The loop
takes one *task* (a script, a timer callback, an event, a message), runs it to
completion, then runs every queued *microtask* (promise callbacks, `queueMicrotask`,
the code after an `await`), possibly renders, and starts over. Nothing preempts a
running task.

Three consequences shape all async code:

- **Order is predictable**: synchronous code first, then all microtasks, then the next
  task.
- **Blocking freezes everything**: while one task runs, no input is handled, no timer
  fires and nothing repaints.
- **Shared state changes only at `await` points**: no data races, but every `await` lets
  other code in before your function continues.

This is concurrency without parallelism: many operations in flight, exactly one running.

## Coming from C/C++/Java/Python/JS

- **Go**: goroutines run on several OS threads and are preempted (asynchronously since
  Go 1.14), so a busy goroutine doesn't stall the others, and two goroutines writing one
  variable is a data race. Here one task runs at a time and is never interrupted: no data
  races, but a busy loop stalls the whole page. `await` is the nearest thing to a channel
  receive: it parks the function, not the thread.
- **Java**: think of a single-threaded executor, or Swing's event dispatch thread. Blocking
  it freezes the UI in exactly the same way.
- **Python**: `asyncio` is the same model, cooperative switching at `await` on one
  thread. The differences: Promises start running when created, and the browser owns the
  loop, so there is no `asyncio.run`.
- **C/C++**: an epoll or libuv loop dispatching callbacks, plus a second, higher-priority
  queue (microtasks) drained after every callback.

## How it works

One turn of the loop, as the HTML standard describes it:

1. Pick a task from one of the task queues. Timers, messages, network events and user
   input have separate queues, and the browser chooses which one to serve.
2. Run the task to completion.
3. Microtask checkpoint: run microtasks until the queue is empty, including the ones
   queued meanwhile.
4. If a frame is due, render: `requestAnimationFrame` callbacks, style, layout, paint.

Within one queue the order is fixed: timers created in the same task with the same delay
fire in creation order, and microtasks are first in, first out. Across queues, don't
depend on the order. Each Web Worker has its own loop on its own thread, which is how a
page gets real parallelism (see the storage and workers lesson).

## Design notes

Keep tasks short. Browsers report tasks over 50 ms as *long tasks*, the point where users
notice input lag. For heavy CPU work, either split it into chunks that end the task
between them (`setTimeout(0)`, or `scheduler.yield()` where available) or move it to a
Worker and keep the main thread for the UI.

Treat every `await` as a place where the world may change. Do read-modify-write steps
synchronously after the last `await`, or funnel them through one promise chain. The bug
in example 3 is the in-memory twin of a lost update in SQL, where the fix is
`SELECT ... FOR UPDATE` or a single `UPDATE ... SET balance = balance - 30`.

Go chose preemptive goroutines on all cores plus explicit synchronization; JavaScript chose
one thread per page and no locks. Neither is free: Go pays with data races to guard
against, JavaScript with a UI that any long loop can freeze.
