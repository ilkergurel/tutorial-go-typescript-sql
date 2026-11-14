---
id: ts.async.cancellation
title: "Cancellation with AbortController and AbortSignal.timeout; async iterators; debounce"
track: ts
module: async
order: 4
level: intermediate
minutes: 14
since: es2018
prerequisites: [ts.async.combinators]
compare_with: [go.concurrency.context, go.iterators.writing, ts.services.fetch, ts.react.effects-hooks]
tags: [async, abortcontroller, abortsignal, async-iterators, debounce, cancellation]
summary: >
  A Promise can't be cancelled, so the platform passes an AbortSignal into the
  work instead, much like Go's context. Pull-based async iterators and
  debouncing stop or skip work at the source.
examples:
  - id: abort-signal
    file: examples/01_abort_signal.ts
    title: "abort(), AbortSignal.timeout() and AbortSignal.any()"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "9-21"
        note: >
          What fetch does with a signal: on abort, stop the work and reject with
          signal.reason. Cancellation is cooperative, as with ctx.Done() in Go: a
          function that ignores its signal can't be cancelled.
      - lines: "10"
        note: >
          throwIfAborted() throws the reason if the signal has already fired.
          Check before starting expensive work, as Go code checks ctx.Err().
      - lines: "15-19"
        note: >
          Remove the listener when the work completes. A long-lived signal, one
          per page for example, would otherwise collect a listener per request.
      - lines: "26-31"
        note: >
          e is unknown. The abort reasons the platform creates are DOMExceptions,
          and their name says why; a reason you pass yourself arrives as is.
      - lines: "34-38"
        note: >
          The controller cancels and the signal observes, the same split as Go's
          context.WithCancel returning ctx and cancel. abort() without an
          argument makes the reason a DOMException named AbortError.
      - lines: "40-41"
        note: >
          AbortSignal.timeout(ms) is context.WithTimeout. Its reason is a
          TimeoutError, so callers can tell "too slow" from "cancelled".
      - lines: "43-48"
        note: >
          any() aborts when the first of its signals does, with that signal's
          reason. An aborted signal stays aborted: later calls fail at once, so
          new work needs a new controller.
  - id: async-iterators
    file: examples/02_async_iterators.ts
    title: "Async generators: break stops the producer, a signal stops it from outside"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "15-28"
        note: >
          An async generator is pull-based: the next page is fetched only when
          the consumer asks for more items. Work nobody asks for never starts.
      - lines: "19"
        note: >
          The cancellation check sits between pages, so cancellation takes effect
          at the next check: C was already fetched and still arrives after the
          abort at B.
      - lines: "25-27"
        note: >
          finally runs however the loop ends: completion, break or a thrown
          error. This is where a real generator releases its reader, cursor or
          connection.
      - lines: "31-36"
        note: >
          break makes for await call the generator's return(), which runs its
          finally: page 2 is never fetched. Go 1.23 iterators work the same way:
          yield returns false and the iterator must stop.
      - lines: "39-50"
        note: >
          Cancellation from outside: the generator throws AbortError at its next
          check, runs finally, and the error surfaces at the for await loop.
  - id: debounce-latest
    file: examples/03_debounce_latest.ts
    title: "Search as you type: abort the previous request, debounce the input"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "8-9"
        note: >
          Both sides are lowercased with the Turkish rules (İ to i, I to ı), so
          "izm" finds İzmir. Plain toLowerCase() turns İ into i plus a combining
          dot, and the prefix never matches.
      - lines: "22-35"
        note: >
          Latest wins: each search aborts the one before. Without this, a slow
          answer for "a" could arrive after the answer for "ant" and overwrite
          the newer results.
      - lines: "31-33"
        note: >
          Tell your own abort from a real failure by asking the signal, not by
          parsing the error. Real failures still propagate.
      - lines: "37-43"
        note: >
          Debounce: each call clears the pending timer, so only a call followed
          by ms of quiet runs. There is no built-in; this is the whole thing.
      - lines: "45-50"
        note: >
          Without debounce, three keystrokes start three requests and two are
          aborted. With it, the same three keystrokes start one.
use_cases:
  - "Leaving a page or closing a dialog: abort its requests so late responses don't update UI that is gone"
  - "Search as you type: debounce the input and abort the previous request"
  - "A deadline on any async call with AbortSignal.timeout, merged with a user's cancel through AbortSignal.any"
  - "Paginated APIs and streams consumed with for await and stopped early with break"
  - "Removing many event listeners at once: addEventListener(type, fn, { signal })"
strengths:
  - "One standard cancellation token for fetch, event listeners, streams and your own functions, as context.Context is in Go, instead of a cancel() method per library"
  - "The reason says why: AbortError for abort(), TimeoutError for AbortSignal.timeout(), or any value you pass to abort(reason)"
  - "Async generators stop when the consumer stops: break runs the producer's finally, so cleanup sits next to the code that acquired the resource"
weaknesses:
  - "Cooperative only: there is no promise.cancel(), and code that ignores its signal keeps running"
  - "Nothing passes the signal for you: every function in the chain needs a signal parameter, the same discipline as ctx in Go"
  - "Debounce and throttle are not built in, and a hand-written debounce needs a cancel path for when the UI goes away"
  - "AbortSignal.any() is recent: available in all major browsers only since 2024"
pitfalls:
  - title: "Reporting an abort as an error"
    detail: "An aborted request is the expected outcome of cancelling. Check the signal before showing an error message."
    bad: "catch (e) { showError(e); }"
    good: "catch (e) { if (signal.aborted) return; showError(e); }"
  - title: "Reusing an aborted controller"
    detail: "A signal is one-shot: once aborted it stays aborted, and every API given it rejects at once. Create a new AbortController per operation."
  - title: "Rejecting without stopping the work"
    detail: "An abort handler that only rejects leaves the timer, stream or socket running. Stop the work itself (clearTimeout, reader.cancel(), close) and remove the listener when the work completes."
  - title: "Debounced calls that outlive their component"
    detail: "The pending timer fires after the user navigated away. Keep a way to clear it, and call it from the effect cleanup together with abort()."
quiz:
  - q: "fetch(url, { signal: AbortSignal.timeout(3000) }) takes too long. What is err.name in the catch block?"
    options: ["AbortError", "TimeoutError", "TypeError"]
    answer: 1
    why: "AbortSignal.timeout aborts with a TimeoutError DOMException, and fetch rejects with the signal's reason."
  - q: "A for await loop over an async generator executes break. What happens to the generator?"
    options: ["It keeps fetching in the background", "Its return() is called: its finally block runs and it stops", "It receives an AbortError"]
    answer: 1
    why: "Leaving a for await loop early calls the iterator's return(), which resumes the generator at a return and runs its finally."
  - q: "The user types i, iz, izm within 50 ms, into a box debounced at 50 ms. How many searches start?"
    options: ["3", "1", "0"]
    answer: 1
    why: "Each call restarts the timer; only the last call is followed by 50 ms of quiet."
  - q: "Search input is debounced. Why abort the previous request as well?"
    options: ["Debounce already cancels requests in flight", "A request already sent can answer after a newer one and overwrite fresher results", "Browsers allow only one fetch at a time"]
    answer: 1
    why: "Debounce only decides when to start. Once two requests overlap, only aborting the old one guarantees the newest result is the one shown."
further_reading:
  - title: "MDN: AbortController"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/AbortController"
  - title: "MDN: AbortSignal.timeout()"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/AbortSignal/timeout_static"
  - title: "MDN: AbortSignal.any()"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/AbortSignal/any_static"
  - title: "MDN: for await...of"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Statements/for-await...of"
  - title: "package context (Go, for comparison)"
    url: "https://pkg.go.dev/context"
---

## Concept

A Promise has no `cancel()`. The platform solves cancellation with a separate pair of
objects: an `AbortController`, which you keep, and its `AbortSignal`, which you pass into
the work. `controller.abort(reason)` marks the signal aborted, fires its `abort` event, and
every API holding the signal stops and rejects with `signal.reason`: `fetch`, streams,
`addEventListener` (the listener is removed), and your own functions if they listen.

Two factories cover the common cases: `AbortSignal.timeout(ms)` aborts itself with a
`TimeoutError`, and `AbortSignal.any([a, b])` aborts when the first of its inputs does.

Stopping work also happens at the source. An async generator produces items only when
the consumer asks, so breaking out of `for await` stops it, and its `finally` cleans up.
Debouncing delays work until the input stops changing, and "latest wins" aborts the
previous request when a new one starts.

## Coming from C/C++/Java/Python/JS

- **Go**: `AbortController` plus `AbortSignal` is `cancel` plus `ctx` from
  `context.WithCancel`; `signal.aborted` is `ctx.Err() != nil`; the `abort` event is
  `<-ctx.Done()`; `signal.reason` is `context.Cause`; `AbortSignal.timeout` is
  `context.WithTimeout`. Signals carry no values and form no tree on their own:
  `AbortSignal.any` is how you derive one from several.
- **Java**: `Future.cancel(true)` interrupts a thread. There is no thread to interrupt here,
  so cancellation is a flag plus an event that the work must observe, like a hand-rolled
  `volatile boolean cancelled`.
- **Python**: `Task.cancel()` raises `CancelledError` inside the task at its next `await`,
  with no cooperation needed. JavaScript can't inject an exception into a running async
  function; the function has to check its signal.
- **JS**: `async function*` with `for await...of` (ES2018) is the async twin of generators
  and `for...of`. Leaving the loop early calls the iterator's `return()`, the hook that
  lets a producer clean up.

## Design notes

Give every async function that does I/O an optional `signal` and pass it down, for example
`load(id: string, { signal }: { signal?: AbortSignal } = {})`. It is the same discipline as
`ctx context.Context` as the first parameter in Go: tedious, explicit, and the only way
cancellation reaches the bottom of the stack.

Decide in one place what an abort means. Usually it means "nobody wants this result
anymore", which is not an error: don't log it, don't show it, don't retry it. A timeout,
on the other hand, often is an error worth reporting, which is why the two reasons have
different names.

For UI input, combine the tools: debounce decides *when* to start, aborting the previous
request decides *which* answer may render, and the component's cleanup aborts whatever is
still running when it goes away. The React effects lesson builds on exactly this.
