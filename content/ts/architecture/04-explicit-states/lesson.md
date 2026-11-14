---
id: ts.architecture.explicit-states
title: "Loading and error states as explicit types; functional core, imperative shell"
track: ts
module: architecture
order: 4
level: advanced
minutes: 15
since: ts2.8
prerequisites: [ts.narrowing.discriminated-unions, ts.react.state, ts.type-level.utility]
compare_with: [ts.react.query, ts.errors.result, go.concurrency.select, design.architecture.business-rules]
tags: [discriminated-unions, async, state-machine, functional-core, effects, testing]
summary: >
  Give every async value one union type (idle, loading, failed, ready, with stale data where
  it applies) and render it exhaustively. Put the decisions in a pure core that returns the
  next state plus effects as data, and keep I/O in a thin shell that only executes them.
examples:
  - id: remote-data
    file: examples/01_remote_data.tsx
    title: "Remote<T>: one generic union, stale data while reloading, exhaustive rendering"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "5-9"
        note: >
          Four states, and each carries only what exists in it: no data while
          idle, an error only when failed. stale is the piece flags usually lose:
          a refresh that fails should keep showing the last good value, marked as
          old, not blank the screen.
      - lines: "12-14"
        note: >
          A mapped type over the status union with Extract gives each handler the
          narrowed variant as its parameter. Adding a fifth status to Remote makes
          every match call without a handler for it a compile error.
      - lines: "15-20"
        note: >
          TypeScript cannot see that handlers[state.status] and state belong to
          the same variant (the "correlated union" limitation), so the call needs
          one cast. Keeping it inside a tiny generic helper means callers never
          cast.
      - lines: "22-33"
        note: >
          Each branch renders its own state with its own markup: aria-busy while
          loading, role="alert" for errors. With flags, these would be nested
          conditionals that can disagree.
      - lines: "38-50"
        note: >
          The timeline is the life of a dashboard widget. Rendering each state in
          isolation is the cheapest UI test there is: no fetch, no timers, every
          state reachable directly.
  - id: functional-core
    file: examples/02_functional_core.ts
    title: "A functional core: (state, event) in, (state, effects) out, tested by replaying events"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "16-17"
        note: >
          Effects are data. The core says "fetch attempt 2" or "sleep 100 ms";
          it never calls fetch or setTimeout, which is what keeps it pure.
      - lines: "25-42"
        note: >
          Every rule lives here: when to retry, how long to wait, when to give up,
          which replies are stale. The declared return type doubles as the
          exhaustiveness check: an Event variant without a case makes the end of
          the function reachable and tsc reports TS2366.
      - lines: "29-33"
        note: >
          A reply is accepted only from the attempt the core is waiting for. The
          race between a slow first request and its retry is settled by
          comparing two numbers, not by aborting in the right order.
      - lines: "44-59"
        note: >
          The test is a list of events and a loop. The late reply from attempt 1
          and the second open are both ignored; returning the same state object
          makes "nothing happened" checkable with ===.
  - id: imperative-shell
    file: examples/03_imperative_shell.ts
    title: "The imperative shell: perform effects, turn results into events"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "16-25"
        note: >
          The same transitions as the previous example, written as early returns.
          Narrowing works through them: after the open and timer checks, e is
          loaded or rejected, so e.attempt is known to exist.
      - lines: "28-34"
        note: >
          The view is a pure function of state too. The final branch compiles only
          because every other status was handled above: s is narrowed to ready.
      - lines: "38-54"
        note: >
          The shell has no if about business rules. It runs step, shows the
          result, and turns each effect into real I/O whose outcome comes back as
          an event. In React this is a reducer plus an effect runner, or an
          external store read with useSyncExternalStore.
      - lines: "56-61"
        note: >
          The fake server fails once. The shell output matches what the core's
          replay predicted, with a real 100 ms timer in between.
use_cases:
  - "Every widget that loads data: dashboards, lists, detail pages; one Remote<T> type and one rendering helper for all of them"
  - "Background refresh: keep showing stale data, marked as such, while a refetch runs or after it fails"
  - "Retry and backoff policies, request deduplication and stale-response handling that you want to unit-test without timers"
  - "Multi-step flows (upload, then process, then poll) where the order of events is the hard part"
strengths:
  - "Unlike isLoading/error/data flags, a union cannot express 'loading and failed at once', and each branch can only read fields that exist in it"
  - "A pure core is tested by replaying events: no fake timers, no fetch mocks, deterministic in any test runner"
  - "Race conditions become ordinary data comparisons (is this reply from the attempt I'm waiting for?) instead of cancellation choreography"
  - "The shell is small and generic, so the code that touches the outside world, the code most likely to need integration tests, is the least of it"
weaknesses:
  - "More types up front: state, event and effect unions plus a step function, where a useEffect with three useState calls looks shorter on day one"
  - "Effects as data add an interpreter (the shell) and an indirection when reading code: 'fetch' in the core is a request, not a call"
  - "TypeScript cannot relate a handler table to the variant it handles without one cast in the helper"
  - "For plain server data, TanStack Query already implements this state machine (status, fetchStatus, retries); hand-rolling it is for flows the library doesn't cover"
pitfalls:
  - title: "Three flags for one async value"
    detail: "isLoading, error and data as separate state allow combinations the UI cannot render sensibly, and a refetch that sets isLoading usually hides data that is still valid. Use one union, with stale data where it applies."
    bad: "const [isLoading, setLoading] = useState(false);\nconst [error, setError] = useState<string>();\nconst [data, setData] = useState<number>();"
    good: "const [state, setState] = useState<Remote<number>>({ status: \"idle\" });"
  - title: "I/O inside the decision code"
    detail: "A reducer or step function that calls fetch or setTimeout can't be tested without mocks and runs twice under Strict Mode. Return an effect description and let the shell perform it."
    bad: "case \"rejected\":\n  setTimeout(retry, 100); // the core now owns a timer\n  return { status: \"backoff\" };"
    good: "case \"rejected\":\n  return [{ status: \"backoff\", ... }, [{ type: \"sleep\", ms: 100 }]];"
  - title: "Accepting any reply"
    detail: "Without a request id in the state, a slow first response can overwrite the result of its retry. Tag requests and ignore replies that don't match the one you are waiting for."
quiz:
  - q: "A refresh of already-loaded data fails. With Remote<T> as in the first example, what should the widget show?"
    options: ["An empty state with the error", "The last good value, marked as stale, plus the error", "A spinner until the next successful load"]
    answer: 1
    why: "The failed variant carries stale for exactly this case. The data was valid a moment ago; hiding it on a failed refresh punishes the user for a network blip."
  - q: "Why does step() return effects such as { type: 'sleep', ms: 100 } instead of calling setTimeout itself?"
    options: ["setTimeout is not allowed in TypeScript reducers", "So the core stays pure: tests replay events and assert on returned data, with no timers or mocks", "Effects objects are faster than function calls"]
    answer: 1
    why: "Describing the effect keeps every decision in a pure function; the shell turns descriptions into real I/O. The test in the second example needs nothing but a loop."
  - q: "In the core, a 'loaded' event arrives with attempt 1 while the state is { status: 'loading', attempt: 2 }. What does step return?"
    options: ["The ready state with attempt 1's data", "The same state object and no effects", "A failed state"]
    answer: 1
    why: "The reply belongs to a request the core already gave up on. Returning the identical object marks the event as ignored, which the replay test can check with ===."
further_reading:
  - title: "TypeScript Handbook: Narrowing (discriminated unions, exhaustiveness)"
    url: "https://www.typescriptlang.org/docs/handbook/2/narrowing.html"
  - title: "TypeScript Handbook: Mapped types"
    url: "https://www.typescriptlang.org/docs/handbook/2/mapped-types.html"
  - title: "TypeScript Handbook: Utility types (Extract)"
    url: "https://www.typescriptlang.org/docs/handbook/utility-types.html#extracttype-union"
  - title: "React: Extracting state logic into a reducer"
    url: "https://react.dev/learn/extracting-state-logic-into-a-reducer"
  - title: "MDN: ARIA aria-busy"
    url: "https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Reference/Attributes/aria-busy"
---

## Concept

An async value is never just "the data". Before the first request it is idle; then it is
loading; then it has data or an error; and on every refresh it is loading **again while still
holding the old data**. Give that one generic type and use it everywhere:

```ts
type Remote<T> =
  | { status: "idle" }
  | { status: "loading"; stale?: T }
  | { status: "failed"; error: string; stale?: T }
  | { status: "ready"; data: T };
```

Lesson ts.react.state used the same idea for one form's lifecycle. At the scale of an app,
two more ideas make it stick:

- **Exhaustive rendering.** A `match(state, handlers)` helper whose handler object is a
  mapped type over the status union makes "forgot the error state" a compile error at every
  widget, not a review comment.
- **Functional core, imperative shell.** Split the code that *decides* from the code that
  *does*. The core is a pure function `step(state, event) → [state, effects]`, where effects
  are data: "fetch attempt 2", "sleep 100 ms". The shell is a small loop that performs those
  effects (fetch, timers, rendering) and feeds the outcomes back as events. Retries, backoff,
  stale replies and give-up rules all live in the core, where a test is a list of events.

TanStack Query is this pattern, packaged for server data: its result is a union on `status`
(`"pending" | "error" | "success"`) plus `fetchStatus` (`"fetching" | "paused" | "idle"`), so
"success and fetching" is the stale-while-refreshing state. Use it for plain server reads and
write your own core for flows it doesn't model: uploads with processing steps, wizards,
polling jobs, anything with rules about order.

## Coming from C/C++/Java/Python/JS

- **C:** a tagged union and a `switch` in an event loop, with the compiler checking the tag
  instead of a comment. Effects-as-data is the command queue an embedded main loop drains.
- **Java:** the State pattern plus Command objects, without the class per state. Java 21's
  sealed interfaces with pattern-matching `switch` are the closest equivalent to the union
  and the exhaustive match.
- **Go:** a goroutine that owns the state and `select`s over events is the shell; the
  function it calls to decide the next state can be pure, which makes it table-testable.
- **Python:** a dataclass per state plus `match` (3.10); the pure step function is what you'd
  write to test a state machine without `asyncio` in the test.
- **JS:** Redux reducers are a functional core without effects; redux-saga and the Elm
  architecture (`update` returning a model plus commands) are the full pattern.

## How it works

`Handlers<T, R>` iterates the status union with a mapped type and uses `Extract` to pick the
matching variant: for `"failed"` the handler receives
`{ status: "failed"; error: string; stale?: T }`. Inside `match`, though, `handlers[state.status]`
is typed as a union of four function types, and TypeScript cannot prove that the one it
picks fits this particular `state`. That is the correlated-union limitation; the cast in the
helper is the standard workaround, and the helper's signature keeps every call checked.

In the core, the declared return type is the exhaustiveness check. If `Event` gains a
variant with no `case`, control can fall off the end of `step` and tsc reports TS2366
("Function lacks ending return statement and return type does not include 'undefined'").
Returning the **same** state object for ignored events gives the shell (and React, which
compares with `Object.is`) a cheap "nothing changed" signal.

## Design notes

This app already models its runner this way: `RunState` in `src/features/runner/types.ts` is
`idle | running | done | failed`, and the run panel switches on `kind`. The learning feature
doesn't: `LessonNotes` tracks `body`, `loaded` and `saved: 'no' | 'saving' | 'yes'` as three
`useState` calls, so only the `key` its parent sets keeps it from showing one lesson's note
while the next one loads (lesson ts.architecture.state). One `Remote<string>` for the note plus a save status would
make that state unrepresentable.

Choose the depth by the flow. A single fetch with no rules: `Remote<T>` or TanStack Query, no
core. Retries, timeouts, ordering constraints, anything you'd want to replay in a test: a
core with effects as data. Keep the shell boring; if it starts making decisions ("only retry
if..."), move that decision into `step` and add an event for it.
