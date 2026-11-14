---
id: ts.architecture.patterns
title: "Idiomatic TS patterns (strategy, adapter, observer, factory) — not ported Java"
track: ts
module: architecture
order: 3
level: advanced
minutes: 14
since: ts4.9
prerequisites: [ts.mental-model.structural, ts.functions.higher-order, ts.generics.keyof]
compare_with: [design.patterns.classic, design.paradigms.polymorphism, go.interfaces.implicit, ts.objects-classes.classes]
tags: [patterns, strategy, adapter, observer, factory, satisfies, structural-typing]
summary: >
  The classic patterns still solve real problems in TypeScript, but functions are values
  and types are structural, so a strategy is a function, an adapter is a function returning
  an object literal, an observer is subscribe-returns-unsubscribe, and a factory needs no class.
examples:
  - id: strategy
    file: examples/01_strategy.ts
    title: "Strategy: a function type, a record of strategies, and composition"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "4"
        note: >
          The whole Strategy interface. Any function with this shape qualifies,
          inline arrows included; nothing has to declare that it implements
          anything. Intl.Collator's compare takes strings, so line 12 adapts it
          to Lessons in one arrow.
      - lines: "11-16"
        note: >
          satisfies (TS 4.9) checks every entry against Compare and gives a and b
          their types by context, but keeps the object's own type, so keyof
          still knows the exact keys. Annotating the variable as
          Record<string, Compare> would widen SortKey to string.
      - lines: "18-30"
        note: >
          Chaining and reversing are functions from strategies to strategies.
          The Java equivalents, Comparator.thenComparing and reversed(), are
          default methods on an interface for the same reason.
      - lines: "42-47"
        note: >
          The Turkish collator puts Ç after C, İ after I and Ş after S. String
          comparison with < (and sort() without a comparator) orders by UTF-16
          code unit, which sends all three after Z.
      - lines: "48-49"
        note: >
          An unknown strategy name is a compile error. tsc 7 prints the union in
          its own stable order ("level" | "minutes" | "title"); tsc 6 printed it
          in declaration order.
  - id: adapter-factory
    file: examples/02_adapter_factory.ts
    title: "Adapter and factory: two backends behind one port, and the cast that hides a gap"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-7"
        note: >
          The port is declared by the consumer, in the notes feature, in the
          terms it needs. The SDK's callback style and key scheme never leak past
          the adapter.
      - lines: "22-35"
        note: >
          The adapter is a function returning an object literal. The declared
          return type is checked against NotesStore here, and contextual typing
          gives id and body their types without annotations.
      - lines: "37-45"
        note: >
          The in-memory adapter keeps its inferred type and uses satisfies for
          the check. Either form fails at the definition when a method is
          missing or has the wrong signature.
      - lines: "47-50"
        note: >
          The factory is the one place that knows the concrete adapters, like
          this app's pickApi() choosing between the backend and localStorage.
      - lines: "59-65"
        note: >
          as unknown as tells the checker to stop comparing shapes. The missing
          save surfaces as a TypeError at the first call, possibly in
          production; satisfies would have failed the build.
  - id: observer
    file: examples/03_observer.ts
    title: "Observer: a typed event map, unsubscribe functions, and EventTarget with AbortSignal"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-6"
        note: >
          The event map is the contract: event names as keys, payload types as
          values. Adding an event is one line here, and every emit and on is
          checked against it.
      - lines: "9-20"
        note: >
          K extends keyof E ties the payload to the event name, so a listener
          for "finished" receives exactly that payload. The mapped type gives
          each event its own listener set.
      - lines: "14"
        note: >
          subscribe returns its own undo. This is the contract React's
          useSyncExternalStore expects, and the shape of useEffect cleanup:
          return the function from the effect and the subscription ends with it.
      - lines: "34-45"
        note: >
          EventTarget is built into browsers and Node. Its listeners receive a
          plain Event, hence the cast to CustomEvent; in exchange, one
          AbortController removes every listener registered with its signal.
use_cases:
  - "Pluggable behaviour such as sort orders, pricing rules or retry policies: a record of functions keyed by a union, with satisfies"
  - "Wrapping a third-party SDK, a legacy callback API or a different backend behind the interface your feature declares"
  - "Notifying several parts of the UI about runs, saves or connection changes without them importing each other"
  - "Choosing an implementation at startup (backend vs localStorage, real vs fake) in one factory function"
strengths:
  - "Unlike Java before lambdas, a strategy needs no interface declaration and no class: a function type is the interface and any matching function implements it"
  - "Structural typing lets an object literal satisfy an interface with no implements clause, so adapters for tests are a few lines"
  - "Closures give truly private state to factory-made objects, enforced at run time, without classes or #private"
  - "Mapped types over an event map give a fully typed observer in about ten lines, where a Java version needs listener interfaces per event"
weaknesses:
  - "Without implements, nothing ties an object to an interface unless you write a return type or satisfies; a mismatch then surfaces only where the object is first used"
  - "Function-valued strategies are invisible in stack traces and debuggers when they are anonymous; name the important ones"
  - "A hand-written emitter has no built-in error isolation: a throwing listener stops the remaining listeners in that emit"
  - "EventTarget's listeners are typed as Event, so payloads need casts or a typed wrapper"
pitfalls:
  - title: "Porting the class hierarchy"
    detail: "An interface with one method and a class per implementation is the Java encoding of a function. In TS it is ceremony: more names, more files, and `this` binding problems when a method is passed as a callback."
    bad: "interface SortStrategy { compare(a: Lesson, b: Lesson): number }\nclass TitleSort implements SortStrategy { compare(a: Lesson, b: Lesson) { return a.title.localeCompare(b.title); } }"
    good: "type Compare = (a: Lesson, b: Lesson) => number;\nconst byTitle: Compare = (a, b) => turkish.compare(a.title, b.title);"
  - title: "Casting an adapter to the port"
    detail: "A double cast compiles even when methods are missing. Let the compiler compare shapes: annotate the return type or use satisfies."
    bad: "const local = { /* ... */ } as unknown as LearningApi;"
    good: "const local = { /* ... */ } satisfies LearningApi;"
  - title: "Forgetting to unsubscribe"
    detail: "A listener that outlives its component keeps it alive and runs on unmounted state. Return the unsubscribe function from useEffect, or pass an AbortSignal to addEventListener."
quiz:
  - q: "Why does `const sorters = { ... } satisfies Record<string, Compare>` keep `keyof typeof sorters` precise, where `const sorters: Record<string, Compare> = { ... }` does not?"
    options: ["satisfies checks the value but leaves its inferred type alone, so the literal keys survive", "satisfies is erased by the compiler", "Record<string, ...> only allows string values"]
    answer: 0
    why: "An annotation replaces the inferred type with Record<string, Compare>, whose keys are any string. satisfies validates against the type and keeps the narrower inferred one."
  - q: "An adapter object is missing one method of the port interface. Which version fails to compile?"
    options: ["`const a = { ... } as unknown as Port`", "`const a = { ... } satisfies Port`", "Neither: structural typing ignores missing methods"]
    answer: 1
    why: "satisfies compares the shapes and reports the missing property. The double cast tells the checker not to compare at all."
  - q: "What should a subscribe function return so that it fits React's useSyncExternalStore and useEffect cleanup?"
    options: ["The listener it was given", "A function that removes the listener", "A Promise that resolves on the first event"]
    answer: 1
    why: "Both APIs expect an unsubscribe function: React calls it when the component unmounts or resubscribes."
further_reading:
  - title: "TypeScript 4.9 release notes: the satisfies operator"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-4-9.html"
  - title: "TypeScript Handbook: Mapped types"
    url: "https://www.typescriptlang.org/docs/handbook/2/mapped-types.html"
  - title: "MDN: EventTarget.addEventListener() (the signal option)"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/EventTarget/addEventListener"
  - title: "MDN: Intl.Collator"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/Collator"
  - title: "React: useSyncExternalStore"
    url: "https://react.dev/reference/react/useSyncExternalStore"
---

## Concept

The Gang of Four patterns describe problems that don't go away: swap an algorithm at run
time, fit a foreign API to your own, notify interested parties, hide which implementation
you got. What changes in TypeScript is the encoding. Java expressed them with interfaces
and classes because, until Java 8, a method couldn't be passed as a value, and a
type had to be declared to be implemented. TypeScript has neither restriction:

- **Strategy** is a **function type**. A family of strategies is a record of functions keyed
  by a union, checked with `satisfies`. Combining strategies (`thenBy`, `reversed`) is a
  higher-order function.
- **Adapter** is a **function from their shape to yours**: it takes the SDK, the callback API
  or the other backend and returns an object literal that fits the port your feature
  declared. Structural typing means nothing has to say `implements`; a return type or
  `satisfies` makes the compiler check the fit where the adapter is defined.
- **Observer** is **`subscribe(listener)` returning an unsubscribe function**, typed with an
  event map. The platform has one built in: `EventTarget`, whose listeners an
  `AbortSignal` can remove in bulk.
- **Factory** is a **function that returns an object**. Closures hold private state,
  `ReturnType<typeof create>` names the result when you need a type, and choosing among
  implementations is a `switch` or a conditional over a union.

Classes still have a place: many instances sharing methods, `instanceof` checks (error
types), state that must be truly private with `#fields`, or a framework that requires
them. Reach for one when it earns its keep, not because the pattern book drew one.

## Coming from C/C++/Java/Python/JS

- **Java:** `Comparator<T>` is already a functional interface, and since Java 8 a lambda is
  the idiomatic strategy. TypeScript drops the interface name: `(a: T, b: T) => number` is
  the type. Spring's bean factories become ordinary functions called once in `main.tsx`.
- **C++:** `std::function` or a template parameter (policy-based design) instead of a
  virtual base class; closures replace functor objects with state.
- **Python:** first-class functions and duck typing, with `typing.Protocol` as the closest
  thing to a TS interface. The difference is that tsc checks the duck before run time.
- **Go:** the same journey happened in the standard library. `sort.Interface` needed a type
  with three methods; `slices.SortFunc` takes a plain function. Go's implicitly satisfied
  interfaces are TypeScript's structural typing restricted to methods.
- **JS:** these were always functions and objects. TypeScript adds checked shapes, typed
  event maps and `satisfies`.

## How it works

All four rely on two features covered earlier. **Structural typing**
(ts.mental-model.structural): an object literal with the right members is a `NotesStore`,
whoever made it. **Contextual typing** (ts.functions.higher-order): when a function
expression appears where a function type is expected, its parameters take their types
from it, which is why `(a, b) => ...` in the `sorters` record needs no annotations.

`satisfies` (TS 4.9) checks an expression against a type without changing the
expression's type. That matters for strategy tables and config objects: you want both
"every entry is a `Compare`" and "the keys are exactly title, level and minutes". An
annotation gives the first and loses the second; `satisfies` gives both.

## Design notes

This app uses all four, mostly well. `src/features/runner/CodeRunner.ts` calls itself a
strategy: a one-method `CodeRunner` interface, two classes and a factory `runnerFor`. The
`BrowserTsRunner` class earns its shape (it owns a worker and a lazily built import map);
`ServerRunner` holds no state, and a function would do. `src/features/learning/api.ts`
has two adapters of one port, `learningApi` (fetch) and `localLearningApi` (localStorage),
with `pickApi()` as the factory. It assigns the local one through
`as unknown as typeof learningApi`. That cast is unnecessary today: with it removed, tsc
7.0.2 accepts the assignment, because the shapes match. Kept, it guarantees that the day
they stop matching, the compiler stays silent. `satisfies typeof learningApi` on the object
literal keeps the check. `src/features/learning/store.ts` is an observer whose `subscribe`
returns an unsubscribe function, exactly the contract `useSyncExternalStore` asks for.

Choose the plain function first and promote it when there is a reason: a second method
that shares state, a lifecycle to manage, a need for `instanceof`. Most code never gets
there, and code that does gets a class for a reason someone can name.
