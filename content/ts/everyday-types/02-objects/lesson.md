---
id: ts.everyday-types.objects
title: "Arrays, readonly tuples, object types, optional and readonly properties"
track: ts
module: everyday-types
order: 2
level: basic
minutes: 12
since: es2023
prerequisites: [ts.everyday-types.primitives]
compare_with: [go.composite.slices-and-maps, go.composite.structs, design.paradigms.immutability, ts.objects-classes.spread]
tags: [arrays, tuples, readonly, optional-properties, object-types]
summary: >
  Arrays, tuples and object types describe data; readonly and ? describe what may
  change and what may be missing. Both are compile-time only and shallow, so know
  when Object.freeze or a copy is the real guarantee.
examples:
  - id: arrays-tuples
    file: examples/01_arrays_tuples.ts
    title: readonly arrays, tuples and the in-place sort bug
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-8"
        note: >
          sort() (like reverse() and splice()) mutates the array it is called on, so
          the caller's data changes behind its back. In Go a slice parameter shares
          its backing array in the same way.
      - lines: "10-17"
        note: >
          readonly number[] (also written ReadonlyArray<number>) is the same runtime
          array with the mutating methods removed from its type. toSorted (ES2023,
          needs lib es2023+; this app's own tsconfig.app.json uses ES2020 libs, where
          you'd write [...scores].sort()) returns a sorted copy.
      - lines: "19-27"
        note: >
          A tuple type fixes length and per-position types; readonly forbids
          writes. The labels lat and lon are documentation only: they don't change
          compatibility and destructuring can use any names.
      - lines: "29-32"
        note: >
          Object.entries yields [key, value] tuples, and destructuring in the
          callback parameter unpacks them. This is TS's answer to Go's multiple
          return values.
  - id: object-types
    file: examples/02_object_types.ts
    title: Optional and readonly properties, and their limits
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-11"
        note: >
          `minutes?: number` means the key may be absent, so every read is
          number | undefined and the caller picks a default. Unlike Go, there is no
          zero value: a missing number is undefined, not 0.
      - lines: "13-19"
        note: >
          readonly applies to the property, not to the value in it. The tags
          reference can't be replaced, but the array can be pushed to. Use
          readonly string[] as well when the contents must not change either.
      - lines: "21-25"
        note: >
          A known hole: readonly properties are ignored when checking assignability,
          so a mutable type can alias the same object. Readonly arrays are stricter:
          readonly T[] is not assignable to T[] (TS4104).
      - lines: "27-37"
        note: >
          Object.freeze is the runtime guarantee, and its return type is
          Readonly<T>. In strict-mode code (every ES module) a write throws a
          TypeError; in sloppy scripts it is silently ignored. It is shallow as
          well: nested arrays and objects stay mutable.
use_cases:
  - "readonly T[] parameters on every function that only reads an array, so callers know their data is safe"
  - "Readonly tuples for fixed pairs: coordinates, [key, value], a hook's [value, setValue]"
  - "Optional properties for genuinely absent data (no estimate yet), not as a way to avoid modelling states"
  - "Object.freeze for module-level constants and configuration that must not change at run time"
strengths:
  - "Unlike Java's unmodifiable views (which throw at run time), readonly arrays remove the mutators from the type, so the mistake is a compile error"
  - "Tuples give fixed-length, per-position types without a class, which Java needs a record for and Go a struct or multiple returns"
  - "Optional properties make absence part of the type, so a missing field is handled where it is read, not discovered as null later"
weaknesses:
  - "readonly is shallow and compile-time only; deep immutability needs Readonly types at every level, as const, or a freeze at run time"
  - "readonly properties are not enforced on assignment to a mutable type, so an alias can write to them"
  - "Arrays and objects are reference types with no copy-on-assign, unlike Go structs; sharing is the default and copying is explicit"
pitfalls:
  - title: "Mutating an array you were handed"
    detail: "sort, reverse and splice change the caller's array. Use toSorted, toReversed, toSpliced (ES2023) or copy first, and type read-only parameters as readonly T[]."
    bad: "function top(xs: number[]) { return xs.sort((a, b) => b - a)[0]; }"
    good: "function top(xs: readonly number[]) { return xs.toSorted((a, b) => b - a)[0]; }"
  - title: "Treating readonly as immutability"
    detail: "readonly stops assignments through that type only. Nested values remain mutable, and another alias without readonly can still write."
  - title: "Optional property where a union of states belongs"
    detail: "`{ data?: T; error?: string }` allows both or neither. If exactly one must be present, use a discriminated union (lesson ts.narrowing.discriminated-unions)."
quiz:
  - q: "A function takes `xs: readonly string[]`. Can you pass it a `string[]`?"
    options: ["Yes", "No"]
    answer: 0
    why: "A mutable array can always be viewed read-only. The reverse, readonly string[] to string[], is a compile error (TS4104)."
  - q: "`interface C { readonly tags: string[] }`. Which statement compiles?"
    options: ["c.tags = []", "c.tags.push(\"x\")", "Neither"]
    answer: 1
    why: "readonly protects the property binding, not the array it points to. Make it readonly string[] to block push too."
  - q: "What does Object.freeze give you that a readonly type does not?"
    options: ["Deep immutability", "Enforcement at run time, one level deep", "Nothing; they are the same"]
    answer: 1
    why: "freeze makes the object's own properties non-writable at run time; nested objects are not frozen."
  - q: "In `type LatLon = readonly [lat: number, lon: number]`, what do the labels change?"
    options: ["Only readability and editor hints", "They turn the tuple into an object with lat and lon", "They make the tuple incompatible with [number, number]"]
    answer: 0
    why: "Tuple labels are documentation: the type is still a two-element array of numbers."
further_reading:
  - title: "TypeScript Handbook: Object Types (optional, readonly, tuples)"
    url: "https://www.typescriptlang.org/docs/handbook/2/objects.html"
  - title: "TypeScript Handbook: Everyday Types (arrays)"
    url: "https://www.typescriptlang.org/docs/handbook/2/everyday-types.html"
  - title: "MDN: Array.prototype.toSorted()"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Array/toSorted"
  - title: "MDN: Object.freeze()"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Object/freeze"
---

## Concept

Four building blocks cover most data you will type:

- **Arrays**: `string[]` (or `Array<string>`), and `readonly string[]`
  (`ReadonlyArray<string>`), which has no `push`, `sort` or other mutating methods.
- **Tuples**: arrays with a fixed length and a type per position, `[string, number]`,
  optionally labelled and readonly: `readonly [lat: number, lon: number]`.
- **Object types**: inline `{ id: string; title: string }`, or named with `interface`
  or `type`. `Record<string, T>` (or an index signature `{ [key: string]: T }`) types
  dictionaries; with `noUncheckedIndexedAccess` their reads include `undefined`.
- **Property modifiers**: `?` marks a property that may be absent, `readonly` one that
  can't be assigned through this type.

`readonly` and `?` are checker-only, like every type. At run time an array is an array
and an object is an object; `Object.freeze` is the only built-in runtime enforcement.

## Coming from C/C++/Java/Python/JS

- C++: `readonly T[]` is closest to `const std::vector<T>&`: the callee can read but not
  modify. Like C++ `const` it is shallow for pointers inside, and it is weaker: a
  `readonly` property can be aliased as writable without any cast (example 2).
- Java: `final` fields ≈ `readonly` properties; `List.copyOf`/unmodifiable views ≈
  `readonly T[]`, except Java enforces at run time with an exception and TS at compile
  time with a missing method. Records ≈ readonly object types.
- Go: slices share their backing array just as JS arrays are shared references, but Go
  structs are copied on assignment; JS objects never are. There is no zero value: an
  absent optional property reads as `undefined`.
- Python: tuples are immutable at run time; TS tuples are ordinary arrays with a fixed
  type. `typing.Final` and frozen dataclasses are the nearest relatives of `readonly`
  and `Object.freeze`.
- JS: nothing changes at run time. `toSorted`, `toReversed`, `toSpliced` and `with`
  (ES2023) are the copying versions of the mutating array methods.

## Design notes

Make read-only the default in signatures: a function that only reads an array or object
should say so with `readonly T[]` or `Readonly<T>`, which costs nothing and documents
the contract at every call site. Return fresh values rather than mutating inputs (the
copying ES2023 methods and spread make that cheap to write). Reserve `?` for data that
can legitimately be absent; when fields come and go together, model the states as a
discriminated union instead. Reach for `Object.freeze` when something outside the type
system (another module, a library, the console) could write to a shared value.
