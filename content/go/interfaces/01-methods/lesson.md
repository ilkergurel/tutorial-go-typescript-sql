---
id: go.interfaces.methods
title: "Methods: value vs pointer receivers and method sets"
track: go
module: interfaces
order: 1
level: basic
minutes: 13
since: go1.0
prerequisites: [go.basics.pointers, go.composite.structs]
compare_with: [ts.objects-classes.classes, go.composite.structs]
tags: [methods, receivers, method-sets, pointers]
summary: >
  A receiver is passed like any other argument: a value receiver gets a copy,
  a pointer receiver shares the caller's value. Pick one kind per type, because
  mixed receivers let the compiler accept code that silently works on copies.
examples:
  - id: receivers
    file: examples/01_receivers.go
    title: Value receivers copy, pointer receivers share
    runtime: go
    check: exact
    annotations:
      - lines: "7-11"
        note: >
          IncCopy compiles, runs and changes nothing the caller can see: the
          receiver is an ordinary parameter, copied like any other. The
          compiler doesn't warn. Counter mixes both kinds only to show the
          difference; the next example shows why real types shouldn't.
      - lines: "17"
        note: >
          Go takes the address for you when the variable is addressable, so
          value and pointer methods look identical at the call site. That is
          also why the lost increment on line 15 is easy to miss in review.
      - lines: "21-24"
        note: >
          range hands you a copy of each element. Since Go 1.22 k is a fresh
          variable per iteration, but it is still a copy, so Inc changes
          nothing in the slice. Indexing (line 26) reaches the element itself.
      - lines: "30-34"
        note: >
          Map elements are not addressable, so Go can't insert the &: the
          commented call fails with "cannot call pointer method Inc on
          Counter". Copy out, modify, store back, or keep *Counter values in
          the map.
  - id: mixed-receivers
    file: examples/02_mixed_receivers.go
    title: "Mixed receivers: interfaces and method values capture copies"
    runtime: go
    check: exact
    annotations:
      - lines: "5-9"
        note: >
          With String on the value, both Hits and *Hits satisfy fmt.Stringer.
          That looks convenient, and it is the trap: the compiler now accepts
          code that quietly works on a copy.
      - lines: "19-22"
        note: >
          Storing h in an interface copies it. A method value with a value
          receiver evaluates and saves the receiver immediately, the same rule
          as a defer's arguments. Both print hits=0 after the increment.
      - lines: "25-29"
        note: >
          With pointer receivers throughout, the copy can't happen by accident:
          the commented assignment fails with "method String has pointer
          receiver", and v.String binds &v, so all three agree.
      - lines: "31-32"
        note: >
          The one hole: fmt.Println takes any and looks for String at run
          time. A Visits value has no String in its method set, so fmt prints
          the raw struct. Pass &v, or hand *Visits around everywhere.
  - id: non-struct-types
    file: examples/03_non_struct_types.go
    title: Methods on slices and maps, and nil pointer receivers
    runtime: go
    check: exact
    annotations:
      - lines: "8-13"
        note: >
          A slice value is a small header (pointer, length, capacity). A value
          receiver could overwrite existing elements, but its append would be
          lost with the copied header. Growing needs *Tags, and consistency
          moves String to *Tags too.
      - lines: "15-19"
        note: >
          Map, channel and func values already refer to shared state, so value
          receivers are idiomatic for them. Writing to a nil map still panics,
          which is why line 40 makes one.
      - lines: "27-32"
        note: >
          A pointer-receiver method may run with a nil receiver: it is only a
          parameter. Here nil means "empty tree", so the recursion needs no nil
          checks at the call sites. Java would throw a NullPointerException
          before the method body ran.
      - lines: "46-48"
        note: >
          empty.Sum() is a legal call on a nil pointer. Lesson
          go.interfaces.assertions shows what happens when such a nil pointer
          is stored in an interface.
use_cases:
  - "Pointer receivers for anything with mutable state or a sync.Mutex inside: services, repositories, caches, builders"
  - "Value receivers for small immutable values: time.Time-style types, Celsius, Money, IDs"
  - "Methods on non-struct types: http.HandlerFunc is a func type with a ServeHTTP method; http.Header is a map type"
  - "nil receivers as a cheap default: an empty tree or list that still answers Len and Sum"
strengths:
  - "Unlike Java or JS, where a method always receives a reference, a value receiver guarantees the method can't change the caller's data; time.Time relies on that"
  - "Methods attach to any named type, so a plain function becomes an http.Handler through http.HandlerFunc instead of a wrapper class"
  - "A method value binds its receiver: f := c.Inc keeps working when passed around, unlike JS where const f = obj.method loses this"
  - "nil receivers are legal, so a nil pointer can be a working empty value instead of needing a null-object class"
weaknesses:
  - "Call sites look the same for both receiver kinds, so a wrong value receiver is invisible where the method is used"
  - "The compiler accepts a value receiver that modifies its copy; only review, tests and go vet's copylocks check (for mutexes) catch it"
  - "Mixing receivers is legal: the one-kind-per-type rule is a convention from the FAQ and review guidelines, not a compiler check"
pitfalls:
  - title: "A value receiver on a struct that holds a sync.Mutex"
    detail: "Each call locks its own copy of the mutex, so nothing is protected. go vet reports \"passes lock by value\"."
    bad: "func (s Store) Get(k string) string { s.mu.Lock(); defer s.mu.Unlock(); return s.m[k] }"
    good: "func (s *Store) Get(k string) string { s.mu.Lock(); defer s.mu.Unlock(); return s.m[k] }"
  - title: "Calling a mutating method on a range copy"
    detail: "for _, c := range items gives copies; c.Inc() changes the copy. Index into the slice, or store pointers."
    bad: "for _, c := range counters { c.Inc() }"
    good: "for i := range counters { counters[i].Inc() }"
  - title: "fmt ignores a pointer-receiver String on a value"
    detail: "fmt.Println(v) prints the raw struct when String is declared on *T. Pass &v. The same applies to Error methods and errors."
  - title: "Naming the receiver this or self"
    detail: "Go convention is a short name derived from the type, the same in every method: (c *Counter), (t *Tree)."
quiz:
  - q: "func (c Counter) Inc() { c.n++ }. After var c Counter; c.Inc(); c.Inc(), what is c.n?"
    options: ["0", "1", "2", "It doesn't compile"]
    answer: 0
    why: "A value receiver is a copy; both increments change copies that are discarded."
  - q: "Inc has a pointer receiver. Which call does not compile?"
    options: ["c.Inc() on a local variable c", "s[0].Inc() on a slice element", "m[\"k\"].Inc() on a map element", "p.Inc() on a *Counter p"]
    answer: 2
    why: "Go inserts & only for addressable operands, and map elements are not addressable."
  - q: "Hits has Inc with a pointer receiver and String with a value receiver. var s fmt.Stringer = h; h.Inc(); fmt.Println(s) prints what?"
    options: ["The count before Inc", "The count after Inc", "Nothing: the assignment doesn't compile"]
    answer: 0
    why: "Assigning h to the interface copied it; Inc changed h, not the copy inside s."
  - q: "type Stack []int needs a Push that appends. Which receivers should Stack's methods use?"
    options: ["Value receivers: slices are references", "Pointer receivers for every method", "Pointer for Push, value receivers for the rest"]
    answer: 1
    why: "append changes the length, which only a pointer receiver can report back; the FAQ then asks for the same kind on every method."
further_reading:
  - title: "Go FAQ: Should I define methods on values or pointers?"
    url: "https://go.dev/doc/faq#methods_on_values_or_pointers"
  - title: "Go FAQ: Why do T and *T have different method sets?"
    url: "https://go.dev/doc/faq#different_method_sets"
  - title: "Go Code Review Comments: Receiver Type"
    url: "https://go.dev/wiki/CodeReviewComments#receiver-type"
  - title: "The Go Programming Language Specification: Method sets"
    url: "https://go.dev/ref/spec#Method_sets"
  - title: "The Go Programming Language Specification: Method values"
    url: "https://go.dev/ref/spec#Method_values"
---

## Concept

A method is a function with a receiver: `func (c *Counter) Inc()`. The receiver is a
parameter like any other, so a value receiver (`c Counter`) gets a copy and a pointer
receiver (`c *Counter`) shares the caller's value. Methods can be declared on any named
type defined in the same package: structs, but also slices, maps, functions and numbers
(not pointer or interface types). They can't be added to types from other packages, such
as `int` or `time.Time`.

The method set of `T` holds its value-receiver methods; the method set of `*T` holds
both kinds. Method sets decide which interfaces a type satisfies (the next lesson). At
call sites Go hides the difference: it takes `&c` for you when `c` is addressable, and
dereferences a pointer when the method wants a value.

The working rule is one decision per type, not per method. If any method must mutate the
receiver, or the type holds a mutex, or it is large, every method gets a pointer
receiver. If the type is a small value that never changes in place, every method gets a
value receiver.

## Coming from C/C++/Java/Python/JS

- Java/Python/JS: `this` and `self` are always references. Go's value receivers are the
  unusual part, as the Go FAQ itself points out.
- C++: a pointer receiver is like a non-const member function; a value receiver is like a
  member function that receives its own copy of `*this`. There is no `->`: Go takes
  addresses and dereferences at call sites automatically.
- JS: `const f = obj.method; f()` loses `this`. In Go, `f := c.Inc` is a method value
  with the receiver bound; for a value receiver, the receiver is copied at that moment.
- Python: the receiver is an explicit parameter like `self`, but you name it after the
  type (`c`, `t`), never `this` or `self`.

## How it works

    you have                                  value-receiver method   pointer-receiver method
    addressable T (variable, slice element)   x.M()                   (&x).M(), inserted by Go
    non-addressable T (map element, literal)  x.M()                   compile error
    *T                                        (*p).M(), inserted      p.M()
    T assigned to an interface needing M      ok, holds a copy        compile error: use &x

The last row has a reason: an interface holds its own copy of a `T`, and a pointer
method changing that copy would change nothing you can see. The FAQ's example is `io.Copy(buf, os.Stdin)` with a `bytes.Buffer` value,
which would copy into a copy of `buf`.

## Design notes

Decide per type, in this order, which follows the Go FAQ and the Code Review Comments.
A type that mutates, holds a `sync.Mutex`, or is large gets pointer receivers on every
method. Map, channel and func types get value receivers. Small immutable values get
value receivers. When in doubt, use pointers. The standard library's honest exception is
`time.Time`: value receivers everywhere except `UnmarshalJSON`, `UnmarshalText`,
`UnmarshalBinary` and `GobDecode`, which must write into the value. That is the one good
reason to mix. Mixing for convenience brings back the copies example 2 shows, and the
compiler won't point them out.
