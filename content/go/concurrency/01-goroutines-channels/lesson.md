---
id: go.concurrency.goroutines-channels
title: "Goroutines and channels: unbuffered, close, ranging"
track: go
module: concurrency
order: 1
level: intermediate
minutes: 12
since: go1.0
prerequisites: [go.basics.functions, go.composite.slices-and-maps]
compare_with: [ts.async.event-loop, design.paradigms.concurrency]
tags: [concurrency, goroutines, channels, csp]
summary: >
  A goroutine is a concurrently running function; a channel passes values
  between goroutines. An unbuffered channel is a handoff: both sides wait
  until the value changes hands.
examples:
  - id: worker
    file: examples/01_worker.go
    title: Three workers receiving from one channel
    runtime: go
    check: sorted-lines
    nondeterminism_note: >
      Which worker takes which job is the scheduler's decision and varies run to
      run. Sorted-lines comparison makes the check about the *set* of events:
      every job 1-6 handed to some worker, each exactly once. Output is also
      post-sorted in the example itself for a fully stable lesson view.
    annotations:
      - lines: "25-28"
        note: >
          Launching three goroutines is just `go f()`. Each runs the same loop;
          the runtime schedules them on OS threads for you.
      - lines: "30-32"
        note: >
          An unbuffered channel send blocks until a receiver takes the value:
          the sender knows the work was handed off, unlike a queue drop.
      - lines: "33"
        note: >
          Close signals "no more values". The range loop ends when the channel
          is closed AND drained. Only the sender should close: a send on a
          closed channel panics, and so does closing it twice.
      - lines: "35"
        note: >
          wg.Wait() blocks until the counter hits zero. The counter was
          incremented before `go` ran the goroutine — decrementing happens in
          the goroutine. Workers share the map only to collect results, under a
          mutex; the channel itself passed each job to exactly one worker.
use_cases:
  - "Fan-out work across workers while results stream back through a channel"
  - "Signaling completion (close a done channel) without locks"
strengths:
  - "Goroutines are cheap (KB-scale stacks): thousands are normal, unlike Java's thread-per-request"
  - "Channels pass ownership of data between goroutines: no lock to forget"
  - "The compiler checks send/receive direction in typed channels"
weaknesses:
  - "Wrong ownership (who closes, who reads) produces deadlocks or panics at runtime, not compile time"
  - "A nil channel blocks forever; a send on a closed channel panics"
  - "Unbuffered channels couple sender to receiver timing; sometimes you want a buffer"
pitfalls:
  - title: "Closing a channel from the receiver side"
    detail: "The sender then panics on send. Close belongs to the sender: it means 'I will send no more'."
  - title: "Forgetting wg.Wait() or leaking goroutines"
    detail: "main returning kills all goroutines instantly, half-finished. Use WaitGroup or range over the channel."
  - title: "Sharing a variable across goroutines instead of sending it"
    detail: "Concurrent map writes and read/write races are real. Race detector: go test -race."
quiz:
  - q: "What does a receive from a closed channel yield after it is drained?"
    options: ["It blocks forever", "The zero value, immediately", "A panic"]
    answer: 1
    why: "Closed + drained channels return zero values with ok=false; that's how range terminates."
  - q: "Who should close a channel?"
    options: ["The receiver", "The sender", "Either, it is symmetric"]
    answer: 1
    why: "Close means 'no more sends coming', which only the sender can know."
further_reading:
  - title: "Go Concurrency Patterns (Pike)"
    url: "https://go.dev/talks/2012/concurrency.slide"
  - title: "Effective Go: channels"
    url: "https://go.dev/doc/effective_go#channels"
  - title: "package sync"
    url: "https://pkg.go.dev/sync"
---

## Concept

`go f()` starts a goroutine: a function running concurrently with the caller. Channels are
typed conduits: `ch <- v` sends, `v := <-ch` receives. An unbuffered channel makes both
sides wait for the handoff; a closed channel signals "no more values", and `for v := range ch`
ends when the channel closes.

## Coming from C/C++/Java/Python/JS

- Java: goroutines ≈ virtual threads; channels ≈ BlockingQueue, but first-class in the
  language with select (later lesson) and no checked InterruptedException.
- C/C++: no mutex to forget, no data race through the channel — the value itself moves.
- JS: like posting messages to a Web Worker, but with shared memory processes cheaply and
  many goroutines instead of two isolated heaps.

## Design notes

Share memory by communicating: send the data, don't guard it with a mutex when ownership
actually transfers. A useful rule this lesson plants: whoever *sends* owns the channel and
closes it; receivers only read. When several senders exist, wrap close in a sync.Once or
collect via WaitGroup first — covered in the sync lesson.
