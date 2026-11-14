---
id: ts.narrowing.basic
title: "typeof, instanceof and in narrowing"
track: ts
module: narrowing
order: 1
level: basic
minutes: 13
since: ts5.5
prerequisites: [ts.everyday-types.unions-enums, ts.everyday-types.null]
compare_with: [go.interfaces.assertions, design.paradigms.generics-absence]
tags: [narrowing, typeof, instanceof, in, control-flow, unknown]
summary: >
  TypeScript follows your runtime checks (typeof, instanceof, in, equality and
  truthiness) and narrows a union branch by branch. Each check tests exactly
  what JS tests: typeof null is "object", truthiness also drops 0 and "", and
  instanceof needs a class.
examples:
  - id: typeof-truthiness
    file: examples/01_typeof_truthiness.ts
    title: typeof, truthiness and == null
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-9"
        note: >
          Inside the if, code is a number. After the return, number can no
          longer reach the last line, so there code is a string. The same
          variable has a different type at different points: that is control
          flow analysis.
      - lines: "11-19"
        note: >
          typeof null is "object", a JS accident the compiler models: in the
          object branch tags is string[] | null, so tags.length is TS18047.
          Arrays are "object" too; Array.isArray is the precise array check.
      - lines: "21-27"
        note: >
          The naive version reports a stock of 0 as unknown, because 0 is falsy.
          tsc accepts both functions: truthiness is a legal narrowing, just a
          wrong one for numbers and strings. Objects are never falsy, so for
          them it is safe.
      - lines: "29-33"
        note: >
          name == null is true for exactly null and undefined, which makes it
          the one loose comparison worth keeping. ESLint's eqeqeq rule allows it
          with { null: "ignore" }. An empty string passes through as a string.
      - lines: "35-38"
        note: >
          The last line prints an array so the empty string stays visible as "".
          ilgaz becomes İLGAZ because tr-TR maps i to İ.
  - id: instanceof-in
    file: examples/02_instanceof_in.ts
    title: instanceof for classes, in for shapes and unknown JSON
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-10"
        note: >
          A class exists at runtime as a constructor with a prototype, which is
          what instanceof tests. Classes and built-in constructors (Date, Map,
          Error) can be checked this way; types that exist only for the
          compiler can't.
      - lines: "12-18"
        note: >
          Under strict, a catch variable is unknown, which is why describe
          accepts unknown. instanceof narrows it step by step. Swap the two ifs and the HttpError
          branch never runs, because every HttpError is also an Error.
      - lines: "23-32"
        note: >
          Type aliases and interfaces are erased, so instanceof has nothing to
          test (TS2693). "iban" in p narrows to the members that declare iban.
          An optional property would put a member in both branches.
      - lines: "36-45"
        note: >
          v starts as unknown. typeof plus !== null gives object; "id" in v
          gives object and Record<"id", unknown> (TS 4.9; before that, v stayed
          object); typeof v.id narrows the property. Lesson 03 packages such
          checks into a reusable type predicate.
  - id: control-flow
    file: examples/03_control_flow.ts
    title: Where narrowing holds, and where it stops
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-7"
        note: >
          Since TS 4.4 a condition stored in a const still narrows, as long as
          the narrowed value is a const, a readonly property or a parameter that
          the function never assigns.
      - lines: "9-16"
        note: >
          The assignment narrows prefix to string. The callback could run at any
          later time, so TS keeps the narrowing only because no assignment
          follows (TS 5.4). Add prefix = null after the map and prefix.trim()
          becomes TS18047: 'prefix' is possibly 'null'.
      - lines: "18-20"
        note: >
          Since TS 5.5 the compiler infers the predicate c is string for this
          arrow, so filter returns string[]. Before 5.5 the result stayed
          (string | null)[].
      - lines: "22-28"
        note: >
          No predicate is inferred for !!s: false could mean undefined or 0, so
          the type can't drop undefined. At runtime the 0 score really is gone,
          a bug the TS 5.5 release notes use as their example. Compare with
          !== undefined.
use_cases:
  - "Values from spreadsheets, forms and query strings that arrive as string | number"
  - "catch (e): narrow unknown to Error or to your own error classes with instanceof"
  - "Unions of object shapes you don't control and that have no tag field: narrow with in"
  - "A first pass over JSON.parse output before it reaches a type predicate or a schema validator"
strengths:
  - "Ordinary JS checks drive the types: the same if does the runtime test and changes the static type, with no cast and no new variable"
  - "Narrowing follows control flow: early returns, assignments, &&, ||, ternaries and conditions stored in a const all count"
  - "The compiler models JS quirks such as typeof null === \"object\", turning a classic runtime bug into a compile error"
weaknesses:
  - "A narrowing is only as precise as its runtime test: a truthiness check also removes 0 and \"\", and the compiler accepts it"
  - "instanceof needs a runtime class, so it can't test interfaces or type aliases, and it fails for objects from another realm (iframe, window)"
  - "Narrowing is local: it doesn't cross a function boundary unless you write a type predicate (lesson 03)"
  - "in proves that a key exists, not what type its value has; optional properties end up on both sides"
pitfalls:
  - title: "Truthiness checks on numbers and strings"
    detail: "if (qty) treats 0 as missing, and if (name) treats \"\" as missing. Compare with !== undefined, or use == null to exclude both null and undefined and nothing else."
    bad: "qty ? `${qty} left` : \"unknown\""
    good: "qty !== undefined ? `${qty} left` : \"unknown\""
  - title: "typeof x === \"object\" lets null through"
    detail: "typeof null is \"object\". Add x !== null, or use Array.isArray when you mean an array."
    bad: "if (typeof tags === \"object\") return tags.length;"
    good: "if (Array.isArray(tags)) return tags.length;"
  - title: "instanceof across realms"
    detail: "An array or Error created in another iframe or window fails instanceof Array or instanceof Error in yours, because each realm has its own constructors. Array.isArray works across realms."
    bad: "value instanceof Array"
    good: "Array.isArray(value)"
  - title: "Checking the base class first"
    detail: "if (e instanceof Error) before if (e instanceof HttpError) makes the second branch dead code: every HttpError is an Error. Order the checks from the most specific class to the most general."
quiz:
  - q: "tags: string[] | null. Inside if (typeof tags === \"object\") { ... }, what is the type of tags?"
    options: ["string[]", "string[] | null", "object", "never"]
    answer: 1
    why: "typeof null is \"object\", so null survives the check. TS models this and reports TS18047 if you use tags.length there."
  - q: "qty: number | undefined. Which condition is true for qty = 0?"
    options: ["qty", "qty !== undefined", "Boolean(qty)"]
    answer: 1
    why: "0 is falsy, so the truthiness checks treat it like undefined. Only the explicit comparison keeps it."
  - q: "type Card = { last4: string }. Why does p instanceof Card fail to compile?"
    options: ["Card must be declared as an interface", "Type aliases and interfaces are erased; instanceof needs a runtime constructor", "instanceof only works on values of type unknown"]
    answer: 1
    why: "instanceof walks the prototype chain looking for a constructor's prototype. Card has no runtime existence, so TS2693 says it only refers to a type."
  - q: "arr: (string | null)[]. In TS 5.5 or later, what is the type of arr.filter((x) => x !== null)?"
    options: ["(string | null)[]", "string[]", "unknown[]"]
    answer: 1
    why: "Since TS 5.5 the compiler infers the type predicate x is string for the arrow, and filter's predicate overload returns string[]."
further_reading:
  - title: "TypeScript Handbook: Narrowing"
    url: "https://www.typescriptlang.org/docs/handbook/2/narrowing.html"
  - title: "TypeScript 4.9 release notes: unlisted property narrowing with in"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-4-9.html"
  - title: "TypeScript 5.5 release notes: inferred type predicates"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-5-5.html"
  - title: "MDN: typeof"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/typeof"
  - title: "MDN: instanceof"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/instanceof"
---

## Concept

A union type says what a value might be. Narrowing is the compiler following your runtime
checks to work out what it is on a given line. Every branch of an `if`, every early
`return` and every assignment updates the type; the handbook calls this **control flow
analysis**. The checks it understands are the ones you already write in JS:

- `typeof x === "string"` and the other `typeof` results: `"number"`, `"bigint"`,
  `"boolean"`, `"symbol"`, `"undefined"`, `"object"`, `"function"`
- truthiness (`if (x)`) and equality (`===`, `!==`, `== null`)
- `x instanceof C` for classes
- `"key" in x` for object shapes

Each narrows exactly as far as the runtime test goes, quirks included. The next lesson
covers the most useful special case, a literal tag field; lesson 03 shows how to turn
your own checks into type predicates.

## Coming from C/C++/Java/Python/JS

- Java: similar to pattern-matching `instanceof` (`if (o instanceof String s)`), but
  there is no new variable: `x` itself has the narrower type inside the branch. And it
  works for any union, not only along a class hierarchy.
- C/C++: no counterpart. Checking a tagged union's tag doesn't change the static type,
  and `dynamic_cast` hands you a new pointer.
- Python: the same idea as mypy or pyright narrowing after `isinstance()`. JS's `typeof`
  is much coarser than `isinstance`, and there is no runtime check for an interface.
- Go: a type switch (`switch v := x.(type)`) binds a new variable per case and works on
  interface values. TS narrows the variable you already have, for any union.
- JS: nothing changes at runtime. TypeScript reads checks you already write, and some
  classic bugs (`typeof null`, `catch` values that aren't `Error`s) now show up at
  compile time.

## Design notes

Narrowing is why idiomatic TypeScript needs few casts: write the check you'd write in JS
anyway, and the type follows it. When you own the data shape, prefer a literal tag field
over `in` checks on incidental keys (next lesson); it is more precise and survives
refactoring. When a check gets long, name it: first as a `const` (TS 4.4), then as a type
predicate (lesson 03), which is the only way to carry a narrowing across a function
boundary. Data from outside the program (`JSON.parse`, `localStorage`, `postMessage`)
starts as `unknown` and gets narrowed step by step or validated by a schema
(`ts.services.zod`), never cast.
