---
id: ts.generics.functions
title: "Generic functions and types, extends constraints, defaults"
track: ts
module: generics
order: 1
level: intermediate
minutes: 14
since: ts5.4
prerequisites: [ts.functions.higher-order, ts.narrowing.discriminated-unions]
compare_with: [go.generics.parameters, go.generics.methods-when, ts.generics.keyof]
tags: [generics, type-parameters, constraints, inference, const-type-parameters, NoInfer]
summary: >
  A type parameter relates the types of several values, usually an input and the
  output, so every call keeps its precise type. Constraints state what T must have
  while T keeps the rest; defaults, const type parameters and NoInfer tune inference.
examples:
  - id: type-parameters
    file: examples/01_type_parameters.ts
    title: any vs unknown vs a type parameter
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-14"
        note: >
          Three signatures for "first element". any switches checking off for
          every caller, unknown is honest but makes every caller narrow, and T
          relates the input to the output: it appears twice, which is what makes
          a type parameter worth having.
      - lines: "16-21"
        note: >
          any spreads: the result of firstAny is any, so calling a number method
          on a string compiles and throws at run time.
      - lines: "22-27"
        note: >
          unknown refuses every use until you narrow it (TS2571). The generic
          version returns string | undefined, so string methods work and a
          number annotation is rejected. "izmir".toLocaleUpperCase("tr") is
          İZMİR; the locale-free toUpperCase gives IZMIR.
      - lines: "29-35"
        note: >
          Each type parameter is inferred from its own argument. With nothing to
          infer from, first([]) would infer never and return undefined, so pass
          the type argument. Type arguments are all or nothing: pair<string>(...)
          is TS2558 (expected 2), unless the missing ones have defaults.
      - lines: "37-46"
        note: >
          A const type parameter (TS 5.0) infers literal types as if the caller
          had written as const. The constraint must be a readonly array, or the
          inferred readonly tuple doesn't fit and inference falls back to
          string[]. (typeof locales)[number] turns the tuple into a union.
  - id: constraints
    file: examples/02_constraints.ts
    title: Constraints keep the caller's type
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-14"
        note: >
          Both functions require an id. Only the generic one returns what it was
          given: T extends HasId sets a lower bound, and T itself carries every
          other property of the caller's objects through to Map<string, T>.
      - lines: "16-27"
        note: >
          The lossy version forgot price (TS2339). An argument without id fails
          the constraint: T falls back to HasId, and the object literal then has
          an excess property (TS2353).
      - lines: "29-32"
        note: >
          A constraint is a minimum, not the type. The caller chooses T, which
          may be a subtype with more required properties, so { id: "new" } is
          not a T. Note that T appears only in the return type here, the
          warning sign from example 3.
      - lines: "34-42"
        note: >
          T is inferred from items and then flows into the callback, so p is the
          full product type without an annotation. Intl.Collator("tr") sorts in
          the Turkish alphabet (Ç after C, I before İ, Ş after S); the default
          sort compares UTF-16 code units, which puts Ilık before Çay.
  - id: generic-types
    file: examples/03_generic_types.ts
    title: Generic types, defaults, NoInfer and the return-only trap
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-4"
        note: >
          A generic type alias. E = Error is a default: Result<number> means
          Result<number, Error>. Parameters with defaults must come after the
          ones without, as in function parameters.
      - lines: "6-18"
        note: >
          parsePrice relies on the default, parsePort narrows the error to two
          string literals. The union is discriminated by ok (lesson
          ts.narrowing.discriminated-unions), so r.value and r.error are only
          reachable on the right branch.
      - lines: "20-27"
        note: >
          NoInfer (TS 5.4) blocks inference from fallback, so T comes from the
          options list alone and "fr" is rejected. Without it, every argument is
          an inference candidate and T silently widens to include "fr".
      - lines: "29-34"
        note: >
          T appears only in the return type, so it relates nothing: it is a cast
          with nicer syntax. JSON.parse returns any, which is assignable to T, so
          port is typed number while holding the string "8080", and + concatenates.
      - lines: "35-40"
        note: >
          The honest signature returns unknown and makes the caller check. For
          real data, validate with a schema at the boundary (lesson
          ts.services.zod) and let the schema produce the type.
use_cases:
  - "Collection helpers that return what they were given: first, indexById, groupBy, sortBy, chunk"
  - "Containers and wrappers: Result<T, E>, Page<T>, Cache<K, V>, a typed event emitter"
  - "Functions that take callbacks over the caller's element type, such as sortBy(items, (p) => p.name)"
  - "Accepting anything with a required shape while returning the caller's exact type: T extends { id: string }"
strengths:
  - "Inference does the work: callers rarely write type arguments, and the return type follows the arguments"
  - "Constraints can be any type, including an object shape, a literal union or a function type; Java bounds and Go constraints are declared interfaces"
  - "Type parameters are erased: one JavaScript function serves every T, with no boxing as in Java and no per-type instantiation as in C++ templates"
  - "const type parameters and NoInfer let a library shape inference without asking callers for as const or explicit type arguments"
weaknesses:
  - "Erased means no new T(), no T.name and no runtime check that a value is a T; pass a factory, a schema or a type guard instead"
  - "A type parameter used only in the return type compiles and lies: the caller picks any T"
  - "No partial inference: supply every type argument or none (defaults aside)"
  - "Deeply generic signatures produce long error messages; two plain functions are often clearer than one clever generic"
pitfalls:
  - title: "A generic return type with no matching input"
    detail: "The handbook's rule: a type parameter that appears once relates nothing. parseJson<T> is an unchecked cast; return unknown and validate."
    bad: "function parseJson<T>(text: string): T { return JSON.parse(text); }"
    good: "function parseJson(text: string): unknown { return JSON.parse(text); }"
  - title: "Constraining a type parameter instead of using it"
    detail: "When T is the whole array, element reads resolve through the constraint and the return type degrades to any. Make the element the type parameter (the handbook calls this pushing type parameters down)."
    bad: "function first<T extends any[]>(arr: T) { return arr[0]; }"
    good: "function first<T>(arr: readonly T[]): T | undefined { return arr[0]; }"
  - title: "Generic where a plain type would do"
    detail: "function greet<S extends string>(s: S): void relates nothing to S. Write (s: string): void; add the type parameter when a second position needs it."
  - title: "Fabricating a T inside the function"
    detail: "Returning { id: \"new\" } as T compiles with the cast and breaks for any caller whose T has more required fields. Take a factory parameter, or return the constraint type."
    bad: "function create<T extends HasId>(): T { return { id: \"new\" } as T; }"
    good: "function create<T extends HasId>(make: (id: string) => T): T { return make(\"new\"); }"
quiz:
  - q: "first<T>(items: readonly T[]): T | undefined. What is the type of first([\"a\", \"b\"])?"
    options: ["unknown", "any", "string | undefined", "\"a\" | \"b\" | undefined"]
    answer: 2
    why: "T is inferred from the array literal, whose elements widen to string. Getting the literals needs a const type parameter or as const."
  - q: "Why is function make<T extends HasId>(): T { return { id: \"x\" }; } an error?"
    options: ["Object literals can't be returned from generic functions", "T may be a subtype of HasId with more required properties", "HasId must be a class to be used as a constraint"]
    answer: 1
    why: "The constraint is a lower bound chosen by the caller; { id } satisfies HasId but not every T."
  - q: "Which signature is a cast in disguise?"
    options: ["<T>(items: T[]) => T | undefined", "<T>(text: string) => T", "<T extends string>(options: T[], fallback: NoInfer<T>) => T"]
    answer: 1
    why: "Its T appears only in the return type, so nothing ties it to a real value; the caller asserts it."
  - q: "pick<T extends string>(options: T[], fallback: T) is called as pick([\"tr\", \"en\"], \"fr\"). What happens?"
    options: ["It compiles: T is \"tr\" | \"en\" | \"fr\"", "Error: \"fr\" is not assignable", "T is inferred as string"]
    answer: 0
    why: "Both arguments are inference candidates. NoInfer<T> on fallback makes the call an error."
further_reading:
  - title: "TypeScript Handbook: Generics"
    url: "https://www.typescriptlang.org/docs/handbook/2/generics.html"
  - title: "TypeScript Handbook: More on Functions (generic functions, guidelines)"
    url: "https://www.typescriptlang.org/docs/handbook/2/functions.html"
  - title: "TypeScript 5.0 release notes: const type parameters"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-5-0.html"
  - title: "TypeScript 5.4 release notes: the NoInfer utility type"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-5-4.html"
  - title: "Go tutorial: Getting started with generics"
    url: "https://go.dev/doc/tutorial/generics"
---

## Concept

A **type parameter** is a placeholder the caller fills in, usually by inference:
`function first<T>(items: readonly T[]): T | undefined`. Its job is to *relate* types:
the element type of the input is the type of the output. A parameter that appears once
relates nothing and is a sign the function doesn't need to be generic.

- **Constraints** (`T extends HasId`) set a minimum. Inside the function you may use what
  the constraint guarantees; outside, the caller's full type flows through. A constraint
  is a lower bound, so the function can't invent a value of type `T`.
- **Generic types** take parameters too: `type Result<T, E = Error>`, `interface Page<T>`,
  `class Cache<K, V>`. A **default** (`E = Error`) fills in an argument the user omits.
- **Inference** picks each type parameter from the arguments. When it guesses wrong, you
  can pass type arguments explicitly (all of them, or none), mark a parameter
  `const` (TS 5.0) to keep literal types, or wrap a parameter in `NoInfer<T>` (TS 5.4) so
  it is checked against `T` but doesn't help infer it.

Type parameters are erased like every other type. One JavaScript function runs for every
`T`, which is why `new T()` and `x instanceof T` don't exist.

## Coming from C/C++/Java/Python/JS

- Java: `<T extends Comparable<T>>` reads the same, and Java also erases. Differences: the
  bound can be any TS type (an object shape, a union of literals, a function type), not
  only a declared interface; there are no wildcards, because variance is computed from
  structure; and primitives need no boxing.
- C++: templates generate code per instantiation and check the body per use. TS checks a
  generic body once, against the constraint, and generates nothing. The closest C++ idea
  is a concept, minus the code generation.
- Python: `def first[T](items: Sequence[T]) -> T | None` (PEP 695) is the same idea, also
  erased, also checked only by a separate tool.
- Go: `func First[T any](items []T) (T, bool)` matches closely. Go constraints are
  interfaces listing methods and type sets; TS constraints are arbitrary types. TS
  methods have always been able to declare their own type parameters, which Go methods
  gained in 1.27 (lesson go.generics.methods-when).
- JS: nothing changes at run time; generics exist only in the checker.

## Design notes

Start concrete. Write `(items: readonly Product[]) => Product[]`, and introduce `T` the
second time you need the same logic for a different type, when the relation between
input and output is obvious. Apply the handbook's three rules: use the type parameter
itself rather than constraining a bigger one, use as few type parameters as possible,
and make each one appear at least twice.

Prefer the constraint that states what the body actually uses (`T extends { id: string }`),
and return `T` so callers keep their type. Reach for `const` type parameters and
`NoInfer` when you write library-style helpers whose callers shouldn't need `as const`
or explicit arguments. And when a type parameter would only describe data arriving from
outside (`fetchJson<T>()`), stop: that is a validation problem, solved at run time by a
schema, not by a type argument.
