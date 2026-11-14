---
id: go.concurrency.patterns
title: "Patterns: worker pool, fan-out/fan-in, pipelines, semaphores, rate limiting"
track: go
module: concurrency
order: 5
level: advanced
minutes: 14
since: go1.25
prerequisites: [go.concurrency.context, go.concurrency.sync]
compare_with: [ts.async.combinators, sql.transactions.skip-locked, design.paradigms.concurrency]
tags: [concurrency, pipelines, worker-pool, fan-out, fan-in, semaphore, rate-limiting]
summary: >
  Most concurrent Go programs combine a few shapes: pipelines of stages joined by
  channels, fan-out to workers and fan-in of their results, a bound on how many
  things run at once, and a limit on how often they start.
examples:
  - id: pipeline
    file: examples/01_pipeline.go
    title: A pipeline with fan-out, fan-in and a cancellation path
    runtime: go
    check: exact
    annotations:
      - lines: "12-25"
        note: >
          A stage returns a receive-only channel and starts the goroutine that
          fills it. That goroutine owns the channel, so it closes it, and
          downstream range loops end cleanly.
      - lines: "17-21"
        note: >
          Every send also selects on ctx.Done(). If the consumer stops
          reading, a plain send would block forever and leak this goroutine;
          with the select, one cancel() releases it.
      - lines: "32-42"
        note: >
          Fan-out: three workers range over the same channel and each value
          goes to exactly one of them. A slow stage gets parallel without any
          change to the stages around it.
      - lines: "43-45"
        note: >
          Fan-in needs one extra goroutine that waits for all workers and then
          closes out. Closing inside a worker would make the others panic on
          their next send; never closing would leave main's range loop waiting
          forever.
      - lines: "50-51"
        note: >
          If main returned early, say on the first bad value, the deferred
          cancel would unblock every stage still trying to send. Without it,
          those goroutines would leak.
      - lines: "57"
        note: >
          Workers finish in any order. When order matters, carry an index with
          each value or write results into a slice by index, as the next
          example does.
  - id: bounded
    file: examples/02_bounded.go
    title: "Bounding concurrency: worker pool vs semaphore"
    runtime: go
    check: exact
    annotations:
      - lines: "15-23"
        note: >
          The tracker only measures concurrency. time.Sleep stands in for I/O so
          that tasks overlap; nothing here depends on it for ordering.
      - lines: "26"
        note: >
          Each index is written by exactly one goroutine, and wg.Wait orders
          those writes before the Println, so the slice needs no lock and the
          results keep input order.
      - lines: "29-43"
        note: >
          Worker pool: a fixed set of goroutines and one job channel; the
          concurrency is the number of workers. It suits workers that hold
          something costly, such as a connection or a large buffer, and reuse
          it across jobs.
      - lines: "47-56"
        note: >
          Semaphore: one goroutine per job and a channel of capacity 3 as the
          gate. The send acquires a slot and blocks while three are taken; the
          deferred receive releases it. All ten goroutines exist at once, but
          only three work at a time.
  - id: rate-limit
    file: examples/03_rate_limit.go
    title: "Rate limiting: a steady ticker and a token bucket"
    runtime: go
    check: exact
    annotations:
      - lines: "14-21"
        note: >
          A loop that waits for a tick before each request can't go faster than
          the ticker. Only lower bounds on elapsed time are reliable, so that
          is all the example prints.
      - lines: "24-27"
        note: >
          The bucket is a buffered channel of tokens. Starting it full lets a
          burst of three requests through without waiting.
      - lines: "31-45"
        note: >
          The refill goroutine adds one token per tick and drops it when the
          bucket is full, so idle time can't bank more than three requests.
          The done channel stops it; without that, every limiter you create
          leaks a goroutine and a ticker.
      - lines: "47-51"
        note: >
          The first three requests take the burst; the next two wait for
          refills. In production, golang.org/x/time/rate's Limiter.Wait(ctx)
          does this and lets a waiting request be canceled.
use_cases:
  - "Pipeline: parse, enrich, write, e.g. read a CSV export, look up each row, then batch inserts into PostgreSQL"
  - "Worker pool: a fixed number of workers, each holding a database connection, processing jobs claimed with FOR UPDATE SKIP LOCKED"
  - "Semaphore: cap concurrent calls to an external API, or concurrent image resizes at the number of cores"
  - "Rate limiting: stay under a third-party quota such as 50 requests per second"
strengths:
  - "Built from language parts (goroutines, channels, select), with no executor framework to configure, unlike Java's ThreadPoolExecutor and its queue and rejection policies"
  - "Unbuffered channels give backpressure for free: a fast stage waits for a slow one, where Java's Executors.newFixedThreadPool queues work in an unbounded LinkedBlockingQueue"
  - "A buffered channel is a semaphore with no extra type, and goroutines are cheap enough that one per item behind a semaphore is a sound design"
weaknesses:
  - "Every pattern needs a cancellation path and a clear owner for every close. Getting either wrong deadlocks or leaks at run time; the compiler doesn't notice"
  - "Fan-out loses order; restoring it means carrying indexes or sequence numbers"
  - "No rate limiter in the standard library: bursts, per-key limits and Wait(ctx) come from golang.org/x/time/rate or your own code"
  - "Each channel hop is a synchronisation. When the per-item work is tiny, a plain loop can beat a pipeline; measure before splitting cheap work into stages"
pitfalls:
  - title: "Closing a shared output channel in a worker"
    detail: "The first worker to finish closes it and the others panic with \"send on closed channel\". Close it from one goroutine after wg.Wait()."
    bad: "wg.Go(func() { for n := range in { out <- n * n }; close(out) })"
    good: "go func() { wg.Wait(); close(out) }()"
  - title: "One goroutine per item with no bound"
    detail: "Ten thousand URLs become ten thousand simultaneous connections, file descriptors and memory spikes. Bound it with a pool, a semaphore or errgroup.SetLimit (lesson 6)."
    bad: "for _, u := range urls { go fetch(u) }"
    good: "g.SetLimit(8)\nfor _, u := range urls { g.Go(func() error { return fetch(ctx, u) }) }"
  - title: "Stages without a cancellation path"
    detail: "When the consumer returns early, every upstream goroutine blocks on its next send forever. Select on ctx.Done() for each send and defer cancel() in the consumer."
  - title: "Rate limiting with time.Sleep"
    detail: "Sleeping after each request makes the rate drift by the request's own duration, and nothing can cancel the sleep. Use a ticker, or rate.Limiter.Wait(ctx)."
quiz:
  - q: "In a fan-in, who should close the merged output channel?"
    options: ["Each worker, when it finishes", "The consumer", "One goroutine that waits for all workers, then closes it"]
    answer: 2
    why: "Only after every sender is done is it safe to close; a closing worker would make the others panic on send."
  - q: "A pipeline's consumer returns after the first bad value. What keeps the upstream goroutines from leaking?"
    options: ["The garbage collector", "Each send also selects on ctx.Done(), and the consumer cancels", "Closing the first stage's input"]
    answer: 1
    why: "Blocked goroutines are never collected. They exit only if something unblocks them, and ctx.Done() is that something."
  - q: "slots := make(chan struct{}, 3) is used as a semaphore. What does slots <- struct{}{} do when three tokens are already in it?"
    options: ["Blocks until another goroutine receives one", "Drops the token", "Panics"]
    answer: 0
    why: "A send on a full buffered channel blocks; receiving a token is the release."
further_reading:
  - title: "Go Concurrency Patterns: Pipelines and cancellation"
    url: "https://go.dev/blog/pipelines"
  - title: "Advanced Go Concurrency Patterns"
    url: "https://go.dev/blog/io2013-talk-concurrency"
  - title: "package time: Ticker"
    url: "https://pkg.go.dev/time#Ticker"
  - title: "package golang.org/x/time/rate"
    url: "https://pkg.go.dev/golang.org/x/time/rate"
---

## Concept

Four shapes cover most concurrent Go code:

- **Pipeline**: stages connected by channels. Each stage is a goroutine that receives
  from its input, does one thing, sends to its output, and closes the output when done.
- **Fan-out / fan-in**: several goroutines read the same channel (fan-out); their
  outputs are merged back into one channel (fan-in).
- **Bounded concurrency**: a **worker pool** (N goroutines pulling from a job channel) or
  a **semaphore** (one goroutine per job, a buffered channel admitting N at a time).
- **Rate limiting**: start work no faster than a fixed rate, with a ticker or a token
  bucket.

The rules that make them safe are the same everywhere: the goroutine that sends on a
channel owns it and closes it, every blocking send also watches `ctx.Done()`, and the
consumer cancels when it stops early.

## Coming from C/C++/Java/Python/JS

- **C / Unix**: a pipeline is `grep | sort | uniq` inside one process, with channels as
  the pipes and blocking sends as their backpressure.
- **Java**: a worker pool is `Executors.newFixedThreadPool(n)` without the executor:
  the job channel is the queue and `close` is `shutdown()`. A semaphore is
  `java.util.concurrent.Semaphore`.
- **Python**: `ThreadPoolExecutor(max_workers=n)` ≈ worker pool; `asyncio.Semaphore` ≈
  the channel semaphore. Goroutines run in parallel, so CPU-bound stages scale across
  cores.
- **JS**: bounding concurrency is what `p-limit` or chunked `Promise.all` do; Node's
  `stream.pipeline` is the pipeline. There, everything shares one thread; here, each
  stage can run on its own core.

## How it works

The pipeline example has this shape:

```
        ┌─► worker 1 ─┐
gen ────┼─► worker 2 ─┼──► out ──► main
        └─► worker 3 ─┘
```

Each value from `gen` goes to exactly one worker, because a receive takes the value off
the channel. The merged `out` must be closed exactly once, after all three workers have
stopped sending, which is why a fourth small goroutine waits on the WaitGroup and then
closes it.

A worker pool and a semaphore bound concurrency the same way, from different ends: the
pool creates N goroutines and feeds them jobs; the semaphore creates one goroutine per
job and lets N through a gate. A third option is `errgroup.Group.SetLimit` (next lesson),
which adds error handling. `golang.org/x/sync/semaphore` offers a weighted semaphore
whose `Acquire(ctx, n)` can be canceled.

## Design notes

Start with the simplest thing: a loop. Add a pipeline when stages have different costs
and you want them to overlap, such as reading the next CSV rows while the database
insert of the previous batch is in flight. Add fan-out only to the slow stage.

Prefer a semaphore or `errgroup.SetLimit` when jobs are independent and short-lived;
prefer a worker pool when workers hold a resource (a connection, a buffer, a
prepared statement) worth reusing.

Make every stage cancelable, and make ownership obvious from the signature: return
`<-chan T` from the function that creates and closes the channel, and accept `<-chan T`
in stages that only read.
