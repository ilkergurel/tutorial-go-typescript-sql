---
id: go.basics.pointers
title: "Pointers without arithmetic; new(expr) vs &T{}"
track: go
module: basics
order: 7
level: basic
minutes: 12
since: go1.26
prerequisites: [go.basics.functions]
compare_with: [go.composite.structs, ts.objects-classes.spread]
tags: [pointers, escape-analysis, new, nil, optional-fields]
summary: >
  Go passes everything by value; a pointer is how a callee changes the
  caller's data. There is no pointer arithmetic, returning &local is safe,
  and Go 1.26's new(expr) points at any value.
examples:
  - id: value-vs-pointer
    file: examples/01_value_vs_pointer.go
    title: Copies vs pointers, and why returning &local is safe
    runtime: go
    check: exact
    annotations:
      - lines: "10"
        note: >
          A struct parameter is a full copy. The increment changes the copy,
          which disappears when the function returns.
      - lines: "12"
        note: >
          With a pointer the callee changes the caller's struct. c.Hits is
          shorthand for (*c).Hits: no -> operator.
      - lines: "14-19"
        note: >
          In C this would return a dangling pointer. Go's escape analysis sees
          that c outlives the call and allocates it on the heap; go build
          -gcflags=-m prints "moved to heap: c".
      - lines: "32-34"
        note: >
          Copying a pointer copies the address: alias and p name the same
          struct, so the write through alias shows up through p.
      - lines: "36-39"
        note: >
          The zero value of a pointer is nil. Dereferencing it panics, and
          pointer arithmetic doesn't exist outside package unsafe.
  - id: new-expr
    file: examples/02_new_expr.go
    title: Pointers to values for optional fields; new(expr) in Go 1.26
    runtime: go
    check: exact
    annotations:
      - lines: "5-10"
        note: >
          A pointer field can say "not set", which "" and 0 can't. This is the
          usual shape for PATCH requests and optional JSON fields.
      - lines: "23-25"
        note: >
          &"api" and &0 don't compile: only addressable operands, such as
          variables, have an address. Codebases wrote this generic helper to
          copy the value into a variable first.
      - lines: "29"
        note: >
          Since Go 1.26, new accepts an expression and returns a pointer to a
          new variable holding its value, which retires the helper.
      - lines: "33-35"
        note: >
          new(int) still works and points at a zeroed int; new(7) points at a
          fresh int holding 7.
      - lines: "37"
        note: >
          &T{...} remains the idiomatic way to get a pointer to a new struct:
          taking the address of a composite literal is allowed on purpose.
use_cases:
  - "Letting a function modify its argument: func (c *Counter) Inc()"
  - "Avoiding copies of large structs in hot paths (measure first)"
  - "Optional values: *string and *int fields for PATCH bodies and nullable columns"
  - "Constructors returning *T: func NewServer(cfg Config) *Server"
strengths:
  - "No pointer arithmetic and garbage collection: no buffer overruns or use-after-free through pointers, unlike C"
  - "Escape analysis decides stack vs heap, so returning &local is safe and idiomatic"
  - "Value semantics are the default, so a copy never aliases by accident, unlike Java and JS where every object is a reference"
weaknesses:
  - "nil pointer dereferences still crash at run time; the type system can't mark a pointer as non-nil"
  - "Pointer vs value is a choice at every API, and a struct that mixes both kinds of methods confuses readers"
  - "Heap allocations caused by escaping pointers cost garbage-collection time in hot code"
pitfalls:
  - title: "Taking the address of a loop element copy"
    detail: "In for _, v := range items, v is a copy; &v points at that copy, not into the slice. Use &items[i] to point into the slice."
    bad: "for _, v := range items { ptrs = append(ptrs, &v) }"
    good: "for i := range items { ptrs = append(ptrs, &items[i]) }"
  - title: "Using pointers 'for speed' by default"
    detail: "Small structs are cheap to copy, and a pointer can force a heap allocation. Use pointers when you need sharing or mutation, and measure before optimizing."
  - title: "Dereferencing optional fields without a nil check"
    detail: "*p.Name panics when Name is nil. Check first, or give the type a method that does."
quiz:
  - q: "func inc(c Counter) { c.Hits++ } is called with a variable c. Does the caller's c.Hits change?"
    options: ["Yes", "No, the function got a copy"]
    answer: 1
    why: "Go passes arguments by value; only a pointer lets the callee modify the caller's struct."
  - q: "Is `func newT() *T { t := T{}; return &t }` safe?"
    options: ["No, t lives on the stack", "Yes, escape analysis moves t to the heap"]
    answer: 1
    why: "The compiler allocates variables whose address escapes on the heap."
  - q: "Which expression gives a *int pointing at 5 in Go 1.26?"
    options: ["&5", "new(5)", "*int(5)"]
    answer: 1
    why: "new(expr) creates a variable initialized to the value and returns its address; &5 doesn't compile."
further_reading:
  - title: "The Go Programming Language Specification: Address operators"
    url: "https://go.dev/ref/spec#Address_operators"
  - title: "The Go Programming Language Specification: Allocation"
    url: "https://go.dev/ref/spec#Allocation"
  - title: "Go 1.26 Release Notes"
    url: "https://go.dev/doc/go1.26"
---

## Concept

Every assignment and every function argument in Go copies the value. A pointer `*T`
holds the address of a `T`; `&x` takes an address and `*p` dereferences, while field
access and method calls dereference automatically. There is no pointer arithmetic, and
the garbage collector owns memory: the compiler's escape analysis puts a variable on the
heap when its address outlives the function, so returning `&local` is safe. `new(T)`
returns a pointer to a zeroed `T`; since Go 1.26 `new(expr)` returns a pointer to a new
variable holding the value of `expr`.

## Coming from C/C++/Java/Python/JS

- C/C++: no arithmetic, no dangling pointers, no `->`, no `free`. `&local` is safe.
- Java/Python/JS: objects there are always references. In Go a struct is a value until you
  take its address, so `b := a` copies the whole struct.
- TS: `?:` optional properties correspond to pointer fields (`*string`), where `nil` means
  absent and `""` is a real value.

## Design notes

Use a pointer when the callee must modify the value, when the value must be shared, or
when it is genuinely large; otherwise pass values. Pick one receiver kind per type
(usually pointer receivers for anything with mutable state). Pointer fields are the
standard way to express "optional" in structs that map to JSON or SQL, and `new(expr)`
removes the boilerplate of filling them. Let escape analysis do its job and check
`-gcflags=-m` only when a profile points at allocations.
