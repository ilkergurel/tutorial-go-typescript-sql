---
id: go.composite.slices-and-maps
title: "Slices and maps: len/cap, append, aliasing, nil maps, comma-ok"
track: go
module: composite
order: 1
level: basic
minutes: 14
since: go1.23
prerequisites: [go.basics.functions]
compare_with: [ts.objects-classes.spread, sql.tables-types.jsonb-arrays]
tags: [slices, maps, append, aliasing, nil, comma-ok]
summary: >
  A slice is a small header (pointer, length, capacity) over an array, so
  slicing shares memory and append may or may not. A map returns the zero
  value for missing keys, tells you with comma-ok, and iterates in random order.
examples:
  - id: slice-header
    file: examples/01_slice_header.go
    title: Slices share their backing array until append runs out of room
    runtime: go
    check: exact
    annotations:
      - lines: "6-8"
        note: >
          arr[1:3] copies nothing: s points at arr[1], has length 2, and
          capacity 4 because arr has four elements from index 1 on.
      - lines: "10-14"
        note: >
          Writes through s land in arr. append with spare capacity writes into
          the same array too, overwriting arr[3], which other code may still
          be using.
      - lines: "16-18"
        note: >
          Two more elements don't fit, so append allocates a new array and
          copies. From then on s and arr are independent. Whether append
          copied depends on capacity, not on anything visible at the call.
      - lines: "20-24"
        note: >
          The classic bug: a and b both append into base's spare capacity, so
          the second append overwrites the first. a[3] is 2, not 1.
      - lines: "26-30"
        note: >
          base[:3:3] sets capacity to 3, so each append must copy and a and b
          get their own arrays. slices.Clone(base) is the other way out.
  - id: maps
    file: examples/02_maps.go
    title: "Maps: comma-ok, key normalization, random order, nil maps"
    runtime: go
    check: exact
    annotations:
      - lines: "14-17"
        note: >
          A missing key yields the zero value, not an exception or undefined.
          The second result, ok, is the only way to tell "stored 0" from
          "not there".
      - lines: "19-25"
        note: >
          Keys are compared byte for byte, so case-insensitive keys need a
          normalized form. strings.ToLower maps ILIK to ilik, merging
          lukewarm with bone marrow; unicode.TurkishCase maps I to ı and İ
          to i, which is right for Turkish text and wrong for English.
      - lines: "26-27"
        note: >
          fmt sorts map keys when printing (since Go 1.12), which is why this
          output is stable. That says nothing about range.
      - lines: "29-32"
        note: >
          The runtime randomizes iteration order on purpose, so code that
          depends on it breaks in tests instead of after an upgrade.
          slices.Sorted(maps.Keys(m)) (Go 1.23) gives a stable order.
      - lines: "34-36"
        note: >
          A nil map behaves like an empty one for reads, len, delete and
          range, but storing into it panics. A struct with a map field needs
          a constructor or a lazy make.
  - id: slices-and-functions
    file: examples/03_slices_and_functions.go
    title: Passing slices to functions, range copies, nil vs empty
    runtime: go
    check: exact
    annotations:
      - lines: "8-13"
        note: >
          The slice header is copied into the parameter, but it points at
          the caller's array, so element writes are visible to the caller.
      - lines: "15-17"
        note: >
          append changes the parameter's length (and maybe its pointer); the
          caller's header is untouched. If the caller's slice had spare
          capacity, the new element even lands in its array, invisibly.
      - lines: "19-20"
        note: >
          Return the slice, exactly like append: xs = add(xs, 4). The compiler
          rejects a bare append(xs, 4) whose result is unused.
      - lines: "30-32"
        note: >
          The range value is a copy of each element. To change elements,
          index: for i := range xs { xs[i] *= 10 }.
      - lines: "35-42"
        note: >
          A nil slice and an empty one both have length 0 and work with range
          and append. They differ in == nil and in encoding/json, which writes
          null for nil and [] for empty; API clients often care.
use_cases:
  - "Slices: every ordered collection, buffers, results of queries, function arguments of variable length"
  - "Maps: lookups by key, counting, grouping, de-duplication; map[K]struct{} as a set"
  - "Full slice expression or slices.Clone: handing out part of a slice that the receiver may append to"
  - "Sorted keys of a map: stable output for logs, tests, golden files and APIs"
strengths:
  - "Slicing is O(1) and copies nothing, unlike Python's or JavaScript's slice, which copy"
  - "comma-ok separates a missing key from a stored zero value without exceptions (Python's KeyError) or undefined"
  - "Zero values are usable: a nil slice can be appended to and ranged over, a nil map can be read, so empty collections need no setup in most code"
  - "Randomized map order makes order-dependent code fail early, in tests, unlike Java's HashMap whose order merely happens to be stable"
weaknesses:
  - "Whether append shares or copies depends on capacity, so aliasing bugs appear only for some inputs"
  - "No ordered map in the standard library; sorted output means sorting the keys"
  - "Maps are not safe for concurrent writes: the runtime aborts with 'concurrent map writes'. Use a sync.Mutex or sync.Map"
  - "nil and empty slices look the same in Go code but differ in JSON (null vs [])"
pitfalls:
  - title: "Two appends into one slice's spare capacity"
    detail: "Both results share the array and the second append overwrites the first. Cap the capacity with s[:len(s):len(s)] or copy with slices.Clone before appending."
    bad: "a := append(base, 1); b := append(base, 2)"
    good: "a := append(slices.Clone(base), 1); b := append(slices.Clone(base), 2)"
  - title: "Writing to a nil map in a zero-value struct"
    detail: "var c Cache; c.items[\"k\"] = v panics. Initialize the map in a constructor (NewCache) or with if c.items == nil { c.items = map[string]V{} }."
  - title: "Keeping a small slice of a large array"
    detail: "header := data[:16] keeps the whole data array reachable. Copy what you keep (slices.Clone) when the source is big."
  - title: "Changing a struct stored in a map"
    detail: "m[k].Count++ does not compile (cannot assign to struct field in map): map values are not addressable. Read, modify and store back, or store pointers."
    bad: "m[\"a\"].Count++"
    good: "v := m[\"a\"]; v.Count++; m[\"a\"] = v"
quiz:
  - q: "base := make([]int, 3, 10); a := append(base, 1); b := append(base, 2). What is a[3]?"
    options: ["1", "2", "It panics"]
    answer: 1
    why: "Both appends write index 3 of base's array, which has room. a and b share it, so the second write wins."
  - q: "m is map[string]int without the key \"k\". What does v, ok := m[\"k\"] give?"
    options: ["0, false", "0, true", "a panic", "nil, false"]
    answer: 0
    why: "Missing keys yield the zero value; ok reports whether the key was present."
  - q: "In Python a[1:3] creates a new list. What does a[1:3] create in Go?"
    options: ["A copy of two elements", "A slice that shares a's backing array", "An array of length 2"]
    answer: 1
    why: "Slicing builds a new header over the same array; writes through either are visible to both."
  - q: "var m map[string]int. Which operation panics?"
    options: ["m[\"x\"]", "len(m)", "delete(m, \"x\")", "m[\"x\"] = 1"]
    answer: 3
    why: "A nil map behaves like an empty read-only map; only storing into it panics."
further_reading:
  - title: "Go Slices: usage and internals"
    url: "https://go.dev/blog/slices-intro"
  - title: "Arrays, slices (and strings): The mechanics of 'append'"
    url: "https://go.dev/blog/slices"
  - title: "Go maps in action"
    url: "https://go.dev/blog/maps"
  - title: "The Go Programming Language Specification: Slice expressions"
    url: "https://go.dev/ref/spec#Slice_expressions"
---

## Concept

Go's two built-in collections cover nearly everything. A **slice** `[]T` is a
three-word header: a pointer into an underlying array, a length, and a capacity (how
many elements the array has from that pointer on). Slicing makes a new header over
the same array; `append` writes into the spare capacity when there is some and
allocates a bigger array when there isn't. An array `[N]T` is a fixed-size value that
is copied on assignment, and you rarely use it directly.

A **map** `map[K]V` is a hash table with comparable keys. Reading a missing key
returns the zero value of `V`; `v, ok := m[k]` tells you whether the key was there.
Iteration order is unspecified and deliberately randomized. Maps and slices are both
cheap to pass around: you copy a small header, not the data.

## Coming from C/C++/Java/Python/JS

- **Python/JS**: `a[1:3]` and `arr.slice(1, 3)` copy; Go's `a[1:3]` aliases. Python
  dicts and JS `Map` keep insertion order; Go maps keep none.
- **Java**: a slice is like `ArrayList.subList`, a view that writes through, except
  that `append` may silently detach it. A missing map key gives a zero value, not
  `null`, and deleting during `range` is allowed instead of throwing
  `ConcurrentModificationException`.
- **C**: a slice is `struct { T *ptr; size_t len, cap; }` passed by value, and
  `append` is a `realloc` that may move the data. The same rule applies: after a
  possible reallocation, use the returned value.
- **C++**: like `std::vector`, but a sub-slice is a view (`std::span`) that can still
  grow into its parent's storage.

## How it works

```
arr := [5]int{1, 2, 3, 4, 5}
s := arr[1:3]          s = {ptr: &arr[1], len: 2, cap: 4}

arr  [ 1 | 2 | 3 | 4 | 5 ]
           ^ s[0]
           |--len--|
           |------cap------|
```

`append(s, x)` checks `len < cap`. If so, it stores `x` at `arr[3]` and returns a
header with `len` 3, same pointer. If not, it allocates a larger array, copies, and
returns a header pointing there. The growth factor is an implementation detail; the
spec only promises a "sufficiently large" array.

## Design notes

Every slice rule follows from "the header is a value, the array is shared". So:
always assign the result of `append`; return slices from functions that grow them;
and when you hand out a slice of your internal state, give a copy (`slices.Clone`) or
cap it (`s[:n:n]`) so the receiver cannot write into your array. Preallocate with
`make([]T, 0, n)` when you know the size. For maps, decide the key's canonical form
before inserting: for user-entered Turkish text that means choosing between
`strings.ToLower` and `strings.ToLowerSpecial(unicode.TurkishCase, s)` based on the
text's language, not the server's locale.
