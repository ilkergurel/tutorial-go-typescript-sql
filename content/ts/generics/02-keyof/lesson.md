---
id: ts.generics.keyof
title: "keyof, indexed access types, generic React components"
track: ts
module: generics
order: 2
level: advanced
minutes: 15
since: es2023
prerequisites: [ts.generics.functions, ts.everyday-types.unions-enums]
compare_with: [ts.type-level.utility, ts.react.components, go.generics.parameters]
tags: [keyof, indexed-access, generics, react, generic-components, Object.keys]
summary: >
  keyof T is the union of T's property names and T[K] the type stored under K, so a
  generic K extends keyof T ties a key to its value type. The same idea makes React
  components generic over the caller's row type, with every column checked.
examples:
  - id: keyof-indexed
    file: examples/01_keyof_indexed.ts
    title: keyof, T[K], and why Object.keys returns string[]
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-11"
        note: >
          keyof Product is the union of its property names as literal types;
          Product["price"] reads the type stored under a key, and [number] reads
          an array's element type. Both are computed from Product, so renaming a
          field updates every type derived from it.
      - lines: "13-27"
        note: >
          K extends keyof T captures the literal key the caller passed, so
          get(cay, "price") is a number. The loose version accepts any key and
          must return the union of every property type, leaving the caller to
          narrow. A typo is a compile error in both.
      - lines: "29-36"
        note: >
          T[K][] follows the key: pluck(rows, "name") is string[], and
          pluck(rows, "tags") is string[][]. The Turkish locale uppercases simit
          to SİMİT; toUpperCase() would give SIMIT.
      - lines: "38-47"
        note: >
          Object.keys returns string[] on purpose. Structural typing lets a
          Product carry extra properties (stock here), so typing the keys as
          (keyof Product)[] would be a lie. Indexing with a plain string is
          TS7053; compare the key against known names first, which narrows it.
  - id: correlated-keys
    file: examples/02_correlated_keys.ts
    title: A generic key keeps key and value together
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "9-12"
        note: >
          keyof Settings and Settings[keyof Settings] are two independent
          unions, so any key may pair with any field's value type, including
          "pageSize" with "tr-TR".
      - lines: "13-16"
        note: >
          K records which key was passed, and value must be Settings[K] for that
          K. The mismatched call is TS2345; the message shows the literal widened
          to string.
      - lines: "18-23"
        note: >
          The loose call compiled and stored a string where every reader expects
          a number. Nothing at run time noticed, which is why the type has to.
      - lines: "25-35"
        note: >
          A mapped type, { [K in keyof T]: ... }, builds one property per key,
          and T[K] gives each handler its own parameter type: v is a number for
          pageSize and a boolean for darkMode. Lesson ts.type-level.utility
          covers mapped types properly.
  - id: generic-table
    file: examples/03_generic_table.tsx
    title: A generic React table typed by keyof
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "7-17"
        note: >
          In generic code keyof T is string | number | symbol, and a React key
          can't be a symbol, so columns use keyof T & string. render receives
          the whole row, typed T, which avoids the harder problem of typing a
          per-column value.
      - lines: "19-22"
        note: >
          A generic comparator over any key. T[keyof T] could be anything, so it
          compares String() of each value with a Turkish collator; numeric: true
          also orders "9" before "10".
      - lines: "24-41"
        note: >
          A generic component is a generic function that returns JSX. In a .tsx
          file a generic arrow must be written <T,>(props) => ..., because <T>
          parses as a JSX tag. Rows are keyed by rowKey, not by index, so sorting
          can't mix up row state.
      - lines: "53"
        note: >
          A fixed time zone keeps the output identical on every machine; tr-TR
          medium dates abbreviate months: 1 Haz 2023.
      - lines: "55-70"
        note: >
          T is inferred as Customer from rows, so rowKey's c is a Customer,
          sortBy must be a Customer key, each column key is checked, and render
          gets c.joined as a Date. <Table<Customer> ...> states T explicitly when
          inference can't.
      - lines: "71-73"
        note: >
          flushSync makes React render synchronously, so the DOM can be read on
          the next line. The rows come out in Turkish order: Çağrı, Işıl, Şule.
use_cases:
  - "Typed property helpers: get, pluck, groupBy(rows, key), sortBy(rows, key)"
  - "Updating settings or form state with correlated key and value: set(form, key, value)"
  - "Reusable components over the caller's data: tables with column definitions, selects, lists, filters"
  - "Deriving types from data: keyof typeof labels, (typeof ROUTES)[number]"
strengths:
  - "Property names become types, so renaming a field breaks every column definition and sort key at compile time; Java needs reflection or string constants, Go struct tags are unchecked strings"
  - "Generic components infer T from props: JSX call sites need no type arguments, and callbacks such as render get the row type for free"
  - "Indexed access types derive one type from another (Product[\"price\"], Row[\"tags\"][number]) instead of restating it"
weaknesses:
  - "Object.keys and for...in give string, not keyof T; iterating keys safely needs a known key list or a narrowing check"
  - "keyof T in generic code includes number and symbol, so string operations need keyof T & string"
  - "Wrapping a generic component in memo or forwardRef loses its type parameter: memo(Table) infers T as unknown"
  - "Typing a per-column value (a formatter taking T[K]) needs mapped types and runs into correlated-union limits inside the component"
pitfalls:
  - title: "Independent key and value unions"
    detail: "(key: keyof S, value: S[keyof S]) accepts every pairing of key and value. Capture the key in a type parameter."
    bad: "function set(s: Settings, key: keyof Settings, value: Settings[keyof Settings]) { ... }"
    good: "function set<K extends keyof Settings>(s: Settings, key: K, value: Settings[K]) { ... }"
  - title: "Casting Object.keys to keyof T"
    detail: "The cast compiles, and at run time the array also holds the extra properties the object carried in. Iterate a list of the keys you know."
    bad: "for (const k of Object.keys(p) as (keyof Product)[]) show(k, p[k]);"
    good: "for (const k of [\"name\", \"price\"] as const) show(k, p[k]);"
  - title: "Generic arrow components in .tsx"
    detail: "const Table = <T>(props: TableProps<T>) => ... is parsed as a JSX element and fails to compile (TS17008). Add a trailing comma or use a function declaration."
    bad: "const Table = <T>(props: TableProps<T>) => <table />;"
    good: "const Table = <T,>(props: TableProps<T>) => <table />;"
  - title: "The array index as the row key in a sortable list"
    detail: "When rows are reordered, index keys make React reuse the wrong row's state. Take a key function (rowKey) from the caller."
quiz:
  - q: "type M = { [k: string]: boolean }. What is keyof M?"
    options: ["string", "string | number", "never"]
    answer: 1
    why: "JavaScript converts numeric keys to strings, so obj[0] is obj[\"0\"] and a string index signature also accepts numbers."
  - q: "Why does Object.keys(product) return string[] instead of (keyof Product)[]?"
    options: ["A bug kept for compatibility", "A Product value may carry more properties than the type lists", "keyof is not available at run time"]
    answer: 1
    why: "Structural typing allows extra properties, so the runtime key list can be longer than keyof Product."
  - q: "set(s: Settings, key: keyof Settings, value: Settings[keyof Settings]). What does set(s, \"pageSize\", \"tr-TR\") do?"
    options: ["Compiles, and stores a string in pageSize", "TS2345", "Throws at run time"]
    answer: 0
    why: "Key and value are separate unions. With <K extends keyof Settings> and value: Settings[K] the call is rejected."
  - q: "In a .tsx file, which declaration of a generic component compiles?"
    options: ["const List = <T>(p: Props<T>) => <ul />", "const List = <T,>(p: Props<T>) => <ul />", "Neither: components can't be generic"]
    answer: 1
    why: "<T> alone starts a JSX element. A trailing comma, a constraint (<T extends object>) or a function declaration avoids the ambiguity."
further_reading:
  - title: "TypeScript Handbook: Keyof Type Operator"
    url: "https://www.typescriptlang.org/docs/handbook/2/keyof-types.html"
  - title: "TypeScript Handbook: Indexed Access Types"
    url: "https://www.typescriptlang.org/docs/handbook/2/indexed-access-types.html"
  - title: "TypeScript Handbook: Generics (using type parameters in generic constraints)"
    url: "https://www.typescriptlang.org/docs/handbook/2/generics.html"
  - title: "TypeScript 2.9 release notes: generic type arguments in JSX elements"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-2-9.html"
  - title: "react.dev: Rendering Lists (keys)"
    url: "https://react.dev/learn/rendering-lists"
---

## Concept

Two type operators turn property names into types:

- **`keyof T`** is the union of `T`'s keys as literal types: `keyof Product` is
  `"id" | "name" | "price" | "tags"`. For a string index signature it is
  `string | number`, because JavaScript converts numeric keys to strings.
- **Indexed access `T[K]`** is the type stored under `K`: `Product["price"]` is `number`,
  `Product[keyof Product]` the union of all property types, and `T[number]` an array's
  element type.

Both arrived in TypeScript 2.1. Combined with a type parameter they give the key a memory:
`function get<T, K extends keyof T>(obj: T, key: K): T[K]` infers `K` as the literal key
you passed, so the result has that property's type, and a typo is a compile error.

A React component is a function, so it can be generic too. `function Table<T>(props:
TableProps<T>)` infers `T` from the `rows` you pass, and every other prop (column keys,
sort key, render callbacks) is checked against that row type. TS 2.9 added explicit
type arguments in JSX, `<Table<Customer> ... />`, for the cases inference can't settle.

## Coming from C/C++/Java/Python/JS

- Java: there is no type for "a field name of `Product`". You use strings plus
  reflection, or method references (`Product::getPrice`) that can't be enumerated. `keyof`
  makes the field name itself a checked type.
- C++: closest are pointers to members (`&Product::price`), which do carry the member's
  type, but there is no way to get "all member names" as a type.
- Python: `Literal["id", "name"]` must be written by hand; `TypedDict` keys aren't
  derived automatically the way `keyof` derives them.
- Go: struct tags such as `json:"price"` are strings nobody checks, and generics can't
  abstract over field names. Code that needs "any field of T" uses reflection.
- JS/React: `Object.keys` and `for...in` behave as always, and TypeScript types their keys
  as `string` on purpose. Generic components are ordinary function components with a type
  parameter; nothing changes at run time.

## Design notes

Use `K extends keyof T` whenever a function takes a key and a value or returns a
property's value; it is the cheapest way to keep pairs consistent. Accept that
`Object.keys` returns `string[]`: when you need to iterate keys, iterate a list you control
(`["name", "price"] as const`) and let the list's literal types do the narrowing.

For components, make generic the ones that render the caller's data (tables, lists,
selects), and give them a small surface: rows, a key function, column definitions whose
`key` is `keyof T & string`, and render callbacks that receive the whole typed row.
Per-column value types (`format: (value: T[K]) => string`) are possible with mapped
types but quickly need helper functions to stay type-safe inside the component; reach for
them only when the row-level callback really isn't enough. Generic components also don't
survive `memo` or `forwardRef` with their type parameter intact, so memoize the data you
pass in rather than the component, or cast the wrapped component back to its generic
signature in one place.
