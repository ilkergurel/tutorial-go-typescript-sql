---
id: ts.everyday-types.null
title: "null/undefined under strictNullChecks; ?. and ??"
track: ts
module: everyday-types
order: 3
level: basic
minutes: 12
since: es2021
prerequisites: [ts.everyday-types.primitives, ts.mental-model.tsconfig]
compare_with: [go.basics.pointers, sql.querying.null-logic, go.postgres.querying, design.paradigms.generics-absence]
tags: ["null", "undefined", strictNullChecks, optional-chaining, nullish-coalescing]
summary: >
  Under strictNullChecks, null and undefined are not members of every type: a value
  that may be absent says so (T | undefined, T | null), and the compiler makes you
  handle it. ?. and ?? are the operators that make handling it short.
examples:
  - id: strict-null-checks
    file: examples/01_strict_null_checks.ts
    title: Absence in the type, narrowing, and the ! escape hatch
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-7"
        note: >
          Two kinds of absence: `string | null` means the key is always there and
          null is a legitimate value (a nullable column); `phone?: string` means the
          key may be missing. Say which one you mean.
      - lines: "14-22"
        note: >
          find() returns User | undefined, and the compiler refuses member access
          until a check narrows it. In Java this line compiles and throws a
          NullPointerException at run time; in Go a nil pointer dereference panics.
      - lines: "24-31"
        note: >
          Narrowing is per value: mehmet is a User inside the if, but its nickname is
          still string | null, so the same rule applies one level down.
      - lines: "33-38"
        note: >
          `!` asserts "not null or undefined" and is erased, so it checks nothing at
          run time. When the assertion is wrong you get exactly the crash
          strictNullChecks was meant to prevent. Every ! deserves a reviewer's look.
  - id: optional-chaining
    file: examples/02_optional_chaining.ts
    title: "?. and ?? versus ||"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "11-14"
        note: >
          || falls back on every falsy value, so a deliberate pageSize of 0 and an
          empty title are replaced. ?? (ES2020) falls back only on null and
          undefined, which is almost always what a default means.
      - lines: "16-22"
        note: >
          ?. stops the whole chain at the first null or undefined and yields
          undefined, so one ?? at the end supplies the fallback. The Turkish locale
          gives IŞIL for Işıl; plain toUpperCase() would give the same here, but
          not for a name with a dotted i.
      - lines: "24-29"
        note: >
          `?.()` calls a function only if it exists: the idiomatic way to invoke an
          optional callback without an if.
      - lines: "31-35"
        note: >
          ??= (ES2021) assigns only when the current value is null or undefined:
          lazy initialisation in one line. The second assignment is skipped.
  - id: null-vs-undefined
    file: examples/03_null_vs_undefined.ts
    title: null vs undefined at the edges (JSON, Go, defaults)
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-4"
        note: >
          `x == null` is true for both null and undefined, the one place where loose
          equality is idiomatic. Use === everywhere else.
      - lines: "6-8"
        note: >
          JSON has no undefined: JSON.stringify drops such properties and writes
          null in arrays. A value that must survive a round trip as "empty" has to
          be null.
      - lines: "10-17"
        note: >
          This is what a Go handler sends: a nil *string becomes null, and an
          omitempty field disappears, which TS models as an optional property.
          pgx scans SQL NULL into a nil pointer, so NULL arrives as null too.
      - lines: "19-25"
        note: >
          Default values fire for undefined only. greet(null) prints "Merhaba, null"
          and the null nickname survives destructuring. Normalise null with ?? if
          you want the default.
use_cases:
  - "Lookups that can miss: Array.find, Map.get, index reads with noUncheckedIndexedAccess"
  - "Nullable database columns and JSON fields from a Go API, typed T | null"
  - "Optional configuration and callbacks, read with ?. and defaulted with ??"
strengths:
  - "Unlike Java, where any reference can be null and Optional is opt-in, or Go with its nil pointers and maps, the compiler tracks which values may be absent and refuses unchecked access"
  - "Narrowing is ordinary code: an if, an early return or ?? changes the type, without wrapper types like Optional<T>"
  - "?. and ?? make the safe version as short as the unsafe one"
weaknesses:
  - "Two kinds of absence (null and undefined) with different behaviour in JSON, defaults and typeof; teams must agree which means what"
  - "`!` and `as` switch the check off locally with no runtime trace when they are wrong"
  - "Only as good as the data's types: JSON.parse returns any, so a field typed string can still be null at run time"
pitfalls:
  - title: "|| for defaults"
    detail: "0, \"\" and false are valid values that || replaces. Use ?? unless you really mean every falsy value."
    bad: "const size = input.pageSize || 50;"
    good: "const size = input.pageSize ?? 50;"
  - title: "Truthiness checks on numbers and strings"
    detail: "if (count) skips 0 and if (name) skips \"\". Compare with undefined or null explicitly when 0 or empty are legitimate."
  - title: "Expecting defaults to replace null"
    detail: "Parameter and destructuring defaults apply to undefined only. A null from JSON or the database passes straight through."
  - title: "Sprinkling ! to make errors go away"
    detail: "Each ! is a claim the compiler can't verify. Prefer a check that handles the absent case, or throw a descriptive error at the point you know it can't happen."
quiz:
  - q: "`input.pageSize` is 0. What is `input.pageSize || 50`?"
    options: ["0", "50"]
    answer: 1
    why: "|| replaces every falsy value, and 0 is falsy. `input.pageSize ?? 50` would keep 0."
  - q: "user is undefined. What does `user?.address.city` evaluate to?"
    options: ["It throws a TypeError", "undefined", "null"]
    answer: 1
    why: "?. short-circuits the entire rest of the chain, so .address.city is never evaluated."
  - q: "`function f(x: number | null = 1) { return x; }`. What does f(null) return?"
    options: ["1", "null"]
    answer: 1
    why: "Default parameters are applied only when the argument is undefined."
  - q: "What does the ! in `map.get(k)!.name` do at run time?"
    options: ["Throws a clear error if the value is missing", "Nothing; it is erased and the access may still crash"]
    answer: 1
    why: "The non-null assertion is type syntax only. If the value is undefined, .name throws a TypeError."
further_reading:
  - title: "TypeScript Handbook: Everyday Types (null and undefined)"
    url: "https://www.typescriptlang.org/docs/handbook/2/everyday-types.html#null-and-undefined"
  - title: "TSConfig reference: strictNullChecks"
    url: "https://www.typescriptlang.org/tsconfig/#strictNullChecks"
  - title: "MDN: Optional chaining (?.)"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/Optional_chaining"
  - title: "MDN: Nullish coalescing operator (??)"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/Nullish_coalescing"
---

## Concept

JavaScript has two "no value" values: `undefined` (a variable or property that was never
set, a missing argument, a function without `return`) and `null` (an explicit "empty").
With **`strictNullChecks`** (part of `strict`, on by default since TypeScript 6.0) neither
belongs to any other type. A value that may be absent says so in its type, as
`T | undefined`, `T | null` or an optional property `x?: T`, and the compiler rejects
member access until a check narrows the type back to `T`.

Two operators (ES2020) make the checks short:

- **`a?.b`** evaluates to `undefined` if `a` is null or undefined, skipping the rest of the
  chain. It also works for calls (`f?.()`) and index reads (`a?.[i]`).
- **`a ?? b`** gives `b` only if `a` is null or undefined; `??=` (ES2021) assigns on the
  same condition.

## Coming from C/C++/Java/Python/JS

- Java: without `strictNullChecks` TypeScript works like Java, where null is a member of
  every reference type. With it, it works like Kotlin's `String` vs `String?`, without
  needing an `Optional<T>` wrapper.
- Go: nil pointers, maps and interfaces are unchecked, and a missing map key silently
  gives the zero value. TS has no zero values: absent is `undefined`, and the type says
  when that can happen.
- C/C++: like a pointer the compiler forces you to null-check before dereferencing.
- Python: `Optional[str]` / `str | None` under mypy, with `?.` and `??` as operators
  Python lacks.
- SQL: TS `null` is not SQL `NULL`. `null === null` is true in JS, whereas `NULL = NULL`
  is NULL in SQL; use `IS NOT DISTINCT FROM` there.

## Design notes

Give each kind of absence one meaning: `undefined` (or an optional property) for "not
set / not provided", `null` only where the data itself has nulls, typically JSON from a
Go API and nullable database columns. Normalise at the boundary (`row.nickname ?? undefined`
or the reverse) instead of letting both flow through the program. Prefer narrowing to
assertions: an early `if (x === undefined) return ...` documents the absent case and keeps
the rest of the function on the happy path. And default with `??`, never with `||`, unless
you really want to replace `0`, `""` and `false` as well.
