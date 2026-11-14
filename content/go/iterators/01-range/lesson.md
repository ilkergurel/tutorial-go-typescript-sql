---
id: go.iterators.range
title: "Range over int and range over functions (iter.Seq/Seq2)"
track: go
module: iterators
order: 1
level: intermediate
minutes: 14
since: go1.23
prerequisites: [go.basics.control-flow, go.generics.parameters]
compare_with: [ts.async.cancellation, go.timeline.1-21-to-1-27]
tags: [iterators, range, iter, yield, pull]
summary: >
  Since Go 1.23, range accepts a function that pushes values into a yield
  callback. Break and return still mean what they always meant, the iterator
  must stop when yield returns false, and iter.Pull turns push into pull.
examples:
  - id: push
    file: examples/01_push.go
    title: A push iterator, early exit, and Seq2 from the standard library
    runtime: go
    check: exact
    annotations:
      - lines: "12-22"
        note: >
          An iter.Seq[int] is only a named func(yield func(int) bool). The
          iterator owns the loop and its state lives in local variables; the
          range body becomes yield. The deferred cleanup runs when the iterator
          returns, whether the sequence ended or the consumer stopped.
      - lines: "25-30"
        note: >
          The output interleaves iterator and body on one call stack: no
          goroutine, no buffer. break makes the pending yield call return false,
          Countdown returns, and its cleanup prints before the next statement
          after the loop runs.
      - lines: "32-36"
        note: >
          Seq2 yields pairs. slices.Backward walks from the end and yields the
          original indexes, without copying or reversing the slice.
      - lines: "38-41"
        note: >
          maps.Keys yields keys in random order; slices.Sorted collects and
          sorts them, the usual way to print a map deterministically. Sorting is
          by bytes, so çay lands after simit.
  - id: yield-contract
    file: examples/02_yield_contract.go
    title: The yield contract, and why return and defer still behave
    runtime: go
    check: exact
    annotations:
      - lines: "5-10"
        note: >
          careless keeps calling yield after the body has finished. It compiles,
          and it even works for every loop that runs to the end.
      - lines: "13-19"
        note: >
          The contract: check what yield returns and stop when it is false.
      - lines: "21-28"
        note: >
          The compiler turns the body into a function, yet return v returns from
          firstPositive. break, continue, return, goto and labels keep their
          ordinary meaning.
      - lines: "30-35"
        note: >
          The defers belong to deferInBody, not to the synthesized yield
          function: they run after "loop done", last in first out, exactly as
          in a plain for loop.
      - lines: "41-42"
        note: >
          The early return makes yield return false, careless calls it again,
          and the runtime panics. The bug is in the iterator, but it shows up in
          the first loop that stops early.
  - id: pull
    file: examples/03_pull.go
    title: iter.Pull for consuming two sequences in lockstep
    runtime: go
    check: exact
    annotations:
      - lines: "9-11"
        note: >
          A range loop drives one push iterator. Merging needs "the next value
          from whichever side is smaller", so both sides must be pulled on demand.
      - lines: "13-16"
        note: >
          Pull runs the iterator on a separate coroutine and returns next and
          stop. If you stop pulling before the end, stop lets the iterator
          return and run its own cleanup; defer stop() is the idiom.
      - lines: "18-32"
        note: >
          A textbook merge written against next. Merge still returns a push
          iterator, so its callers simply range over it.
      - lines: "37-39"
        note: >
          slices.Values turns a slice into a Seq; slices.Collect drains a Seq
          into a new slice.
      - lines: "42-48"
        note: >
          next returns a value and ok. After the end it keeps returning the
          zero value and false, and calling stop then is still allowed.
use_cases:
  - "Exposing a collection without exposing its storage: All() iter.Seq2[K, V] on a cache, tree or ordered map"
  - "Producing lines, rows or API pages lazily, with early exit and cleanup kept inside the iterator"
  - "iter.Pull for lockstep work: merging, zipping or comparing two sequences"
  - "for i := range n for plain counted loops (Go 1.22)"
strengths:
  - "Unlike a Java Iterator with hasNext/next, the iterator is an ordinary loop whose state lives in local variables; a tree walk needs no explicit stack"
  - "Push iterators need no generator objects and no goroutines: the loop body runs as an ordinary function call on the same stack"
  - "One range syntax covers slices, maps, strings, channels, integers and functions, and the standard library speaks iter.Seq (maps.Keys, slices.Values, strings.Lines)"
weaknesses:
  - "The yield contract is checked only at run time: an iterator that ignores false compiles and panics later, in someone else's loop"
  - "Walking two sequences at once needs iter.Pull, which switches coroutines on every value and needs a stop call"
  - "The standard library has no Map, Filter or Take for iter.Seq; you write those adapters yourself (next lesson)"
  - "Panics in the loop body show the iterator's frames in the stack trace, which makes traces deeper than for a plain loop"
pitfalls:
  - title: "Ignoring what yield returns"
    detail: "Every yield call must be checked. A Seq that calls yield after false panics with \"range function continued iteration after function for loop body returned false\"."
    bad: "for _, v := range items { yield(v) }"
    good: "for _, v := range items { if !yield(v) { return } }"
  - title: "Dropping stop from iter.Pull"
    detail: "If you stop calling next before the sequence ends, the iterator stays suspended, holding whatever it opened. Always keep stop and defer it."
    bad: "next, _ := iter.Pull(seq)"
    good: "next, stop := iter.Pull(seq); defer stop()"
  - title: "Replacing iterators with goroutines and channels"
    detail: "Before Go 1.23 a lazy sequence was often a goroutine sending on a channel. It is slower, and the goroutine leaks when the consumer stops early. Return an iter.Seq instead."
  - title: "A go line older than 1.23"
    detail: "Range over a function follows the language version in go.mod: with go 1.22 the compiler reports \"cannot range over ... requires go1.23 or later\"."
quiz:
  - q: "The body of a range-over-func loop runs break. What does the iterator observe?"
    options: ["The pending yield call returns false", "The iterator is terminated immediately", "Nothing; further values are silently ignored"]
    answer: 0
    why: "The iterator must check that result and return; its defers then run as usual."
  - q: "What does return x do inside the body of a range-over-func loop?"
    options: ["Returns from the synthesized yield function only", "It is a compile error", "Returns from the enclosing function, as in any for loop"]
    answer: 2
    why: "The compiler preserves the meaning of return, break, continue, goto and defer in the body."
  - q: "When do you need iter.Pull?"
    options: ["To range over a map", "To consume two sequences in lockstep, or take values one at a time on demand", "Whenever a sequence is large"]
    answer: 1
    why: "range drives one push iterator to completion or break; Pull lets the consumer decide when to ask for the next value."
  - q: "An iterator keeps calling yield after it returned false. When is that caught?"
    options: ["At run time: that yield call panics", "At compile time", "Never; the extra values are dropped"]
    answer: 0
    why: "The yield function the compiler synthesizes checks its state and panics."
further_reading:
  - title: "The Go Programming Language Specification: For statements with range clause"
    url: "https://go.dev/ref/spec#For_range"
  - title: "Range Over Function Types"
    url: "https://go.dev/blog/range-functions"
  - title: "package iter"
    url: "https://pkg.go.dev/iter"
---

## Concept

Go 1.22 let `range` count over an integer and gave every iteration fresh variables
(see the control-flow lesson). One detail worth adding: in `for i := range n`, `i`
has the type of `n`, so ranging over a `uint8` gives `uint8` values. Go 1.23 added the
third form: `range` over a function. The function must have one of three shapes:
`func(yield func() bool)`, `func(yield func(V) bool)` or `func(yield func(K, V) bool)`.
Package `iter` names the last two `Seq[V]` and `Seq2[K, V]`.

Such a function is a push iterator. It runs its own loop and calls `yield` once per
element; the compiler turns the loop body into that `yield`. When the body breaks or
returns, `yield` returns false and the iterator must stop. Sometimes the consumer
needs to decide when to take the next value, for example to walk two sequences
together. `iter.Pull` converts a push iterator into a pull pair: `next` to take one
value, `stop` to end early.

## Coming from C/C++/Java/Python/JS

- Python: a `Seq` is a generator turned inside out. A Python generator suspends at
  `yield` until someone calls `next()`; a Go iterator calls `yield`, and the loop body
  runs inside that call. `iter.Pull` gives you the `next()` style.
- JS: close to a generator consumed by `for...of`, where `break` calls the iterator's
  `return()`. There is no generator object, though: a `Seq` is a plain function, and
  calling it again usually walks the sequence again.
- Java: `Iterator<T>` is pull, with state in fields between `hasNext()` and `next()`.
  `Iterable.forEach(Consumer)` is push but can't stop early; `yield`'s bool result is
  what Go adds.
- C++: like a visitor callback that returns "keep going". C++23's `std::generator` is
  the coroutine-based pull equivalent.

## How it works

The compiler rewrites `for v := range seq { body }` into roughly
`seq(func(v V) bool { body; return true })`. A `break` becomes `return false`.
Extra bookkeeping makes `return`, labeled `break`, `goto` and `defer` in the body act
on the enclosing function, and makes the synthesized `yield` panic if it is called
again after returning false. No goroutine is involved. `iter.Pull` is different: it
runs the iterator on a coroutine and switches stacks on every `next`, which is why it
costs more and needs `stop`.

## Design notes

Give collection types methods that return `iter.Seq` or `iter.Seq2`, conventionally
named `All`, `Keys`, `Values` or `Backward`, instead of exposing the underlying slice
or map. Callers range over them, or collect with `slices.Collect`. Check every `yield`.
Prefer push; reach for `iter.Pull` only when you consume two sequences at once or
hand out values on demand, and always `defer stop()`. Code from before Go 1.23 often
used a goroutine and a channel for the same job. Replacing it with an iterator removes
the goroutine leak on early exit and the scheduling cost per element.
