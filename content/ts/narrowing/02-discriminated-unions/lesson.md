---
id: ts.narrowing.discriminated-unions
title: "Discriminated unions: the single most important pattern"
track: ts
module: narrowing
order: 2
level: basic
minutes: 12
since: ts2.0
prerequisites: [ts.everyday-types.unions-enums, ts.narrowing.basic]
compare_with: [go.interfaces.assertions, sql.modeling.anti-patterns]
tags: [unions, narrowing, discriminated-unions, exhaustiveness]
summary: >
  Give every variant of a union a shared literal tag field. The compiler then
  narrows the variant from the tag alone, and a switch over the tag can be
  checked for exhaustiveness with never.
examples:
  - id: run-result
    file: examples/01_run_result.ts
    title: A run result as a discriminated union
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-12"
        note: >
          status is a literal type "ok", not string. That literal is the
          discriminator: comparing it anywhere narrows the whole object type.
      - lines: "9-14"
        note: >
          The failed variant carries an error and no rows. Impossible states
          like "ok with error message" or "failed with rows" are unrepresentable.
      - lines: "18-24"
        note: >
          Narrowing happens on the tag alone — no type assertions, no casts.
          Inside case "failed", TypeScript knows error exists and rows doesn't.
      - lines: "25-30"
        note: >
          The never default is a compile-time exhaustiveness check: add a
          variant without extending this switch and tsc refuses to build.
use_cases:
  - "API responses (ok/failed), state machines (idle/loading/ready/error), UI events"
  - "Anywhere you would otherwise have optional fields that are only sometimes present"
strengths:
  - "Unlike optional-field soup (rows?: T[]; error?: string), variants document which fields belong together"
  - "switch + never catches every unhandled variant at compile time, for free"
  - "Narrowing works through any check on the tag: switch, if, ternary, destructuring"
weaknesses:
  - "Requires discipline at the boundaries: network JSON needs runtime validation (zod) before you can trust the tag"
  - "Deep unions of unions can get noisy; keep variants flat where you can"
pitfalls:
  - title: "Typing the tag as string instead of a literal"
    detail: "status: string narrows nothing. The tag must be a literal type ('ok' | 'failed')."
  - title: "Optional fields instead of variants"
    detail: "rows?: T[] invites checking nothing. With variants, using rows in the failed branch is a compile error."
  - title: "Assertions (as) instead of narrowing"
    detail: "res as FailedResult silences the compiler and lies at runtime. Narrow from the tag; assert only when you truly know more than tsc."
quiz:
  - q: "What makes a union 'discriminated'?"
    options: ["It uses unions of objects", "A literal-typed shared field identifies each variant", "It uses never"]
    answer: 1
    why: "The literal tag is what lets control-flow analysis narrow the variant everywhere it's compared."
  - q: "Adding a 'cancelled' variant without touching the switch — when does it break?"
    options: ["At runtime, silently", "At compile time, via the never check", "It doesn't break"]
    answer: 1
    why: "The default branch assigns the variant to never, which no type except never satisfies, so tsc errors."
further_reading:
  - title: "TypeScript Handbook: Narrowing"
    url: "https://www.typescriptlang.org/docs/handbook/2/narrowing.html"
  - title: "TypeScript Handbook: Discriminated Unions"
    url: "https://www.typescriptlang.org/docs/handbook/literal-types.html"
---

## Concept

A discriminated union is a union of object types that share one field — the *tag* — whose
type is a string (or number) literal unique to each variant. Checking the tag narrows the
object type: `if (r.status === "failed")` gives you the failed variant, with its fields and
nothing else.

## Coming from C/C++/Java/Python/JS

- C: a tagged union (`struct { enum tag; union {…} }`) done by the type system, with the
  forgetting-to-check-the-tag bug made impossible.
- Java/Python: sealed interfaces / subclasses and instanceof; discriminated unions are the
  data-only, serialization-friendly version.
- JS: you've written this shape as plain objects with a `type` field (Redux actions). The
  difference is TypeScript checking the tag and the exhaustiveness.

## Design notes

Model a value as *exactly one variant* and make illegal states unrepresentable: a request
either has rows or an error, never both, never neither. Combined with a `never`-checked
switch, the compiler forces every consumer to grow when the union grows — this is the
frontend's equivalent of Go's "accept interfaces, return structs": a default shape for
everything that crosses feature boundaries.
