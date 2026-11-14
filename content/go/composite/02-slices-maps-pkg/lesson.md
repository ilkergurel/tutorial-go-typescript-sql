---
id: go.composite.slices-maps-pkg
title: "The slices and maps packages, copy, clear"
track: go
module: composite
order: 2
level: basic
minutes: 12
since: go1.23
prerequisites: [go.composite.slices-and-maps]
compare_with: [go.iterators.writing, ts.functions.higher-order]
tags: [slices, maps, sort, cmp, copy, clear, iterators]
summary: >
  Since Go 1.21 the generic slices and maps packages, plus the min, max and
  clear built-ins, replace most hand-written search, sort, de-dup, filter and
  merge loops. Go 1.23 iterators connect them.
examples:
  - id: search-sort
    file: examples/01_search_sort.go
    title: Search and sort without writing the loop
    runtime: go
    check: exact
    annotations:
      - lines: "17-18"
        note: >
          Three loops you no longer write. Index returns -1 when absent;
          Max panics on an empty slice, so check len first when it can be empty.
      - lines: "20-22"
        note: >
          Sort works in place on any ordered element type. BinarySearch needs
          sorted input and returns where the value is (or would be inserted)
          plus whether it was found, instead of Java's negative-index encoding.
      - lines: "31-38"
        note: >
          The comparison returns negative, zero or positive, like Java's
          Comparator. cmp.Or (Go 1.22) returns its first non-zero argument,
          which turns "by region, then by population" into one expression.
          SortFunc is not stable; use SortStableFunc to keep input order of ties.
      - lines: "43-46"
        note: >
          IndexFunc and ContainsFunc take a predicate. IndexFunc returns -1
          when nothing matches, so check before indexing.
      - lines: "48-51"
        note: >
          String order is byte order of the UTF-8 encoding, so Ç (0xC3...) and
          İ (0xC4...) sort after Z. Turkish alphabetical order needs a
          collator: golang.org/x/text/collate with language.Turkish, outside
          the standard library.
  - id: edit-copy-clear
    file: examples/02_edit_copy_clear.go
    title: "Edit in place: Compact, Insert, Delete, DeleteFunc, copy, clear"
    runtime: go
    check: exact
    annotations:
      - lines: "11-13"
        note: >
          slices.Values turns tags into an iterator and Sorted collects it into
          a new sorted slice, leaving tags alone (slices.Sort would sort in
          place). Compact, like Unix uniq, only drops adjacent repeats, hence
          the sort first.
      - lines: "15-16"
        note: >
          Insert shifts elements up and may reallocate: always use the result,
          as with append.
      - lines: "18-22"
        note: >
          Delete shifts the tail down in the same array and zeroes the freed
          slots (Go 1.22), which also lets the garbage collector reclaim what
          they pointed to. before still covers four slots and shows that.
      - lines: "24-27"
        note: >
          DeleteFunc is an in-place filter: no second slice, but the input's
          array is reused, so don't keep using the old slice.
      - lines: "29-32"
        note: >
          copy copies min(len(dst), len(src)) elements and returns that count.
          It never grows dst: a destination made with make([]T, 0, n) has
          length 0 and receives nothing.
      - lines: "34-37"
        note: >
          Slices cannot be compared with == (only with nil), so slices.Equal
          does it. Clone allocates a new array, so writes to c don't reach nums.
      - lines: "39-41"
        note: >
          clear on a slice sets each element to its zero value and keeps the
          length. To empty a slice for reuse, use s = s[:0] instead.
  - id: maps-pkg
    file: examples/03_maps_pkg.go
    title: "maps: Clone, Copy, Keys, DeleteFunc, Equal, Collect, clear"
    runtime: go
    check: exact
    annotations:
      - lines: "14-18"
        note: >
          Clone then Copy is the merge-with-defaults idiom: keys from user
          overwrite. Clone is shallow: a value that is a slice or map is
          shared between the clone and the original.
      - lines: "20-22"
        note: >
          maps.Keys and maps.Values return iterators (iter.Seq), not slices,
          and in random order. slices.Sorted collects and sorts in one call;
          slices.Collect collects without sorting.
      - lines: "24-29"
        note: >
          DeleteFunc removes every entry the predicate accepts. Maps can't be
          compared with ==, so maps.Equal compares keys and values.
      - lines: "31-33"
        note: >
          Iterators compose: slices.All yields index/element pairs and
          maps.Collect builds a map from any key/value sequence.
      - lines: "35-38"
        note: >
          clear on a map deletes every entry and the map stays writable, unlike
          assigning nil. That matters when other code holds the same map.
use_cases:
  - "Membership, position and extremes: slices.Contains, Index, IndexFunc, Max, Min"
  - "Sorting by several keys: slices.SortFunc with cmp.Or(cmp.Compare(...), ...)"
  - "De-duplicating: slices.Sort then slices.Compact, or collect into a map[T]struct{}"
  - "Configuration merges and copies: maps.Clone, maps.Copy; stable output with slices.Sorted(maps.Keys(m))"
  - "Resetting a cache or buffer without reallocating: clear(m), s = s[:0]"
strengths:
  - "Generic and type-checked: slices.SortFunc receives typed elements, unlike sort.Slice's index-based less function, and its docs note it is usually faster"
  - "One idiom for everything ordered: cmp.Compare and cmp.Or give Java Comparator chaining without a builder API"
  - "Iterators (Go 1.23) connect the packages: maps.Keys feeds slices.Sorted without an intermediate slice in your code"
  - "In-place operations (Delete, Compact, DeleteFunc) avoid allocations, and zero the tail so the GC can reclaim it"
weaknesses:
  - "In-place operations mutate the shared array: other slices of it see shifted or zeroed elements"
  - "No locale-aware string sorting in the standard library; Turkish order needs golang.org/x/text/collate"
  - "No map/filter/reduce returning new slices: a transformation is still a loop, or DeleteFunc for filtering in place"
  - "maps.Clone and slices.Clone are shallow; deep copies are hand-written"
pitfalls:
  - title: "Compact without sorting"
    detail: "Compact only removes adjacent duplicates: [go sql go] stays as it is. Sort first, or use a map if order must be kept."
  - title: "Ignoring the result of Insert, Delete, Compact or DeleteFunc"
    detail: "They return the new slice header; the old variable keeps the old length and now sees shifted or zeroed elements."
    bad: "slices.Delete(s, 0, 1)"
    good: "s = slices.Delete(s, 0, 1)"
  - title: "copy into a zero-length destination"
    detail: "copy(dst, src) with dst := make([]T, 0, len(src)) copies nothing. Use make([]T, len(src)), or simply slices.Clone(src)."
  - title: "Sorting user-visible Turkish names with slices.Sort"
    detail: "Byte order puts Çorum and İzmir after Zonguldak. Collate with golang.org/x/text/collate (language.Turkish), or sort in PostgreSQL with a Turkish ICU collation."
quiz:
  - q: "s := []string{\"go\", \"sql\", \"go\"}; s = slices.Compact(s). What is s?"
    options: ["[go sql]", "[go sql go]", "[sql go]"]
    answer: 1
    why: "Compact only collapses runs of adjacent equal elements. Sort first to remove all duplicates."
  - q: "dst := make([]int, 0, 3); n := copy(dst, []int{1, 2, 3}). What is n?"
    options: ["3", "0", "It panics"]
    answer: 1
    why: "copy copies min(len(dst), len(src)) elements; len(dst) is 0. Capacity doesn't count."
  - q: "What does clear(s) do to a slice of length 3?"
    options: ["Sets its length to 0", "Zeroes the three elements, length stays 3", "Sets s to nil"]
    answer: 1
    why: "For slices, clear zeroes elements up to the length. For maps it deletes all entries."
  - q: "How do you sort cities by region, then by population descending?"
    options: ["Two calls to slices.Sort", "slices.SortFunc with cmp.Or(strings.Compare(a.Region, b.Region), cmp.Compare(b.Pop, a.Pop))", "sort.Strings followed by slices.Reverse"]
    answer: 1
    why: "cmp.Or returns the first non-zero comparison, so the population only breaks ties within a region; swapping a and b sorts descending."
further_reading:
  - title: "package slices"
    url: "https://pkg.go.dev/slices"
  - title: "package maps"
    url: "https://pkg.go.dev/maps"
  - title: "package cmp"
    url: "https://pkg.go.dev/cmp"
  - title: "The Go Programming Language Specification: Built-in functions (copy, clear)"
    url: "https://go.dev/ref/spec#Built-in_functions"
  - title: "package golang.org/x/text/collate"
    url: "https://pkg.go.dev/golang.org/x/text/collate"
---

## Concept

Go 1.21 added generic `slices` and `maps` packages, the `cmp` package with
`cmp.Compare` (and `cmp.Or` in 1.22), and the `min`, `max` and `clear` built-ins. Go
1.23 added iterators, so `maps.Keys`, `slices.Values` and `slices.All` produce
sequences that `slices.Sorted`, `slices.Collect` and `maps.Collect` consume. Together
they cover the loops every codebase used to repeat: search (`Contains`, `Index`,
`BinarySearch`), sort (`Sort`, `SortFunc`, `SortStableFunc`), edit in place (`Insert`,
`Delete`, `DeleteFunc`, `Compact`, `Reverse`), compare (`Equal`) and copy (`Clone`). The
older built-ins remain: `copy` moves elements between slices, `clear` resets a map or
zeroes a slice.

Before Go 1.21 you would have written `sort.Slice(s, func(i, j int) bool {...})` or a
loop for each of these. `go fix` (Go 1.26) rewrites many such loops for you, as the
modernizers lesson shows.

## Coming from C/C++/Java/Python/JS

- **Java**: `list.sort(Comparator.comparing(City::region).thenComparing(...))` becomes
  `slices.SortFunc` with `cmp.Or`. `Arrays.binarySearch` encodes "not found" as a
  negative insertion point; `slices.BinarySearch` returns `(pos, found)`.
- **Python**: `sorted(xs, key=...)` has no direct equivalent: Go comparison functions
  are three-way, so compare the derived keys with `cmp.Compare`. `{**defaults, **user}`
  is `maps.Clone` plus `maps.Copy`.
- **JS**: `filter` returns a new array, `slices.DeleteFunc` filters in place.
  `Object.assign` ≈ `maps.Copy`; `structuredClone` is deep, `maps.Clone` is shallow.
- **C++**: `std::sort`, `std::lower_bound` and `std::unique` map to `Sort`,
  `BinarySearch` and `Compact`, except that `Compact` returns the shortened slice
  instead of an iterator you must pass to `erase`.

## Design notes

Reach for the library before writing a loop; it states intent in one line and its
edge cases (empty input, ties, zeroing) are documented. Remember which calls mutate:
`Sort`, `Delete`, `Compact`, `Insert` and `DeleteFunc` work on the slice you pass,
while `Sorted`, `Clone` and `Collect` allocate. Pick ordered output deliberately:
`fmt` sorts map keys for you, `range` and `maps.Keys` never do. For text shown to
users, byte order is rarely the right order; when the data lives in PostgreSQL, sort
there with a Turkish collation, and in Go use `golang.org/x/text/collate`.
