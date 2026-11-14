---
id: go.interfaces.implicit
title: "Interfaces are satisfied implicitly"
track: go
module: interfaces
order: 2
level: basic
minutes: 12
since: go1.0
prerequisites: []
compare_with: [ts.narrowing.guards]
tags: [interfaces, methods, implicit]
summary: >
  A Go type satisfies an interface just by having its methods — no implements
  declaration. Small interfaces plus value vs pointer receivers shape idiomatic
  Go APIs.
examples:
  - id: implicit
    file: examples/01_implicit.go
    title: "Satisfaction without declaration; receiver rules"
    runtime: go
    check: exact
    annotations:
      - lines: "5-13"
        note: >
          Account satisfies Teller just by having a Balance method; nothing says
          "implements". Teller is declared where it is used and asks for one
          method, and small interfaces are easy to satisfy.
      - lines: "28-29"
        note: >
          No implements keyword anywhere: Account converts to Teller, and
          Celsius prints through its String method, by method set alone.
      - lines: "31-33"
        note: >
          Meter has a pointer receiver (line 23), so only *Gauge has it in its
          method set: var m Meter = &g compiles, var m Meter = g would not.
use_cases:
  - "Accepting narrow interfaces in functions so callers can pass anything with the right method"
  - "io.Reader/io.Writer composition: your types plug into the whole stdlib for free"
  - "Testing: a small interface is trivially satisfied by a fake"
strengths:
  - "Decoupling without ceremony: packages never import each other just to name a shared interface"
  - "Interfaces defined at the consumer are as small as that consumer needs"
  - "Retrofitting: existing types gain new interface compatibility without edits"
weaknesses:
  - "No self-documentation: nothing at the type's declaration lists what it satisfies"
  - "Method-set rules (value vs pointer receivers) are subtle and fail at compile time far from the cause"
  - "Accidental satisfaction: a type can match an interface by coincidence with wrong semantics"
pitfalls:
  - title: "Storing a value where the pointer satisfies the interface"
    detail: "If methods use pointer receivers, T does not satisfy the interface — only *T does. A slice of values fails to compile where a slice of pointers works."
  - title: "Declaring interfaces next to implementations, Java-style"
    detail: "Go style is to declare the interface where it is USED, with only the methods that call site needs. Big Java-style interfaces force implementers to promise more than anyone relies on."
quiz:
  - q: "How does a Go type declare that it implements an interface?"
    options: ["implements keyword", "It just has the methods", "A compile-time assertion", "In the package clause"]
    answer: 1
    why: "Satisfaction is implicit and structural: the method set is the contract."
  - q: "type S has a method with a pointer receiver. Which satisfies interface I { m() }?"
    options: ["S only", "*S only", "Both", "Neither"]
    answer: 1
    why: "Pointer-receiver methods belong only to the pointer's method set, so *S satisfies I and S does not."
  - q: "Where is it idiomatic to declare an interface?"
    options: ["Next to the concrete type", "In a shared interfaces package", "In the package that consumes it", "Nowhere — inline struct types"]
    answer: 2
    why: "Consumer-declared interfaces stay minimal and decouple packages."
further_reading:
  - title: "Effective Go: interfaces"
    url: "https://go.dev/doc/effective_go#interfaces"
  - title: "Go blog: The Laws of Reflection (method sets)"
    url: "https://go.dev/blog/laws-of-reflection"
---

## Concept

Go interfaces are satisfied *implicitly*: a type implements an interface when
its method set contains the interface's methods — no `implements` keyword, no
registration. This inverts the usual dependency direction. The interface is
declared by whoever needs it, often with just one or two methods, and any type
that happens to fit can be used without knowing about the interface.

The fine print is method sets: methods with a value receiver belong to both
`T` and `*T`; methods with a pointer receiver belong only to `*T`. So a type
whose methods use pointer receivers satisfies an interface only through its
pointer.

## Coming from C/C++/Java/Python/JS

Java (and C#, TypeScript) demand an up-front `implements` declaration — the
type must import and name its interface, creating a compile-time dependency
from implementation to abstraction. C++ does it with abstract base classes and
the same coupling. Python/JS have structural typing at runtime (duck typing)
but no compiler to check it; Go is "duck typing with a proof": structural like
Python, but verified statically. If you have ever written `class Foo implements
Bar` only to satisfy a linter, Go deletes that line.

## Design notes

The standard library's tiny interfaces (`io.Reader` is one method) only work
because satisfaction is implicit: your type becomes an `io.Reader` the moment
it has a matching `Read`, with zero declarations. Consumer-side declaration
keeps interfaces minimal — an interface with 30 methods where callers use 2 is
a Java habit, not a Go one.
