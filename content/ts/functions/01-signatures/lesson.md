---
id: ts.functions.signatures
title: "Parameter and return types; optional, default and rest parameters"
track: ts
module: functions
order: 1
level: basic
minutes: 12
since: ts3.4
prerequisites: [ts.everyday-types.primitives, ts.everyday-types.null]
compare_with: [go.basics.functions]
tags: [functions, parameters, inference, optional-parameters, default-parameters, rest-parameters]
summary: >
  Parameters are the one place TypeScript cannot infer, so you annotate them;
  return types are inferred, and you pin them where they are a contract.
  ?, = and ... decide how a function may be called, and an options object
  takes over once optional arguments pile up.
examples:
  - id: annotate
    file: examples/01_annotate.ts
    title: Annotate parameters; pin return types where they are a contract
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-6"
        note: >
          A parameter's type is an input to inference, not a result of it. The
          compiler won't guess from how the body uses it, so under strict
          (noImplicitAny) a missing annotation is an error. Without strict it
          silently becomes any and nothing that touches it is checked.
      - lines: "8-11"
        note: >
          The return type is inferred as string. For small internal functions,
          writing it out adds nothing and goes stale when the body changes.
      - lines: "13-15"
        note: >
          Plain JS lets you pass fewer arguments (the rest become undefined) or
          more (they are ignored). TypeScript checks the count against the
          signature at every call site.
      - lines: "17-25"
        note: >
          With ": number | undefined" the bad return is an error inside the
          function. Without the annotation, return null would compile and widen
          the inferred type to number | null | undefined, and the errors would
          appear in callers instead. The Turkish decimal comma is a real trap
          here: Number("12,5") is NaN and parseFloat("12,5") is 12.
      - lines: "27-34"
        note: >
          To infer digitSum's return type the compiler would need digitSum's
          return type. It reports TS7023 instead of guessing; one annotation
          breaks the cycle.
      - lines: "36"
        note: >
          toLocaleUpperCase("tr-TR") maps i to İ and ı to I. toUpperCase() uses
          the locale-independent Unicode mapping and turns both into I.
  - id: optional-default
    file: examples/02_optional_default.ts
    title: Optional and default parameters
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-8"
        note: >
          title?: string makes the argument omittable, so inside the function
          its type is string | undefined and you must narrow it before calling
          string methods. One function covers both call shapes where Java would
          need two overloads.
      - lines: "10-14"
        note: >
          With a default, callers see locale?: string, but inside the function
          locale is a plain string, inferred from "tr-TR". No narrowing needed.
      - lines: "16-21"
        note: >
          JS evaluates the default expression on every call that needs it, so
          each call gets its own array. Python evaluates def f(x=[]) once and
          shares that list across calls; that bug can't happen here.
      - lines: "25-33"
        note: >
          Only undefined (passed explicitly or by leaving the argument out)
          selects the default. null is a real value: JS would hand it to
          toLocaleLowerCase, which throws a TypeError. Here tsc rejects the call
          before that can happen.
      - lines: "35-37"
        note: >
          The same input lower-cased with two locales has different lengths.
          The locale is part of a text function's meaning, which is why it is
          a parameter with an explicit default instead of the machine's locale.
  - id: rest-options
    file: examples/03_rest_options.ts
    title: Rest parameters, spread arguments and an options object
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-7"
        note: >
          ...amounts collects all remaining arguments into a number[]. Called
          with no arguments it is an empty array, never undefined. Same idea as
          Java's varargs and Go's ...T.
      - lines: "16-20"
        note: >
          number[] has no known length, so tsc can't prove that two numbers
          arrive for min and max. as const turns the literal into the tuple
          readonly [0, 100], whose length is part of its type.
      - lines: "22-25"
        note: >
          The naive version. To change only the third setting, the caller must
          pass undefined twice, and the call site no longer says what 1 means.
      - lines: "27-37"
        note: >
          Destructuring in the parameter list with a default per field gives
          named, order-free arguments. The trailing = {} makes the whole object
          optional. Adding a fourth option later breaks no caller.
      - lines: "39-40"
        note: >
          Both functions produce the same delays. Only the call sites differ,
          and only one of them can be read without opening the definition.
use_cases:
  - "Annotate every parameter of a function declaration; annotate return types on exported functions, recursive functions, and results other modules rely on"
  - "Optional parameter for one trailing argument most callers leave out (a title, a separator)"
  - "Default parameter for a configuration-like argument with a sensible value (locale = \"tr-TR\", pageSize = 50)"
  - "Rest parameters for variadic helpers: total(...amounts), log(level, ...parts)"
  - "An options object once a function has two or more optional settings: retry policies, formatters, fetch wrappers"
strengths:
  - "Unlike Java, one function covers calls with and without an argument; optional and default parameters replace chains of telescoping overloads"
  - "Unlike plain JS, a wrong argument count or type fails at compile time (TS2554, TS2345) instead of arriving as undefined at runtime"
  - "Unlike Python, a default expression is evaluated on each call, so a [] or {} default is never shared between calls"
  - "Options objects give named arguments with per-field defaults, which Go and Java only get through option structs or builders"
weaknesses:
  - "Checks are compile-time only: plain JS callers, or data cast with as, can still pass anything, so validate input at the boundaries"
  - "Defaults fire only for undefined; a null from a JSON API is not replaced"
  - "Positional optionals scale badly: f(a, undefined, undefined, true) compiles and is unreadable"
  - "An inferred return type changes silently when the body changes; without an annotation the error shows up at distant call sites"
pitfalls:
  - title: "Expecting a default to replace null"
    detail: "Defaults apply only when the argument is undefined. JSON APIs send null for absent values. Convert at the boundary with ?? undefined; tsc rejects passing string | null to a defaulted string parameter, which shows you where."
    bad: "lower(text, settings.locale) // locale: string | null -> TS2345"
    good: "lower(text, settings.locale ?? undefined)"
  - title: "Case-mapping Turkish text without a locale"
    detail: "toLowerCase()/toUpperCase() use the locale-independent Unicode mapping: I becomes i, and İ becomes i plus a combining dot. toLocaleLowerCase() without an argument uses the runtime's default locale, so results differ between machines. Pass the locale, ideally as a parameter with a default."
    bad: "name.toLowerCase()"
    good: "name.toLocaleLowerCase(\"tr-TR\")"
  - title: "Boolean and undefined holes in positional calls"
    detail: "render(rows, undefined, true, false) compiles but hides which flag is which, and reordering the parameters breaks every caller silently if the types match. Use an options object once there is more than one optional argument."
    bad: "delaysPositional(undefined, undefined, 1)"
    good: "delays({ factor: 1 })"
quiz:
  - q: "function f(a: string, b?: number) — inside f, what is the type of b?"
    options: ["number", "number | undefined", "number | null", "any"]
    answer: 1
    why: "A missing argument arrives as undefined, so the checker adds undefined to the declared type and makes you narrow before using b as a number."
  - q: "Given function lower(text: string, locale = \"tr-TR\"), which call uses the default locale?"
    options: ["lower(\"I\", null)", "lower(\"I\", undefined)", "lower(\"I\", \"\")"]
    answer: 1
    why: "Only undefined, explicit or from a missing argument, selects a default. null is rejected by tsc, and an empty string is a real argument value."
  - q: "function f(x: string | undefined) and function g(x?: string). Which of f() and g() compiles?"
    options: ["Only f()", "Only g()", "Both", "Neither"]
    answer: 1
    why: "f's parameter is required; its type merely allows undefined, so f() is TS2554 and callers must write f(undefined). The ? on g's parameter makes the argument omittable."
further_reading:
  - title: "TypeScript Handbook: More on Functions"
    url: "https://www.typescriptlang.org/docs/handbook/2/functions.html"
  - title: "TSConfig reference: noImplicitAny"
    url: "https://www.typescriptlang.org/tsconfig/#noImplicitAny"
  - title: "MDN: Default parameters"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Functions/Default_parameters"
  - title: "MDN: Rest parameters"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Functions/rest_parameters"
  - title: "MDN: String.prototype.toLocaleLowerCase()"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/String/toLocaleLowerCase"
---

## Concept

TypeScript infers almost everything inside a function, including locals and the return
type. Parameters are the exception: their types are what inference starts from, so you
write them. Under `strict`, a parameter without a type is an error (TS7006), not an
implicit `any`. Two exceptions: a parameter with a default value takes its type from the
default, and a callback whose function type is already known gets its parameter types
from it (next lesson).

The return type is inferred from the `return` statements. Annotate it when it is a
contract: exported functions, recursive functions (which can't be inferred), and results
other code depends on. A wrong `return` is then an error at the definition, not at every
caller.

Three parameter forms change how a function may be called:

- `x?: T`: callers may leave it out; inside, `x` is `T | undefined`.
- `x = value`: callers may leave it out or pass `undefined`; inside, `x` is `T`, inferred
  from the default. The default expression runs on each call that needs it.
- `...xs: T[]`: any number of trailing arguments, collected into a real array.

Once a function has two or more optional settings, replace them with one options object,
destructured in the parameter list with a default per field.

## Coming from C/C++/Java/Python/JS

- Java: optional and default parameters replace telescoping overloads, and `...xs: T[]`
  is varargs. There is one implementation per function name; the next lesson shows TS
  overload signatures and why you rarely need them.
- C++: default arguments must be trailing. A TS default can sit anywhere (callers then pass
  `undefined` to get it) and can refer to earlier parameters: `(w: number, h = w)`.
- Python: an options object with destructuring plays the role of keyword arguments, and
  the mutable-default trap (`def f(x=[])`) doesn't exist. Python checks the argument
  count at runtime; TypeScript checks it at compile time, and plain JS not at all.
- JS: the runtime semantics are unchanged. TypeScript adds arity and type checks at call
  sites and the `T | undefined` view of optional parameters, then erases all of it.
- Go: no optional or default parameters and no overloading. Go uses an options struct or
  separate functions; `...T` is the same idea as a rest parameter.

## Design notes

Annotate at the boundaries and let inference handle the inside. Exported functions get
explicit return types so that a change in the body can't silently change the module's
API. The `isolatedDeclarations` option (TS 5.5) enforces this for every export whose type
isn't trivially obvious, so other tools can generate `.d.ts` files without running the
type checker.

Keep positional parameters for what every caller passes. One trailing optional parameter
is fine. After that, an options object names each argument at the call site, lets each
setting default independently, and lets you add a setting without breaking callers. It's
the TypeScript version of Go's options struct.

Text functions should take the locale as a parameter with an explicit default (`"tr-TR"`)
instead of relying on the runtime's default locale. The default locale differs between
your laptop, CI and a user's browser; an explicit parameter keeps the result the same
everywhere.
