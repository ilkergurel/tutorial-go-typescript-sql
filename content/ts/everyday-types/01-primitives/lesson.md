---
id: ts.everyday-types.primitives
title: "Primitives, literal types, any vs unknown vs never"
track: ts
module: everyday-types
order: 1
level: basic
minutes: 13
since: ts4.9
prerequisites: [ts.mental-model.tsconfig]
compare_with: [go.basics.types-conversion, go.basics.strings-runes, sql.tables-types.ids, sql.tables-types.text-numbers]
tags: [primitives, number, bigint, string, literal-types, any, unknown, never]
summary: >
  Seven primitive types, one of them a single double-precision number type, plus
  literal types that turn values into types. any switches checking off, unknown
  forces a check, and never is the type with no values.
examples:
  - id: numbers-literals
    file: examples/01_numbers_literals.ts
    title: number, bigint and literal types
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-4"
        note: >
          There is no int: number is Java's double / Go's float64 for everything.
          Integers are exact only up to Number.MAX_SAFE_INTEGER (2^53 - 1).
      - lines: "6-9"
        note: >
          The full-stack trap: Postgres bigint and Go int64 go beyond 2^53, and
          JSON.parse rounds to the nearest double without an error. The annotation
          says number and is right; the value is still wrong. Send such ids as
          strings.
      - lines: "11-15"
        note: >
          bigint (ES2020) is arbitrary precision with its own literal suffix n. As
          with Go's numeric types (and unlike Java's int-to-long-to-double widening)
          the two never mix implicitly: tsc reports it at compile time, where plain
          JavaScript would throw a TypeError at run time.
      - lines: "17-26"
        note: >
          A literal type is a type with one value. const keeps "GET" as its type
          because the binding can't change; let widens to string because it can,
          so a let can't be passed where only "GET" | "POST" is allowed.
      - lines: "28-31"
        note: >
          The fix is an annotation with the union, not a cast: next can change, but
          only to a valid Method.
  - id: strings-utf16
    file: examples/02_strings_utf16.ts
    title: "string: UTF-16 code units and Turkish case mapping"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "5-9"
        note: >
          A JS string is a sequence of UTF-16 code units, so length is not a
          character count and not a byte count. Go's len(s) counts UTF-8 bytes (9
          here); a VARCHAR(n) in Postgres counts characters.
      - lines: "11-13"
        note: >
          Unicode's default lowercase of İ (U+0130) is i followed by U+0307 COMBINING
          DOT ABOVE, so the string grows by one and no longer equals "istanbul".
      - lines: "15-24"
        note: >
          toLocaleLowerCase("tr") / toLocaleUpperCase("tr") apply the Turkish rules
          (I <-> ı, İ <-> i). Pass the locale explicitly: without an argument the
          result depends on the user's browser settings.
      - lines: "26-28"
        note: >
          Characters outside the Basic Multilingual Plane (emoji, some CJK) take two
          code units. Spread or for...of iterates code points; slicing by index can
          cut one in half.
  - id: any-unknown-never
    file: examples/03_any_unknown_never.ts
    title: any vs unknown vs never
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "5-13"
        note: >
          any is assignable to and from everything and allows any operation, so
          loose.count + 1 compiles and concatenates. Worse, any spreads: total is
          any too, and the missing items field is only found when the code runs.
      - lines: "15-26"
        note: >
          unknown accepts every value but allows nothing until you narrow it.
          typeof, the null check and `"count" in v` (narrowing since TS 4.9) prove
          the shape step by step, then v.count is usable.
      - lines: "28-32"
        note: >
          never is the empty type. A function that cannot return normally returns
          never, which is assignable to every type, so `return fail(...)` fits a
          function returning number.
      - lines: "38-39"
        note: >
          An intersection of incompatible primitives reduces to never: no value is
          both a string and a number. The same type shows up when a switch has
          handled every member of a union.
use_cases:
  - "number for counts, money in minor units (kuruş) and measurements; bigint or strings for 64-bit database ids"
  - "Literal unions for closed sets of strings: HTTP methods, statuses, event names"
  - "unknown for anything from outside the program: JSON.parse, catch variables, postMessage data"
  - "never for helpers that always throw and for exhaustiveness checks"
strengths:
  - "Literal types give enum-like safety to plain strings with zero runtime cost, which Java needs an enum class and Go a named type plus constants for"
  - "unknown is a safe top type: unlike Java's Object (which you downcast with a cast that may throw) you narrow with ordinary runtime checks the compiler follows"
  - "bigint and number never mix implicitly, so a precision loss can't slip in through arithmetic"
weaknesses:
  - "One floating-point number type: no int, no decimal; money and 64-bit ids need care (minor units, bigint, strings)"
  - "string.length counts UTF-16 code units, which is neither Go's bytes nor the characters a user sees"
  - "any is contagious and silent, and it enters through common APIs (JSON.parse, older libraries); it takes lint rules to keep it out"
pitfalls:
  - title: "Large ids through JSON.parse"
    detail: "Ids above 2^53 - 1 are silently rounded. Have the server send them as strings (Go: `json:\"id,string\"`, SQL: id::text) and keep them as strings in TS."
  - title: "toLowerCase() on Turkish text"
    detail: "\"ISPARTA\".toLowerCase() is \"isparta\", and \"İ\".toLowerCase() has a combining dot. Use toLocaleLowerCase(\"tr\") for Turkish user input, and plain toLowerCase() only for ASCII identifiers."
    bad: "login.toLowerCase() === stored"
    good: "login.toLocaleLowerCase(\"tr\") === stored"
  - title: "Annotating parsed JSON instead of narrowing it"
    detail: "`const x: Payload = JSON.parse(s)` turns any into a promise nobody checked. Type it unknown and narrow, or validate with a schema."
  - title: "Using the wrapper types"
    detail: "String, Number and Boolean (capitalized) are the object wrapper types. Always write string, number, boolean."
quiz:
  - q: "`let status = \"ok\";` What is the type of status?"
    options: ["\"ok\"", "string", "any"]
    answer: 1
    why: "let can be reassigned, so the literal widens to string. const status = \"ok\" would have type \"ok\"."
  - q: "On which value can you call .toUpperCase() without any check, and have tsc accept it?"
    options: ["A value of type any", "A value of type unknown", "Neither"]
    answer: 0
    why: "any disables checking (and may crash at run time); unknown requires narrowing first, for example typeof v === \"string\"."
  - q: "What is \"İ\".toLowerCase().length?"
    options: ["1", "2"]
    answer: 1
    why: "Outside the Turkish locale, U+0130 lowercases to i plus U+0307 COMBINING DOT ABOVE. toLocaleLowerCase(\"tr\") gives the single letter i."
  - q: "A Postgres bigint id 9007199254740993 arrives as a JSON number. What does JSON.parse return?"
    options: ["9007199254740993", "9007199254740992", "A bigint", "It throws"]
    answer: 1
    why: "JSON numbers become doubles; the nearest double to 2^53 + 1 is 2^53, and no error is raised."
further_reading:
  - title: "TypeScript Handbook: Everyday Types"
    url: "https://www.typescriptlang.org/docs/handbook/2/everyday-types.html"
  - title: "TypeScript Handbook: More on Functions (unknown, never)"
    url: "https://www.typescriptlang.org/docs/handbook/2/functions.html"
  - title: "MDN: String.prototype.toLocaleLowerCase()"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/String/toLocaleLowerCase"
  - title: "MDN: BigInt"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/BigInt"
---

## Concept

TypeScript's primitive types are JavaScript's: `string`, `number`, `bigint`, `boolean`,
`symbol`, `null` and `undefined` (always lowercase). On top of them sit **literal
types**: `"GET"`, `42` and `true` are types with exactly one value, and unions of them
(`"GET" | "POST"`) are how TypeScript models closed sets.

Three special types frame the rest of the type system:

- **`any`** opts out. Everything is assignable to it, it is assignable to everything, and
  every operation on it compiles. It is the escape hatch from untyped JavaScript.
- **`unknown`** is the safe top type. Everything is assignable to it, but you can do
  nothing with it until a runtime check narrows it.
- **`never`** is the bottom type, with no values. It is the return type of functions that
  cannot return and the type that is left after every case has been handled.

## Coming from C/C++/Java/Python/JS

- Java/C/C++/Go: there is no `int`, `long` or `float`. `number` is a 64-bit double, and
  `bigint` is a separate arbitrary-precision type, closer to `BigInteger` than to `long`.
- Java: there are no boxed `Integer` types to think about; `String`/`Number` (capitalized)
  exist but are wrapper types you never annotate with.
- Go: a string is UTF-8 bytes there and `len` counts bytes; here a string is UTF-16
  code units, like Java's `String.length()`. Go's `strings.ToLower` ignores Turkish
  rules too (I becomes i); `strings.ToLowerSpecial(unicode.TurkishCase, s)` is its
  `toLocaleLowerCase("tr")`.
- Python: `Any` in mypy is TS's `any`; `object` is the closest thing to `unknown`;
  `NoReturn`/`Never` is `never`. `Literal["GET", "POST"]` is a literal union.
- JS: same values, same `typeof` results. TypeScript adds literal types and the three
  special types.

## Design notes

Treat `any` as a code smell to contain, not a convenience: it disables checking for
everything it touches, and it spreads through assignments and return values. At the
edges of the program (JSON, `catch`, messages from other windows) use `unknown` and
narrow once, so the types inside the program are real. Prefer literal unions to plain
`string` whenever the set of valid values is known; the next lessons show how they
replace enums. And because `number` is a double, decide deliberately how money and
database ids cross the wire: minor units as integers for money, strings for 64-bit ids.
