---
id: go.basics.defer
title: "defer: LIFO order and when arguments are evaluated"
track: go
module: basics
order: 6
level: basic
minutes: 12
since: go1.22
prerequisites: [go.basics.functions]
compare_with: [ts.errors.try-catch, go.errors.panic-strategy]
tags: [defer, recover, panic, cleanup, os-exit]
summary: >
  defer schedules a call for when the function returns, in reverse order,
  with its arguments evaluated immediately. Deferred closures can rewrite
  named results; os.Exit skips them all.
examples:
  - id: lifo-args
    file: examples/01_lifo_args.go
    title: Last in, first out, and arguments evaluated at defer time
    runtime: go
    check: exact
    annotations:
      - lines: "6-8"
        note: >
          Deferred calls run when main returns, not at the end of each loop
          iteration, and in reverse order. A loop that opens a file per
          iteration and defers Close keeps every file open until return.
      - lines: "10-11"
        note: >
          The arguments of a deferred call are evaluated when the defer
          statement runs, so this line prints 1 although x is 2 by then.
      - lines: "12-14"
        note: >
          A closure reads x when it finally runs, so it prints 2. Pick the form
          that captures the value you mean.
  - id: named-result
    file: examples/02_named_result.go
    title: Deferred closures can change named results, and recover from panics
    runtime: go
    check: exact
    annotations:
      - lines: "11-16"
        note: >
          The return statement assigns port and err first; then the deferred
          closure runs and can replace err. One place adds context to every
          error path.
      - lines: "17-24"
        note: >
          All three returns pass through the deferred wrapper, including the
          range check added later. Nobody can forget the context.
      - lines: "29-36"
        note: >
          recover stops a panic only when called directly by a deferred
          function. Here the integer division by zero becomes an ordinary
          error value.
  - id: exit-skips-defer
    file: examples/03_exit_skips_defer.go
    title: os.Exit (and log.Fatal) skip deferred calls
    runtime: go
    check: exact
    annotations:
      - lines: "9"
        note: >
          This cleanup never prints: the expected output has a single line.
      - lines: "11-14"
        note: >
          os.Exit ends the process immediately, without running deferred calls.
          log.Fatal and log.Fatalf call os.Exit(1), so in a server they skip
          the cleanup you deferred.
use_cases:
  - "Releasing resources right after acquiring them: defer f.Close(), defer mu.Unlock(), defer rows.Close()"
  - "Adding context to every error of a function in one deferred closure"
  - "Turning a panic into an error at a boundary (an HTTP handler, a worker goroutine)"
  - "Timing a function: defer logDuration(time.Now())"
strengths:
  - "Cleanup sits next to the acquisition, so it can't drift away from it, unlike a finally block at the bottom of a long Java method"
  - "Runs on every return path and during panics, which try/finally also does, but without the extra nesting"
  - "Argument evaluation at defer time makes defer logDuration(time.Now()) work with no closure"
weaknesses:
  - "Defers are per function, not per block: in loops they pile up until the function returns"
  - "os.Exit and log.Fatal bypass them, so cleanup depends on how the program ends"
  - "An error returned by a deferred Close is easy to ignore; handling it needs a named result and a closure"
pitfalls:
  - title: "defer in a loop"
    detail: "Each iteration's defer waits for the function to return. Move the loop body into its own function so each defer runs per iteration."
    bad: "for _, p := range paths { f, _ := os.Open(p); defer f.Close() }"
    good: "for _, p := range paths { if err := process(p); err != nil { return err } } // process defers Close"
  - title: "log.Fatal in a function with defers"
    detail: "It calls os.Exit(1); deferred cleanup never runs. Return the error and exit from main after cleanup instead."
  - title: "Expecting a deferred call to see later changes"
    detail: "defer fmt.Println(x) captures x now. Use a closure if you want the value at return time."
quiz:
  - q: "x := 1; defer fmt.Println(x); x = 2. What prints when the function returns?"
    options: ["1", "2", "Nothing"]
    answer: 0
    why: "Arguments are evaluated when the defer statement executes."
  - q: "Three defers A, B, C are registered in that order. In which order do they run?"
    options: ["A B C", "C B A", "Unspecified"]
    answer: 1
    why: "Deferred calls run last in, first out."
  - q: "Where does recover() stop a panic?"
    options: ["Anywhere in the panicking goroutine", "Only when called directly by a deferred function", "In main only"]
    answer: 1
    why: "Called anywhere else, recover returns nil and the panic continues."
further_reading:
  - title: "The Go Programming Language Specification: Defer statements"
    url: "https://go.dev/ref/spec#Defer_statements"
  - title: "Defer, Panic, and Recover"
    url: "https://go.dev/blog/defer-panic-and-recover"
  - title: "package os: Exit"
    url: "https://pkg.go.dev/os#Exit"
---

## Concept

`defer f(args)` schedules `f` to run when the surrounding function returns, whether it
returns normally or panics. Deferred calls run in last-in, first-out order. The function
value and its arguments are evaluated when the `defer` statement executes; only the call
is delayed. Because deferred functions run after the return values are set, a deferred
closure can read and change named results, and it is the only place where `recover`
stops a panic.

## Coming from C/C++/Java/Python/JS

- Java/JS: `defer` replaces most `try/finally` blocks, and it sits next to the
  acquisition instead of at the end of the method.
- Python: similar to a `with` block's exit, but tied to the function, not to a block.
- C++: like a destructor-based scope guard (RAII), but explicit and scoped to the whole
  function.
- `os.Exit` skipping defers matches Java's `System.exit` skipping `finally` and Node's
  `process.exit`. Python's `sys.exit` differs: it raises `SystemExit`, so `finally` and
  `with` blocks still run.

## Design notes

Write the `defer` on the line right after a successful acquisition (`f, err :=
os.Open(p); if err != nil { return err }; defer f.Close()`). If a loop acquires
resources, put the loop body in its own function. Use `recover` only at boundaries you
own, such as an HTTP handler or a worker goroutine, and convert the panic to an error
there; recovering everywhere hides bugs. In `main`, prefer returning errors to a single
exit point over calling `log.Fatal` deep in the code, so deferred cleanup still runs.
