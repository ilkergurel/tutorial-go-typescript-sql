---
id: ts.errors.try-catch
title: "try/catch with unknown; custom error classes and cause"
track: ts
module: errors
order: 1
level: intermediate
minutes: 13
since: es2022
prerequisites: [ts.narrowing.basic, ts.objects-classes.classes]
compare_with: [go.errors.errors-as-values, go.errors.is-as-join, go.basics.defer, ts.errors.result]
tags: [errors, try-catch, unknown, Error, cause, instanceof, finally, return-await]
summary: >
  JavaScript can throw any value, so under strict TypeScript the catch variable is
  unknown and must be narrowed. Throw Error subclasses that carry data, wrap with cause
  instead of replacing, and remember that try/catch only sees rejections it awaits.
examples:
  - id: unknown
    file: examples/01_unknown.ts
    title: Anything can be thrown, so the catch variable is unknown
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-15"
        note: >
          throw accepts any value. Strings and plain objects carry no stack trace and
          no name, and they still turn up from old libraries and hand-written fetch
          wrappers. Your own code should throw Error objects; typescript-eslint's
          only-throw-error rule enforces it.
      - lines: "17-22"
        note: >
          The narrowing every catch needs: instanceof Error first, then typeof for
          primitives. Only the name is printed because the message text belongs to
          the engine: JSON.parse's SyntaxError reads differently in Chrome, Firefox and
          Safari.
      - lines: "27-29"
        note: >
          With strict on, useUnknownInCatchVariables (TS 4.4) types e as unknown, so
          reading e.message is TS18046 until you narrow. Before 4.4 it was any and the
          line compiled, then printed undefined for the string throws.
      - lines: "33-34"
        note: >
          A catch clause can't be typed as Error, because nothing tells the checker
          what a call may throw: TypeScript has no throws clause. Only any and unknown
          are allowed, and you narrow inside.
      - lines: "36-38"
        note: >
          Promise.prototype.catch is declared with (reason: any) in the standard lib,
          so the strict flag doesn't reach it: e.message compiles and is undefined.
          Annotate e: unknown yourself; typescript-eslint's
          use-unknown-in-catch-callback-variable rule enforces it.
  - id: custom-errors
    file: examples/02_custom_errors.ts
    title: Error subclasses with data, wrapped with cause and found again
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-18"
        note: >
          Fields carry what a caller can act on, like a Go error type. Set name
          explicitly: toString and stack traces print it, and a minifier may rename the
          class, so constructor.name isn't reliable. Older code adds
          Object.setPrototypeOf(this, new.target.prototype) after super: that worked
          around ES5 output, where instanceof failed for Error subclasses. TS 7 no
          longer accepts target ES5 (TS5108), so the line can go.
      - lines: "25-33"
        note: >
          The import layer knows the line number, not why the lookup failed, so it
          wraps: { cause: e } (ES2022) keeps the original error attached, like Go's %w.
          Throwing a new error without cause would discard it.
      - lines: "35-42"
        note: >
          Go's errors.As written by hand: follow cause until a match or a non-Error.
          cause is typed unknown, which is honest, since anything can be a cause, and
          the loop condition narrows it again at each step.
      - lines: "44-53"
        note: >
          Handle the failure this code understands and rethrow everything else
          untouched, the equivalent of Go's "handle it or return it". A catch that logs
          and swallows every error would also swallow the TypeError from a real bug.
  - id: await-finally
    file: examples/03_await_finally.ts
    title: "return vs return await inside try, and finally"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "10-18"
        note: >
          return save(id) hands the pending promise out of the try block, which then
          ends; the rejection arrives later, where this catch can't see it. finally
          runs too early as well, before the save has failed.
      - lines: "19-27"
        note: >
          return await keeps the function suspended inside the try, so the rejection is
          thrown there and caught, and finally runs after the save settled.
          typescript-eslint's return-await rule (default in-try-catch) requires this
          form where error handling depends on it and the bare return everywhere else.
      - lines: "29-36"
        note: >
          The logs show the order. With await, save fails and then finally runs;
          without it, finally runs first and the error reaches the caller instead.
          finally runs on every path, which makes it JavaScript's counterpart of Go's
          defer for cleanup: release a lock, close a reader, clear a spinner.
use_cases:
  - "Wrapping third-party code that throws (JSON.parse, new URL, SDKs) and turning its failures into your own error types"
  - "Domain errors callers branch on: NotFoundError with the id, ValidationError with the field, ConflictError with the current version"
  - "Adding context at a layer boundary (which file, which line, which request) with { cause } instead of losing the original"
  - "Cleanup with finally: releasing locks, closing readers, resetting loading flags"
strengths:
  - "Unlike Go's if err != nil, failures propagate without code at every level, so the happy path reads straight through"
  - "unknown in catch forces the narrowing that Java's typed catch blocks do for you, without pretending to know what was thrown"
  - "cause (ES2022) gives a standard error chain, the counterpart of Go's %w and Java's getCause(), that tools and consoles understand"
  - "Error subclasses carry data and work with instanceof, like Java exception classes, without a declared hierarchy to maintain"
weaknesses:
  - "Nothing in a signature says a function can throw, unlike Go's error result or Java's checked exceptions; callers find out from docs or production"
  - "Any value can be thrown, so every catch needs narrowing code, and non-Error throws arrive without a stack"
  - "Promise rejection callbacks still receive any; the strict flag covers only catch clauses"
  - "instanceof checks compare constructors, so they fail across realms (an iframe, Node's vm) and with two copies of a library in the bundle"
pitfalls:
  - title: "Reading e.message without narrowing"
    detail: "Under strict it's a compile error; with catch (e: any) or a .catch callback it compiles and gives undefined for non-Error throws. Narrow with instanceof Error, or normalize once with a helper."
    bad: "catch (e: any) { toast(e.message); }"
    good: "catch (e) { toast(e instanceof Error ? e.message : String(e)); }"
  - title: "Wrapping without cause"
    detail: "throw new Error(\"import failed\") inside a catch discards the original error and its stack. Pass it on."
    bad: "catch (e) { throw new ImportError(line); }"
    good: "catch (e) { throw new ImportError(line, { cause: e }); }"
  - title: "return without await inside try"
    detail: "The try block ends before the promise rejects, so catch never runs and finally runs too early. Use return await inside try/catch/finally."
    bad: "try { return save(id); } catch { ... }"
    good: "try { return await save(id); } catch { ... }"
  - title: "Catching everything and logging"
    detail: "A catch that logs and continues also hides TypeErrors and RangeErrors from bugs. Catch what you can handle, rethrow the rest, and leave the last word to a top-level handler or an error boundary (lesson ts.errors.result)."
quiz:
  - q: "Under strict, what is the type of e in try { ... } catch (e) { }?"
    options: ["any", "Error", "unknown"]
    answer: 2
    why: "useUnknownInCatchVariables, part of strict since TS 4.4, types it unknown because any value can be thrown."
  - q: "Promise.reject(\"timeout\").catch((e) => e.message). What happens?"
    options:
      - "Compile error: e is unknown"
      - "It compiles and evaluates to undefined"
      - "It throws a TypeError"
    answer: 1
    why: "The lib declares the callback's reason as any, so the access compiles; a string has no message property, so the result is undefined."
  - q: "Inside async function f() { try { return save(); } catch { return \"handled\"; } }, save() rejects. What does await f() do?"
    options: ["Returns \"handled\"", "Rejects with save's error", "Resolves to undefined"]
    answer: 1
    why: "Without await, the promise leaves the try block before it rejects, so the catch never sees the rejection; f's promise adopts it."
  - q: "Which line keeps the original error available to callers?"
    options:
      - "throw new Error(`load failed: ${e}`)"
      - "throw new Error(\"load failed\", { cause: e })"
      - "console.error(e); throw new Error(\"load failed\")"
    answer: 1
    why: "Only cause keeps the error object itself, with its type, fields and stack. The first keeps only its text."
further_reading:
  - title: "TypeScript 4.4 release notes: unknown in catch variables"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-4-4.html"
  - title: "TSConfig reference: useUnknownInCatchVariables"
    url: "https://www.typescriptlang.org/tsconfig/#useUnknownInCatchVariables"
  - title: "MDN: try...catch"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Statements/try...catch"
  - title: "MDN: Error: cause"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Error/cause"
  - title: "MDN: Error (custom error types)"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Error"
---

## Concept

`throw` in JavaScript accepts any value: an `Error`, a string, a plain object, `undefined`.
TypeScript can't know what a call will throw, because signatures have no throws clause, so
since TS 4.4 the `strict` family includes `useUnknownInCatchVariables` and the catch
variable is `unknown`. You narrow it before use: `instanceof Error` for errors,
`typeof` for primitives, a fallback for everything else.

For your own failures, throw subclasses of `Error`. A subclass carries fields a caller can
act on (`id`, `line`), sets `name`, and works with `instanceof`. When a layer adds context,
it wraps instead of replacing: `new ImportError(line, { cause: e })` (ES2022) keeps the
original error attached, and code further up can walk the `cause` chain to find the error
it cares about.

Two rules complete the picture. Catch only what you can handle, and rethrow the rest
unchanged. And in async code, `try`/`catch` sees a rejection only if the `await` happens
inside the `try` block, which is why `return await` matters there.

## Coming from C/C++/Java/Python/JS

- Java: no checked exceptions and no typed catch blocks. `catch (NotFoundException e)` becomes
  `catch (e) { if (e instanceof NotFoundError) ... else throw e; }`. `cause` corresponds to
  `getCause()`.
- Python: `raise ImportFailed(line) from e` is `{ cause: e }`; `except NotFound as e:` is the
  `instanceof` check plus rethrow. Python, like Java, only lets you raise exception objects;
  JavaScript doesn't.
- Go: exceptions unwind for you where Go makes every layer write `if err != nil`.
  `{ cause: e }` is `fmt.Errorf("...: %w", err)`, the `findCause` loop is `errors.As`, and
  `finally` covers what `defer` does for cleanup. What you lose is the visible error result
  in every signature.
- C++: you can `throw 42;` there too, but catch clauses are typed and `catch (...)` gives you
  nothing to inspect. JavaScript has the one untyped catch, and you inspect the value.
- JS: nothing new at run time. TypeScript only stops you from pretending you know what was
  thrown.

## Design notes

Throw `Error` objects only, and give each failure callers branch on its own subclass with
the data they need; keep the hierarchy flat, one level under `Error` is usually enough.
Wrap at layer boundaries with `cause`, the same place Go code would add context with
`%w`. Handle errors where you can do something about them (retry, show a message, fall
back) and let the rest travel to one top-level handler that logs them once.

Exceptions are the right tool for bugs and for failures nobody nearby can handle. For
failures a caller is expected to handle every time, such as validation or "not found" in a
lookup, a return type that says so is often better: that is the Result pattern, the next
lesson, together with React's error boundaries for what slips through.
