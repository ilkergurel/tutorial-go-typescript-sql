---
id: ts.objects-classes.classes
title: "Classes: #private vs private, readonly, abstract, parameter properties"
track: ts
module: objects-classes
order: 2
level: intermediate
minutes: 14
since: es2022
prerequisites: [ts.mental-model.erasable, ts.mental-model.structural]
compare_with: [go.interfaces.implicit, go.composite.structs, design.paradigms.polymorphism]
tags: [classes, private-fields, readonly, abstract, override, parameter-properties, useDefineForClassFields]
summary: >
  `#private` is enforced by the JavaScript engine; `private`, `protected`, `readonly` and
  `abstract` are compile-time checks that vanish. Parameter properties generate code and
  interact badly with field initializers, so erasable code declares fields explicitly.
examples:
  - id: private-fields
    file: examples/01_private_fields.ts
    title: "private vs #private, and when a class is nominal"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-10"
        note: >
          Two kinds of private. `private pin` is a TypeScript modifier: tsc checks
          it, the transpiler deletes it, and pin is an ordinary property. `#balance`
          is a JavaScript private field (ES2022): not a property at all, so no
          bracket access, JSON, Object.keys or Proxy can reach it. For targets
          below ES2022, tsc emulates it with a WeakMap.
      - lines: "14-17"
        note: >
          `#balance in x` (ES2022, supported since TS 4.5) is true only for
          objects this class's constructor initialized, so unlike instanceof it
          can't be fooled by Object.create(Account.prototype) or a copied
          prototype. tsc narrows x to Account after the check.
      - lines: "20-27"
        note: >
          The private modifier rejects acc.pin, but the handbook deliberately
          allows acc["pin"] (an escape hatch for tests), and at run time pin
          shows up in JSON.stringify, Object.keys and every log line. #balance
          shows up nowhere; only the getter exposes it.
      - lines: "29-33"
        note: >
          A private or #private member makes a class nominal: no object literal
          can provide it, so the look-alike is rejected. With only `private pin`
          the message would be "Property 'pin' is private in type 'Account' but
          not in type ...". Two different classes with the same private member
          are not interchangeable either.
      - lines: "35-45"
        note: >
          Without private members a class is just a shape, compared structurally
          like any type. tsc accepts the literal as a Point, but instanceof says
          false. A function that takes a Point and branches on instanceof will
          misjudge well-typed callers.
  - id: parameter-properties
    file: examples/02_parameter_properties.ts
    title: Parameter properties and initialization order
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-10"
        note: >
          `constructor(private readonly id: string)` declares a field and
          assigns it, so the transpiler must write `this.id = id` that isn't in
          your source. That is why erasableSyntaxOnly (TS 5.8) reports TS1294 and
          Node's type stripping refuses the file. tsc also reports TS2729 on the
          label initializer; @ts-expect-error silences it so the program runs.
      - lines: "11-13"
        note: >
          Fields are initialized in declaration order before the constructor
          body runs, and the parameter property is assigned in the body, so label
          reads undefined. This is define semantics (useDefineForClassFields), the default for
          target ES2022 and later. With an older target tsc moved initializers
          after the parameter assignments and this printed "Fatura 2026-001":
          the same source changes behaviour when the target is raised.
      - lines: "15-28"
        note: >
          The erasable spelling: a #private field for the state, a readonly
          field assigned in the constructor, and a getter that exposes id without
          a setter. Every line that runs is in the source, in the order it runs.
      - lines: "30-35"
        note: >
          readonly fields may be assigned in their declaration or in the
          constructor, nowhere else. A get accessor without a set is read-only
          as well; both writes are TS2540.
      - lines: "37-39"
        note: >
          readonly is erased. Object.assign's signature does not check it, and
          at run time label is a plain writable property, so the write succeeds.
          The #id field can't be reached this way; Object.freeze(this) at the
          end of the constructor would block the label write too.
  - id: abstract
    file: examples/03_abstract.ts
    title: Abstract classes vs an interface and a function
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-10"
        note: >
          Template method: export() is the fixed algorithm, header() and row()
          are the steps that vary. Abstract members have no body and must be
          implemented by every concrete subclass. protected, like private, is
          checked by tsc only.
      - lines: "12-19"
        note: >
          override is optional when implementing an abstract member. Under
          noImplicitOverride (TS 4.3) it is required when replacing a concrete
          method (TS4114), so writing it everywhere keeps the rule simple.
      - lines: "20-23"
        note: >
          tsc refuses to instantiate the abstract class and lists every missing
          member of an incomplete subclass. The abstract keyword is erased: at
          run time Exporter is an ordinary class, and plain JavaScript could
          call new on it.
      - lines: "31-43"
        note: >
          The same variation point as data: an interface for the shape and a
          function for the algorithm. Any object literal fits, there is no this
          to lose and no base class to read. Most TypeScript code takes this
          route, and tests just pass a different object.
      - lines: "45-47"
        note: >
          The real value of override: if the base method is renamed, every
          subclass that still says override fails to compile instead of silently
          becoming a new, never-called method.
      - lines: "49-52"
        note: >
          Classes exist at run time, interfaces don't. When code must ask "what
          is this value?" at run time (Error subclasses, values from another
          module), a class gives instanceof; an interface needs a discriminant
          property or a type guard.
use_cases:
  - "Many instances that share behaviour and guard an invariant: Money, a date range, a tokenizer with internal state, a rate limiter, a connection pool"
  - "Runtime identity: Error subclasses checked with instanceof"
  - "Platform contracts that require a class: custom elements (extends HTMLElement), React error boundaries"
  - "A fixed algorithm with a few varying steps that need shared protected state (template method)"
strengths:
  - "#private is enforced by the engine: no reflection, bracket access or JSON can reach it, unlike TS private, Python's _name convention or Java private fields opened by reflection"
  - "Methods live once on the prototype, and instanceof works, which plain object types can't offer"
  - "readonly, abstract, override and protected are checked at compile time and cost nothing at run time"
  - "A private member makes a class type nominal exactly where you want it: no object literal can pass as an Account"
weaknesses:
  - "private, protected, readonly and abstract exist only for tsc; JSON, Object.keys, bracket access and plain JavaScript callers ignore them"
  - "Parameter properties and field initializers interact through initialization order, and the outcome depends on target and useDefineForClassFields"
  - "Methods passed as callbacks lose this (lesson ts.functions.higher-order); objects of functions don't have that problem"
  - "Inheritance couples subclasses to base-class internals, and instances don't survive JSON, structuredClone or postMessage as instances"
pitfalls:
  - title: "Treating private as secret"
    detail: "A private field is an ordinary property at run time: it is logged, serialised and reachable with brackets. Use #private for anything that must not leak or be tampered with."
    bad: "class Session { private token = \"\"; }"
    good: "class Session { #token = \"\"; }"
  - title: "Field initializers that read parameter properties"
    detail: "Under define semantics (target ES2022+) fields are initialized before the constructor body assigns parameter properties, so the initializer sees undefined. tsc reports TS2729; the erasable style avoids the trap entirely."
    bad: "class Invoice { label = `Fatura ${this.id}`; constructor(private id: string) {} }"
    good: "class Invoice { readonly label: string; constructor(id: string) { this.label = `Fatura ${id}`; } }"
  - title: "Redeclaring an inherited field to narrow its type"
    detail: "With define semantics a field declared in a subclass is re-initialized after super() returns, overwriting what the base constructor stored. tsc reports TS2612; `declare` narrows the type without emitting a field."
    bad: "class DogHouse extends AnimalHouse { resident!: Dog; }"
    good: "class DogHouse extends AnimalHouse { declare resident: Dog; }"
  - title: "instanceof on a structurally typed class"
    detail: "A class without private members accepts any object of the same shape, so instanceof can be false for a well-typed value. Discriminate with a kind property, or give the class a #private brand."
quiz:
  - q: "Which field is missing from JSON.stringify(obj) and Object.keys(obj)?"
    options: ["private pin", "#pin", "Both", "Neither"]
    answer: 1
    why: "A #private field is not a property, so nothing that enumerates properties sees it. private is erased and leaves an ordinary property."
  - q: "class A { label = `x${this.id}`; constructor(private id: string) {} } with target ES2022. What is new A(\"1\").label?"
    options: ["\"x1\"", "\"xundefined\", and tsc reports TS2729", "It throws a ReferenceError"]
    answer: 1
    why: "Field initializers run before the constructor body, where the parameter property is assigned."
  - q: "Why does erasableSyntaxOnly reject constructor(private id: string)?"
    options: ["private is not allowed on constructor parameters", "It generates a field and an assignment, so deleting the types is not enough", "Parameter properties are deprecated in TypeScript 6"]
    answer: 1
    why: "Erasable syntax can be removed without changing behaviour; a parameter property has runtime effects the transpiler must generate."
  - q: "class Point { x = 0; y = 0 } — does const p: Point = { x: 1, y: 2 } compile, and is p instanceof Point?"
    options: ["Compiles; instanceof is true", "Compiles; instanceof is false", "Does not compile"]
    answer: 1
    why: "Classes are compared structurally unless they have private members, but instanceof checks the prototype chain at run time."
further_reading:
  - title: "TypeScript Handbook: Classes"
    url: "https://www.typescriptlang.org/docs/handbook/2/classes.html"
  - title: "MDN: Private elements (#)"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Classes/Private_elements"
  - title: "MDN: Public class fields"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Classes/Public_class_fields"
  - title: "TSConfig reference: useDefineForClassFields"
    url: "https://www.typescriptlang.org/tsconfig/#useDefineForClassFields"
  - title: "TSConfig reference: noImplicitOverride"
    url: "https://www.typescriptlang.org/tsconfig/#noImplicitOverride"
  - title: "react.dev: Component (error boundaries still need a class)"
    url: "https://react.dev/reference/react/Component"
---

## Concept

A TypeScript class is a JavaScript class plus annotations, and the annotations split into
two groups that behave very differently:

- **JavaScript features** survive into the running code: fields, methods, accessors,
  `static`, `extends`, and **`#private`** fields and methods (ES2022). A `#name` is a
  private slot the engine enforces; outside the class body it is a syntax error, and no
  reflection reaches it.
- **TypeScript modifiers** are checked by tsc and then erased: `private`, `protected`,
  `public`, `readonly`, `abstract`, `override`, `implements`. At run time a `private` field
  is a normal property, a `readonly` one is writable and an abstract class can be
  instantiated.

**Parameter properties** (`constructor(private readonly id: string)`) are the odd one
out: TypeScript-only syntax that generates a field and an assignment. They are not
erasable, which is why `erasableSyntaxOnly` and Node's type stripping reject them, and
they run at a surprising point in the initialization order.

That order, with define semantics (the default for target ES2022 and later): base class
fields, base constructor, then the derived class's fields in declaration order, then the
derived constructor body. A field initializer therefore can't see anything the constructor
body assigns.

## Coming from C/C++/Java/Python/JS

- Java: `private` in TS is closer to a lint rule than to Java's access control, and even
  Java's can be opened by reflection. `#private` is stronger than both. Classes are
  structural unless they have private members, so `implements` is optional: any object
  with the right shape satisfies an interface.
- C++: no multiple inheritance, no destructors (memory is garbage collected), and every
  method is dispatched dynamically, like a virtual function. An abstract method is a pure
  virtual function; `override` is the same keyword with the same purpose.
- Python: `_name` is a convention like TS `private`; `__name` mangling is still reachable.
  `#name` is real privacy. Python's dataclass fields are initialized in `__init__`; JS
  fields are initialized before the constructor body.
- Go: no classes or inheritance. A struct with methods plus an implicitly satisfied
  interface is what most TS code achieves with plain objects and interfaces. Go's
  unexported fields are package-private and enforced by the compiler, roughly between TS
  `private` and `#private`.
- JS/React: classes are the same as in JavaScript. React recommends function components;
  error boundaries are the remaining case that needs a class component.

## Design notes

Reach for a class when you have many instances sharing behaviour and an invariant to guard
(a `Money` that never holds a fractional kuruş, a tokenizer's position), when you need
runtime identity (`instanceof` for error types), or when a platform demands one (custom
elements, error boundaries). For everything else, plain data plus functions in a module
is simpler: module scope already hides what isn't exported, closures give privacy, and
there is no `this` to lose.

When you do write a class, write it erasably: declare fields explicitly, assign them in
the constructor, use `#private` for state that must stay inside and a getter to expose it
read-only. Turn on `erasableSyntaxOnly` and `noImplicitOverride`. Prefer composition
(pass a `Format` object) to an abstract base class unless the varying steps genuinely
need shared protected state; an interface plus a function is easier to test and has no
initialization order to get wrong.
