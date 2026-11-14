---
id: go.generics.methods-when
title: "Generic methods (Go 1.27) and when an interface wins"
track: go
module: generics
order: 2
level: advanced
minutes: 15
since: go1.27
prerequisites: [go.generics.parameters, go.interfaces.implicit]
compare_with: [ts.generics.functions, design.paradigms.polymorphism, design.paradigms.generics-absence]
tags: [generics, generic-methods, interfaces, reflection, go1.27]
summary: >
  Since Go 1.27 a method can declare its own type parameters, which makes
  left-to-right chains like list.Filter(even).Map(strconv.Itoa) possible. Such
  methods never satisfy interfaces, so interfaces still own run-time polymorphism.
examples:
  - id: generic-method
    file: examples/01_generic_method.go
    title: A method with its own type parameter, and the function it replaces
    runtime: go
    check: exact
    annotations:
      - lines: "23-30"
        note: >
          E comes from the receiver; F is Map's own type parameter, declared
          after the method name exactly as on a function. In a module whose
          go.mod says go 1.26 or older this line fails with "generic method
          requires go1.27 or later".
      - lines: "32-33"
        note: >
          Before Go 1.27 the only option was a package-level function. It still
          works, and it is still how the slices and maps packages are shaped.
      - lines: "42-43"
        note: >
          Same result, two shapes. The nested calls read inside out: the last
          step, label, is written first. The chain reads in execution order. F
          is inferred at each step from the function argument.
      - lines: "45-47"
        note: >
          Used as a value, a generic method must be instantiated first, like a
          generic function. List[int].Map[string] is a method expression: an
          ordinary func whose first parameter is the receiver. nums.Map without
          type arguments, or a target type to infer them from, does not compile.
      - lines: "49-51"
        note: >
          Go 1.27 added N as a method on *rand.Rand, next to the older generic
          function rand.N. Int is inferred from the argument, so a
          time.Duration bound yields a time.Duration. Only the types are printed;
          the values depend on the generator.
  - id: interface-gap
    file: examples/02_interface_gap.go
    title: Generic methods don't satisfy interfaces and are invisible to reflect
    runtime: go
    check: exact
    annotations:
      - lines: "11"
        note: >
          An ordinary method. It satisfies fmt.Stringer the usual way (line 26).
      - lines: "14"
        note: >
          A generic method on a generic type. Called directly, U is inferred
          from strconv.Itoa (line 24).
      - lines: "16-20"
        note: >
          Writing Convert[U any] inside an interface is a compile error ("interface
          method must have no type parameters"), so an interface can only name one
          instance. Even that is not enough: a generic method never satisfies an
          interface method.
      - lines: "29-31"
        note: >
          The commented line fails to compile: "wrong type for method Convert,
          have Convert[U any](func(int) U) Box[U], want Convert(func(int) string)
          Box[string]". The assertion from any compiles and answers false at run
          time.
      - lines: "33-37"
        note: >
          reflect can't instantiate code at run time, so it lists String only.
          Anything that finds methods by reflection, such as text/template's
          {{.Convert}}, can't call a generic method.
  - id: interface-wins
    file: examples/03_interface_wins.go
    title: "Over-genericized code, the interface fix, and where a type parameter pays"
    runtime: go
    check: exact
    annotations:
      - lines: "16-26"
        note: >
          A type switch on any(s) inside a generic function means the behavior
          differs per type, which is what methods are for. The set is closed,
          the compiler can't check the switch is exhaustive (hence the panic),
          and the logic duplicates the Area methods.
      - lines: "28-36"
        note: >
          Total only calls Area, so a plain interface parameter is enough.
          One slice holds both shapes, and a shape added in another package
          works without touching Total.
      - lines: "38-47"
        note: >
          Here the type parameter does work: Largest returns S, the caller's
          own type. It also accepts a []Circle as is, while Total([]Circle{...})
          doesn't compile, because []Circle is not a []Shape.
      - lines: "51"
        note: >
          Circle | Rect is not a type. Unions exist only inside constraints, so
          no variable or slice can hold "a Circle or a Rect". Unlike TypeScript,
          Go has no union types; mixed values need an interface.
      - lines: "54-55"
        note: >
          c is a Circle, so c.R needs no type assertion. With an interface
          result the caller would get a Shape back and have to assert.
use_cases:
  - "Fluent APIs on your own types: query builders, result lists, pipelines that change the element type step by step"
  - "Typed operations that belong to one type's namespace instead of the package's: (*rand.Rand).N, List.Map vs a MapList function"
  - "Adding a stricter type parameter next to a looser receiver: Set[T comparable] can have SortedBy[K cmp.Ordered](key func(T) K)"
  - "Interfaces, still, for run-time polymorphism: mixed collections, plugins, ports in a layered design, test fakes"
strengths:
  - "Operations chain left to right in execution order, where pre-1.27 code nested MapList(MapList(...)) inside out"
  - "No package-level name clashes: List.Map and Tree.Map instead of MapList and MapTree"
  - "Interfaces stay simple and fast: dynamic dispatch never needs code generated at run time, so Go keeps pure ahead-of-time compilation, where C# typically instantiates generic virtual methods in its JIT"
weaknesses:
  - "A generic method never satisfies an interface; Java and C# both allow generic interface methods (<R> R accept(Visitor<R> v))"
  - "Invisible to reflect, so text/template and other reflection-driven code can't call them"
  - "Requires go 1.27 in go.mod, which raises the minimum Go version for every user of a library that adopts them"
  - "As values they must be instantiated: nums.Map alone is not a func value"
pitfalls:
  - title: "Expecting a generic method to implement an interface"
    detail: "A type with Read[B ~[]byte](p B) (int, error) does not implement io.Reader, although Read[[]byte] would match. Methods that must satisfy an interface stay non-generic; put type parameters on the type or on the interface instead."
    bad: "func (r *Src) Read[B ~[]byte](p B) (int, error)"
    good: "func (r *Src) Read(p []byte) (int, error)"
  - title: "A type parameter that only calls methods"
    detail: "If the body only calls methods of T and T never appears in the result, the type parameter adds nothing over an interface parameter."
    bad: "func Describe[T fmt.Stringer](v T) string { return v.String() }"
    good: "func Describe(v fmt.Stringer) string { return v.String() }"
  - title: "Type switches inside generic code"
    detail: "switch any(v).(type) in a generic function means the behavior is per type. Give the types a method and accept an interface; the set stays open and the compiler checks each implementation."
  - title: "Union constraints of your own structs"
    detail: "[T Circle | Rect] looks like a sum type but isn't one: you can't store a Circle | Rect anywhere. If you need a value that is one of several types, use an interface (optionally with an unexported marker method to close the set)."
quiz:
  - q: "Which declaration compiles in a module whose go.mod says go 1.27?"
    options: ["type Mapper interface { Map[F any](func(int) F) []F }", "func (l List[E]) Map[F any](f func(E) F) List[F]", "Both", "Neither"]
    answer: 1
    why: "Methods of concrete types may declare type parameters since Go 1.27; interface methods still may not."
  - q: "type Src struct{} has func (Src) Read[B ~[]byte](p B) (int, error). Does Src implement io.Reader?"
    options: ["Yes, B is inferred as []byte", "Only *Src does", "No, a generic method never satisfies an interface method"]
    answer: 2
    why: "Interface satisfaction compares method signatures, and a generic method's signature has type parameters. Go 1.27 defines it this way so that no code has to be instantiated at run time."
  - q: "func Describe[T fmt.Stringer](v T) string { return v.String() }. What would you write instead?"
    options: ["func Describe(v fmt.Stringer) string", "Nothing; generics are faster", "func Describe[T any](v T) string"]
    answer: 0
    why: "The body only calls a method and T never flows back out, which is exactly what an interface parameter expresses."
  - q: "When does a type parameter beat an interface parameter?"
    options: ["When the function only calls methods", "When the caller's concrete type must come back out, as in Largest[S Shape](xs []S) S", "Whenever performance matters"]
    answer: 1
    why: "With an interface result the caller gets a Shape and must assert; with S they get their own type back."
further_reading:
  - title: "Go 1.27 Release Notes: changes to the language"
    url: "https://go.dev/doc/go1.27"
  - title: "Generic Methods"
    url: "https://go.dev/blog/generic-methods"
  - title: "The Go Programming Language Specification: Method declarations"
    url: "https://go.dev/ref/spec#Method_declarations"
  - title: "When To Use Generics"
    url: "https://go.dev/blog/when-generics"
---

## Concept

Since Go 1.27 a method may declare type parameters of its own, after the method name
and in addition to any its receiver type has:
`func (l List[E]) Map[F any](f func(E) F) List[F]`. Calls infer them like function
calls; used as a value, the method must be instantiated first. Two rules decide where
the feature fits. Interface methods still can't declare type parameters, and a generic
method never satisfies an interface method. reflect can't see generic methods either.
So generic methods organize code around a type and make chains read left to right;
they never take part in run-time polymorphism.

That leaves the older and more frequent question: should this code have a type
parameter at all, or is an interface the right tool?

## Coming from C/C++/Java/Python/JS

- Java: `<R> List<R> map(Function<T, R> f)` has always been allowed, interface methods
  included, because erasure compiles one body for every `R`. Go compiles real
  instantiations, so it would have to know every `R` an interface call could need.
- C#: generic virtual methods exist; the runtime typically instantiates them on
  demand in its JIT. Go has no JIT.
- C++: member function templates can't be virtual, for the same reason as Go's rule.
- TypeScript: generic methods are unrestricted because types are erased. TS also has
  real union types; Go's `Circle | Rect` exists only inside a constraint.
- Python: duck typing makes the question moot; `typing.Protocol` is the closest thing
  to a Go interface.

## How it works

Put a value whose type has `func (Empty) Nop[T any](x T) T` into an `any`, then ask
`x.(interface{ Nop(string) string })`. Saying yes would mean running `Nop[string]`,
code nobody compiled, because the compiler can't see which assertions will happen at
run time. The Go FAQ lists the ways out: recompile at link time, add a JIT, fall back
to a slow boxed version, or rule that generic methods never satisfy interfaces. The
FAQ entry predates Go 1.27 and rejects all four. Go 1.27 took the last one, on the
grounds that methods are worth having for code organization even without interfaces.
That makes a generic method an ordinary compile-time instantiation, the same
machinery generic functions use.

## Design notes

Use a generic method when an operation belongs to one of your types and you would
otherwise name a package function after the type (`MapList`, `MapTree`), or when a
chain reads better than nesting. Keep package functions for operations that apply to
many types, which is why `slices` and `maps` stay as they are. Any method that has to
satisfy an interface (`io.Reader`, `fmt.Stringer`, your own repository port) must be
non-generic; if it needs a type parameter, put it on the type or the interface
(`type Converter[T, U any] interface{ Convert(T) U }`).

Signs of over-genericized code: a type parameter used only to call methods on it; a
type switch on `any(v)` inside a generic function; a union constraint of your own
struct types; a type parameter that is only ever instantiated with one type. The
opposite smell is a helper taking `any` and forcing every caller to assert. The Go
team's rule of thumb holds: write the concrete code first, and add a type parameter
when you are about to write the same code again for another type.
