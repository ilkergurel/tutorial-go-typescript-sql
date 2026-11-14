---
id: design.paradigms.side-by-side
title: "Paradigms side by side: Go, TS, SQL"
track: design
module: paradigms
order: 1
level: intermediate
minutes: 13
since: go1.23
prerequisites: [go.composite.slices-and-maps, ts.mental-model.structural, sql.basics.set-based]
compare_with: [sql.basics.set-based, ts.mental-model.structural, go.interfaces.implicit, ts.everyday-types.unions-enums, sql.constraints.fk-check, go.design.copying]
tags: [paradigms, comparison, explicit, structural-typing, declarative, constraints]
summary: >
  One report written three times: Go spells out the loop, the map and the sort; TypeScript
  describes shapes and lets the checker find mismatches; SQL describes the result and lets
  the planner choose. Each puts the same rule in a different place: run time, compile time,
  write time.
examples:
  - id: go-explicit
    file: examples/01_explicit.go
    title: "Go: write the algorithm, check the value yourself"
    runtime: go
    check: exact
    annotations:
      - lines: "10-17"
        note: >
          Go's way to spell an enum: a named type and some constants. The set is
          not closed. "pased" on line 63 is an untyped constant, and untyped
          constants are assignable to any type they can represent, so the
          compiler accepts it. Compare line 2 of the TS example.
      - lines: "30-33"
        note: >
          You choose the data structure (a hash map) and how it is filled. A
          missing key yields the zero Tally, so there is no "first time" branch:
          Go's zero values do the work JavaScript's `?? 0` would.
      - lines: "34-41"
        note: >
          The rule lives here, in code you wrote, and runs at run time. The
          compiler doesn't check that a switch covers every constant, so the
          default branch is what turns a bad value into an error value.
      - lines: "44-46"
        note: >
          Map iteration order is random, so sorting is part of the algorithm, not
          a nicety. maps.Values with slices.SortedFunc (Go 1.23) collects and
          sorts in one call; cmp.Or (Go 1.22) chains the sort keys.
      - lines: "63-64"
        note: >
          The bad value compiles and is found only when report runs, as an
          ordinary return value the caller must look at.
  - id: ts-typed-shapes
    file: examples/02_typed_shapes.ts
    title: "TypeScript: describe the shapes, let the checker work"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "2"
        note: >
          A literal union is a closed set the checker enforces on every
          expression it can see. This one type replaces Go's named type, its
          constants and its default branch, but only at compile time: it is
          erased before the code runs.
      - lines: "4-8"
        note: >
          An interface names a shape, not a class. Any object with these three
          members is a Run, whoever built it; readonly is checked by the compiler
          only.
      - lines: "16-21"
        note: >
          The same map-then-sort as Go, as a pipeline: Map.groupBy (ES2024) builds
          the map, filter counts, sort orders. It reads declaratively but runs
          exactly as written, in this process, over data already fetched. That is
          the difference from the SQL version.
      - lines: "36-37"
        note: >
          Where Go compiled "pased", tsc refuses it (tsc 7.0.2 output). Mistakes
          in code are caught before anything runs.
      - lines: "39-42"
        note: >
          Mistakes in data are not. JSON.parse returns any, the cast is accepted
          unchecked, and the run with status "pased" is silently counted as
          neither passed nor failed: the last output line. At a trust boundary,
          validate (lesson ts.services.zod) instead of casting.
  - id: sql-declarative
    file: examples/03_declarative.sql
    title: "SQL: describe the result and the rule, let the database do the rest"
    runtime: sql
    check: exact
    annotations:
      - lines: "5"
        note: >
          The rule is part of the table. It is checked on every INSERT and UPDATE
          from every client: this script, a Go service, psql, a migration.
      - lines: "12-17"
        note: >
          The whole report as a description: which groups, which counts, which
          order. FILTER (WHERE ...) counts only matching rows (PostgreSQL 9.4).
          There is no loop, no map and no variable to get wrong.
      - lines: "20-22"
        note: >
          The plan shows the algorithm the Go example wrote by hand: HashAggregate
          is the map, Sort is slices.SortedFunc. On a bigger table with an index
          the planner may choose differently, and the query text stays the same.
      - lines: "25"
        note: >
          The bad value is refused when it is written, with SQLSTATE 23514. In this
          runtime the first error ends the script; in an application, the Go layer
          translates this into a 4xx (lesson design.paradigms.errors).
use_cases:
  - "Deciding where a rule belongs: a CHECK or foreign key for invariants every client must respect, a Go switch for workflow decisions, a TS union for UI states and API shapes"
  - "Moving a report from a Go loop into one GROUP BY query, or reviewing a pull request that does"
  - "Translating an idiom when you switch languages mid-task: map plus sort in Go, Map.groupBy plus sort in TS, GROUP BY plus ORDER BY in SQL"
strengths:
  - "Go: every step and its cost is on the page; a reviewer sees the map, the copy back into it and the sort, with no stream pipeline hiding allocations as in Java"
  - "TS: a misspelled status in source fails the build (TS2345), where Go compiles \"pased\" as a Status and only the default branch catches it, at run time"
  - "SQL: a CHECK constraint rejects bad data from every client, psql included; the Go and TS checks protect only their own code paths"
  - "SQL: the same query text can be run as a hash aggregate or a sort, chosen per data size; the algorithm in the Go loop is fixed when you write it"
weaknesses:
  - "Go: no closed sets; named string types accept any constant, and the compiler doesn't check a switch for completeness"
  - "TS: types are erased, so data from JSON.parse, fetch or a cast is never checked unless you validate it"
  - "SQL: a bad value is refused at write time, far from the code that built it, as a SQLSTATE the application has to translate"
  - "The stack carries three spellings of one concept (Go constants, a TS union, a CHECK list) that drift apart unless one is generated from another or a test compares them"
pitfalls:
  - title: "Casting data at the trust boundary"
    detail: "A cast tells the checker to believe you. Everything after it is type-checked against a shape the data may not have, as the fromApi line shows. Parse unknown input with a schema instead."
    bad: "const runs = (await res.json()) as Run[];"
    good: "const runs = RunList.parse(await res.json()); // a zod schema: throws on bad data"
  - title: "Treating a Go named type as a closed enum"
    detail: "Status(\"pased\") compiles, and so does any untyped constant where a Status is expected. Convert strings from outside through one function that returns an error for unknown values, and keep a default branch in every switch."
    bad: "s := Status(r.FormValue(\"status\"))"
    good: "s, err := ParseStatus(r.FormValue(\"status\")) // returns an error for anything but passed/failed"
  - title: "Fetching rows to aggregate them in application code"
    detail: "The Go and TS versions need every run in memory first. Against a database, send the GROUP BY and receive three rows instead of three million (lesson sql.basics.set-based)."
  - title: "Expecting the same string order in all three"
    detail: "Go's cmp.Compare and JavaScript's < compare bytes and UTF-16 code units, so \"Çay\" sorts after \"Su\". PostgreSQL orders by the column's collation; this database's default (the builtin provider) compares code points, so 'Çay' < 'Su' is false there too. Turkish alphabetical order needs a locale: Intl.Collator(\"tr\") or localeCompare(b, \"tr\") in TS, golang.org/x/text/collate in Go, COLLATE \"tr-x-icu\" in SQL."
quiz:
  - q: "The value \"pased\" is offered to all three examples. Which one rejects it before any code runs?"
    options:
      - "Go: the named type Status rejects it"
      - "TypeScript, for the literal written in source"
      - "SQL: the CHECK constraint"
    answer: 1
    why: "tsc rejects the literal at compile time. Go accepts the untyped constant and the switch catches it at run time; SQL refuses it when the row is written."
  - q: "fromApi carries status \"pased\" through `as Run[]`. What happens?"
    options:
      - "JSON.parse throws because the data doesn't match Run"
      - "tsc reports an error on the cast"
      - "Nothing checks it; report counts the run as neither passed nor failed"
    answer: 2
    why: "Types are erased and a cast from any is unchecked; only a run-time validation would catch it."
  - q: "Which version can switch from hashing to sorting without anyone editing it?"
    options:
      - "The Go loop"
      - "The TS pipeline"
      - "The SQL query"
    answer: 2
    why: "The query describes the result; the planner chooses HashAggregate or a sort from statistics and indexes. The Go and TS code run the algorithm they spell out."
  - q: "Where must \"status is passed or failed\" live so that a psql session can't break it?"
    options:
      - "In the Go service's switch"
      - "In the TS union"
      - "In a CHECK constraint on the table"
    answer: 2
    why: "Only the database sees every write. Application checks are for good error messages; the constraint is the guarantee."
further_reading:
  - title: "Effective Go"
    url: "https://go.dev/doc/effective_go"
  - title: "The Go Programming Language Specification: Assignability"
    url: "https://go.dev/ref/spec#Assignability"
  - title: "TypeScript Handbook: Type Compatibility"
    url: "https://www.typescriptlang.org/docs/handbook/type-compatibility.html"
  - title: "TypeScript Handbook: Literal Types"
    url: "https://www.typescriptlang.org/docs/handbook/2/everyday-types.html#literal-types"
  - title: "PostgreSQL docs: Check Constraints"
    url: "https://www.postgresql.org/docs/current/ddl-constraints.html#DDL-CONSTRAINTS-CHECK-CONSTRAINTS"
  - title: "PostgreSQL docs: Aggregate Expressions (FILTER)"
    url: "https://www.postgresql.org/docs/current/sql-expressions.html#SYNTAX-AGGREGATES"
---

## Concept

The three languages of this stack were designed around different ideas, and code reads
best when it follows the idea of the language it is written in.

- **Go is simple and explicit.** You write the algorithm: the loop, the map, the sort, the
  check, the error return. The language has few features and no hidden control flow (no
  exceptions, no inheritance, no implicit conversions between named types), so what a
  function does and what it costs is on the page. Rules live in code that runs.
- **TypeScript is structural and type-driven.** The runtime is JavaScript: dynamic,
  objects as bags of properties, functions as values. On top sits a type system that
  describes shapes, not classes. You design by writing the types first, a literal union
  for the statuses, a readonly array for the input, and the checker finds every place
  that disagrees. Types are erased, so they constrain the code you compile, never the
  data that arrives at run time.
- **SQL is declarative and set-based.** You describe the result and the invariants; the
  planner chooses the algorithms, and constraints guard every write from every client.
  The unit of work is a set of rows, and the data outlives every program that touches it.

The examples solve one small problem in each: group test runs by requirement, count
passes and failures, worst first. Then each is offered the same bad status, `"pased"`.
Go finds it at run time, in a branch you wrote. TypeScript finds it at compile time if
it is in the source, and not at all if it arrives as data. SQL finds it at write time,
whoever writes.

## Coming from C/C++/Java/Python/JS

- **C**: Go is the closest relative here: structs are values, loops are explicit, errors
  are return values. It adds garbage collection, slices, maps, and interfaces that types
  satisfy without declaring it.
- **Java**: Go drops class hierarchies and exceptions; TypeScript keeps classes but rarely
  needs them, because any object of the right shape fits an interface; SQL resembles the
  Streams API in spirit, but it runs next to the data and picks its own execution plan.
- **C++**: TypeScript's structural checks feel like templates and concepts applied to
  every type, except that nothing is generated: the checker runs, then the types vanish.
- **Python**: TypeScript is JavaScript with a mypy that runs on every keystroke and
  blocks the build. Go shares Python's "one obvious way" culture, enforced by `gofmt` and
  a compiler.
- **JS**: TypeScript changes nothing at run time, which is both its strength (all your
  idioms carry over) and the reason the `fromApi` line goes unnoticed.

## How it works

The same concept, spelled three ways:

```text
concept           Go                          TypeScript                  SQL
record            struct (a value)            interface / type (a shape)  table row
collection        slice, map                  array, Map                  table (a set of rows)
closed set        named type + consts (open)  literal union (closed)      CHECK, enum, lookup table
grouping          loop + map                  Map.groupBy                 GROUP BY
ordering          slices.SortFunc             .sort / toSorted            ORDER BY
invariant         code and tests              types, at compile time      constraints, at write time
failure           error value                 throw, or a Result union    SQLSTATE error, rollback
who chooses       you                         you                         the planner
```

## Design notes

In a Go + TypeScript + PostgreSQL system, let each language do what it was built for.
Invariants that must hold whoever writes (not null, unique, references exist, status in
a set) belong in the schema, because only the database sees every write. Filtering,
joining and aggregating belong in queries, next to the data and its indexes. Go owns
workflows, side effects and I/O: calling other services, translating database errors
into API responses, deciding what happens next. TypeScript owns the UI's model, with
types that mirror the API contract and validation where data enters, because the types
will not be there at run time.

The price is that one concept now has three spellings: a CHECK list, Go constants and a
TS union. Make one of them the source and derive the others (sqlc generates Go from the
schema, an OpenAPI document generates TS types), or add a test that compares them.

Each language also has a style that fights it: getters and setters on Go structs, deep
class hierarchies in TypeScript, row-by-row loops around SQL. The rest of this module
takes one problem at a time (errors, polymorphism, concurrency, absence, immutability)
and shows each language's idiomatic answer next to the others; design.patterns.anti-patterns
collects the styles to avoid.
