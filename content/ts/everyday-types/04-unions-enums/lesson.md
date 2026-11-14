---
id: ts.everyday-types.unions-enums
title: "Enums vs literal unions with as const; type aliases vs interfaces"
track: ts
module: everyday-types
order: 4
level: basic
minutes: 13
since: ts5.8
prerequisites: [ts.everyday-types.primitives, ts.mental-model.erasable]
compare_with: [go.basics.variables, sql.tables-types.jsonb-arrays, ts.narrowing.discriminated-unions]
tags: [enums, literal-unions, as-const, erasableSyntaxOnly, type-alias, interface]
summary: >
  Enums are the one everyday TypeScript feature that generates code, with quirks to
  match; a literal union derived from an `as const` array or object gives the same
  safety and is plain erasable JavaScript. Use interfaces for object shapes and type
  aliases for unions.
examples:
  - id: enums-at-runtime
    file: examples/01_enums_at_runtime.ts
    title: What an enum is at run time, and where it leaks
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-11"
        note: >
          A numeric enum compiles to an object holding both Debug -> 0 and 0 ->
          "Debug". Code that iterates Object.keys or Object.values over it sees six
          entries for three members, a classic source of duplicated dropdown options.
      - lines: "13-18"
        note: >
          Since TS 5.0 an out-of-range literal is an error, but a value of type
          number is still assignable to a numeric enum. A 7 from the database
          type-checks as a Level, and the reverse lookup gives undefined.
      - lines: "20-33"
        note: >
          String enums are an exception to structural typing: the literal
          "active" is not assignable to Status even though Status.Active is
          exactly "active" at run time. Every value from JSON needs a cast, and
          the cast checks nothing.
  - id: as-const
    file: examples/02_as_const.ts
    title: "The erasable replacement: as const and a derived union"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-5"
        note: >
          as const (TS 3.4) makes the array a readonly tuple of literal types, and
          (typeof LEVELS)[number] reads the union of its elements back out. One
          source of truth: add a level to the array and the type follows.
      - lines: "7-11"
        note: >
          includes on a tuple of literals only accepts those literals, so checking
          an arbitrary string needs a widened view. The `s is Level` return type is
          a type guard (lesson ts.narrowing.guards): a true result narrows s.
      - lines: "13-17"
        note: >
          The other side of the Turkish rule: "INFO".toLocaleLowerCase("tr") is
          "ınfo" with a dotless ı, which matches nothing. Use the locale for words
          people read and plain toLowerCase() for ASCII identifiers.
      - lines: "19-31"
        note: >
          The object form gives named members (Status.Active) like an enum, but the
          type is the plain union "active" | "suspended". A value and a type may
          share the name Status because they live in separate namespaces.
  - id: type-vs-interface
    file: examples/03_type_vs_interface.ts
    title: interface vs type alias
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-16"
        note: >
          Interfaces describe object shapes and extend each other; a union of them
          needs a type alias, because an interface can only name an object type.
          This interface-per-variant plus union is the discriminated union you will
          use constantly.
      - lines: "18-21"
        note: >
          Two interface declarations with the same name merge. That is how
          libraries let you add fields to Window or to a request type, and also how
          a stray duplicate name silently changes a type. Type aliases can't be
          reopened: declaring one twice is a "Duplicate identifier" error.
      - lines: "30-36"
        note: >
          extends checks the new members against the base and reports the conflict
          where it is written. An intersection reports nothing: the member becomes
          never, and the error appears later, at whichever value fails to fit.
use_cases:
  - "Closed sets of values that cross the wire (statuses, roles, log levels): a const array or object plus its derived union"
  - "Runtime lists of the allowed values (validation, dropdowns) from the same as const array as the type"
  - "interface for object shapes, especially ones others may extend or augment; type for unions, tuples, functions and computed types"
strengths:
  - "Literal unions cost nothing at run time and accept plain strings from JSON without casts, unlike string enums"
  - "as const keeps the runtime list and the type in one place, which Go needs a named type, iota constants and a hand-written slice for"
  - "Unlike Java's enum classes, a union of string literals is just data: it serialises to itself and needs no valueOf/name mapping"
weaknesses:
  - "Unions have no namespace of named members unless you add the as const object; refactoring a string value means finding its uses by type, not by symbol"
  - "Enums give named members and IDE navigation, but emit code, break Node's type stripping and erasableSyntaxOnly, and numeric ones accept any number"
  - "type vs interface overlaps heavily, so codebases need a convention; declaration merging can surprise when two files reuse a name"
pitfalls:
  - title: "Iterating a numeric enum"
    detail: "Object.keys/values on a numeric enum include the reverse-mapping keys. Iterate an as const array instead."
  - title: "Casting JSON strings to string enums"
    detail: "`value as Status` compiles for any string and validates nothing. With a literal union, validate once with a guard like isLevel."
    bad: "const s = row.status as Status;"
    good: "if (!isStatus(row.status)) throw new Error(`bad status ${row.status}`);"
  - title: "Locale-aware lowercasing of identifiers"
    detail: "toLocaleLowerCase() with a Turkish default locale turns \"INFO\" into \"ınfo\". Lowercase ASCII identifiers with toLowerCase(); keep the \"tr\" locale for human text."
  - title: "Intersections instead of extends for overriding a member"
    detail: "`A & { id: number }` with A.id: string yields never instead of an error. Use interface extends (or Omit<A, \"id\"> & ...) when a member's type changes."
quiz:
  - q: "With erasableSyntaxOnly on, which declaration is an error?"
    options:
      - "type Level = \"debug\" | \"info\""
      - "const Level = { Debug: \"debug\" } as const"
      - "enum Level { Debug, Info }"
      - "interface Level { name: string }"
    answer: 2
    why: "enum (including const enum) generates JavaScript, so tsc reports TS1294. The others are erasable."
  - q: "`enum Status { Active = \"active\" }`. Does `const s: Status = \"active\"` compile?"
    options: ["Yes", "No"]
    answer: 1
    why: "String enum types are nominal: only Status.Active is assignable, not the equal string literal."
  - q: "`const LEVELS = [\"debug\", \"info\"] as const`. What is `(typeof LEVELS)[number]`?"
    options: ["string", "\"debug\" | \"info\"", "readonly string[]"]
    answer: 1
    why: "as const keeps the literal types; indexing the tuple type with number gives the union of its elements."
  - q: "When must you use a type alias rather than an interface?"
    options: ["To name a union", "To name an object shape", "To extend another type"]
    answer: 0
    why: "An interface can only declare an object type; unions, tuples and primitive aliases need type."
further_reading:
  - title: "TypeScript Handbook: Enums (Objects vs Enums)"
    url: "https://www.typescriptlang.org/docs/handbook/enums.html"
  - title: "TypeScript Handbook: Everyday Types (type aliases vs interfaces, literal types)"
    url: "https://www.typescriptlang.org/docs/handbook/2/everyday-types.html"
  - title: "TSConfig reference: erasableSyntaxOnly"
    url: "https://www.typescriptlang.org/tsconfig/#erasableSyntaxOnly"
  - title: "TypeScript 5.8 release notes: --erasableSyntaxOnly"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-5-8.html"
---

## Concept

TypeScript offers two ways to model a closed set of values:

- **`enum`**: named members compiled to a JavaScript object. Numeric enums also get a
  reverse mapping; string enums are nominal (only `Status.Active` is a `Status`).
- **A literal union**: `type Level = "debug" | "info" | "warn"`. To get a runtime list or
  named members as well, declare an array or object `as const` and derive the union
  from it with `(typeof X)[number]` or `(typeof X)[keyof typeof X]`.

`enum` is one of the few TypeScript constructs that is not erasable, which is why
`erasableSyntaxOnly` (TS 5.8) rejects it, `const enum` included, and why Node's type
stripping refuses to run it. The `as const` pattern gives the same compile-time safety
in plain JavaScript.

The second half of the lesson is the everyday choice between **`interface`** and
**`type`**. Both name object shapes. Only `type` can name a union, tuple, function or
computed type; only `interface` can be extended with `extends` checks and reopened by
declaration merging.

## Coming from C/C++/Java/Python/JS

- Java: enums there are classes with behaviour. TS string enums are closer to a set of
  constants with nominal typing; literal unions are closer to a sealed set of string
  values with no class at all.
- C/C++: numeric enums behave like C's `enum`: any integer value converts in, so the
  enum type does not guarantee a valid member. Even C++'s `enum class` is stricter; it
  needs a `static_cast`.
- Go: there are no enums. `type Level string` plus constants (or `iota`) is the idiom,
  and like a numeric TS enum it accepts any value of the underlying type through a
  conversion. `as const` plus a union is the TS idiom that plays the same role.
- Python: `Literal["debug", "info"]` is the literal union; `enum.Enum` is closer to TS
  enums, minus the reverse-mapping quirk.
- SQL: Postgres has `CREATE TYPE ... AS ENUM` and CHECK constraints; a literal union is
  the TS-side mirror of a `CHECK (status IN ('active', 'suspended'))`.

## Design notes

Default to literal unions. Declare the values once `as const`, derive the type, and
validate untrusted strings with a guard built on the same array. That keeps the code
erasable (Node, esbuild and every other per-file tool run it unchanged), JSON values
flow in without casts, and the list of allowed values is available at run time for
validation and UI. Reach for an `enum` only when a codebase already uses them
consistently.

For object shapes, pick one convention and keep it. The handbook's heuristic is "use
`interface` until you need features from `type`": interfaces keep their names in error
messages and `extends` is checked, while unions, tuples, functions and mapped types
need `type`. The difference matters in two places: an interface
can be merged and augmented, which libraries rely on, and `extends` reports a conflicting
member where an intersection silently produces `never`.
