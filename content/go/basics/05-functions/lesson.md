---
id: go.basics.functions
title: "Functions: multiple returns, named results, variadic, closures"
track: go
module: basics
order: 5
level: basic
minutes: 13
since: go1.21
prerequisites: [go.basics.control-flow]
compare_with: [ts.functions.signatures, ts.functions.higher-order, design.paradigms.errors]
tags: [functions, errors, variadic, closures, multiple-returns]
summary: >
  Go functions return several values, and (value, error) is the idiom every
  API follows. Variadic parameters are slices, and closures capture variables,
  not copies.
examples:
  - id: multi-return
    file: examples/01_multi_return.go
    title: (value, error), named results and the min/max builtins
    runtime: go
    check: exact
    annotations:
      - lines: "12-17"
        note: >
          The error is a second return value, so the caller can't use the
          result without at least seeing it. On failure, return the zero value
          plus the error.
      - lines: "20-29"
        note: >
          Named results document what comes back and start at their zero value.
          A naked return returns them as they are, handy here, confusing in long
          functions. min and max are builtins since Go 1.21.
      - lines: "32-37"
        note: >
          q and err exist only inside each if. errors.Is compares against the
          ErrDivByZero sentinel, which keeps working when someone wraps the error.
      - lines: "39"
        note: >
          The standard library uses the same (value, error) shape everywhere, so
          callers handle every failure the same way.
  - id: variadic-closures
    file: examples/02_variadic_closures.go
    title: Variadic parameters are slices; closures keep state
    runtime: go
    check: exact
    annotations:
      - lines: "8-14"
        note: >
          Inside the function nums is a plain []int. sum() with no arguments
          gets a nil slice, which ranges zero times.
      - lines: "42-45"
        note: >
          xs... passes the slice itself, not a copy, so double writes into the
          caller's backing array. Callers who don't expect that get a surprise.
      - lines: "22-29"
        note: >
          The closure captures the variable count, not its value at creation.
          Each call to counter makes a new count, so counters don't share state.
      - lines: "47-48"
        note: >
          Arguments are evaluated left to right, so the output is 1 2 3 1: the
          second counter starts fresh.
      - lines: "50-51"
        note: >
          Functions are values: assign them, pass them, return them. A function
          type like func(string) string is often all the interface you need.
use_cases:
  - "(T, error) results for anything that can fail: parsing, I/O, lookups"
  - "Named results when a deferred function must adjust them (see the defer lesson)"
  - "Variadic helpers: fmt.Println, append, slices.Concat, option lists"
  - "Closures for counters, memoization, and callbacks passed to sort or http handlers"
strengths:
  - "Errors are visible in every signature; unlike Java's unchecked exceptions, a call that can fail looks different from one that can't"
  - "Multiple returns remove out-parameters (C) and throwaway tuple or result classes (Java)"
  - "Closures are lightweight and capture by reference, like JS closures, with no 'effectively final' rule as in Java lambdas"
weaknesses:
  - "No default parameters and no overloading: options become structs, variadic option functions, or several functions with different names"
  - "if err != nil repeats in every caller; there is no ? operator as in Rust"
  - "Variadic calls with slice... share the caller's backing array, an aliasing rule that is easy to forget"
pitfalls:
  - title: "Ignoring the error with _"
    detail: "v, _ := f() compiles and silently drops failures. Reserve _ for errors that genuinely can't happen, and say why in a comment."
  - title: "Naked returns in long functions"
    detail: "A bare return far from the result declarations hides what is returned. Keep naked returns for short functions."
  - title: "Expecting xs... to copy"
    detail: "The callee receives the caller's slice; writes are visible to the caller. Copy with slices.Clone when that's not wanted."
quiz:
  - q: "Inside func f(nums ...int), what is the type of nums?"
    options: ["...int", "[]int", "[N]int"]
    answer: 1
    why: "A variadic parameter is an ordinary slice inside the function."
  - q: "c1, c2 := counter(), counter(); fmt.Println(c1(), c1(), c2()). What prints?"
    options: ["1 2 3", "1 2 1", "1 1 1"]
    answer: 1
    why: "Each counter() call creates its own captured count variable."
  - q: "What should a function return together with a non-nil error?"
    options: ["Whatever partial result it has", "The zero value of the result type, unless the docs say otherwise", "nil, always"]
    answer: 1
    why: "Callers must not use the result when err != nil; returning the zero value makes accidental use harmless."
further_reading:
  - title: "The Go Programming Language Specification: Function types"
    url: "https://go.dev/ref/spec#Function_types"
  - title: "Effective Go: Functions"
    url: "https://go.dev/doc/effective_go#functions"
  - title: "The Go Programming Language Specification: Min and max"
    url: "https://go.dev/ref/spec#Min_and_max"
---

## Concept

A Go function can return several values. The convention for anything that can fail is
`(result, error)` with the error last; callers check it before touching the result. Results
can be named, which documents them and lets a `defer` change them. A final parameter
`...T` accepts any number of `T` and arrives as a `[]T`. Functions are first-class
values, and function literals are closures over the variables they mention.

## Coming from C/C++/Java/Python/JS

- C: multiple returns replace out-parameters and `errno`.
- Java: no checked or unchecked exceptions for ordinary failures; the error is part of the
  return type. No overloading and no default arguments.
- Python: multiple returns look like tuple returns, but they aren't a tuple value; you
  can't store `f()` in one variable when f returns two results.
- JS/TS: closures work the same way (capture by reference). Rest parameters `...xs`
  correspond to variadic `...T`.

## Design notes

Return errors, don't print them: the caller knows whether a failure matters. Keep
signatures small; when you need optional parameters, take an options struct rather than
inventing overloads. Name results when they clarify (`(lo, hi int)`) or when a deferred
function must touch them, and otherwise leave them unnamed. When a function only needs
one behavior from its caller, accept a function value instead of an interface.
