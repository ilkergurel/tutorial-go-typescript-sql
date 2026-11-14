---
id: ts.async.combinators
title: "Promise.all / allSettled / race / any; concurrency limits"
track: ts
module: async
order: 3
level: intermediate
minutes: 13
since: es2021
prerequisites: [ts.async.promises, ts.narrowing.discriminated-unions]
compare_with: [go.concurrency.errgroup, go.concurrency.patterns, go.concurrency.select]
tags: [async, promises, promise-all, allsettled, concurrency-limit]
summary: >
  The four combinators differ in one thing: what counts as done and what counts as
  failure. Pick by failure semantics, remember that none of them cancels anything,
  and bound concurrency yourself when there are many items.
examples:
  - id: four-combinators
    file: examples/01_four_combinators.ts
    title: "Same three calls, four combinators"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-11"
        note: >
          A fake call with fixed timings so the order is reproducible: orders
          fails at 10 ms, prefs succeeds at 30 ms, profile at 50 ms.
      - lines: "22-27"
        note: >
          Promises are eager: the three calls start here, once. A combinator only
          subscribes to promises, so handing the same three to four combinators
          repeats no work.
      - lines: "30"
        note: >
          all rejects with the first rejection, at 10 ms, without waiting for the
          rest and without stopping them: prefs and profile still finish (the
          indented lines after it).
      - lines: "31"
        note: >
          race settles like the first input to settle, success or failure. Here
          that is a failure, which is why race is mostly used for deadlines.
      - lines: "32"
        note: >
          any takes the first success and ignores failures until every input
          has failed.
      - lines: "33-40"
        note: >
          allSettled never rejects. It waits for everything and gives one
          {status, value} or {status, reason} per input, in input order.
      - lines: "43-50"
        note: >
          When every input fails, any rejects with an AggregateError whose errors
          are in input order: mirror-1 comes first although mirror-2 failed first.
  - id: typed-results
    file: examples/02_typed_results.ts
    title: "Tuple types, narrowing settled results, and a race deadline"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "21-23"
        note: >
          Given an array literal, Promise.all is typed as a tuple: user is User
          and orders is Order[], no casts. With Go's errgroup you would declare
          the result variables and assign them inside each goroutine.
      - lines: "25-27"
        note: >
          Each settled result is a discriminated union on status. Checking
          "fulfilled" makes value available; "rejected" makes reason available.
      - lines: "28-32"
        note: >
          reason is typed any in TypeScript's own lib files, so nothing forces a
          check. Narrow it like a catch variable before reading .message.
      - lines: "36-42"
        note: >
          race against a timer gives a deadline but cancels nothing. Clear the
          timer in finally, or it outlives the call (in Node it would keep the
          process alive).
      - lines: "44-54"
        note: >
          The slow call reports after the timeout: losing a race does not stop
          the work. Stopping it needs an AbortSignal, the next lesson's topic.
  - id: concurrency-limit
    file: examples/03_concurrency_limit.ts
    title: "A concurrency limit: N workers, one shared index"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "8-15"
        note: >
          Each worker claims the next index with next++. That runs synchronously,
          so two workers can never take the same item, without the atomic counter
          Go would need. Results are stored by index, so their order matches the
          input whatever the finish order.
      - lines: "16-17"
        note: >
          limit workers, joined with Promise.all: Go's errgroup with SetLimit(2),
          or a buffered channel as a semaphore. Unlike errgroup.WithContext, a
          failure here does not stop the other workers.
      - lines: "21-30"
        note: >
          A plain counter is enough to measure how many uploads are in flight:
          there is one thread, so no increment is lost.
      - lines: "37-39"
        note: >
          Promise.all over map starts everything at once: 5 here, 5,000 in
          production. That is how a page floods an API or trips a rate limit.
use_cases:
  - "Promise.all: independent requests that are all required, such as a user and their permissions before rendering"
  - "allSettled: dashboards and batch jobs where partial results are useful and failures are reported per item"
  - "any: redundant sources (mirrors, fallback endpoints) where one answer is enough"
  - "race: deadlines and first-response-wins; for timeouts, AbortSignal.timeout is now the better tool"
  - "A concurrency limit: uploads, crawls, imports, anything with many items and a server on the other end"
strengths:
  - "Combinators are ordinary functions over promises: no executor or thread pool to configure, unlike Java's CompletableFuture.allOf with an Executor"
  - "Types follow the data: Promise.all on a tuple keeps each position's type, and allSettled results narrow on status"
  - "A concurrency limit takes about ten lines and no locks, because next++ cannot race on one thread"
weaknesses:
  - "No combinator cancels the losers: after Promise.all rejects, the other requests keep running, whereas Go's errgroup.WithContext cancels a shared context"
  - "Failure values are untyped: allSettled's reason and AggregateError.errors are any"
  - "No built-in concurrency limit: Promise.all starts everything at once, so you write a pool or pull in a library"
pitfalls:
  - title: "Promise.all over thousands of items"
    detail: "Every call starts immediately. Bound the concurrency."
    bad: "await Promise.all(ids.map(fetchOne)); // 10,000 requests at once"
    good: "await mapLimit(ids, 6, fetchOne);"
  - title: "Expecting Promise.all to stop the other calls"
    detail: "It stops waiting, not working. Pass one AbortSignal to every call and abort it when the first one fails."
  - title: "Using race for 'first success'"
    detail: "race settles with the first result, and a fast failure wins. any is the combinator for 'first success'."
  - title: "Using all when partial results are acceptable"
    detail: "One failed widget then blanks the whole dashboard. Use allSettled and render what succeeded."
quiz:
  - q: "Three mirrors serve the same file; you want the first successful download. Which combinator?"
    options: ["Promise.all", "Promise.race", "Promise.any", "Promise.allSettled"]
    answer: 2
    why: "any fulfills with the first success and only rejects when every mirror failed. race could settle with a fast failure."
  - q: "Promise.all([a, b, c]) rejected because b failed. What happens to a and c?"
    options: ["They are cancelled", "They keep running; their results are ignored", "They are retried"]
    answer: 1
    why: "Promises have no cancellation. The combinator stops waiting; the work goes on unless you abort it with a signal."
  - q: "What is the type of r in: const [r] = await Promise.allSettled([getUser()])?"
    options: ["User", "User | Error", "PromiseSettledResult<User>"]
    answer: 2
    why: "A union of { status: \"fulfilled\"; value: User } and { status: \"rejected\"; reason: any }; narrow on status."
  - q: "In mapLimit, why is const i = next++ safe without a lock when several workers run?"
    options: ["JavaScript numbers are atomic", "The read and the increment run in one synchronous step, and nothing interrupts it", "Promise.all serializes the workers"]
    answer: 1
    why: "Run-to-completion: other workers only run at await points, and there is none between the read and the write."
further_reading:
  - title: "MDN: Promise concurrency"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise#promise_concurrency"
  - title: "MDN: Promise.allSettled()"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise/allSettled"
  - title: "MDN: Promise.any() and AggregateError"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Promise/any"
  - title: "errgroup (Go, for comparison)"
    url: "https://pkg.go.dev/golang.org/x/sync/errgroup"
---

## Concept

A Promise combinator takes several promises and returns one. The four built-ins differ only
in **when they settle** and **what counts as failure**:

- `Promise.all`: fulfills with every value, in input order, or rejects with the **first**
  rejection.
- `Promise.allSettled` (ES2020): always fulfills, after **every** input settled, with one
  `{ status, value | reason }` per input.
- `Promise.any` (ES2021): fulfills with the **first success**; rejects with an
  `AggregateError` only when every input failed.
- `Promise.race`: settles like the **first input to settle**, success or failure.

None of them starts or stops work. The promises were running before the combinator saw
them and keep running after it settled. And none of them limits how many run at once:
that part you write yourself.

## Coming from C/C++/Java/Python/JS

- **Go**: `errgroup.Group.Wait` is `Promise.all` (first error wins), but `errgroup.WithContext`
  also cancels the others through the context; JavaScript needs an explicit `AbortSignal`
  for that. `SetLimit(n)` is the concurrency limit that `Promise.all` lacks. `select` over
  several channels is the nearest thing to `race`.
- **Java**: `CompletableFuture.allOf` and `anyOf`. Note that `anyOf` completes with the first
  future to complete, failure included, so it is `race`, not `any`.
- **Python**: `asyncio.gather` is `Promise.all`, and `gather(..., return_exceptions=True)`
  is close to `allSettled`. `asyncio.wait(..., return_when=FIRST_COMPLETED)` is a race. A
  `Semaphore` is the usual concurrency limit.

## Design notes

Choose by the failure semantics the *caller* needs: "all or nothing" is `all`, "whatever
worked" is `allSettled`, "any one answer" is `any`. `race` is a building block for
deadlines, and `AbortSignal.timeout` (next lesson) does that job better because it can
stop the work too.

For many items, the worker pool in example 3 is the standard shape: a fixed number of
async loops pulling from a shared cursor, which is the same design as a Go worker pool
reading from a channel. Make it stop on failure by sharing one `AbortController`: abort it
in the first failing worker, pass its signal into every call, and check it before taking
the next item.
