---
id: ts.narrowing.guards
title: "Type guards, assertion functions, satisfies, never for exhaustiveness"
track: ts
module: narrowing
order: 3
level: intermediate
minutes: 14
since: ts4.9
prerequisites: [ts.narrowing.discriminated-unions]
compare_with: [go.interfaces.implicit]
tags: [narrowing, guards, satisfies, never]
summary: >
  Beyond discriminated unions: custom type predicates teach the checker your
  own invariants, assertion functions throw instead of branching, satisfies
  validates without widening, and never catches a missed case at compile time.
examples:
  - id: guards
    file: examples/01_guards.ts
    title: "Predicates, assertions, satisfies and never"
    runtime: ts-node
    check: exact
    annotations:
      - lines: "6-8"
        note: >
          `shape is Circle` is a type predicate: when it returns true the
          checker narrows shape to Circle at every call site.
      - lines: "12-14"
        note: >
          An assertion function either returns normally or throws. After a call
          that returned, the checker treats shape as a Circle, so the code
          continues without an else branch.
use_cases:
  - "Parse unknown JSON and narrow it to a tagged union before use"
  - "DOM event handlers: narrow EventTarget to HTMLInputElement"
  - "Config objects validated with satisfies while keeping literal types"
strengths:
  - "Custom predicates teach the checker project-specific invariants once, then every call site narrows"
  - "never-exhaustiveness turns 'we added a variant' into a compile error at the switch, not a bug in production"
  - "satisfies validates without widening, keeping inference precise"
weaknesses:
  - "A predicate can lie: the compiler trusts the runtime check it names, so the check must be tested"
  - "Assertion functions couple call sites to throw-based control flow"
  - "Chained predicates read worse than a discriminated union; prefer tagging data"
pitfalls:
  - title: "Writing a predicate whose runtime check is weaker than the type"
    detail: "`x is Circle` that only tests typeof x === 'object' narrows incorrectly and the compiler cannot tell. Keep the predicate next to its type and test it like any other invariant."
  - title: "Forgetting the never guard when a union grows"
    detail: "A switch without a default: never branch compiles fine with the new variant and silently does nothing. Add the guard from day one so the next variant breaks the build instead."
quiz:
  - q: "What does `x is Circle` on a boolean-returning function do?"
    options: ["Casts x to Circle", "Narrows x to Circle where the function returns true", "Asserts at runtime", "Nothing — documentation only"]
    answer: 1
    why: "It is a type predicate: the checker narrows the argument in the true branch of every call site."
  - q: "What is the point of `satisfies`?"
    options: ["It widens the value to the constraint", "It validates the shape while keeping the narrower literal type", "It replaces interfaces", "It is a runtime check"]
    answer: 1
    why: "satisfies checks assignability without widening, so literal keys and tuple types survive."
further_reading:
  - title: "TypeScript handbook: narrowing"
    url: "https://www.typescriptlang.org/docs/handbook/2/narrowing.html"
  - title: "TypeScript 4.9 release notes: satisfies"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-4-9.html"
---

## Concept

Discriminated unions cover most narrowing, but four tools handle the rest:

- **Type predicates** (`x is T`) package a runtime check so the compiler can
  reuse it. Write once, narrow everywhere.
- **Assertion functions** (`asserts x is T`) throw instead of returning false —
  the right shape for "this is a programmer error, not a value to handle".
- **`satisfies`** checks a value against a type *without widening* it: the
  config object keeps its literal keys, so later reads stay precise.
- **`never`** in the default branch of an exhaustive switch turns a forgotten
  variant into a compile error — the check that keeps switches honest when the
  union grows.

## Coming from C/C++/Java/Python/JS

Java's `instanceof` narrows too, but only along class hierarchies; TS predicates
narrow along any shape you can express. Python's `isinstance` is runtime-only —
no checker learns from it unless you use `typing.TypeGuard`. C++ has no
equivalent; you cast and pray. JavaScript gives you none of this — TS adds the
whole layer on top of the same runtime. The `never` exhaustiveness trick has no
mainstream analogue: it is a static assertion that doubles as documentation.

## Design notes

Predicates shift trust from the checker to you: the compiler believes whatever
boolean you name, so the predicate's runtime logic must actually correspond to
the type — the one place in this lesson where lying is possible. That is why the
idiom pairs a predicate with a single implementation and a test, instead of
scattering ad-hoc casts.
