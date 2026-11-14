---
id: go.interfaces.assertions
title: "Type assertions, type switches, any, and the nil-interface trap"
track: go
module: interfaces
order: 3
level: intermediate
minutes: 14
since: go1.18
prerequisites: [go.interfaces.implicit, go.interfaces.methods]
compare_with: [ts.narrowing.basic, ts.narrowing.guards, go.generics.parameters, design.paradigms.polymorphism]
tags: [interfaces, type-assertion, type-switch, any, nil]
summary: >
  An interface value is a (dynamic type, value) pair. Assertions and type
  switches read the type back out, and an interface holding a nil pointer is
  not a nil interface.
examples:
  - id: assert-switch
    file: examples/01_assert_switch.go
    title: Comma-ok assertions, the panicking form, and type switches
    runtime: go
    check: exact
    annotations:
      - lines: "11"
        note: >
          In each single-type case, x has that case's type. Cases are tried in
          order, so a value that is both an error and a Stringer takes
          whichever case comes first. fallthrough isn't allowed in a type
          switch.
      - lines: "12-13"
        note: >
          case nil matches only an interface with no dynamic type at all. An
          interface holding a nil pointer skips it, which example 3 shows.
      - lines: "14-15"
        note: >
          With two types in one case Go can't give x a single static type, so
          x stays any. Split the case when you need int arithmetic.
      - lines: "16-17"
        note: >
          Here x is a string. len counts bytes, so the Turkish "ağaç" is 6 bytes
          but 4 runes: ğ and ç take two bytes each in UTF-8.
      - lines: "18-21"
        note: >
          Cases can name interfaces, not just concrete types. time.Duration has
          a String method, so 90s lands in the Stringer case and prints 1m30s.
      - lines: "27-34"
        note: >
          The one-result form v.(int) panics on a mismatch. The message names
          both types; any is an alias, so the runtime still spells it
          interface {}. Use this form only where a mismatch would be a bug.
      - lines: "38-41"
        note: >
          The comma-ok form never panics: a mismatch gives the zero value and
          false, the same shape as a map lookup.
  - id: capability
    file: examples/02_capability.go
    title: Asserting to an interface to discover an optional capability
    runtime: go
    check: exact
    annotations:
      - lines: "11-13"
        note: >
          Asserting to an interface type asks whether the dynamic value also
          has these methods. Signatures must match exactly: http.Flusher's
          Flush returns nothing, so a ResponseWriter wouldn't match this one.
      - lines: "23-26"
        note: >
          The standard library downcasts this way itself: io.WriteString checks
          for io.StringWriter, and io.Copy for WriterTo and ReaderFrom. The
          parameter stays the narrow io.Writer; the extra method is optional.
      - lines: "38-43"
        note: >
          Proof that the upgrade ran: the report lines appear after "(... can
          flush)", because bufio held them until Flush.
  - id: nil-interface
    file: examples/03_nil_interface.go
    title: "The trap: a non-nil error holding a nil pointer"
    runtime: go
    check: exact
    annotations:
      - lines: "10-16"
        note: >
          Returning verr converts a *ValidationError to error, and the
          interface records the type even when the pointer is nil. The result
          is not nil, so every caller's if err != nil fires.
      - lines: "19-24"
        note: >
          The fix is at the source: declare the result as error and return the
          literal nil. Functions that return *ValidationError instead recreate
          the trap as soon as a caller stores the result in an error variable.
      - lines: "28-29"
        note: >
          fmt makes it worse: %v prints <nil> because fmt catches the panic of
          Error on a nil receiver, while err == nil is false. %T shows the
          dynamic type. Calling err.Error() yourself would panic.
      - lines: "36-39"
        note: >
          Every interface type behaves this way. Only the assertion x.(*int)
          reveals the nil pointer inside; short of package reflect, there is no
          operator that asks whether the dynamic value is nil.
use_cases:
  - "Walking JSON decoded into any: encoding/json stores float64, string, bool, nil, []any and map[string]any, and a type switch takes them apart"
  - "Optional capabilities: io.WriterTo and io.ReaderFrom in io.Copy, http.Flusher behind an http.ResponseWriter"
  - "Classifying the value recover() returns: runtime.Error, error, or anything else"
  - "Heterogeneous messages or AST nodes where the set of types is open"
strengths:
  - "Checked at run time, unlike a C cast: a failed assertion returns false or panics with both type names, never reinterprets memory"
  - "Type switches give each case a correctly typed variable, much like TypeScript's narrowing, but over real runtime types instead of erased ones"
  - "Asserting to an interface lets a function accept the narrow io.Writer and still use extra methods when the value has them"
weaknesses:
  - "The compiler can't check a type switch for exhaustiveness: interfaces are open, so a new type silently falls to default"
  - "Every assertion is a spot where static typing ends; code full of any and switches gets the costs of dynamic typing without its brevity"
  - "Wrapper types hide capabilities: a logging ResponseWriter wrapper drops http.Flusher unless it forwards it (http.ResponseController, Go 1.20, finds it through the wrapper's Unwrap method)"
pitfalls:
  - title: "Returning a typed nil pointer as an error"
    detail: "The interface holds (type *T, value nil) and compares unequal to nil. Return the literal nil on success and declare results as error, never as *MyError."
    bad: "var e *MyErr; if bad { e = &MyErr{} }; return e"
    good: "if bad { return &MyErr{} }; return nil"
  - title: "Type-asserting a wrapped error"
    detail: "err.(*MyErr) only looks at the outermost error. Once someone wraps it with %w, the assertion fails. Use errors.As or errors.AsType (lesson go.errors.is-as-join)."
  - title: "The one-result assertion on data you don't control"
    detail: "v.(string) panics on a mismatch, and JSON numbers decoded into any are float64, never int. Use the comma-ok form for input."
  - title: "Comparing interfaces that hold slices or maps"
    detail: "== on two interfaces compares dynamic types and values, and panics with \"comparing uncomparable type []int\" when both hold the same dynamic type and that type can't be compared."
quiz:
  - q: "var x any = \"7\"; n, ok := x.(int). What happens?"
    options: ["Panic", "n == 7, ok == true", "n == 0, ok == false", "Compile error"]
    answer: 2
    why: "The comma-ok form never panics; on a mismatch it returns the zero value and false."
  - q: "func f() error { var p *MyErr; return p }. What is f() == nil?"
    options: ["true", "false", "It doesn't compile", "It panics"]
    answer: 1
    why: "The interface holds the dynamic type *MyErr with a nil value, and an interface is nil only when it has no type."
  - q: "In switch x := v.(type) { case int, int64: ... }, what is x's type inside that case?"
    options: ["int", "int64", "The type of v (for example any)", "It depends on the value"]
    answer: 2
    why: "With more than one type in a case, x keeps the type of the switched expression."
  - q: "Which assertion checks whether a writer can also flush?"
    options: ["w.(bufio.Writer)", "w.(interface{ Flush() error })", "w == Flusher", "reflect is required"]
    answer: 1
    why: "Asserting to an interface type tests the dynamic value's method set without naming its concrete type."
further_reading:
  - title: "The Go Programming Language Specification: Type assertions"
    url: "https://go.dev/ref/spec#Type_assertions"
  - title: "The Go Programming Language Specification: Type switches"
    url: "https://go.dev/ref/spec#Type_switches"
  - title: "Go FAQ: Why is my nil error value not equal to nil?"
    url: "https://go.dev/doc/faq#nil_error"
  - title: "Effective Go: Interface conversions and type assertions"
    url: "https://go.dev/doc/effective_go#interface_conversions"
  - title: "The Go Blog: The Laws of Reflection"
    url: "https://go.dev/blog/laws-of-reflection"
---

## Concept

An interface value holds two things: a dynamic type and a value of that type. A type
assertion `x.(T)` reads them back. If `T` is a concrete type, it asks "is the dynamic
type exactly `T`?"; if `T` is an interface, it asks "does the dynamic value also have
these methods?". The one-result form panics on a mismatch; the comma-ok form `v, ok :=
x.(T)` returns the zero value and `false`. A type switch, `switch x := v.(type)`, runs
the same test against several types in order.

`any` is an alias for `interface{}` (Go 1.18): every type satisfies it, so it carries no
static information at all.

An interface is `nil` only when it has no dynamic type. Store a nil pointer in it and the
interface is no longer nil: it has a type, and the value happens to be a nil pointer.
This is the most common way to return an error that looks like `<nil>` but fails
`err != nil`.

## Coming from C/C++/Java/Python/JS

- Java: `if (o instanceof Foo f)` corresponds to `f, ok := o.(*Foo)`, and pattern-matching
  `switch` to a type switch. A failed one-result assertion panics, like a failed cast
  throws `ClassCastException`.
- C/C++: closer to `dynamic_cast` than to `static_cast`: it's checked, and the comma-ok
  form returns `false` instead of a null pointer. There's no unchecked cast outside
  package `unsafe`.
- TypeScript: `typeof`/`instanceof` narrowing is the same idea, but TS types are erased
  and narrowing is the compiler trusting your check. Go assertions test real runtime
  types, so `as`-style lying isn't possible.
- Python: `isinstance` plus duck typing. Asserting to an interface is the static-typed
  version of `hasattr(w, "flush")`.

## How it works

    statement                           interface holds                    err == nil
    var err error                       (type: none,   value: none)        true
    var p *MyErr; var err error = p     (type: *MyErr, value: nil)         false
    var err error = &MyErr{}            (type: *MyErr, value: an address)  false

The middle row exists on purpose. A nil `*Tree` can have working methods (the previous
lesson's `Sum`), so an interface holding one is a usable value, and Go can't treat it as
"nothing".

## Design notes

Reach for static types first: accept the narrowest interface that works, and use type
parameters (Go 1.18) when the caller knows the concrete type. Keep `any` and type
switches for data that really is heterogeneous: JSON decoded into `any`, values from
`recover()`, messages in a protocol. Asserting to an interface for an optional capability
is idiomatic and widely used in the standard library, so there's no shame in it, but
wrappers hide capabilities they don't forward. For errors, use `errors.As` or
`errors.AsType` instead of assertions, because they see through wrapping. And never
return a concrete pointer type where the caller expects an `error`: return the
interface, and return `nil` literally.
