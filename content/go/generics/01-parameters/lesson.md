---
id: go.generics.parameters
title: "Type parameters, constraints, ~, comparable, cmp.Ordered"
track: go
module: generics
order: 1
level: intermediate
minutes: 14
since: go1.21
prerequisites: [go.interfaces.methods, go.composite.slices-and-maps]
compare_with: [ts.generics.functions, design.paradigms.generics-absence]
tags: [generics, type-parameters, constraints, comparable, cmp]
summary: >
  A type parameter list says which types a function or type accepts; the
  constraint says what the body may do with them. Inference fills in the type
  arguments at most call sites, so Set[T] and Stack[T] read like ordinary code.
examples:
  - id: constraints
    file: examples/01_constraints.go
    title: Constraints, ~ for defined types, cmp.Ordered, and inference
    runtime: go
    check: exact
    annotations:
      - lines: "8-12"
        note: >
          A constraint is an interface, here one that lists types instead of
          methods. The ~ matters: int64 alone means exactly int64, so Kurus
          (line 15) would be rejected. ~int64 admits every type whose underlying
          type is int64. Interfaces like this exist only as constraints; var n
          Number does not compile.
      - lines: "19-25"
        note: >
          The body is checked once, against the constraint, not per call as with
          C++ templates. += compiles because every type in Number supports it.
          var total T is how you spell "the zero value of T".
      - lines: "28-30"
        note: >
          cmp.Ordered (Go 1.21) is the set of types that support < : integers,
          floats and strings, each with ~. The built-in min and max (also Go
          1.21) accept such a T directly.
      - lines: "33-39"
        note: >
          To appears only in the result, so nothing can infer it. Explicit type
          arguments fill the list from the left, which is why To comes first:
          the caller names To and inference supplies From. To(x) is legal
          because every type in Number converts to every other.
      - lines: "42-45"
        note: >
          No type arguments at these call sites. Untyped constants default to
          int and float64; the []Kurus argument makes T Kurus, so the sum is a
          Kurus and prints through its String method. Passing strings fails at
          compile time, at the call.
      - lines: "47-49"
        note: >
          One Clamp serves ints, Kurus and strings. Strings order by their UTF-8
          bytes, not by the Turkish alphabet: ç is 0xC3 0xA7 and sorts after z.
          The standard library has no locale collation.
      - lines: "51-52"
        note: >
          The result is a real []float64, not a []any. Go generics are not
          erased, which is why %T can print the instantiated type.
  - id: set-stack
    file: examples/02_set_stack.go
    title: Set[T comparable] and Stack[T any]
    runtime: go
    check: exact
    annotations:
      - lines: "9-10"
        note: >
          comparable is the constraint for == and !=, exactly what map keys
          need. struct{} values take no memory, so the map is a set.
      - lines: "12-18"
        note: >
          A constructor function exists mostly for inference: callers write
          SetOf("a", "b") and get a Set[string] without naming the type.
      - lines: "20"
        note: >
          A method on a generic type repeats the type parameter in the receiver,
          Set[T], but never the constraint: that comes from the type declaration.
      - lines: "22-30"
        note: >
          Sorting needs <, which comparable does not promise, and a receiver
          cannot tighten T's constraint. A function with a stricter constraint
          can still take a Set[T]. Sorting also makes the random map order
          deterministic.
      - lines: "37-45"
        note: >
          An empty stack has nothing to return; var zero T gives the zero value
          for any T. Where Java would return null, Pop returns that zero value
          plus false: the comma-ok idiom of map lookups.
      - lines: "48-50"
        note: >
          For strings, == compares bytes: "izmir" is not "İzmir". The
          sorted output puts Iğdır before İzmir because I is the byte 0x49 and İ
          starts with 0xC4.
      - lines: "52"
        note: >
          Inference applies to generic functions, never to generic types, so a
          Stack needs its type argument spelled out. var st Stack[rune] is also
          ready to use: the zero value of the struct is an empty stack.
      - lines: "53-60"
        note: >
          Ranging over a string yields runes, so ş and ı are pushed whole and
          the word reverses correctly. The final Pop returns the zero rune and
          false.
  - id: comparable-any
    file: examples/03_comparable_any.go
    title: comparable accepts interface types, and the check moves to run time
    runtime: go
    check: exact
    annotations:
      - lines: "10-13"
        note: >
          Since Go 1.20, interface types such as any satisfy comparable, because
          == is defined on interfaces. A go.mod declaring go 1.19 rejects
          Set[any] at compile time.
      - lines: "18-19"
        note: >
          A map insert hashes the key's dynamic value, and a []int can be
          neither hashed nor compared, so the insert panics. The compile-time guarantee only
          holds when the type argument is a concrete comparable type.
use_cases:
  - "Containers whose code is the same for every element type: Set[T], Stack[T], an LRU cache, a ring buffer"
  - "Algorithms over slices and maps: Clamp, Sum, a Convert or GroupBy helper, the slices and maps packages themselves"
  - "Replacing any plus type assertions in helpers where the caller's type should flow through to the result"
strengths:
  - "The body is type-checked once against its constraint, so a mistake is reported in the generic code, not as a wall of template errors at a call site as in C++"
  - "Unlike Java's erased generics, a Set[int] stores plain ints with no boxing, and T exists at run time: var zero T, T(x) and %T all work"
  - "Inference keeps most call sites free of type arguments: SetOf(\"a\", \"b\"), Sum(prices...)"
  - "Constraints are ordinary interfaces, so there is one concept to learn rather than Java's bounded wildcards or C++'s SFINAE"
weaknesses:
  - "Operators can't be overloaded, so a constraint like Number can only list built-in types; a user-defined Decimal struct can never be added with +"
  - "Generic types are never inferred: there is no Java diamond (new Stack<>()), you write Stack[rune]"
  - "A method can't demand more of T than the type declared, so stricter operations (Sorted) end up as package-level functions"
  - "cmp.Ordered compares strings by bytes; there is no locale-aware ordering for Turkish in the standard library"
pitfalls:
  - title: "Forgetting ~ in a constraint"
    detail: "int64 | float64 means exactly those two types. A defined type such as type Kurus int64 is rejected; the compiler even hints \"possibly missing ~ for int64\"."
    bad: "type Number interface { int | int64 | float64 }"
    good: "type Number interface { ~int | ~int64 | ~float64 }"
  - title: "Trusting comparable with interface type arguments"
    detail: "Set[any] compiles, then panics with \"hash of unhashable type\" the first time someone adds a slice, map or func. Prefer a concrete key type, or derive a comparable key (an ID, a string) from the value."
  - title: "Case-insensitive Turkish keys"
    detail: "Keys compare bytes, so normalize before inserting. strings.ToLower is not Turkish-aware: it maps I to i, turning \"IĞDIR\" into \"iğdir\". Use the Turkish case mapping."
    bad: "set.Add(strings.ToLower(city))"
    good: "set.Add(strings.ToLowerSpecial(unicode.TurkishCase, city))"
  - title: "Using a constraint as a variable type"
    detail: "An interface with type terms (| or ~) is a constraint only: var n Number fails with \"cannot use type Number outside a type constraint\". Use the type parameter T, or a method-only interface if you need a variable."
quiz:
  - q: "type Celsius float64. Which constraint accepts Celsius as a type argument?"
    options: ["interface{ float64 }", "interface{ ~float64 }", "Both", "Neither"]
    answer: 1
    why: "float64 alone means exactly float64; ~float64 admits every type whose underlying type is float64."
  - q: "s := Set[any]{}; s.Add([]int{1}). What happens with Set[T comparable] map[T]struct{}?"
    options: ["A run-time panic", "Compile error: []int is not comparable", "The slice is stored"]
    answer: 0
    why: "The type argument is any, which satisfies comparable since Go 1.20. The slice is the dynamic value, hashed at run time, and slices can't be hashed."
  - q: "Why does func Convert[To, From Number](xs []From) []To list To first?"
    options: ["Pure style", "Result types must come first", "To can't be inferred from the arguments, and explicit type arguments fill the list from the left"]
    answer: 2
    why: "Convert[float64](ints) names To; From is inferred from ints. With From first, the caller would have to name both."
  - q: "Why is Sorted a function instead of a method on Set[T comparable]?"
    options: ["Methods can't have type parameters", "A method can't require more of T (cmp.Ordered) than the type declaration (comparable)", "Functions are faster than methods"]
    answer: 1
    why: "The receiver's type parameters take their constraints from the type. Go 1.27 generic methods can add new type parameters, but they can't tighten T."
further_reading:
  - title: "Tutorial: Getting started with generics"
    url: "https://go.dev/doc/tutorial/generics"
  - title: "An Introduction To Generics"
    url: "https://go.dev/blog/intro-generics"
  - title: "The Go Programming Language Specification: Type parameter declarations"
    url: "https://go.dev/ref/spec#Type_parameter_declarations"
  - title: "package cmp"
    url: "https://pkg.go.dev/cmp"
---

## Concept

A type parameter list in square brackets makes a function or type generic:
`func Sum[T Number](xs ...T) T`. Each parameter has a constraint, which is an
interface. The constraint does two jobs: it limits the type arguments callers may
pass, and it limits what the body may do with values of type `T`. Three constraints
cover most code. `any` allows every type but only assignment and passing around.
`comparable` allows `==` and `!=`, which is what map keys need. `cmp.Ordered` (Go
1.21) allows `<` and friends on integers, floats and strings. When you need your own,
list types with `|`, and write `~T` to include every type whose underlying type is `T`.

At a call site, inference works out the type arguments from the ordinary arguments,
so `SetOf("a", "b")` is a `Set[string]`. You write type arguments yourself only when
there is nothing to infer from: for generic types (`Stack[rune]`) and for type
parameters that appear only in the result (`Convert[float64](ints)`).

## Coming from C/C++/Java/Python/JS

- C++: closer to C++20 concepts than to templates. The body is checked once against
  the constraint; there is no specialization, no SFINAE and no non-type parameters
  (`Array<T, N>` has no Go equivalent).
- Java: no erasure and no boxing. `Set[int]` stores ints, `var zero T` works, and
  there are no wildcards: Go has no subtyping between `[]Circle` and `[]Shape` for
  variance rules to describe.
- TypeScript: `[T any]` instead of `<T>`, and `[T Number]` instead of
  `<T extends Number>`. TS generics are erased after type checking; Go's exist in the
  compiled program (`%T` prints `[]float64`). Go has no conditional or mapped types:
  a constraint describes a set of types and nothing more.
- Python: like a `TypeVar` with a bound, except the compiler enforces it and the
  program won't build if a call breaks it.

## Design notes

Start with concrete code. Reach for a type parameter when you are about to copy a
function for a second element type, or when a helper takes `any` and every caller
follows it with a type assertion. Pick the weakest constraint the body needs: `any`,
then `comparable`, then `cmp.Ordered` or your own union. Put type parameters that
can't be inferred first, so callers name one type and let inference do the rest.
Before writing your own container, check the `slices` and `maps` packages: most
"generic utility" code in older projects is now one standard library call. The
standard library still has no set type, so a small `Set[T comparable]` like the one
here is reasonable. The next lesson covers methods with their own type parameters
(new in Go 1.27) and how to recognize code where an interface is the better tool.
