---
id: ts.mental-model.structural
title: "Structural typing vs nominal typing; where TS runs"
track: ts
module: mental-model
order: 3
level: basic
minutes: 12
since: ts3.8
prerequisites: [ts.mental-model.erasable]
compare_with: [go.interfaces.implicit, go.basics.types-conversion, ts.type-level.branded, design.paradigms.side-by-side]
tags: [structural-typing, nominal-typing, compatibility, excess-property-check]
summary: >
  TypeScript compares types by their members, not their names: anything with the
  right shape fits, including values of unrelated classes. That follows from erasure,
  since at run time an object is only a bag of properties.
examples:
  - id: shapes
    file: examples/01_shapes.ts
    title: Shapes decide compatibility, for better and worse
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "8-23"
        note: >
          Pixel never mentions Point, and a plain object literal works too. In Java
          the class would need `implements Point`; here the checker compares
          members, and extra ones (color) don't matter.
      - lines: "25-36"
        note: >
          Meters and Feet are the same type as far as tsc is concerned, so the units
          bug compiles. Go would reject it: `type Meters float64` and `type Feet
          float64` are distinct named types that need an explicit conversion.
      - lines: "38-51"
        note: >
          A #private (or TS `private`) member makes a class behave nominally: no
          object literal or other class can supply Altitude's own #meters. Since only
          the constructor can produce one, instanceof Altitude is also a reliable
          runtime check.
  - id: extra-properties
    file: examples/02_extra_properties.ts
    title: Extra properties pass the check and leak at run time
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "7-9"
        note: >
          Excess-property checking applies only to a fresh object literal written
          directly against a type. There, an unknown key is almost certainly a typo,
          and nothing could ever read it through that type.
      - lines: "17-22"
        note: >
          Through a variable, DbUser is a subtype of PublicUser (it has at least the
          required fields), so the call compiles. The annotation narrows what the
          code may read, not what the object contains, and JSON.stringify sends the
          hash.
      - lines: "24-28"
        note: >
          Explicit mapping is the fix, as with a DTO in Java or a `json:"-"` tag in
          Go. In TS it is the only fix: no type annotation removes a property.
  - id: function-shapes
    file: examples/03_function_shapes.ts
    title: Function types are structural too
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-6"
        note: >
          map passes (value, index, array); parseInt's parameters (string, radix?)
          line up with the first two, so the types fit and the index becomes the
          radix: parseInt("10", 1) is NaN, parseInt("10", 2) is 2.
      - lines: "12-19"
        note: >
          This is why the rule exists: a callback may ignore trailing arguments, as
          JavaScript callbacks always have. Java's functional interfaces need an
          exact arity; TypeScript accepts fewer parameters.
      - lines: "21-24"
        note: >
          The reverse is unsafe and rejected: a function that needs a third
          argument would receive undefined from a caller that passes two.
use_cases:
  - "Passing a class instance, an object literal or a mock to the same function without interfaces or adapters"
  - "Test doubles: any object with the right methods is a valid fake, with no mocking framework or `implements` needed"
  - "Typing plain JSON data, which has no class to be nominal about"
strengths:
  - "Unlike Java or C#, no `implements` ceremony: the consumer declares the shape it needs and every producer that has it fits, like Go's implicit interfaces"
  - "Matches how JavaScript actually works (duck typing), so existing JS patterns type-check without wrappers"
  - "Object literals, class instances and JSON-derived objects are interchangeable when they have the same fields"
weaknesses:
  - "Same shape, different meaning compiles: Meters vs Feet, UserId vs OrderId (type aliases of string are fully interchangeable). Go's named types and Java's classes catch this; TS needs branded types"
  - "A wider object passes for a narrower type and keeps its extra fields at run time, so types never sanitize data"
  - "A value typed as a class need not be an instance of it: `const p: Pixel = { x: 1, y: 2 }` compiles and `p instanceof Pixel` is false"
pitfalls:
  - title: "Expecting a type annotation to strip fields"
    detail: "Returning a DbUser from a function typed PublicUser still sends every field. Map to a new object explicitly."
    bad: "const body: PublicUser = row; res.send(JSON.stringify(body));"
    good: "const body: PublicUser = { name: row.name };"
  - title: "Using instanceof on a value that was only typed as a class"
    detail: "If a class has only public members, a plain object literal satisfies its type but fails instanceof. Narrow with a discriminant field or construct real instances."
  - title: "ID aliases that mix silently"
    detail: "`type UserId = string` is a name, not a new type. Passing an OrderId where a UserId is expected compiles; brand them (lesson ts.type-level.branded) when mixing them would be a real bug."
quiz:
  - q: "`interface Named { name: string }` and `class Dog { name = \"Karabaş\"; bark() {} }`. Does `const n: Named = new Dog()` compile?"
    options: ["No: Dog does not implement Named", "Yes: Dog has a string name", "Only with a cast"]
    answer: 1
    why: "Compatibility is structural: Dog has every member Named requires, and extra members are allowed."
  - q: "`const u: PublicUser = { name, passwordHash }` fails, but `const u: PublicUser = row` with the same two fields compiles. Why?"
    options:
      - "Excess-property checks apply only to fresh object literals"
      - "Variables are never type-checked"
      - "row is implicitly any"
    answer: 0
    why: "A literal's extra key is unreachable through the target type and probably a typo; a variable may be used elsewhere with its full type, so only the shape is compared."
  - q: "What makes an otherwise structural class type behave nominally?"
    options: ["Declaring it with `class`", "A private or #private member", "Marking its fields readonly"]
    answer: 1
    why: "Private members are only compatible with themselves, so nothing but the class (or a subclass) can supply them."
further_reading:
  - title: "TypeScript Handbook: Type Compatibility"
    url: "https://www.typescriptlang.org/docs/handbook/type-compatibility.html"
  - title: "TypeScript for Java/C# Programmers (structural type system)"
    url: "https://www.typescriptlang.org/docs/handbook/typescript-in-5-minutes-oop.html"
  - title: "TypeScript Handbook: Object Types (excess property checks)"
    url: "https://www.typescriptlang.org/docs/handbook/2/objects.html"
  - title: "MDN: Private properties"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Classes/Private_properties"
---

## Concept

In a **nominal** type system (Java, C#, C++) two types are compatible when their
declarations say so: same class, or a declared subclass or `implements`. In TypeScript's
**structural** system, a type is compatible with another when it has at least the
required members with compatible types. Names are only labels for shapes.

This is a direct consequence of erasure. At run time an interface does not exist and a
JavaScript object is a bag of properties, so the only thing a checker can promise about
a value is its shape. TypeScript chose to model JavaScript as it is, duck typing
included, and to check it statically.

**Where TS runs**: nowhere, strictly speaking. Browsers run the JavaScript that Vite
(esbuild) or this runner produces. Node runs `.ts` files itself by stripping types (on by
default since Node 23.6 and 22.18, stable since 24.12 and 25.2), and Deno and Bun accept
`.ts` directly by transpiling first. In every case the type checker ran earlier, in your
editor or CI, and the runtime sees the shapes only.

## Coming from C/C++/Java/Python/JS

- Java/C#: no `implements` needed, and two classes with the same public members are
  interchangeable. Private members bring nominal behaviour back for classes.
- C++: templates are the closest analogue, checked duck typing at compile time, but in
  TS it applies to every type, not only to template parameters.
- Go: interfaces are structural, exactly as here. Named types are not: `type Meters
  float64` cannot be passed as `Feet` without a conversion. TypeScript has no equivalent
  of Go's named types; a type alias is only another name for the same type.
- Python: `typing.Protocol` is structural typing for type checkers; TS works that way
  for every interface and object type.
- JS: you have always written duck-typed code. TS now checks that the duck has the
  members you use.

## Design notes

Lean on structure: declare the small shape a function needs next to the function (the
consumer-declared interface idea from Go), and accept any producer that has it. That
makes tests trivial, since a literal object is a valid fake. Add nominal guarantees
deliberately where mixing values is a real bug: `#private` members for classes and
branded types (lesson ts.type-level.branded) for IDs and units. And never use types
to control what leaves the process: what goes into a response or a log is decided by
code that copies fields, because a narrower type over a wider object still carries
every property.
