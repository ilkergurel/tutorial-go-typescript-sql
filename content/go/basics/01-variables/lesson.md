---
id: go.basics.variables
title: "var vs :=, zero values, constants and iota"
track: go
module: basics
order: 1
level: basic
minutes: 12
since: go1.0
prerequisites: []
compare_with: [ts.everyday-types.null, sql.querying.null-logic]
tags: [variables, zero-value, constants, iota, shadowing]
summary: >
  Every Go variable starts at a usable zero value, := declares while = assigns,
  and constants are exact compile-time values that iota can enumerate.
examples:
  - id: zero-values
    file: examples/01_zero_values.go
    title: Zero values are usable, not just empty
    runtime: go
    check: exact
    annotations:
      - lines: "14"
        note: >
          There is no "uninitialized" state: var gives every field its zero
          value ("", 0, false, nil). Unlike C, no garbage; unlike Java, a string
          field is "" rather than null.
      - lines: "16-20"
        note: >
          A nil slice is a valid empty slice (append allocates on demand) and a
          nil map reads as empty. Only writing to a nil map panics, as the
          comment on the next line says.
      - lines: "23-24"
        note: >
          := declares and infers the type, inside functions only. With several
          names on the left it needs at least one new one; the others are
          plainly assigned, which is how n changes from 3 to 4 here.
      - lines: "26"
        note: >
          Both right-hand values are read before either variable is written,
          so a swap needs no temporary.
  - id: shadowing
    file: examples/02_shadowing.go
    title: The := shadowing bug and its one-character fix
    runtime: go
    check: exact
    annotations:
      - lines: "21"
        note: >
          Inside the block, := declares brand-new result and err that hide the
          outer ones. The outer variables are never written, and it compiles
          because the inner ones are used. Default go vet doesn't report it.
      - lines: "27"
        note: >
          = assigns the variables declared on lines 16-17. When a block should
          update outer state, declare first and assign inside.
  - id: iota
    file: examples/03_iota.go
    title: Typed enums with iota, and untyped constants
    runtime: go
    check: exact
    annotations:
      - lines: "10-15"
        note: >
          iota is the line index inside one const block, starting at 0. A line
          without an expression repeats the previous one, type included, so all
          four constants are Level values.
      - lines: "17-29"
        note: >
          A String method makes Level a fmt.Stringer, so fmt prints names. The
          fallback keeps an unknown value like Level(7) debuggable.
      - lines: "31-36"
        note: >
          The blank identifier skips iota's 0; the repeated expression then
          yields 1<<10, 1<<20 and 1<<30.
      - lines: "38-43"
        note: >
          Untyped constants are exact values of arbitrary precision. 1<<100 never
          exists at run time; only the result 4 has to fit in an int.
      - lines: "45"
        note: >
          The untyped 2 takes its type from time.Second, a time.Duration. The
          same with a variable n would need time.Duration(n): the next lesson.
use_cases:
  - "var x T when the zero value is what you want, or a declaration without an obvious initial value"
  - ":= for nearly every local variable, especially results of calls: v, err := f()"
  - "const + iota for enum-like sets (log levels, states) with a String method"
  - "Untyped constants for unit math such as KB/MB or timeouts (2 * time.Second)"
strengths:
  - "Zero values remove a whole class of null checks: a zero sync.Mutex, bytes.Buffer or empty struct is ready to use, unlike Java fields that start as null"
  - "Constants are checked at compile time with exact arithmetic, so 1<<100 overflow mistakes surface as compile errors rather than wrapped values at run time"
  - "Declaration forms are few: var, :=, const. No let/var/const scoping puzzles as in pre-ES6 JavaScript"
weaknesses:
  - "iota enums are just integers: any int converts to Level, and nothing checks a switch for exhaustiveness the way TypeScript's never does"
  - ":= in an inner scope silently shadows outer variables, and the default go vet doesn't report it"
  - "A zero value isn't always meaningful: 0 retries or an empty name may be a real setting or a forgotten field, and the type can't tell you which"
pitfalls:
  - title: "Shadowing err or a result inside if/for blocks"
    detail: "x, err := f() inside a block creates new variables; the outer ones keep their old values. Declare outside and use = inside when the block should update them."
    bad: "if ok { result, err := lookup(k) }"
    good: "if ok { result, err = lookup(k) }"
  - title: "Writing to a nil map"
    detail: "Reading a nil map is fine, writing panics. Initialize maps with make (or a literal) before the first write."
    bad: "var m map[string]int; m[\"a\"] = 1"
    good: "m := make(map[string]int); m[\"a\"] = 1"
  - title: "Relying on iota values in storage"
    detail: "Inserting a new constant in the middle renumbers everything after it. Persist names (or explicit numbers), not iota positions."
quiz:
  - q: "What does fmt.Println print for a struct field `Tags []string` that was never assigned?"
    options: ["nil", "[]", "<nil>", "It doesn't compile"]
    answer: 1
    why: "The field's zero value is a nil slice, and fmt prints a nil slice as []."
  - q: "`a, err := f()` appears inside an if block while a and err are already declared outside. What happens?"
    options: ["Compile error: no new variables", "The outer a and err are assigned", "New a and err are declared for the block; the outer ones are untouched"]
    answer: 2
    why: "The block is a new scope, so := declares fresh variables there. The 'no new variables' error only applies within the same scope."
  - q: "In `const ( A = iota * 10; B; C )`, what is C?"
    options: ["2", "20", "0"]
    answer: 1
    why: "B and C repeat the expression iota * 10 with iota = 1 and 2."
further_reading:
  - title: "The Go Programming Language Specification: Declarations and scope"
    url: "https://go.dev/ref/spec#Declarations_and_scope"
  - title: "The Go Programming Language Specification: Constants and iota"
    url: "https://go.dev/ref/spec#Iota"
  - title: "The Go Programming Language Specification: The zero value"
    url: "https://go.dev/ref/spec#The_zero_value"
---

## Concept

Go has three ways to introduce a name: `var x T` (zero value), `x := expr` (declare and
infer, inside functions only) and `const`. There is no uninitialized memory: every
variable starts at its type's zero value, and the standard library designs types so that
the zero value works (a zero `sync.Mutex` is unlocked, a zero `bytes.Buffer` is empty and
ready). Constants are compile-time values with exact arithmetic; `iota` numbers the lines
of a `const` block, which is how Go spells enums.

## Coming from C/C++/Java/Python/JS

- C/C++: no uninitialized locals and no constructors; `var c Config` is fully initialized.
- Java: no `null` for strings or numbers. Only pointers, slices, maps, channels, functions
  and interfaces have `nil` as their zero value.
- Python: names are declared, not created on first assignment, and a typo in an
  assignment is a compile error instead of a new variable.
- JS/TS: `:=` is roughly `let` with inference, but there is no `undefined`. A `const`
  in Go must be a compile-time value; JS `const` only forbids reassignment.
- Java/TS enums are their own types with a closed set of values. Go's `iota` constants
  are named integers: nothing stops `Level(7)`.

## Design notes

Prefer `:=` for locals and `var` when the zero value is the point (`var buf
bytes.Buffer`). Make your own types useful at their zero value where you can: then
callers never need a constructor for the common case. For enums, add a `String` method
and handle the unknown case explicitly, because the compiler won't check that a
`switch` covers every constant. Watch for `:=` inside `if`/`for` blocks whenever you mean
to update an outer variable; it is the most common beginner bug that compiles cleanly.
