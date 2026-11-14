---
id: ts.type-level.utility
title: "Mapped types and the built-in utilities: Partial, Pick, Omit, Record, Awaited..."
track: ts
module: type-level
order: 1
level: advanced
minutes: 14
since: ts4.5
prerequisites: [ts.generics.keyof]
compare_with: [go.composite.structs, go.http.json-validation, ts.react.components, ts.services.openapi]
tags: [mapped-types, utility-types, Partial, Pick, Omit, Record, Awaited, ReturnType, as-const]
summary: >
  Write the domain type once and compute its variants: the create payload, the patch,
  the per-field error map, the result of an async function. Mapped types are the loop
  behind every built-in utility, and you can write your own in one line.
examples:
  - id: derive
    file: examples/01_derive.ts
    title: One source type, its create and patch variants, and two traps
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "12-15"
        note: >
          Omit is opt-out and Pick is opt-in. Add a required dueDate to Requirement and
          every create() call missing it stops compiling, while the patch type ignores
          it until someone decides it is editable. Choose per type which failure mode
          you want.
      - lines: "18-24"
        note: >
          Readonly<Requirement> makes every property readonly, one level deep, at
          compile time only. Passing r1 to update(r: Requirement) still compiles:
          readonly properties are assignable to mutable ones, so the guarantee holds
          only while the value keeps its Readonly type.
      - lines: "29-32"
        note: >
          The patch type has no id, and a fresh object literal gets the excess
          property check, so { id } is rejected. tsc 7 prints the expanded type, with
          the union members in its own order.
      - lines: "34-39"
        note: >
          Omit is declared as Omit<T, K extends keyof any>, so any string is accepted.
          "createdOn" matches nothing and Oops is Requirement unchanged; the proof is
          that oops.createdAt still type-checks. Pick's K extends keyof T catches the
          same typo.
      - lines: "41-43"
        note: >
          Optional means "may be missing", and without exactOptionalPropertyTypes it
          also admits an explicit undefined. The spread then overwrites title with
          undefined while r3 is typed as a Requirement whose title is a string.
  - id: mapped
    file: examples/02_mapped.ts
    title: Writing your own mapped types
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "9-11"
        note: >
          [K in keyof T] loops over T's keys and T[K] reads each property's type, so
          each validator gets its own parameter type: v is a string for title and a
          number for priority. Mapping over keyof T keeps T's readonly and ? markers,
          which is how Partial and Readonly are built.
      - lines: "13-19"
        note: >
          The mapped type makes the rule set exhaustive: a form field without a
          validator is TS2739, the same check you would otherwise write as a unit
          test.
      - lines: "21-28"
        note: >
          In generic code, validators[key] has the type (value: T[typeof key]) => ...
          and value[key] is exactly T[typeof key], so the call needs no cast. The loop
          walks the validators, not the value: structural typing lets the value carry
          extra properties, and they have no rule to run.
      - lines: "30-31"
        note: >
          FieldErrors<T> has every key optional, so only failing fields appear. This
          shape is what a form component renders next to each input.
      - lines: "33-36"
        note: >
          A - removes a modifier, as Required<T> removes ? with -?. The as clause (TS
          4.1) renames or filters keys; mapping a key to never drops it. The condition
          is a conditional type over a template literal type, the next lesson's topic.
      - lines: "43-46"
        note: >
          Types compose inside out: Public drops _etag, then Mutable removes readonly
          from what is left, so copy.id can be reassigned.
  - id: from-values
    file: examples/03_from_values.ts
    title: Types read off values and functions with typeof, ReturnType and Awaited
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-5"
        note: >
          as const makes the array a readonly tuple of literals, and [number] reads
          the union of its elements. The list stays available at run time for menus
          and validation, which a type-only union is not.
      - lines: "14-29"
        note: >
          typeof in a type position reads the declared type of a value. ReturnType
          alone would give a Promise; Awaited unwraps it the way await does (TS 4.5),
          and an indexed access walks into the result. This is how you type the
          output of a library that doesn't export it.
      - lines: "39-44"
        note: >
          A Record over a finite union requires every key, so a new status breaks the
          build at every table that has to handle it. A Map or Partial<Record<...>>
          would let the missing entry through to run time.
use_cases:
  - "Request types derived from the entity: Omit for create payloads, Partial<Pick<...>> for PATCH bodies"
  - "Per-field maps over a form or DTO: validators, error messages, dirty flags, column definitions"
  - "Lookup tables that must cover a whole union: labels, colors, status transitions as Record<Status, ...>"
  - "Typing what a library returns without its cooperation: Awaited<ReturnType<typeof client.get>>, Parameters<typeof fn>, ComponentProps<typeof Button>"
strengths:
  - "One source of truth: renaming or adding a field in the entity updates every derived type, where Java needs a separate CreateRequest and UpdateRequest class (or MapStruct code generation) kept in sync by hand"
  - "A PATCH type is Partial<Pick<...>> in one line; Go needs a second struct with pointer fields to tell a missing field from a zero value"
  - "Mapped types are ordinary code you can read and write, so a project-specific utility (Validators<T>, FieldErrors<T>) costs one line instead of a code generator"
  - "Record over a union makes lookup tables exhaustive at compile time, the type-level version of a switch with a never check"
weaknesses:
  - "Omit doesn't check its keys and doesn't distribute over unions, two silent failures in the most used utility"
  - "Readonly is shallow and compile-time only; nothing freezes the object, and a Readonly<T> passes for a mutable T"
  - "Hovering a derived type often shows Partial<Pick<...>> instead of the resulting fields, and error messages print the expanded form, which gets long"
  - "Partial makes every field optional with no way to say which combinations are valid; a patch that must change at least one field needs a hand-written type or a runtime check"
pitfalls:
  - title: "A typo in Omit removes nothing"
    detail: "Omit<T, K extends keyof any> accepts any key, so the type silently stays whole. Use Pick where you can, or a one-line strict version."
    bad: "type Draft = Omit<Requirement, \"id\" | \"createdOn\">; // createdAt still required"
    good: "type StrictOmit<T, K extends keyof T> = Omit<T, K>;"
  - title: "Omit on a discriminated union keeps only the common fields"
    detail: "keyof (A | B) is the keys both variants share, so Omit<TraceEvent, \"id\"> becomes { kind: ... } and every variant's own fields disappear. Apply it per member with a distributive conditional type (next lesson)."
    bad: "type NoId = Omit<TraceEvent, \"id\">;"
    good: "type DistributiveOmit<T, K extends PropertyKey> = T extends unknown ? Omit<T, K> : never;"
  - title: "Partial lets undefined overwrite real values"
    detail: "{ ...r, ...patch } with patch = { title: undefined } stores undefined in a field typed string. Turn on exactOptionalPropertyTypes (lesson ts.mental-model.tsconfig), or drop undefined entries before merging."
  - title: "Record<string, T> lookups claim to always find something"
    detail: "owners[\"REQ-EXPORT\"] is typed string even when the key is missing. Use a Map, Partial<Record<string, T>>, or noUncheckedIndexedAccess so the miss shows up as undefined in the type."
quiz:
  - q: "You add a required dueDate: Date to Requirement. RequirementDraft = Omit<Requirement, \"id\" | \"createdAt\"> and RequirementPatch = Partial<Pick<Requirement, \"title\" | \"status\">>. What changes?"
    options:
      - "Both types gain dueDate"
      - "RequirementDraft requires dueDate, so create() calls without it stop compiling; RequirementPatch is unchanged"
      - "Nothing until you rebuild the derived types"
    answer: 1
    why: "Omit keeps every key it doesn't name, so new fields flow in; Pick keeps only the keys it names. Derived types are recomputed on every check."
  - q: "What does Omit<Requirement, \"createdOn\"> do when Requirement has createdAt but no createdOn?"
    options: ["Compile error TS2344", "Compiles and removes nothing", "Removes createdAt, the closest match"]
    answer: 1
    why: "Omit constrains its keys only to keyof any, so an unknown key is accepted and matches nothing. Pick<Requirement, \"createdOn\"> would be TS2344."
  - q: "async function load(): Promise<Row[]>. What is ReturnType<typeof load>?"
    options: ["Row[]", "Promise<Row[]>", "Awaited<Row[]>"]
    answer: 1
    why: "ReturnType reads the declared return type, which is the Promise. Awaited<ReturnType<typeof load>> gives Row[]."
  - q: "Which mapped type is Required<T>?"
    options: ["{ [K in keyof T]+?: T[K] }", "{ [K in keyof T]-?: T[K] }", "{ -readonly [K in keyof T]: T[K] }"]
    answer: 1
    why: "-? removes the optional modifier from each property; -readonly removes readonly, which is a different utility."
further_reading:
  - title: "TypeScript Handbook: Utility Types"
    url: "https://www.typescriptlang.org/docs/handbook/utility-types.html"
  - title: "TypeScript Handbook: Mapped Types"
    url: "https://www.typescriptlang.org/docs/handbook/2/mapped-types.html"
  - title: "TypeScript Handbook: Typeof Type Operator"
    url: "https://www.typescriptlang.org/docs/handbook/2/typeof-types.html"
  - title: "TypeScript 4.5 release notes: the Awaited type"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-4-5.html"
  - title: "TSConfig reference: exactOptionalPropertyTypes"
    url: "https://www.typescriptlang.org/tsconfig/#exactOptionalPropertyTypes"
---

## Concept

A **mapped type** builds an object type by looping over keys:
`{ [K in keyof T]: T[K] }` copies `T`, and changing the right-hand side, the modifiers or
the keys gives you every variant you need. The built-in utilities are mapped types you
don't have to write:

- `Partial<T>` and `Required<T>` add or remove `?`; `Readonly<T>` adds `readonly`.
- `Pick<T, K>` keeps the keys you name; `Omit<T, K>` keeps the rest.
- `Record<K, V>` builds an object with one `V` per key in `K`.

A second family reads types off other types and values: `typeof value`,
`ReturnType<typeof fn>`, `Parameters<typeof fn>`, and `Awaited<T>` (TS 4.5), which unwraps a
promise the way `await` does. With these the entity type is written once, and the create
payload, the patch body, the form's error map and the API result are computed from it,
so they can't drift apart.

## Coming from C/C++/Java/Python/JS

- Java: a `Requirement` record plus `CreateRequirementRequest` and `UpdateRequirementRequest`
  DTOs, kept in sync by hand or by a code generator. Here each one is a type expression,
  and adding a field to the entity updates them all at compile time.
- Go: a PATCH body is a second struct with pointer fields (`Title *string`) so a missing
  field differs from an empty one, and generics can't map over a struct's fields. No type
  is computed from another; each struct is written out.
- C++: templates compute types too, but standard C++ through C++23 cannot enumerate a
  struct's members, so there is no `Partial<T>`.
- Python: `TypedDict` with `total=False` is a hand-written `Partial`; the `typing` module
  has no `Pick` or `Omit`.
- JS: nothing changes at run time. Every one of these types is erased; the spread in
  `update` is the same code you would write without them.

## Design notes

Derive in one direction: the entity (or the generated API type) is the source, and request
types, view models and form state are computed from it. Choose `Omit` when a new field
should be required everywhere by default and `Pick` when new fields should stay out until
someone opts in; a write payload is usually safer as a `Pick`, since a new server-managed
field then can't slip into the request type.

Stop deriving when the derived type stops being readable. `Partial<Omit<Pick<A, ...>, ...>>`
nested three deep is harder to review than the four fields written out, and a type that
must express rules ("status `approved` requires an owner") isn't a mapping at all: that is
a discriminated union, lesson ts.type-level.branded. Two traps are worth a lint rule or a
local helper: `Omit` checks neither its keys nor union members, and `Partial` admits
`undefined` unless `exactOptionalPropertyTypes` is on.
