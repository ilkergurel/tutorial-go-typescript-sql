---
id: ts.type-level.conditionals
title: "Conditional types, infer, template literal types"
track: ts
module: type-level
order: 2
level: advanced
minutes: 15
since: ts4.1
prerequisites: [ts.type-level.utility, ts.narrowing.guards]
compare_with: [go.generics.parameters, go.generics.methods-when, ts.react.routing]
tags: [conditional-types, infer, template-literal-types, Extract, distributive, Uppercase]
summary: >
  A conditional type is an if-expression over types, distributed over unions; infer
  extracts parts of what matched, including parts of strings. They compute precise types
  for callers, but generic code that must produce them needs casts, which is where to stop.
examples:
  - id: extract-infer
    file: examples/01_extract_infer.ts
    title: Extract, a guard that mirrors it, and infer
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "8-10"
        note: >
          A conditional type with a naked type parameter runs once per union member and
          unions the results, so Extract keeps each event that matches on its own. The
          pattern `requirement.${string}` matches two of the three types.
      - lines: "12-18"
        note: >
          approval accepts exactly one variant. The predicate repeats the type's rule in
          runtime code, and the checker trusts it: if the prefix in the type and the
          prefix in startsWith ever drift apart, nothing reports it. Keep them adjacent.
      - lines: "20-31"
        note: >
          infer V declares a type variable that the match fills in, here the promise's
          value type, and the recursion unwraps Promise<Promise<T>> as well. The built-in
          Awaited is the same idea, generalized to any object with a callable then
          method, which is what await itself unwraps.
      - lines: "33-38"
        note: >
          Comparing the discriminant narrows e to the approved variant, which is exactly
          EventOf<"requirement.approved">. Without the check e is the whole union, and
          the message names the first member that doesn't fit.
      - lines: "39-40"
        note: >
          filter with a type predicate returns RequirementEvent[], so e.id type-checks
          after it. An inline (e) => e.type.startsWith(...) would leave TraceEvent[]: TS
          5.5 infers predicates only from checks that narrow, and startsWith doesn't.
      - lines: "42-47"
        note: >
          Distribution applies only when the checked type is a bare type parameter. One
          more consequence: IsText<never> is never, because never is the empty union and
          there is no member to run the check on.
  - id: template-literals
    file: examples/02_template_literals.ts
    title: Building and parsing string types; Uppercase knows no locale
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-9"
        note: >
          Each union in a template literal type is cross multiplied, so 2 x 2 gives 4
          topics. The product grows fast, and the handbook recommends generating large
          string unions ahead of time rather than in the checker.
      - lines: "11-17"
        note: >
          The leading ${string} matches up to the first colon, Name up to the next slash,
          and Rest takes the remainder, which recurses. ParamNames of the route is
          "reqId" | "testId", and Params maps that union to an object. React Router's
          generatePath types its params argument with the same technique.
      - lines: "19-29"
        note: >
          The runtime loop parses the same grammar the type parses, and the final cast
          asserts they agree. That cast is the trust boundary of every type-level parser:
          test the function, because the checker can't compare the two.
      - lines: "31-36"
        note: >
          After the undefined check m.reqId and m.testId are strings, and a misspelled
          parameter is a compile error with a suggestion instead of an undefined at run
          time.
      - lines: "39-44"
        note: >
          The handbook says these intrinsic types use the JavaScript string functions and
          are not locale aware: Uppercase<"izmir"> is "IZMIR". A Turkish UI shows
          "İZMİR", which doesn't fit that type. Use Uppercase for machine identifiers
          (event names, HTTP methods, environment variables), never for display text.
  - id: type-map
    file: examples/03_type_map.ts
    title: "Knowing when to stop: a conditional return type vs a type map"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-13"
        note: >
          Callers of parseCell get precise types, but inside the body K is still a type
          parameter: checking kind === "number" narrows the value, not K, so Parsed<K>
          stays unresolved and only a cast compiles. The cast accepts the wrong date
          branch just as readily.
      - lines: "15-27"
        note: >
          The type map states the relation as data. Each parser is checked against its
          own entry, so returning the raw string for date is TS2322 at the definition.
      - lines: "28-30"
        note: >
          Indexing a mapped type with a generic key lets the checker relate the call to
          CellTypes[K] directly, so the function needs no cast and no conditional type.
          Prefer this shape whenever the cases can be listed.
      - lines: "32-36"
        note: >
          The cost of the cast at run time: due1 is typed Date but holds a string, and the
          first Date method called on it throws. due2 really is a Date.
use_cases:
  - "Narrowing a union by shape or string pattern: Extract<Event, { type: `user.${string}` }>, Exclude<Status, \"deleted\">"
  - "Reading types out of other types: element types, promise results, parameters of a callback, props of a component"
  - "String-keyed APIs: route params from path patterns, event and topic names, CSS custom property names"
  - "Library signatures where the result depends on an option, e.g. a query hook whose data type depends on a select function"
strengths:
  - "Types can be computed from other types, including strings, which neither Java nor Go generics can do: a route pattern yields its params type, an event union yields per-event handler types"
  - "Distribution over unions makes filters like Extract and Exclude one-liners that keep every variant's own fields"
  - "infer names a nested part without restating the outer structure, so helpers keep working when the outer type changes"
  - "It all runs in the checker, so precise library APIs cost nothing at run time"
weaknesses:
  - "Generic implementations can't produce a conditional type without casts, and casts silently accept wrong branches"
  - "Error messages print the computed types, which get long and hard to read as type-level code grows"
  - "Distribution is implicit: whether a union is split depends on whether the checked type is a bare type parameter"
  - "Recursive types have depth limits (TS2589) and cost editor responsiveness on large inputs"
pitfalls:
  - title: "Accidental distribution"
    detail: "ToArray<string | number> with T extends unknown ? T[] : never gives string[] | number[], not (string | number)[]. Wrap both sides in brackets when you mean the whole union."
    bad: "type ToArray<T> = T extends unknown ? T[] : never;"
    good: "type ToArray<T> = [T] extends [unknown] ? T[] : never;"
  - title: "A conditional return type that forces casts"
    detail: "Inside the function K is unresolved, so every return needs `as Parsed<K>`, and a cast on the wrong branch compiles. List the cases in a type map and index into it, or write overloads."
    bad: "function parseCell<K extends Kind>(k: K, raw: string): Parsed<K> { return raw as Parsed<K>; }"
    good: "function parseCell<K extends keyof CellTypes>(k: K, raw: string): CellTypes[K] { return parsers[k](raw); }"
  - title: "Uppercase and Capitalize for Turkish text"
    detail: "They use JavaScript's locale-independent case mapping, so Capitalize<\"istanbul\"> is \"Istanbul\" while Turkish users expect \"İstanbul\". Keep these types for identifiers and case display text at run time with toLocaleUpperCase(\"tr\")."
  - title: "Type-level parsers on unbounded input"
    detail: "A recursive type that isn't tail recursive stops with TS2589 \"Type instantiation is excessively deep and possibly infinite\" after about a hundred levels (tsc 7.0 fails on a 120-character string); tail-recursive types (TS 4.5) go further. Route patterns are fine; CSV rows, SQL or JSON documents belong in runtime code."
quiz:
  - q: "type IsText<T> = T extends string ? \"text\" : \"other\". What is IsText<\"a\" | 1>?"
    options: ["\"text\"", "\"other\"", "\"text\" | \"other\""]
    answer: 2
    why: "T is a bare type parameter, so the check runs per member: \"a\" gives \"text\", 1 gives \"other\", and the results are unioned."
  - q: "What is Uppercase<\"istanbul\">?"
    options: ["\"ISTANBUL\"", "\"İSTANBUL\"", "string"]
    answer: 0
    why: "The intrinsic string types apply JavaScript's toUpperCase, which is not locale aware; only toLocaleUpperCase(\"tr\") maps i to İ."
  - q: "function f<K extends Kind>(k: K): Parsed<K> { if (k === \"number\") return 1; ... } fails with TS2322. Why?"
    options:
      - "Number literals are never assignable to conditional types"
      - "Checking k narrows the value but not the type parameter K, so Parsed<K> is still unresolved"
      - "The function needs an explicit type argument"
    answer: 1
    why: "Narrowing works on values. K could still be any Kind, or a union of them, so the checker can't pick a branch of Parsed<K>."
  - q: "A template literal type interpolates Verb = \"get\" | \"list\" and then Entity = \"Requirement\" | \"Test\" | \"Link\". How many members does it have?"
    options: ["2", "5", "6"]
    answer: 2
    why: "Unions in a template literal type are cross multiplied: 2 x 3 = 6."
further_reading:
  - title: "TypeScript Handbook: Conditional Types"
    url: "https://www.typescriptlang.org/docs/handbook/2/conditional-types.html"
  - title: "TypeScript Handbook: Template Literal Types (and intrinsic string types)"
    url: "https://www.typescriptlang.org/docs/handbook/2/template-literal-types.html"
  - title: "TypeScript 4.1 release notes: template literal types, key remapping, recursive conditional types"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-4-1.html"
  - title: "TypeScript 4.5 release notes: tail-recursion elimination on conditional types"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-4-5.html"
  - title: "TypeScript 4.6 release notes: indexed access inference improvements"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-4-6.html"
---

## Concept

A **conditional type** `T extends U ? X : Y` is an if-expression evaluated by the checker:
if `T` is assignable to `U`, the result is `X`, otherwise `Y`. Two rules make it useful:

- **Distribution.** When `T` is a bare type parameter and receives a union, the condition
  runs for each member separately and the results are unioned. `Extract` and `Exclude`
  are one-line conditional types built on this.
- **`infer`.** Inside the `extends` clause, `infer X` declares a type variable that the
  match fills in: the element of an array, the value of a promise, a parameter of a
  function. `ReturnType`, `Parameters` and `Awaited` are written this way.

**Template literal types** (TS 4.1) bring string manipulation into the type system. The
type `${Entity}.${Change}`, written between backticks like a template string, builds
every combination of two unions, and `infer` inside a template takes a string apart, so
`"/requirements/:reqId"` can yield the type `{ reqId: string }`. Conditional types can recurse, which makes small parsers possible.

All of this is evaluated by the checker and erased. The goal is precise signatures for
callers; the implementation behind them is ordinary JavaScript.

## Coming from C/C++/Java/Python/JS

- C++: closest to template metaprogramming: `std::conditional_t` is `T extends U ? X : Y`,
  partial specialization plays the role of `infer`, and SFINAE-style tricks have the same
  readability problems. TypeScript adds string types, which C++ templates can't take apart.
- Java and Go: generics substitute types but never compute them; there is no way to say
  "the parameter names in this path string". Go's generics deliberately stop at
  constraints, so code that needs this uses code generation instead.
- Python: `typing` has `Literal` and overloads but no conditional types; a type checker
  can't derive `{ "reqId": str }` from a string.
- JS: none of it exists at run time. A function typed with a conditional type still needs
  runtime code that does what the type claims, and nothing checks that the two agree.

## Design notes

Conditional types pay off at **call sites**: a caller writes
`match("/requirements/:reqId", path)` and gets `reqId` checked. They cost at
**implementations**: a generic function cannot narrow its type parameter, so returning a
conditional type means a cast on every branch, and a cast on the wrong branch compiles. When
the cases can be listed, state them as a type map and index into it (example 3), or write
one overload per case. Keep conditional return types for library signatures whose result
really is computed from the argument types, and test the implementation behind the casts.

Stop when the type is harder to understand than the bug it prevents. Signs: a reviewer
can't predict the type without hovering, error messages scroll, a recursive type runs into
the depth limit (TS2589), or the type-level parser duplicates a runtime parser you also
have to test. Keep type-level programs small and close to the boundary they describe,
and write the runtime half first.
