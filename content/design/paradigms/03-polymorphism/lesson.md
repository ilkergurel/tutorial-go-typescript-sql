---
id: design.paradigms.polymorphism
title: "The same problem: polymorphism (Go interfaces vs TS unions vs modeling variants in SQL)"
track: design
module: paradigms
order: 3
level: intermediate
minutes: 14
since: go1.21
prerequisites: [design.paradigms.side-by-side, go.interfaces.implicit, ts.narrowing.discriminated-unions]
compare_with: [go.interfaces.implicit, go.interfaces.assertions, ts.narrowing.discriminated-unions, ts.narrowing.guards, ts.architecture.patterns, sql.tables-types.jsonb-arrays, sql.modeling.anti-patterns]
tags: [polymorphism, interfaces, discriminated-unions, variants, expression-problem, modeling, comparison]
summary: >
  Discounts come in kinds: Go models them as an open set of types behind an interface,
  TypeScript as a closed union of tagged objects, SQL as rows with a kind column and a
  CHECK per shape. Each choice makes one kind of change cheap and leaves the other
  checked or unchecked.
examples:
  - id: go-interfaces
    file: examples/01_interfaces.go
    title: "Go: an open set of types behind one method"
    runtime: go
    check: exact
    annotations:
      - lines: "10-12"
        note: >
          The interface names the behavior, not the variants. Nothing lists which
          types implement it, so code in other packages can add discounts without
          editing this file (lesson go.interfaces.implicit).
      - lines: "14-20"
        note: >
          Each variant is its own struct, so a Percent can't carry an amount: the
          illegal mix is unrepresentable here too. The builtin max (Go 1.21)
          clamps the price at zero.
      - lines: "22-30"
        note: >
          The open set paying off: Chain is a new variant built from the others,
          and main's loop calls Apply on it without a single edit elsewhere. In
          TypeScript this would mean editing the union.
      - lines: "34-42"
        note: >
          The other axis. An operation the interface doesn't have (here, naming
          the kind) needs a type switch, and Go has no sum types, so the compiler
          can't know the switch misses Chain. It falls through at run time.
      - lines: "56"
        note: >
          json.Marshal writes the dynamic value's fields, not its type:
          {"Pct":15} doesn't say it was a Percent, and decoding it back into a
          Discount fails. At a boundary, a variant has to become tagged data again.
  - id: ts-unions
    file: examples/02_unions.ts
    title: "TypeScript: a closed union of tagged objects"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "2-4"
        note: >
          The union lists every variant, and each is plain data with a kind tag.
          This is the closed set Go can't declare, and the same shape as the
          table's rows and the API's JSON.
      - lines: "6-13"
        note: >
          Each case narrows d, so d.percent exists only under "percent". There is
          no default branch on purpose: with the return type declared, a missing
          case is a compile error (lines 29-30).
      - lines: "22"
        note: >
          JSON.stringify keeps the tag, so the value survives a round trip
          through an API or localStorage unchanged. Compare the Go example's
          {"Pct":15}.
      - lines: "25-27"
        note: >
          A percent discount with an amount doesn't type-check (tsc 7.0.2 output).
          The SQL example's one_shape CHECK refuses the same row at run time.
      - lines: "29-30"
        note: >
          The trade-off of a closed set: adding a variant means editing the union,
          and tsc then points at every switch that doesn't handle it. Go's Chain
          needed no edits, and its type switch got no warning either.
  - id: sql-variant-rows
    file: examples/03_variant_rows.sql
    title: "SQL: a kind column, per-variant columns and a CHECK"
    runtime: sql
    check: exact
    annotations:
      - lines: "4-6"
        note: >
          Rows can't have methods, so the variant is data: a kind column with a
          closed set of values (the TS union's tags) and one nullable column per
          variant field.
      - lines: "8-10"
        note: >
          This CHECK is the discriminated union, enforced on every write: a
          percent row has a percent and no amount, a fixed row the reverse. Without
          it, the table would accept any combination of NULLs.
      - lines: "19-22"
        note: >
          The switch over kind is a CASE in each query that needs it. CASE has no
          exhaustiveness check: a kind with no WHEN gets NULL, silently, so the
          CHECK on line 4 is what keeps this CASE complete.
      - lines: "28"
        note: >
          The row the TS checker rejected at compile time is rejected here at
          write time, with SQLSTATE 23514 and the constraint's name.
use_cases:
  - "Go interfaces for behavior supplied from elsewhere: storage backends, notifiers, payment providers, test fakes"
  - "TS unions for closed sets the UI must handle completely: request states, reducer actions, API payloads with a kind field"
  - "SQL kind column plus CHECK for variants that share most columns, such as discounts or payment methods; a table per variant when each carries a lot of data or its own foreign keys"
strengths:
  - "Go: a new variant needs no edits anywhere else; Chain composes the existing discounts without them knowing it exists"
  - "TS: a switch over the tag is checked for completeness (TS2366), which neither Go's type switch nor SQL's CASE ever is"
  - "TS: variants are plain data, so JSON keeps the tag; Go's json.Marshal of an interface value loses which variant it was"
  - "SQL: the one_shape CHECK makes a mixed row impossible for every writer, a guarantee the TS type gives only to code tsc compiled"
weaknesses:
  - "Go: no sum types; a type switch doesn't know the set of variants, so a forgotten case falls through to a default at run time"
  - "TS: only the module that owns the union can add a variant, and every switch over it must change in the same commit"
  - "SQL: a new kind makes every CASE without ELSE return NULL without warning, and nothing finds those queries for you"
  - "SQL: a single table grows a nullable column per variant field, and the shape CHECK has to be kept in step with the kind list"
pitfalls:
  - title: "Serializing a Go interface value as it is"
    detail: "json.Marshal writes {\"Pct\":15} with no variant name, and json.Unmarshal into a Discount field fails: \"cannot unmarshal object into Go struct field Offer.Discount of type main.Discount\". Put a kind field on the wire and switch on it when decoding, which gives the TS union's shape."
    bad: "json.Marshal(Offer{Code: \"KASIM15\", Discount: Percent{15}}) // {\"Discount\":{\"Pct\":15}}"
    good: "json.Marshal(wireDiscount{Kind: \"percent\", Percent: 15}) // {\"kind\":\"percent\",\"percent\":15}"
  - title: "A default branch that switches off the exhaustiveness check"
    detail: "With a default that returns a value, a new variant compiles and quietly takes that branch. Leave the default out (with a declared return type), or assign the narrowed value to never in it so a missing case is a compile error."
    bad: "default: return priceKurus;"
    good: "default: { const unhandled: never = d; throw new Error(`unknown discount ${JSON.stringify(unhandled)}`); }"
  - title: "PostgreSQL table inheritance for variants"
    detail: "CREATE TABLE fixed_discounts (...) INHERITS (discounts) looks like subclassing, but the docs list the catch: unique constraints, primary keys and foreign keys apply to single tables, not their children. The same code can exist in two child tables, and no foreign key can point at \"any discount\". Use one table with a CHECK, or a base table plus a table per variant sharing its key."
  - title: "Sealing a Go interface by convention only"
    detail: "When the set of variants really is closed (AST nodes, protocol messages), add an unexported method to the interface, as go/ast does with exprNode(). Only types in your package can then satisfy it, and a type switch with a panicking default documents that the set is complete."
quiz:
  - q: "A new discount kind is added in all three languages. Where does a compiler point at the code that must handle it?"
    options:
      - "Go's kind() type switch"
      - "TS's apply() switch"
      - "SQL's CASE expression"
    answer: 1
    why: "Only the TS union is a closed type the checker knows; Go's type switch and SQL's CASE fall through to a default or NULL at run time."
  - q: "What does json.Marshal produce for a Percent{15} held in a Discount interface?"
    options:
      - "{\"kind\":\"percent\",\"Pct\":15}"
      - "{\"Pct\":15}"
      - "An error: interfaces can't be marshaled"
    answer: 1
    why: "Marshal encodes the dynamic value's exported fields; the type name is not part of the output."
  - q: "Another package needs to add a discount type without editing yours. Which model allows that?"
    options:
      - "The Go interface"
      - "The TS union"
      - "The SQL CHECK on kind"
    answer: 0
    why: "Go interfaces are satisfied implicitly by any type with the methods. The union and the CHECK list every variant in one place."
  - q: "After the kind CHECK is widened to allow 'bogo', what does the example's CASE return for a bogo row?"
    options:
      - "An error"
      - "NULL"
      - "The price unchanged"
    answer: 1
    why: "A CASE with no matching WHEN and no ELSE returns NULL."
further_reading:
  - title: "The Go Programming Language Specification: Interface types"
    url: "https://go.dev/ref/spec#Interface_types"
  - title: "The Go Programming Language Specification: Type switches"
    url: "https://go.dev/ref/spec#Type_switches"
  - title: "TypeScript Handbook: Narrowing (discriminated unions, exhaustiveness checking)"
    url: "https://www.typescriptlang.org/docs/handbook/2/narrowing.html#exhaustiveness-checking"
  - title: "PostgreSQL docs: Inheritance (and its caveats)"
    url: "https://www.postgresql.org/docs/current/ddl-inherit.html"
  - title: "PostgreSQL docs: Conditional Expressions (CASE)"
    url: "https://www.postgresql.org/docs/current/functions-conditional.html"
---

## Concept

Polymorphism is one operation over values of several kinds. Every design for it answers
two questions: who can add a new kind, and who can add a new operation, and which of
those changes the compiler checks. The three languages answer differently.

- **Go: an open set of types behind an interface.** The interface names the behavior;
  any type with the methods is a variant, in any package, without declaring it. Adding a
  variant is free. Adding an operation means a method on every type, or a type switch
  the compiler can't check, because Go has no sum types.
- **TypeScript: a closed union of tagged objects.** The union lists every variant, each
  is plain data with a literal `kind`, and a switch on the tag narrows to the variant's
  fields. Adding an operation is a new function with a switch the checker verifies;
  adding a variant means editing the union, after which tsc points at every switch that
  misses it. TypeScript can also do Go's style (objects with methods, matched
  structurally), and uses it for strategies and adapters (lesson ts.architecture.patterns).
- **SQL: variants are rows.** A row has no methods, so the kind is a column, each
  variant's fields are columns, and a CHECK states which columns go with which kind.
  Operations are CASE expressions in queries, and CASE checks nothing.

All three examples price the same 200,00 TL basket with the same discounts. Go then adds
a variant without editing anything; TypeScript shows what adding one costs; SQL refuses
a row whose shape doesn't match its kind.

## Coming from C/C++/Java/Python/JS

- **Java**: a Go interface is a Java interface without `implements`. The TS union is Java
  21's sealed interface with records and a pattern-matching `switch`, which the compiler
  also checks for completeness. JPA's inheritance strategies map to SQL designs:
  `SINGLE_TABLE` is this example, `JOINED` is a table per variant sharing the base key.
- **C++**: virtual functions are the Go side; `std::variant` with `std::visit` is the TS
  side, including the compile error when a visitor misses an alternative.
- **C**: the TS union and the SQL table are both the tagged union you write by hand
  (`struct { enum kind; union { ... }; }`), with the "forgot to check the tag" bug
  removed by the checker and by the CHECK respectively. Go's interface is a struct of
  function pointers that the compiler builds for you.
- **Python**: `typing.Protocol` is Go's model; dataclasses with `match` are the union,
  with completeness checked only by a type checker such as mypy (`assert_never`).
- **Rust**: `enum` with `match` is TypeScript's union; traits are Go's interfaces.

## How it works

The two axes, side by side (the "expression problem"):

```text
                     add a variant                         add an operation
Go interface         a new type, anywhere; no edits        a method on every type, or a
                                                           type switch nobody checks
TS union             edit the union; tsc flags every       a new function; its switch
                     switch that misses it                 is checked for completeness
SQL kind column      widen the CHECK, add columns; every   a CASE in the query; no ELSE
                     CASE silently yields NULL for it      means NULL for unknown kinds
```

At the boundaries between them, the closed, tagged form wins: JSON and table rows carry
a `kind` field, because data has no methods. A Go service decodes `{"kind":"percent",...}`
with a switch on the tag into its interface types, works with them through methods, and
encodes them back with the tag.

## Design notes

Pick the model by the change you expect. When other code must be able to plug in new
kinds (payment providers, storage backends, fakes in tests), use an interface: that is
what Go's are for. When the set is fixed and every consumer must handle every case
(request states, messages between client and server, the statuses of a run), use a
closed form: a TS union, and in Go an interface sealed with an unexported method plus a
type switch whose default panics, since the compiler won't check it for you.

Design the wire format as the TS union and let each side map it. The Go side converts
tags to types at the edge and keeps the conversion in one function, the same "translate
once per boundary" rule as for errors.

In the schema, a single table with a kind column and a shape CHECK suits variants that
share most columns. Move to a base table plus one table per variant when variants carry
many fields or need their own foreign keys, and to a jsonb payload when variant
attributes are numerous and rarely filtered on (lesson sql.tables-types.jsonb-arrays).
Avoid the "polymorphic foreign key" (a `target_type` plus `target_id` pair), which no
foreign key can check; sql.modeling.anti-patterns shows the alternatives.
