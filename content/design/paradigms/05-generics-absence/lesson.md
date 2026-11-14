---
id: design.paradigms.generics-absence
title: "The same problem: generics, and absence (zero values vs undefined/null vs NULL)"
track: design
module: paradigms
order: 5
level: advanced
minutes: 14
since: go1.21
prerequisites: [design.paradigms.side-by-side, go.generics.parameters, ts.everyday-types.null, sql.querying.null-logic]
compare_with: [go.basics.variables, go.composite.slices-and-maps, go.generics.parameters, ts.generics.functions, ts.everyday-types.null, sql.querying.null-logic, go.postgres.querying, sql.aggregation.group-by]
tags: [absence, zero-values, "null", "undefined", generics, aggregates, comparison]
summary: >
  "The cheapest offer" can be missing because the product isn't listed, has no offers,
  or has an unknown price. Go answers with zero values plus an ok bool, TypeScript with
  T | undefined, SQL with NULL in every type and no row at all.
examples:
  - id: go-zero-values
    file: examples/01_zero_values.go
    title: "Go: a zero value plus a bool"
    runtime: go
    check: exact
    annotations:
      - lines: "11-17"
        note: >
          A generic function can't return nil for a T that may be int64, and
          there is no Optional type: `var zero T` is the only "nothing" it can
          make. So absence travels as a second result, the comma-ok shape of map
          reads and type assertions. slices.Min (Go 1.21) would panic on an empty
          slice instead.
      - lines: "26"
        note: >
          Reading a missing key yields the zero value, a nil slice, which ranges
          and measures exactly like an empty one. Only the comma-ok bool tells
          "ayran isn't listed" from "kahve has no offers".
      - lines: "28-35"
        note: >
          Three situations, three branches. The free promotion prints 0 as a
          real price because ok, not the value, says whether there was one.
      - lines: "38-42"
        note: >
          What the bools were for: without them, an unlisted product, a product
          without offers and a free offer all come back as 0. Never let a zero
          value stand for "missing" when zero is a valid answer.
  - id: ts-undefined
    file: examples/02_undefined.ts
    title: "TypeScript: absence in the type, narrowed before use"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "2-5"
        note: >
          The generic version returns T | undefined: the type itself says the
          result may be missing, and strictNullChecks makes callers narrow it.
          The catch: if T can itself be undefined, a missing result and a stored
          undefined look the same, where Go's bool would still tell them apart.
      - lines: "3"
        note: >
          The guard is required, not cosmetic: [].reduce(f) with no initial
          value throws a TypeError.
      - lines: "16-20"
        note: >
          Map.get returns number[] | undefined, and the early return narrows
          prices to number[] for the rest of the loop body. This is Go's
          comma-ok, expressed in the type instead of a second variable.
      - lines: "26-27"
        note: >
          JavaScript's built-in answer for "the minimum of nothing" is Infinity,
          a number that looks like a price. And JSON has no Infinity, so it
          arrives at the server as null.
      - lines: "28-29"
        note: >
          || treats 0, "" and false as missing too, so the free offer turns into
          "no offers". ?? replaces only null and undefined.
  - id: sql-null
    file: examples/03_null.sql
    title: "SQL: NULL in every type, and the difference between NULL and no row"
    runtime: sql
    check: exact
    annotations:
      - lines: "5"
        note: >
          A nullable column is SQL's third kind of absence, "unknown": the price
          exists but isn't known. In Go this column needs *int32 or
          sql.Null[int32], in TS number | null.
      - lines: "12-19"
        note: >
          kahve has no offers, so the LEFT JOIN gives it one row of NULLs and min
          returns NULL. For su, min skips the NULL price and returns the free 0.
          count(column) counts non-null values, so offers and priced differ.
      - lines: "22"
        note: >
          No row at all is a different answer from a NULL. In Go,
          QueryRow(...).Scan returns pgx.ErrNoRows here, which a repository translates to its
          own ErrNotFound.
      - lines: "25-26"
        note: >
          Without GROUP BY, an aggregate always returns exactly one row, even
          over no input: min is NULL, count is 0. Code that expects "no row" for
          an unknown product never gets it.
      - lines: "29"
        note: >
          coalesce is SQL's ??, and like min it works for any type. Use it to
          present absence, as here, not to store a made-up value in its place.
use_cases:
  - "Lookups that may find nothing: comma-ok or (T, bool) in Go, T | undefined in TS, an empty result in SQL, kept distinct from a found zero"
  - "Optional fields across the stack: a nullable column, a *T or sql.Null[T] field in Go, a JSON null, number | null in TS"
  - "Reports over groups that may be empty: LEFT JOIN plus aggregates, where NULL and count = 0 mean \"nothing to measure\""
strengths:
  - "Go: every variable is usable at its zero value, and no read of a missing map key or unset field can crash, unlike a null dereference in Java"
  - "TS: absence is part of the type under strictNullChecks, so the compiler makes callers handle undefined, which Go's zero values never force"
  - "SQL: aggregates skip unknown values by definition, so one missing price doesn't turn the whole minimum into NULL"
weaknesses:
  - "Go: a zero value is indistinguishable from a real 0 or \"\" unless you add a bool, a pointer or sql.Null[T]"
  - "TS: two absent values (undefined and null) and falsy-based operators (||) make it easy to treat 0 as missing"
  - "SQL: NULL is in every column type unless you add NOT NULL, and it propagates through expressions and three-valued logic"
pitfalls:
  - title: "A generic helper that returns a bare zero value"
    detail: "The caller can't tell \"no element\" from an element equal to the zero value. Return (T, bool), as map reads and type assertions do."
    bad: "func MinOf[T cmp.Ordered](xs []T) T { var zero T; if len(xs) == 0 { return zero }; return slices.Min(xs) }"
    good: "func MinOf[T cmp.Ordered](xs []T) (T, bool)"
  - title: "omitempty erasing a real zero on the way to TypeScript"
    detail: "encoding/json's omitempty drops a 0 int64, so a free price disappears from the JSON and the TS client reads undefined: \"no offers\". Use a pointer, which encodes nil as null and 0 as 0."
    bad: "Cheapest int64 `json:\"cheapest,omitempty\"` // 0 -> {}"
    good: "Cheapest *int64 `json:\"cheapest\"` // nil -> null, 0 -> 0"
  - title: "Defaulting with || in TypeScript"
    detail: "|| replaces every falsy value: 0, \"\", false and NaN as well as null and undefined. Default with ?? unless replacing those is really what you mean."
    bad: "const shown = cheapest || \"no offers\";"
    good: "const shown = cheapest ?? \"no offers\";"
  - title: "Expecting no row from an aggregate"
    detail: "SELECT min(...) WHERE ... returns one row even when nothing matches, so pgx's QueryRow never reports ErrNoRows for it. Scan into a pointer and check for nil, or count the rows explicitly."
quiz:
  - q: "Why does MinOf return (T, bool) instead of just T?"
    options:
      - "Generic functions must return two values"
      - "The zero value of T can be a real answer, like a free price of 0"
      - "slices.Min requires it"
    answer: 1
    why: "Without the bool, an empty input and a minimum of 0 return the same value."
  - q: "How many rows does SELECT min(price_kurus) FROM offers WHERE product = 'ayran' return?"
    options:
      - "None"
      - "One row, with NULL"
      - "One row, with 0"
    answer: 1
    why: "An aggregate without GROUP BY always returns one row; min over no input is NULL."
  - q: "cheapest is 0, a free offer. What does `cheapest || \"no offers\"` evaluate to?"
    options:
      - "0"
      - "\"no offers\""
      - "undefined"
    answer: 1
    why: "0 is falsy, so || falls through to the right side; ?? would keep the 0."
  - q: "su has offers 500, 0 and NULL. What does min(price_kurus) return for it?"
    options:
      - "NULL, because one price is unknown"
      - "0"
      - "500"
    answer: 1
    why: "Aggregate functions other than count(*) ignore NULL inputs."
further_reading:
  - title: "The Go Programming Language Specification: The zero value"
    url: "https://go.dev/ref/spec#The_zero_value"
  - title: "The Go Programming Language Specification: Index expressions (comma-ok)"
    url: "https://go.dev/ref/spec#Index_expressions"
  - title: "TypeScript Handbook: null and undefined"
    url: "https://www.typescriptlang.org/docs/handbook/2/everyday-types.html#null-and-undefined"
  - title: "MDN: Nullish coalescing operator (??)"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/Nullish_coalescing"
  - title: "MDN: Math.min()"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Math/min"
  - title: "PostgreSQL docs: Aggregate Functions"
    url: "https://www.postgresql.org/docs/current/functions-aggregate.html"
---

## Concept

"Missing" is not one thing. In the example there are four products and four answers:
`ayran` isn't listed, `kahve` is listed with no offers, `simit` has a cheapest offer, and
`su` has a free one, so 0 is a real price. SQL adds a fifth case, an offer whose price is
unknown. Each language spells these differently, and the spelling shapes its generic code.

- **Go has zero values, not nulls.** Every type has a zero value, and a missing map key,
  an unset field and `var zero T` in generic code all produce it. Absence is therefore a
  separate signal: the comma-ok bool, a `(T, bool)` or `(T, error)` result, or a type
  that can say "nothing" (`*T`, `sql.Null[T]`) where a field may be absent.
- **TypeScript has two absent values in the type.** `undefined` is "never set" and `null`
  is "explicitly empty". Under strictNullChecks they belong to no other type, so a value
  that may be missing says so (`T | undefined`), and the checker makes you narrow before
  use. Generic code returns `T | undefined`.
- **SQL has NULL in every type, and empty results.** NULL means unknown and is a member of
  every column type unless NOT NULL removes it. Aggregates skip it, and over no input they
  return it (except `count`). Separately, a query can return no row at all, which is not
  the same answer as a row holding NULL.

Each language also has a built-in "minimum of nothing", and none of them is "nothing":
Go's `slices.Min` panics, JavaScript's `Math.min()` returns `Infinity`, and SQL's `min`
returns NULL.

## Coming from C/C++/Java/Python/JS

- **Java**: `null` belongs to every reference type, as NULL belongs to every SQL type.
  Go's `(T, bool)` is an `Optional<T>` split into two values, and TypeScript under
  strictNullChecks works like Kotlin's `String` versus `String?`.
- **C/C++**: Go's zero values are C's zero-initialized statics applied to everything,
  so there is no uninitialized memory. `std::optional<T>` is the closest relative of Go's
  `(T, bool)`; `T | undefined` is an optional the compiler forces you to check.
- **Python**: `min([])` raises `ValueError`, much like Go's panic, and `min(xs, default=None)`
  is the TS `T | undefined` version. `dict.get(k)` returning `None` is `Map.get`.
- **JS**: TypeScript changes no run-time behavior: `Math.min()` is still `Infinity`,
  `[].reduce(f)` still throws, and `JSON.stringify` still turns `Infinity` into `null`.
- **Generics in general**: Java erases `T`, so every `T` can be null; C# has
  `default(T)`, which is Go's `var zero T`; TypeScript's `T | undefined` is ambiguous
  only when `T` itself includes `undefined`.

## How it works

The same situations, in each spelling:

```text
situation           Go                             TypeScript                SQL
not listed          offers[p]: nil, ok == false    Map.get -> undefined      no row
listed, no offers   MinOf -> (0, false)            minOf -> undefined        min() -> NULL, count 0
free offer          MinOf -> (0, true)             minOf -> 0                min() -> 0
price unknown       *int32 nil, sql.Null[int32]    number | null             NULL, skipped by min()
minimum of nothing  slices.Min panics              Math.min() is Infinity    min() is NULL
```

Crossing layers, the spellings have to be translated: a NULL column scans into a nil
pointer in Go (pgx refuses to scan NULL into a plain `int32`), `encoding/json` writes the
nil pointer as `null`, and TypeScript reads `number | null`.

## Design notes

Give each kind of absence one meaning and keep it through every layer. "Not listed" is
no row in SQL, a not-found error in Go and a 404 for the client. "No offers" is NULL from
the aggregate, a nil pointer in Go, `null` in JSON. A free offer is 0 everywhere, and no
layer may turn it into "missing": `omitempty` on an integer, `||` in TypeScript and
`coalesce(price, 0)` in SQL are the three usual ways it happens.

Prefer NOT NULL in the schema, and allow NULL only where "unknown" is a real state
(lesson sql.querying.null-logic). In Go, make types useful at their zero value, but don't
let the zero value stand for "not set" when zero is a legal value; use a pointer or
`sql.Null[T]` for those fields. In TypeScript, use `undefined` for "not provided" and
`null` only where the data carries it, typically JSON from a Go API and nullable columns.

For generic helpers, follow each language's convention: `(T, bool)` in Go, because a
type parameter has no nil; `T | undefined` in TypeScript, switching to a `{ found, value }`
result only when `T` may itself include `undefined`; and in SQL, the built-in
polymorphic functions (`min`, `coalesce`) already return NULL of whatever type they're
given.
