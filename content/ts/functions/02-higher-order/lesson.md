---
id: ts.functions.higher-order
title: "Function types, callbacks, closures, this, arrow vs function, void vs undefined"
track: ts
module: functions
order: 2
level: basic
minutes: 14
since: ts3.4
prerequisites: [ts.functions.signatures]
compare_with: [go.basics.functions, go.interfaces.methods]
tags: [functions, callbacks, closures, this, arrow-functions, void, overloads]
summary: >
  Function types describe callbacks, and contextual typing fills in an inline
  callback's parameter types. Arrows capture this, methods passed as values
  lose it, void in a function type means "result ignored", and a union
  parameter usually beats overloads.
examples:
  - id: function-types
    file: examples/01_function_types.ts
    title: Function types, contextual typing and void callbacks
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-4"
        note: >
          A function type is structural: any function taking two strings and
          returning a number fits, with no functional interface to declare as in
          Java. Parameter names don't take part in compatibility.
      - lines: "6-9"
        note: >
          The annotation on byLength flows into the arrow: a and b are strings
          without being annotated, and the return value is checked against
          number. That flow is called contextual typing.
      - lines: "11-13"
        note: >
          sort() mutates the array in place and returns it, so copy first.
          sort(undefined) uses the default order, which is why an optional
          comparator can be passed straight through.
      - lines: "15-21"
        note: >
          The default sort compares UTF-16 code units: ç (U+00E7), ı (U+0131)
          and ş (U+015F) all come after z. Intl.Collator("tr") applies the
          Turkish alphabet. Its compare is bound to the collator, so it survives
          being passed as a bare value, unlike the methods in the next example.
      - lines: "22-24"
        note: >
          An inline callback gets its types from the parameter it is passed to.
          || chains comparators: a 0 from byLength (same length) falls through
          to the Turkish comparison.
      - lines: "26-31"
        note: >
          A void return in a function TYPE means the caller won't use the
          result, so a function returning a number fits. This is what lets you
          write forEach((x) => arr.push(x)). The value still exists at runtime:
          typeof result is "number".
      - lines: "32-33"
        note: >
          Because the result's type is void, tsc refuses to branch on it. The
          compiler won't let you depend on a value the type says to ignore.
      - lines: "35-39"
        note: >
          A function literal declared with : void must not return a value, and
          () => undefined demands an actual undefined. Use void for callbacks
          whose result you ignore; use undefined when callers handle it as a
          value. Since TS 5.1, a function declared : undefined may omit return.
  - id: closures-this
    file: examples/02_closures_this.ts
    title: Closures, this, and methods passed as values
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-10"
        note: >
          Each makeCounter() call creates a fresh count, and the returned arrow
          keeps that variable alive, so a and b don't share state: 1 2 1. The
          closure captures the variable itself, which it may reassign; Java
          lambdas only accept effectively final locals.
      - lines: "15-18"
        note: >
          In a method (and any function declared with function), this is set by
          the call: basket.add(x) passes basket. A bare reference to the method
          carries no receiver.
      - lines: "20-23"
        note: >
          An arrow has no this of its own; it uses the one around it, which in
          a field initializer is the new instance. The cost: one function per
          instance instead of one shared on the prototype.
      - lines: "26-32"
        note: >
          forEach calls basket.add with this undefined (modules and classes are
          strict mode), so this.items throws. tsc accepts the line; the
          type-aware typescript-eslint rule unbound-method reports it. Go's
          method values bind the receiver; JS method references don't.
      - lines: "33-36"
        note: >
          Three fixes, in the order you'll meet them: an arrow field, an arrow
          wrapper at the call site (the most common), and bind.
      - lines: "38-44"
        note: >
          A this parameter is a fake first parameter that esbuild and tsc erase.
          It only works on function declarations and expressions (arrows can't
          declare one). It catches the direct detached call, not a method
          passed as a callback.
  - id: overloads
    file: examples/03_overloads.ts
    title: Overloads and the union parameter you usually want instead
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-10"
        note: >
          Callers see the two overload signatures; the third signature belongs
          to the implementation and is invisible to them. tsc checks only that
          the implementation is compatible with the overloads, not that each
          branch returns what its overload promises.
      - lines: "12-14"
        note: >
          The call resolves to the first overload that matches, so single is a
          number and many is a number[]. Callers don't have to narrow the
          result, which is the one thing overloads buy you.
      - lines: "16-23"
        note: >
          A string | string[] argument matches neither overload on its own, so
          TS2769. The caller has to narrow first and call twice. Handbook
          rule: prefer union parameters to overloads when possible.
      - lines: "25-30"
        note: >
          When the return type doesn't depend on the argument type, a union
          parameter does the same job with one signature and accepts the mixed
          case directly.
      - lines: "32-35"
        note: >
          lib.dom.d.ts declares createElement with overloads: a generic one
          keyed by tag name plus a fallback taking any string and returning
          HTMLElement.
use_cases:
  - "Typing callbacks and handlers: comparators, listeners, React props such as onSelect: (id: string) => void"
  - "Closures for private state: counters, caches, debounced handlers"
  - "Arrow functions wherever the surrounding this is needed, including class fields passed as callbacks"
  - "Overloads only when the return type depends on the argument type: parse one vs parse many, createElement(tag)"
strengths:
  - "Function types are structural: any function with matching parameters and return fits, unlike Java's named functional interfaces or C#'s delegate types"
  - "Contextual typing types inline callbacks from the API they are passed to, so callback code carries almost no annotations"
  - "A void return in a callback type accepts any function, so forEach((x) => arr.push(x)) needs no wrapper"
  - "this parameters document and check what a standalone function expects as this, at zero runtime cost"
weaknesses:
  - "A method passed as a value loses this, and tsc does not catch it; Python's obj.method and Go's method values bind the receiver, JS's don't"
  - "Overload implementations are checked loosely: a branch can return number[] for the string overload and callers will believe it is a number"
  - "A void-typed result can still be a value at runtime, including a rejected Promise nobody awaits"
  - "Arrow-function fields cost one function object per instance and are not on the prototype, so subclasses can't call them through super"
pitfalls:
  - title: "Passing a method as a callback"
    detail: "arr.forEach(obj.method) runs the method with this undefined; it throws as soon as the method touches this. Wrap it in an arrow, bind it, or make it an arrow field. typescript-eslint's unbound-method rule (recommended-type-checked) reports it."
    bad: "items.forEach(basket.add)"
    good: "items.forEach((item) => basket.add(item))"
  - title: "Async callbacks where void is expected"
    detail: "An async function returns a Promise, which a void-returning slot accepts and ignores: forEach neither waits for it nor sees its rejection. Use for...of with await, or Promise.all. typescript-eslint's no-misused-promises (recommended-type-checked) reports it."
    bad: "ids.forEach(async (id) => await save(id))"
    good: "await Promise.all(ids.map((id) => save(id)))"
  - title: "Optional parameters in callback types"
    detail: "onRow: (row: Row, index?: number) => void tells every implementation that index may be missing. If you always pass it, declare it required: a function that takes fewer parameters is still assignable, so implementations can ignore index anyway."
    bad: "type OnRow = (row: Row, index?: number) => void"
    good: "type OnRow = (row: Row, index: number) => void"
quiz:
  - q: "type Listener = (e: string) => void. Does const f: Listener = (e) => e.length compile?"
    options: ["No: a void function must not return a value", "Yes: void in a function type means the result is ignored", "Only with an explicit : number on the arrow"]
    answer: 1
    why: "Contextual typing with a void return type doesn't forbid returning something; the value is ignored. Only a function literal declared with : void must not return a value."
  - q: "class C { n = 1; get() { return this.n; } } — what does [0].map(new C().get) do?"
    options: ["Returns [1]", "Fails to compile", "Throws a TypeError at runtime", "Returns [undefined]"]
    answer: 2
    why: "map calls the bare method with this undefined, and reading this.n throws. tsc does not check this for method references, so it compiles."
  - q: "When are overload signatures the right tool?"
    options: ["Whenever a parameter accepts several types", "When the return type depends on which argument type was passed", "To give parameters default values", "To bind a method to its instance"]
    answer: 1
    why: "If the return type is the same for every argument shape, a union parameter is simpler and also accepts a union argument, which overloads reject."
further_reading:
  - title: "TypeScript Handbook: More on Functions"
    url: "https://www.typescriptlang.org/docs/handbook/2/functions.html"
  - title: "TypeScript Handbook: Type Compatibility"
    url: "https://www.typescriptlang.org/docs/handbook/type-compatibility.html"
  - title: "MDN: this"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/this"
  - title: "MDN: Arrow function expressions"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Functions/Arrow_functions"
  - title: "MDN: Closures"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Closures"
---

## Concept

Functions are values, and their types are written like arrows:
`(a: string, b: string) => number`. Only the parameter types, their order and the return
type matter; names are documentation. When TypeScript already knows which function type
a position expects (an annotated variable, a parameter of the function you're calling),
it types the function literal from it. This is **contextual typing**, and it's why inline
callbacks rarely need annotations.

A closure captures variables, not snapshots of their values. Arrow functions also take
`this` from where they are defined. Functions declared with `function`, and methods, get
`this` from each call, so `obj.method` passed as a value arrives without `obj`.

`void` and `undefined` differ as return types. In a function *type*, `void` means "the
caller ignores the result", and any function fits. A function literal declared `: void`
must not return a value. `undefined` is a value type: the function returns `undefined`
and the caller may rely on it.

Overload signatures give one function several call signatures, so the return type can
follow the argument type. When the return type is the same for every argument shape, a
union parameter does the job better.

## Coming from C/C++/Java/Python/JS

- Java: no functional interfaces to name. Any function with the right shape fits a
  function type. Closures may reassign captured locals; there's no "effectively final"
  rule. Overloads are signatures over a single body, not separate methods.
- C++: an arrow capturing variables is like a `[&]` lambda, but the garbage collector keeps
  captured variables alive, so nothing dangles.
- Python: `obj.method` is a bound method in Python. In JS it is a bare function that has
  forgotten `obj`, which is the single most surprising difference in this lesson.
- Go: `func(a, b string) int` is the same idea as a function type. Go's method values
  (`f := basket.Add`) bind the receiver, and Go has no overloading at all.
- JS/React: the runtime is unchanged. TypeScript checks callback shapes such as React props
  (`onChange: (value: string) => void`). Function components and hooks avoid `this`
  entirely, which is one reason modern TS code rarely meets these problems.

## Design notes

Prefer plain functions and arrows, and keep `this` inside classes. A class method that
will be handed to someone else (an event listener, `forEach`) is either an arrow field or
gets wrapped at the call site. Enable the type-aware typescript-eslint presets
(`recommended-type-checked`) to catch the cases tsc lets through: `unbound-method` and
`no-misused-promises`.

Type callback results as `void` when you ignore them; that keeps `(x) => arr.push(x)`
legal. Name a function type (`type Compare`) when it's part of an API, so signatures and
error messages refer to it by name.

Before writing overloads, try the alternatives in order: a union parameter (same return
type), two functions with different names (`parseAmount`, `parseAmounts`, often the
clearest), and generics (the generics module). Keep overloads for return types that
genuinely follow the argument type, list the most specific signature first, and remember
that the implementation's branches are not checked against each overload.
