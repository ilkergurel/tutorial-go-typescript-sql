---
id: ts.errors.result
title: "The Result pattern vs exceptions, compared with Go; React error boundaries"
track: ts
module: errors
order: 2
level: advanced
minutes: 15
since: ts2.0
prerequisites: [ts.errors.try-catch, ts.narrowing.discriminated-unions, ts.react.components]
compare_with: [go.errors.errors-as-values, go.errors.is-as-join, go.errors.panic-strategy, design.paradigms.errors, ts.architecture.explicit-states]
tags: [errors, result, discriminated-unions, exceptions, error-boundary, react, componentDidCatch]
summary: >
  A Result type puts expected failures in the signature, where the checker makes callers
  look before they use the value. Exceptions stay for bugs, and a React error boundary
  catches the render-time ones so one broken widget doesn't blank the page.
examples:
  - id: result
    file: examples/01_result.ts
    title: A Result whose error is a union of known problems
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-5"
        note: >
          The whole pattern is a discriminated union and two constructors; no library
          is needed. ok() returns Result<T, never> and err() Result<never, E>, and
          both fit any Result with matching T or E, because never is assignable to
          everything.
      - lines: "11-15"
        note: >
          The failure modes are spelled out as data a caller can branch on, with the
          details each one needs. A thrown exception would carry this as a message
          string or an untyped property.
      - lines: "17-24"
        note: >
          The signature tells the whole story: a Link or one of three problems. With
          exceptions the signature would say Link, and the failures would live in the
          doc comment.
      - lines: "26-36"
        note: >
          An exhaustive switch over the problem kinds: add a fourth kind and this
          function no longer compiles (TS2366), so every new failure gets a message.
      - lines: "38-45"
        note: >
          The value is unreachable until ok is checked: TS2339. Go's (Link, error) lets
          you read the zero-value Link without looking at err; here the type removes
          that option.
  - id: compose
    file: examples/02_compose.ts
    title: Early returns, error unions, batch errors, and the exception boundary
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "7-15"
        note: >
          The boundary between the two styles. JSON.parse throws, so the expected
          failure (SyntaxError) becomes a Result here, once. Anything else is
          unexpected and stays an exception, the same split as Go's errors and panics.
      - lines: "17-24"
        note: >
          if (!parsed.ok) return parsed; is Go's if err != nil { return nil, err }:
          the failed Result passes through unchanged, since its error type fits the
          wider union. The return type lists every way readIds can fail.
      - lines: "25-28"
        note: >
          A Record over the error union is an exhaustive message table: a new error
          code without a message is a compile error.
      - lines: "30-34"
        note: >
          For batch input, check everything and return all problems at once, as Go's
          errors.Join does. An exception would stop at the first bad item.
      - lines: "36-46"
        note: >
          The caller handles each failure where it occurs, with plain control flow.
          Note the cost: every call site has to unwrap, which exceptions spare you.
  - id: error-boundary
    file: examples/03_error_boundary.test.tsx
    title: A React error boundary, proven by two tests
    runtime: ts-test
    check: contains
    annotations:
      - lines: "1-7"
        note: >
          This example runs as a Vitest test in jsdom, which is how you'd check a
          boundary in a real project. In React 18 development builds, an error a
          boundary catches is also reported to window error handlers (react.dev,
          componentDidCatch caveats), so a page's own window.onerror sees it too.
          StackLab's browser runner skips those reports and counts only errors no
          boundary caught.
      - lines: "9-14"
        note: >
          Error boundaries must be class components; react.dev says there is no
          function-component equivalent yet. getDerivedStateFromError runs during
          rendering and only switches to the fallback. The state is a flag because
          throw null is legal and would look like no error.
      - lines: "15-19"
        note: >
          getDerivedStateFromError must stay pure, so reporting belongs in
          componentDidCatch, with the component stack. It lists the component that
          threw (Coverage here) and its parent components; the JavaScript stack of a
          render error shows React's internals where the parents would be.
      - lines: "26-31"
        note: >
          A real render bug: the API sent tests: null, and the as Row cast believed it.
          Types vanish at run time, so a boundary is still needed where data from
          outside meets rendering code.
      - lines: "33-42"
        note: >
          Each widget has its own boundary, so the broken one shows its fallback and
          the working one stays. With no boundary at all, React removes the whole UI
          from the screen on a render error.
      - lines: "44-49"
        note: >
          Boundaries don't see errors in event handlers or async code: by then React
          isn't rendering. Storing the error in state and throwing it on the next
          render hands it to the nearest boundary; the useErrorBoundary hook from
          react-error-boundary packages the same move.
      - lines: "51-56"
        note: >
          The rejection arrives after the click handler returned, so findByRole waits
          for the re-render that throws. The fallback replaces the button, and the
          report names SaveButton.
use_cases:
  - "Results for failures callers handle every time: validating forms and imports, parsing user input, a lookup that may find nothing, a conflict on save"
  - "Collecting every problem in a batch (CSV import, bulk edit) instead of stopping at the first"
  - "Exceptions for bugs and for failures nobody nearby can handle: a violated invariant, a lost database connection deep in a call chain"
  - "Error boundaries around independent page regions: a dashboard widget, each message in a list, a route's content"
strengths:
  - "Unlike Go's (T, error), the value is unreachable until the check, and E can be a precise union instead of the opaque error interface"
  - "Failure modes appear in signatures and editor hovers, where exceptions are invisible to TypeScript"
  - "Exhaustive switches and Record tables over the error union make a new failure kind a compile error at every place that must handle it"
  - "Error boundaries contain render bugs per subtree, the UI's version of Go's recover at a request boundary"
weaknesses:
  - "Every call site unwraps; without Rust's ? operator, chains of early returns are as verbose as Go's if err != nil"
  - "Two error channels coexist: library code still throws, so boundaries between the styles need care"
  - "Promise<Result<T, E>> can still reject, so async code has both a Result and a rejection to consider"
  - "Error boundaries need a class component, don't catch event-handler or async errors on their own, and don't reset themselves"
pitfalls:
  - title: "Wrapping everything in Result"
    detail: "A Result for failures no caller can handle (a bug, an exhausted connection pool) only adds unwrapping. Keep Results for expected failures and let bugs throw to a boundary."
  - title: "Catching every exception at the edge and returning it as a Result"
    detail: "A catch that converts any error turns a TypeError from a bug into a validation message. Convert the specific failure you expect and rethrow the rest."
    bad: "catch (e) { return err(\"invalid-json\"); }"
    good: "catch (e) { if (e instanceof SyntaxError) return err(\"invalid-json\"); throw e; }"
  - title: "Expecting a boundary to catch event-handler and async errors"
    detail: "Boundaries catch errors thrown while React renders. Handle errors in the handler, or store them in state and throw during render so the nearest boundary takes over."
  - title: "A boundary that never recovers"
    detail: "Once failed, the boundary keeps showing its fallback. Give it a key that changes when the input changes (a new route or record id) so React mounts a fresh one, or use react-error-boundary's resetKeys."
quiz:
  - q: "const r: Result<Link, LinkProblem> = parseLink(line). Why is r.value a compile error?"
    options:
      - "value is private"
      - "r might be the failure variant, which has no value property"
      - "Results must be awaited first"
    answer: 1
    why: "r is a union; value exists only on the { ok: true } member, so you have to narrow with r.ok first."
  - q: "Compared with Go's func parse(s string) (Link, error), what does the TS Result add?"
    options:
      - "Nothing: both are return values"
      - "The value can't be read until the error case is excluded, and the error type can be a precise union"
      - "Automatic propagation like exceptions"
    answer: 1
    why: "Go returns both values and leaves checking to discipline; the discriminated union makes the unchecked read a type error."
  - q: "A button's onClick handler throws. What does the surrounding error boundary do?"
    options:
      - "Shows its fallback"
      - "Nothing: the error didn't happen during rendering"
      - "Retries the click"
    answer: 1
    why: "Boundaries catch errors thrown while React renders; event handler errors propagate like any other exception."
  - q: "Which failure fits an exception better than a Result?"
    options:
      - "The user typed an invalid e-mail"
      - "The requirement ID in the URL doesn't exist"
      - "A switch over your own status union reaches a value that can't exist"
    answer: 2
    why: "Only the last is a bug. The others are expected outcomes a caller must handle every time."
further_reading:
  - title: "TypeScript Handbook: Narrowing (discriminated unions)"
    url: "https://www.typescriptlang.org/docs/handbook/2/narrowing.html#discriminated-unions"
  - title: "react.dev: Catching rendering errors with an error boundary"
    url: "https://react.dev/reference/react/Component#catching-rendering-errors-with-an-error-boundary"
  - title: "react.dev: componentDidCatch (including development vs production behavior)"
    url: "https://react.dev/reference/react/Component#componentdidcatch"
  - title: "react.dev: Preserving and Resetting State (resetting with a key)"
    url: "https://react.dev/learn/preserving-and-resetting-state"
  - title: "The Go Blog: Errors are values"
    url: "https://go.dev/blog/errors-are-values"
---

## Concept

TypeScript has no checked exceptions: `function parseLink(line: string): Link` can throw
anything and the signature doesn't say so. The **Result pattern** moves expected failures
into the return type:

```ts
type Result<T, E> = { ok: true; value: T } | { ok: false; error: E };
```

It is an ordinary discriminated union. The caller must check `ok` before `value` exists,
`E` can be a union of tagged problems the caller switches over exhaustively, and a failed
Result passes upward with a plain `return`. Nothing is unwound; the failure is a value,
as in Go.

That doesn't retire exceptions. Libraries and the platform throw (`JSON.parse`, `fetch` on a
network error), and bugs throw. The working split: Results for outcomes a caller is
expected to handle every time, exceptions for bugs and for failures nobody nearby can do
anything about, and explicit conversion where the two meet.

In React, the last line of defense is an **error boundary**: a class component with
`static getDerivedStateFromError` (render a fallback) and `componentDidCatch` (report the
error). It catches errors thrown while its children render, so one broken widget shows a
message instead of unmounting the page. It doesn't see event handlers, async code,
server rendering or its own errors.

## Coming from C/C++/Java/Python/JS

- Go: `(Link, error)` is the closest relative, with two differences. Go lets you read the
  `Link` without checking `err`, while the union makes that a type error; and Go's `error`
  is an interface inspected with `errors.Is`/`As`, while `E` here is a precise type. Go's
  split between errors and panics is the same split as Results and exceptions, and an
  error boundary is the UI's `recover` at a request boundary (lesson go.errors.panic-strategy).
- Rust: `Result<T, E>` with the `?` operator. TypeScript has no `?`, so propagation is an
  explicit `if (!r.ok) return r;`.
- Java: checked exceptions put failures in signatures too, but force handling through
  try/catch and don't compose with lambdas; a Result is an ordinary value that does.
- C++: `std::expected<T, E>` (C++23) is the same idea in the standard library.
- Python and JS: nothing built in; the pattern is a convention, which TypeScript's
  narrowing makes enforceable.

## Design notes

Choose per failure, not per codebase. A form validator, an import parser or a repository
lookup that may find nothing should return a Result whose error is a small union, and the
UI should map that union to messages in one exhaustive table. A violated invariant or a
lost connection deep in the stack should throw and travel to a boundary that logs it once.
Convert at the seams: wrap throwing library calls where their expected failures become
Results, and throw from a Result only where nobody can handle it.

Place error boundaries where a fallback makes sense to the user: around each route's
content, each independent widget, each item in a long list, not around every component.
Report from `componentDidCatch`, give the boundary a `key` that changes with the data it
shows so it can recover, and forward errors from handlers and effects into render when the
boundary should show them. The boundary is the backstop for bugs, not a substitute for
handling expected failures where they happen.
