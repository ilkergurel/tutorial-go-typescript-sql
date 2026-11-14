---
id: go.iterators.writing
title: "Writing your own iterators; helper functions in slices/maps/strings"
track: go
module: iterators
order: 2
level: intermediate
minutes: 14
since: go1.24
prerequisites: [go.iterators.range, go.generics.parameters]
compare_with: [ts.node.fs-streams, go.files-io.reading-files, go.basics.strings-runes]
tags: [iterators, iter, strings, slices, maps, lazy]
summary: >
  strings.Lines, strings.SplitSeq and slices.Values hand out iter.Seq values;
  small adapters such as Filter, Map and Take compose them into lazy pipelines
  that stop the producer as soon as the consumer is done.
examples:
  - id: strings-seq
    file: examples/01_strings_seq.go
    title: strings.Lines, SplitSeq and FieldsSeq over Turkish text
    runtime: go
    check: exact
    annotations:
      - lines: "26-28"
        note: >
          strings.Lines (Go 1.24) yields every line with its "\n", unlike
          bufio.Scanner, which strips it. The lines concatenate back to the
          exact input; trimming is your job when you parse.
      - lines: "30-34"
        note: >
          TrimSuffix removes the newline Lines kept, and Cut splits once at the
          first ";". A CRLF file would still leave "\r" at the end of each line.
      - lines: "35"
        note: >
          The Turkish case mapping turns i into İ, so simit prints as SİMİT.
          strings.ToUpper would give SIMIT.
      - lines: "36-38"
        note: >
          SplitSeq yields the same pieces as strings.Split but never builds the
          []string. For three tags that's style; inside a loop over a large
          export it removes one allocation per line.
      - lines: "42-43"
        note: >
          FieldsSeq splits on runs of Unicode whitespace, like strings.Fields.
          slices.Collect turns any Seq into a slice when you do need one.
      - lines: "45-46"
        note: >
          Lines and SplitSeq are documented as single-use: the second pass over
          the same value yields nothing, hence 4 and then 0. Call strings.Lines
          again for a second pass; it's cheap.
  - id: pipeline
    file: examples/02_pipeline.go
    title: Filter, Map and Take as generic adapters, composed lazily
    runtime: go
    check: exact
    annotations:
      - lines: "12-20"
        note: >
          An adapter takes a Seq and returns a Seq. Filter's only real duty is
          the yield check: when downstream stops, it stops pulling from upstream.
      - lines: "22-30"
        note: >
          Map changes the element type, so it needs two type parameters. This
          function-over-Seq shape is the one the standard library uses.
      - lines: "32-47"
        note: >
          Take returns right after the n-th value and never pulls the next one.
          Its counter lives inside the returned func, so every range starts from
          zero; decrementing the captured n instead would make the Seq
          single-use.
      - lines: "51-54"
        note: >
          The predicate prints a trace. It counts letters with
          RuneCountInString; len counts bytes, and len("ılık") is 6, so a byte
          check would let a four-letter word through.
      - lines: "57-61"
        note: >
          Building the pipeline runs nothing. The range pulls one value at a
          time through Filter, Map and Take; after the second match, Take's
          return sends false back up the chain, so ıhlamur and kiraz are never
          checked.
  - id: tree-all
    file: examples/03_tree_all.go
    title: An All() method on your own collection, consumed by maps and slices
    runtime: go
    check: exact
    annotations:
      - lines: "11-33"
        note: >
          A plain unbalanced binary search tree: just enough structure to be
          worth iterating over.
      - lines: "35-38"
        note: >
          The convention from package iter: a collection exposes All() returning
          a Seq2 of key and value. Callers never see nodes or pointers.
      - lines: "40-46"
        note: >
          In-order traversal is three lines with yield in the middle. The bool
          result carries "stop" up the recursion, so one false from yield
          unwinds every level. A Java Iterator for the same walk needs a class
          with an explicit stack.
      - lines: "49-52"
        note: >
          Inserting from a map gives a random order and a different tree shape
          on each run, yet the in-order walk is always sorted, so the output is
          deterministic.
      - lines: "53-58"
        note: >
          break after plate 34: walk returns false at every level and no
          further node is visited.
      - lines: "59-60"
        note: >
          All returns a standard Seq2, so maps.Collect, maps.Values and
          slices.Sorted work with no glue code. Byte order puts Iğdır (I) before
          İstanbul (İ).
use_cases:
  - "Parsing text line by line without building a []string: strings.Lines, SplitSeq, FieldsSeq, and the same functions in package bytes"
  - "All, Keys, Values or Backward methods on your own collections"
  - "Small adapters (Filter, Map, Take) over database rows, file lines or API pages, where stopping early saves real work"
  - "slices.Collect, slices.Sorted and maps.Collect at the end of a pipeline when you need a materialized result"
strengths:
  - "Lazy and early-exiting by construction: Take(..., 2) stops the producer after two matches, which a chain of slice-returning helpers can't do"
  - "Everything meets at one type, iter.Seq, so your adapters, your collections and the standard library helpers plug into each other"
  - "A tree iterator is a recursive function with yield in the middle, where Java needs an Iterator class with an explicit stack; Python's generators are the closest match"
weaknesses:
  - "The standard library has no Map, Filter or Take for iter.Seq as of Go 1.27, so every project writes or imports its own"
  - "Nested calls read inside out: Take(Map(Filter(src, long), upper), 2) names the last step first"
  - "Single-use iterators such as strings.Lines and SplitSeq look like reusable values; a second range silently yields nothing"
  - "iter.Seq has no error channel: a sequence that can fail needs a convention, such as yielding (value, error) pairs through a Seq2"
pitfalls:
  - title: "Forgetting that strings.Lines keeps the line ending"
    detail: "Each line ends in \"\\n\" (except possibly the last), and a Windows file with CRLF endings leaves \"\\r\\n\". Trim before comparing or parsing."
    bad: "if line == \"ürün;etiketler\" { ... }"
    good: "line = strings.TrimRight(line, \"\\r\\n\")"
  - title: "Ranging twice over a single-use iterator"
    detail: "strings.Lines, SplitSeq and iterators over readers are single-use. Call the function again for each pass, or collect once with slices.Collect."
    bad: "lines := strings.Lines(s); n := count(lines); for l := range lines { ... }"
    good: "n := count(strings.Lines(s)); for l := range strings.Lines(s) { ... }"
  - title: "Keeping adapter state outside the returned func"
    detail: "Variables declared in the outer function are shared by every range over the Seq. Declare counters and cursors inside the func(yield ...) so each loop starts fresh."
  - title: "Measuring Turkish text with len"
    detail: "len counts bytes: len(\"ılık\") is 6, len(\"Iğdır\") is 7. Use utf8.RuneCountInString for letters, or range over the string."
quiz:
  - q: "for line := range strings.Lines(\"a\\nb\"): which values does line take?"
    options: ["\"a\" and \"b\"", "\"a\\n\" and \"b\"", "\"a\\n\" and \"b\\n\""]
    answer: 1
    why: "Lines keeps each terminating newline; the last line has none because the input doesn't end in one."
  - q: "Take(Map(Filter(src, keep), f), 2) runs over a src with a million elements. How many does src produce?"
    options: ["All million, then Take keeps two", "Exactly two", "Only as many as needed until Take has its two values"]
    answer: 2
    why: "Each value travels through the whole chain before the next is read, and Take's false stops src. Exactly two only if the first two both pass the filter."
  - q: "lines := strings.Lines(text); a := count(lines); b := count(lines). What is b?"
    options: ["0, because Lines returns a single-use iterator", "The same as a", "It panics"]
    answer: 0
    why: "The documentation says single-use; per package iter's convention, calling it again after it finished yields nothing."
  - q: "Why does the tree's walk method return a bool?"
    options: ["To report errors", "So a false from yield stops the whole recursion, not just the current level", "range requires it"]
    answer: 1
    why: "Without it, the parent call would carry on to the right subtree and call yield again after false, which panics."
further_reading:
  - title: "package iter: naming conventions, single-use iterators"
    url: "https://pkg.go.dev/iter"
  - title: "package strings: Lines"
    url: "https://pkg.go.dev/strings#Lines"
  - title: "package slices"
    url: "https://pkg.go.dev/slices"
  - title: "Go 1.24 Release Notes"
    url: "https://go.dev/doc/go1.24"
---

## Concept

Since Go 1.23 the standard library hands out sequences as `iter.Seq[V]` and
`iter.Seq2[K, V]`: `slices.Values` and `slices.All` over a slice, `maps.Keys` and
`maps.All` over a map, and since Go 1.24 `strings.Lines`, `strings.SplitSeq`,
`strings.FieldsSeq` (and their `bytes` twins) over text. At the other end,
`slices.Collect`, `slices.Sorted` and `maps.Collect` turn a sequence back into a slice
or map.

Writing your own takes two forms. A collection gets a method such as `All()` that
returns a Seq built around its internal walk. An adapter is a function that takes a
Seq and returns a new one (`Filter`, `Map`, `Take`). Adapters compose into lazy
pipelines: nothing runs until a `range` pulls, values travel through the whole chain
one at a time, and the first `false` from the consumer stops the producer.

## Coming from C/C++/Java/Python/JS

- Java: like `Stream.filter().map().limit()`, lazy and short-circuiting. Go has no
  stream type: adapters are plain functions over `iter.Seq`, and there is no parallel
  mode.
- Python: like `itertools` with generators: `filter`, `map` and `islice` over a
  generator. Go's `Take` is `islice(seq, n)`.
- JS/Node: close to iterator helpers (`.filter().map().take()`) or a Node stream
  pipeline, but synchronous and without backpressure: the consumer's loop drives
  everything.
- C++: C++20 `std::views::filter | transform | take` composes lazily the same way.

## Design notes

Expose iteration through methods that return `iter.Seq` or `iter.Seq2`, named after
what they walk (`All`, `Keys`, `Values`, `Backward`, or a domain name like `Cities`).
Document single-use iterators, as `strings.Lines` does. Inside every iterator and
adapter, check `yield` and keep mutable state inside the returned function. Keep
adapters few and boring; three or four cover most needs, and when the standard library
already has a function (`slices.Collect`, `slices.Sorted`, `maps.Collect`), use it at
the end of the pipeline. Go 1.27's generic methods would let you wrap a Seq in your
own type and chain `.Filter(...).Map(...)` left to right, but the standard library
stays with functions over `iter.Seq`, which work with every sequence without
wrapping. For text, remember the Turkish rules from earlier lessons: count letters with
`utf8.RuneCountInString`, and change case with `unicode.TurkishCase`.
