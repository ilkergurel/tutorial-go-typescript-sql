---
id: go.basics.control-flow
title: "if with init, the one for, switch without fallthrough, labels"
track: go
module: basics
order: 4
level: basic
minutes: 12
since: go1.22
prerequisites: [go.basics.variables]
compare_with: [ts.narrowing.basic, design.paradigms.side-by-side]
tags: [if, for, switch, labels, loop-variable]
summary: >
  Go has one loop (for), an if that can declare variables for its own scope,
  and a switch whose cases never fall through unless you say so.
examples:
  - id: if-switch
    file: examples/01_if_switch.go
    title: if with an init statement, and switch without break
    runtime: go
    check: exact
    annotations:
      - lines: "9-12"
        note: >
          Handle the error and return early. The happy path stays at the left
          margin instead of nesting inside else branches.
      - lines: "13-22"
        note: >
          A switch with no tag is a tidier if/else-if chain: cases are tested
          top to bottom and the first true one wins.
      - lines: "31-35"
        note: >
          The init statement declares v and ok for the if and its else only.
          The comma-ok form tells a missing key apart from a stored zero.
      - lines: "37-42"
        note: >
          Cases end on their own: no break, and no accidental fall-through as in
          C or Java. One case can list several values.
      - lines: "44-50"
        note: >
          fallthrough jumps into the next case body without testing its
          condition, which is why the second line prints. It must be the last
          statement in a case; real code rarely needs it.
  - id: loops
    file: examples/02_loops.go
    title: One for for everything, labeled break, per-iteration variables
    runtime: go
    check: exact
    annotations:
      - lines: "7-9"
        note: >
          Since Go 1.22, range accepts an integer and counts 0..n-1.
      - lines: "13-15"
        note: >
          A for with only a condition is Go's while loop; with nothing at all
          it loops forever.
      - lines: "19-27"
        note: >
          A label names the outer loop, so break outer leaves both loops at
          once. No flag variable, no goto.
      - lines: "29-39"
        note: >
          Since Go 1.22 every iteration has its own i, also in the three-clause
          form, so the closures print 0 1 2. A module whose go.mod says go 1.21
          or older still gets the old shared variable and prints 3 3 3.
use_cases:
  - "if v, err := f(); err != nil { ... } for errors that matter only in this branch"
  - "Tagless switch for range checks and classification"
  - "for range over slices, maps, strings, channels, integers and iterator functions"
  - "Labeled break/continue for nested searches"
strengths:
  - "One loop keyword covers counting, while, infinite and range loops; less syntax to remember than C's for/while/do-while"
  - "Cases never fall through by accident, the bug Java and C need break (and linters) to prevent"
  - "Init statements keep helper variables scoped to the branch that uses them"
weaknesses:
  - "No ternary operator: a one-line choice becomes a four-line if/else"
  - "switch is not exhaustive over iota constants or interface types; the compiler never warns about a missing case"
  - "The loop-variable fix depends on the go version in go.mod, so old modules keep the old behavior"
pitfalls:
  - title: "break inside switch inside for"
    detail: "A plain break in a switch case leaves the switch, not the loop. Label the loop and break the label."
    bad: "for { switch { case done: break } }"
    good: "loop: for { switch { case done: break loop } }"
  - title: "Assuming per-iteration variables in an old module"
    detail: "The Go 1.22 loop semantics follow the go line in go.mod. Code in a module declaring go 1.21 still shares one variable across iterations."
  - title: "Reading a missing map key without comma-ok"
    detail: "m[k] returns the zero value for a missing key, indistinguishable from a stored zero. Use v, ok := m[k] when the difference matters."
quiz:
  - q: "In Go, what happens at the end of a switch case body?"
    options: ["Execution falls through to the next case", "The switch ends", "It depends on break"]
    answer: 1
    why: "Cases never fall through unless the body ends with fallthrough."
  - q: "for i := 0; i < 3; i++ { fs = append(fs, func() int { return i }) } in a module declaring go 1.27. What do the closures return?"
    options: ["3 3 3", "0 1 2", "It doesn't compile"]
    answer: 1
    why: "Since Go 1.22 each iteration has its own i."
  - q: "How do you leave two nested loops at once?"
    options: ["break 2", "A label on the outer loop and break label", "goto is the only way"]
    answer: 1
    why: "break and continue accept a label naming an enclosing loop."
further_reading:
  - title: "The Go Programming Language Specification: For statements"
    url: "https://go.dev/ref/spec#For_statements"
  - title: "The Go Programming Language Specification: Switch statements"
    url: "https://go.dev/ref/spec#Switch_statements"
  - title: "Fixing For Loops in Go 1.22"
    url: "https://go.dev/blog/loopvar-preview"
---

## Concept

Go keeps three control structures and makes each do more. `if` and `switch` accept an
init statement whose variables live only in that statement. `switch` evaluates cases top
to bottom, stops at the first match and never falls through by default; without a tag it
replaces if/else-if chains. `for` is the only loop: three-clause, condition-only (while),
bare (forever) or `range`. Labels let `break` and `continue` target an outer loop.

## Coming from C/C++/Java/Python/JS

- C/Java: no parentheses around conditions, braces are mandatory, and `switch` needs no
  `break`. There is no `do-while` and no ternary `?:`.
- Python: `range` over an integer works like `range(n)`; labeled `break` covers what
  Python does with flags or `for/else`.
- JS: the Go 1.22 loop variable behaves like `let` in a JS `for` loop (one binding per
  iteration). The pre-1.22 behavior was JS's `var`.

## Design notes

Use the init form when a value only matters for one branch (`if err := f(); err != nil`).
Return early on errors instead of nesting the success path in `else`; this is the shape
most Go code has, and reviewers expect it. Prefer a tagless `switch` over long
if/else-if chains. Keep `fallthrough` for the rare case where one case is a strict
superset of the next, and leave a comment when you use it.
