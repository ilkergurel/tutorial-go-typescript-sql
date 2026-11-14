---
id: design.paradigms.immutability
title: "The same problem: immutability"
track: design
module: paradigms
order: 6
level: advanced
minutes: 14
since: go1.21
prerequisites: [design.paradigms.side-by-side, go.composite.structs, ts.objects-classes.spread]
compare_with: [go.design.copying, go.composite.structs, go.composite.slices-and-maps, ts.objects-classes.spread, ts.everyday-types.objects, ts.react.state, sql.modifying.soft-delete, sql.modeling.temporal, sql.server-side.triggers, sql.transactions.vacuum]
tags: [immutability, copy-on-write, readonly, freeze, append-only, history, comparison]
summary: >
  A price changes, and last month's order must keep the old one: Go protects the old
  list by API and copy-on-write, TypeScript by readonly types and Object.freeze, SQL by
  an INSERT-only table. All three make a change a new value, and all three are shallow
  somewhere.
examples:
  - id: go-copy-on-write
    file: examples/01_copy_on_write.go
    title: "Go: immutable by API, copy-on-write with maps.Clone"
    runtime: go
    check: exact
    annotations:
      - lines: "10-16"
        note: >
          Go's const covers only compile-time constants, so a struct, map or
          slice can't be declared immutable. The guarantee comes from the API: an
          unexported field (outside this package, nobody can reach it) and a
          constructor that clones its input, so the caller keeps no alias to the
          map.
      - lines: "20-24"
        note: >
          A change produces a new PriceList over a cloned map (maps.Clone, Go
          1.21). The old list is never touched, so the order's snapshot and any
          goroutine reading it need no lock. The price is O(n) per change.
      - lines: "27"
        note: >
          The trap: a value receiver looks like it works on a copy, but copying a
          struct copies the map header, not the map. Writing through it changes
          every copy (lesson go.composite.structs).
      - lines: "38-40"
        note: >
          The order still says 1250 after the November change: With built a new
          list instead of editing the one the order holds.
      - lines: "42-45"
        note: >
          "Changing a copy" changed november, because both structs point at one
          map. october is unaffected only because With cloned the map earlier.
  - id: ts-readonly-freeze
    file: examples/02_readonly_freeze.ts
    title: "TypeScript: readonly for the checker, freeze for the runtime"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "2"
        note: >
          Readonly<T> makes every property read-only to the checker, at no
          run-time cost. The const on line 4 is a different thing: it stops
          reassigning the name october, not writing to the object.
      - lines: "7-10"
        note: >
          The TS form of With: spread copies october's properties into a new
          object and overrides one. It copies one level, which is all a flat
          price list needs.
      - lines: "12-13"
        note: >
          A direct write is a compile error (tsc 7.0.2 output). This is where most
          of TypeScript's immutability lives: in the checker.
      - lines: "15-22"
        note: >
          Types are erased and a cast removes readonly, so only Object.freeze
          stops the write at run time. ES modules are strict mode code, where the
          write throws a TypeError; sloppy-mode scripts ignore it silently.
      - lines: "24-27"
        note: >
          Shallow, like Go's struct copy: the outer object is frozen and typed
          readonly, the nested prices object is neither, and tsc accepts the
          write. as const or a recursive readonly type, plus a deep freeze, are
          needed for nested data.
  - id: sql-insert-only
    file: examples/03_insert_only.sql
    title: "SQL: an INSERT-only table, with history as a query"
    runtime: sql
    check: exact
    annotations:
      - lines: "2-7"
        note: >
          Each row is a fact that doesn't change: this sku cost this much from
          this date. The key includes valid_from, so a new price is a new row
          rather than a new value in an old one.
      - lines: "10-16"
        note: >
          A statement-level trigger refuses the statements that would rewrite
          history; TRUNCATE triggers must be FOR EACH STATEMENT anyway. In
          production, REVOKE UPDATE, DELETE and TRUNCATE from the application's
          role first; the trigger also stops the owner's ad-hoc fixes.
      - lines: "20"
        note: >
          The SQL form of With: insert the new version. The October row is still
          there for every order that was priced with it.
      - lines: "23-26"
        note: >
          History is a query, not a copy you had to keep: the newest price on or
          before the order date. The primary key's index on (sku, valid_from)
          serves this lookup.
      - lines: "29-31"
        note: >
          "Current" is also a query: DISTINCT ON keeps the first row per sku in
          the ORDER BY order, here the newest. A view over this query gives the
          application a current_prices table to read.
      - lines: "33"
        note: >
          The trigger's exception: SQLSTATE P0001, raise_exception. Because the
          script runs in one transaction, the error ends it.
use_cases:
  - "Prices, tax rates and exchange rates that orders and invoices must reproduce later: an INSERT-only table with valid_from, read with \"newest on or before\""
  - "Configuration or price lists shared between goroutines: an immutable value swapped as a whole (atomic.Pointer), never edited in place"
  - "UI state in TypeScript and React: readonly types plus spread updates, so a changed reference means changed data"
strengths:
  - "Go: an immutable value can be shared between goroutines without a mutex, since nobody writes it; a new version replaces the old one as a whole"
  - "TS: readonly turns accidental writes into compile errors at every call site at zero run-time cost, and spread makes a copy-on-write update one expression"
  - "SQL: an INSERT-only table keeps every past state answerable with a WHERE clause, and no client can rewrite what an order was charged"
  - "SQL: rows hold values and keys, never pointers, so copying a row can't alias anything, unlike Go maps and JS objects"
weaknesses:
  - "Go: no deep or declared immutability; it is a convention guarded by package boundaries, and maps.Clone costs a full copy per change"
  - "TS: readonly is erased, a cast removes it, and both readonly and Object.freeze are shallow"
  - "SQL: \"current state\" becomes a query that needs an index, corrections need a convention of their own, and the table only grows"
pitfalls:
  - title: "Handing out the internal map or slice"
    detail: "A getter that returns the map gives every caller write access to state the type promised not to change. Return a clone, or an iterator, which has no way to write."
    bad: "func (p PriceList) All() map[string]int64 { return p.prices }"
    good: "func (p PriceList) All() iter.Seq2[string, int64] { return maps.All(p.prices) }"
  - title: "Reading const as immutable"
    detail: "In TypeScript, const forbids reassigning the binding; the object stays writable. In Go, const exists only for numbers, strings, booleans and runes known at compile time. Neither makes data read-only."
    bad: "const october = { simit: 1250 }; october.simit = 0; // compiles and runs"
    good: "const october: Readonly<Record<string, number>> = Object.freeze({ simit: 1250 });"
  - title: "Updating in place where history matters"
    detail: "UPDATE prices SET kurus = 1500 erases what October's orders were charged. Insert a new version with a validity date, or copy the price into the order line when the order is placed; PostgreSQL 18's temporal keys can enforce non-overlapping periods (lesson sql.modeling.temporal)."
    bad: "UPDATE prices SET kurus = 1500 WHERE sku = 'simit';"
    good: "INSERT INTO prices VALUES ('simit', 1500, '2026-11-01');"
  - title: "Personal data in an append-only table"
    detail: "Nothing can be erased from a table that refuses DELETE, but KVKK requires personal data to be erased, destroyed or anonymized once the reason to keep it ends. Keep personal data in mutable tables referenced by key, and keep append-only tables to facts about prices, amounts and states."
quiz:
  - q: "After november := october.With(\"simit\", 1500), what does order.Prices.Price(\"simit\") return?"
    options:
      - "1500: the price list was updated"
      - "1250: With built a new list and left the order's one alone"
      - "0: the order's list was cleared"
    answer: 1
    why: "With clones the map and writes to the clone; the order keeps a reference to the untouched October list."
  - q: "copied := november; copied.SetInPlace(\"ayran\", 0). Why does november's ayran become 0?"
    options:
      - "SetInPlace has a pointer receiver"
      - "Copying the struct copied the map header, so both structs share one map"
      - "maps are copied lazily on first write"
    answer: 1
    why: "Struct assignment copies fields; a map field is a reference to the same hash table."
  - q: "const october = { simit: 1250 }. What does october.simit = 0 do?"
    options:
      - "Compile error: october is const"
      - "Runs and changes the object"
      - "Throws a TypeError at run time"
    answer: 1
    why: "const only prevents reassigning october. readonly types or Object.freeze are needed to protect the object."
  - q: "In the append-only table, how does simit's price change on 1 December?"
    options:
      - "UPDATE the row's kurus"
      - "DELETE the old row, INSERT the new one"
      - "INSERT a row with valid_from 2026-12-01"
    answer: 2
    why: "Every change is a new fact; the trigger refuses UPDATE and DELETE, and queries pick the newest row on or before a date."
further_reading:
  - title: "The Go Programming Language Specification: Constants"
    url: "https://go.dev/ref/spec#Constants"
  - title: "package maps (Clone, All)"
    url: "https://pkg.go.dev/maps"
  - title: "TypeScript Handbook: Object Types (readonly properties)"
    url: "https://www.typescriptlang.org/docs/handbook/2/objects.html#readonly-properties"
  - title: "MDN: Object.freeze()"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Object/freeze"
  - title: "PostgreSQL docs: CREATE TRIGGER"
    url: "https://www.postgresql.org/docs/current/sql-createtrigger.html"
  - title: "PostgreSQL docs: Introduction to MVCC"
    url: "https://www.postgresql.org/docs/current/mvcc-intro.html"
---

## Concept

The simit costs 12,50 TL in October and 15,00 TL from November. An order placed on 20
October must still say 12,50 when someone opens it in December. All three examples solve
this the same way: a change produces a new value and leaves the old one alone, so whoever
holds the old one can trust it. What differs is who enforces that.

- **Go enforces it by API.** `const` exists only for compile-time constants, so maps,
  slices and structs can't be declared read-only. An immutable type keeps its state in
  unexported fields, never writes it after construction, and offers methods that return a
  new value (`With`). Value semantics help, but shallowly: copying a struct copies its
  map and slice headers, not what they point to.
- **TypeScript enforces it in the checker, optionally at run time.** `const` fixes the
  binding, not the object. `readonly`, `Readonly<T>`, `ReadonlyArray<T>` and `as const`
  turn writes into compile errors and are erased. `Object.freeze` enforces at run time,
  one level deep. An update is a new object built with spread, sharing whatever didn't
  change.
- **SQL enforces it in the schema.** Tables are mutable by nature, but a table of facts
  can be append-only: a change is a new row with its validity date, history is a WHERE
  clause, and a trigger (or revoked privileges) refuses UPDATE and DELETE. Underneath,
  PostgreSQL already works this way: an UPDATE writes a new row version and leaves the
  old one for snapshots still reading it (lesson sql.transactions.acid-mvcc).

## Coming from C/C++/Java/Python/JS

- **C++**: Go has no `const` member functions or `const T&` parameters; immutability is
  a property of the type's API instead. TypeScript's `readonly` is the closest thing to
  `const`, and like `const` on a pointer it says nothing about what the pointee allows.
- **Java**: an immutable Go type is a final class with private final fields and
  `withX` methods; Java records are shallowly immutable in the same way, since a record
  holding a `List` can't stop that list from being changed. `Object.freeze` resembles
  an unmodifiable collection, except that it locks the object itself instead of wrapping
  it, and a failed write throws only in strict mode.
- **Python**: frozen dataclasses and tuples are shallow too: a tuple holding a list holds
  a mutable list. `dataclasses.replace(p, simit=1500)` is `With`.
- **JS**: `const` was never immutability; TypeScript adds `readonly` on top without
  changing what runs. React's state rule ("never mutate, replace") is copy-on-write, and
  it is why `prev !== next` can mean "changed".

## How it works

```text
                   Go                            TypeScript                      SQL
fixed binding      const (compile-time values)   const                           -
no writes          unexported fields, no         readonly, Readonly<T> (tsc);    trigger, or REVOKE UPDATE,
                   setters (package boundary)    Object.freeze (run time)        DELETE, TRUNCATE
make a change      With: maps.Clone, then write  { ...old, simit: 1500 }         INSERT a new version
how deep           struct copy is shallow        spread, readonly and freeze     a row holds values and
                                                 are shallow                     keys, never pointers
the past           keep a reference to the old   keep a reference to the old     WHERE valid_from <= date
                   value                         object                          ORDER BY ... LIMIT 1
```

## Design notes

Immutability is how the three languages share data safely. In Go, a value nobody writes
can be read from any goroutine without a lock; the usual pattern for configuration or a
price list is to build a new value and swap it in whole with an `atomic.Pointer`, which is
what MVCC does with row versions. In TypeScript, treat any state other code can see as
immutable, put `readonly` in function signatures, and update by copying the path to the
change; React depends on it. Reserve `Object.freeze` for shared module-level data that
code outside the type system could write.

In the schema, choose per table. Facts (orders, payments, ledger entries, price and rate
histories, audit events) belong in append-only tables: they are never wrong, only
superseded or corrected by a later row. Current state that people edit (a user's
profile, a requirement's title) belongs in ordinary tables, with history kept by a
trigger or a temporal design when it matters (sql.modeling.temporal), not by soft
delete. An append-only table writes no dead row versions, so it doesn't bloat;
autovacuum still visits it, triggered by inserts since PostgreSQL 13, to freeze old rows.

Every form is shallow somewhere, so state the depth you mean. A Go type that holds a
slice must clone it on the way in and on the way out; a TypeScript `Readonly<T>` over
nested data needs a recursive type or `as const`; an SQL row that references a mutable
row by key is only as stable as that row, which is why an order line copies the price
instead of pointing at the current one.
