---
id: design.patterns.classic
title: "Strategy, adapter, decorator/middleware, observer/pub-sub — idiomatically per language"
track: design
module: patterns
order: 1
level: intermediate
minutes: 13
since: ts5.0
prerequisites: [go.interfaces.implicit, ts.functions.higher-order]
compare_with: [ts.architecture.patterns, go.http.middleware, design.paradigms.polymorphism, go.services.listen-notify, sql.server-side.triggers]
tags: [patterns, strategy, adapter, decorator, middleware, observer, outbox]
summary: >
  Four Gang of Four patterns in Go, TypeScript and SQL. The problems they solve are
  permanent; the interfaces, base classes and registries that Java needed to express
  them mostly disappear, and in SQL some patterns become data.
examples:
  - id: func-strategy
    file: examples/01_func_strategy.go
    title: "Go: a strategy is a func type; decorators and an adapter are functions around it"
    runtime: go
    check: exact
    annotations:
      - lines: "14-18"
        note: >
          The strategy "interface" is a function type, so any function of that shape is
          a strategy. The one-line Fee method is the http.HandlerFunc trick: it lets a
          plain function satisfy a one-method interface without a wrapper struct.
      - lines: "22-34"
        note: >
          A decorator takes a FeeRule and returns one, so decorators nest freely and no
          abstract base class is needed. The outermost one decides first: freeAbove
          returns 0 before capped or the carrier are ever asked.
      - lines: "36-43"
        note: >
          The adapter is a closure. Its real job is the boundary conversion: the SDK
          speaks float64 lira, the program speaks integer kuruş, and the rounding happens
          once, here, so float money never leaks inward.
      - lines: "45-46"
        note: >
          The consumer declares the port it needs (go.design.packages). FeeRule satisfies
          it implicitly, and so would any struct with a Fee method: a strategy that needs
          state and several methods can grow into a struct without touching the caller.
      - lines: "56-59"
        note: >
          The whole "pattern catalogue" is four lines of composition. In Java this is an
          interface, three strategy classes, an abstract decorator and two subclasses.
  - id: decorator-order
    file: examples/02_decorator_order.ts
    title: "TypeScript: decorator order is semantics, and what @ really does"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "5-12"
        note: >
          A decorator as a higher-order function: same type in and out. It caches the
          Promise, not the value, so two concurrent calls for one word share a single
          in-flight lookup.
      - lines: "22-30"
        note: >
          The same two decorators in two orders. logged outside cached sees every call
          (three lines); logged inside cached sees only misses (two lines). Middleware
          order in go.http.middleware is the same rule: the outer layer sees more.
      - lines: "32-42"
        note: >
          A TS 5.0 standard decorator is a function from (method, context) to a
          replacement method. It runs once, when the class is defined. Angular and
          NestJS use the older experimentalDecorators form, whose signature differs.
      - lines: "37"
        note: >
          This Map is created once per class definition, so every instance shares it.
          That is why b.computed stays 0 below: b got a's cached answer. Per-instance
          state needs a WeakMap keyed by this, or context.addInitializer.
      - lines: "52-54"
        note: >
          Prints İĞNE twice and "computed: 1 0". The Turkish locale also turns i into
          İ, which plain toUpperCase() would not.
  - id: patterns-in-sql
    file: examples/03_patterns_in_sql.sql
    title: "SQL: strategy as rows, adapter as a view, observer as a trigger writing an outbox"
    runtime: sql
    check: exact
    annotations:
      - lines: "3-10"
        note: >
          In SQL a strategy becomes data: one row per rule. Adding a rule is an INSERT,
          not a deploy, at the price of the type checks and tests that code would get.
      - lines: "18-20"
        note: >
          A view is an adapter: it renames Turkish legacy columns and converts lira to
          kuruş once, and every consumer queries orders in the shape it expects.
      - lines: "22-26"
        note: >
          The join picks the strategy per row and CASE applies it. For the flat rule,
          total_kurus >= NULL is NULL, not true, so CASE falls through to the ELSE branch:
          NULL means "never free" without a special case.
      - lines: "35-42"
        note: >
          A trigger is an observer on every ordinary write path: psql, migrations and
          every service fire it. The event row is written in the same transaction as the order,
          so a rolled-back order leaves no event. That is the outbox pattern.
      - lines: "44-45"
        note: >
          One INSERT of two rows produced two events, because the trigger is FOR EACH
          ROW. A bulk load of a million rows writes a million outbox rows too.
use_cases:
  - "Strategy: pricing, sorting, retry policies, storage backends; a func type in Go, a function type or record of functions in TS, a rules table in SQL when non-developers change the rules"
  - "Adapter: wrapping a vendor SDK, a legacy table (a view) or a second backend behind the port your code declared"
  - "Decorator/middleware: logging, caching, auth, timeouts, metrics around a handler or a client call"
  - "Observer/pub-sub: UI stores, domain events, cache invalidation; an outbox table when the event must survive a crash"
strengths:
  - "Go's implicit interfaces and func types make every pattern here a few lines: unlike Java, the strategy never declares that it implements anything, and a function can satisfy an interface through a method on its type"
  - "Decorators compose by plain function application in both Go and TS, so a stack of five is readable top to bottom, unlike five nested subclasses of an abstract decorator"
  - "SQL patterns are enforced for every client: a view or a trigger applies to psql, migrations and every service, which no application-level adapter can promise"
  - "An outbox row written in the same transaction as the change gives at-least-once delivery without a distributed transaction"
weaknesses:
  - "Function types carry no name in error messages or stack traces: a panic inside a closure shows main.fromSDK.func1, not CarrierPricer.Fee"
  - "Strategies stored as rows lose compile-time checks; a misspelled rule name is a missed join, not a build error"
  - "Triggers are invisible from the application code: a reader of the INSERT has no hint that it also writes an outbox row"
  - "TS standard decorators apply only to classes and their members; a plain function still needs the higher-order-function form"
pitfalls:
  - title: "Decorator order hides calls"
    detail: "A cache outside the logger means cache hits never reach the log, so the log undercounts traffic. Put observers (logging, metrics) outermost unless you mean to measure only misses."
    bad: |
      handler := withCache(withLogging(api)) // hits are never logged
    good: |
      handler := withLogging(withCache(api)) // every request logged, hit or miss
  - title: "Per-class state in a TS decorator"
    detail: "The decorator function runs once per class definition. Anything it creates (a Map, a counter) is shared by every instance, as b.computed = 0 shows. Key per-instance state by this in a WeakMap."
  - title: "An interface per strategy, declared up front"
    detail: "Java habit: IFeeStrategy next to the only implementation. In Go, declare the interface in the consumer when a second implementation or a fake appears (go.design.packages); in TS a function type is already the interface."
  - title: "NOTIFY treated as a durable queue"
    detail: "NOTIFY is delivered at commit to sessions listening at that moment, and is gone otherwise. Use it to wake a worker, and keep the event itself in an outbox table the worker reads."
quiz:
  - q: "Which stack logs only cache misses?"
    options: ["logged(cached(upper))", "cached(logged(upper))", "Both log every call", "Neither logs anything"]
    answer: 1
    why: "The cache sits outside the logger, so a hit returns before the logger is reached. The outer layer sees more calls, the same rule as Go middleware."
  - q: "How does http.HandlerFunc (or FeeRule above) let a plain function satisfy a one-method interface?"
    options: ["A compiler special case for functions", "A named func type with a method that calls the function itself", "Reflection at run time", "An implements clause on the type"]
    answer: 1
    why: "Methods can be declared on any named type, function types included. The method just calls f, and the type now has the method set the interface asks for."
  - q: "An order and its 'order placed' event must be committed together or not at all. Which design gives that?"
    options: ["NOTIFY inside the transaction", "Publishing to a broker after COMMIT", "An outbox row written in the same transaction", "A goroutine started before COMMIT"]
    answer: 2
    why: "The outbox row commits or rolls back with the order. NOTIFY also waits for commit but is lost if no one is listening, and publishing after COMMIT can fail after the order is already saved."
  - q: "Why did b.upper(\"iğne\") leave b.computed at 0?"
    options: ["The decorator ran once per class, so a and b share its cache", "toLocaleUpperCase caches results", "b's method was never decorated", "Class fields are shared between instances"]
    answer: 0
    why: "Standard decorators are evaluated when the class is defined. The returned method is on the prototype, and the Map it closes over is shared by every instance."
further_reading:
  - title: "Effective Go: Interfaces and methods (HandlerFunc)"
    url: "https://go.dev/doc/effective_go#interface_methods"
  - title: "TypeScript 5.0 release notes: Decorators"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-5-0.html#decorators"
  - title: "MDN: EventTarget"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/EventTarget"
  - title: "PostgreSQL: Trigger functions in PL/pgSQL"
    url: "https://www.postgresql.org/docs/current/plpgsql-trigger.html"
  - title: "PostgreSQL: NOTIFY"
    url: "https://www.postgresql.org/docs/current/sql-notify.html"
---

## Concept

The *Design Patterns* book (1994) wrote its examples in C++ and Smalltalk. The problems
it names are still with us: swap an algorithm (**strategy**), fit a foreign shape to yours
(**adapter**), add behaviour around an object without changing it (**decorator**), tell
interested parties that something happened (**observer**). What changes from language to
language is how much ceremony the encoding needs.

- **Strategy.** Go: a `func` type, or a one-method interface when the strategy has state.
  TS: a function type, or a record of functions checked with `satisfies`
  (ts.architecture.patterns). SQL: a row in a rules table, chosen by a join and applied by
  `CASE`; the strategy becomes data.
- **Adapter.** Go: a closure or a small struct that wraps the foreign type and satisfies
  the interface your code declared; a method on a func type (`http.HandlerFunc`) adapts a
  plain function. TS: a function from their shape to yours. SQL: a view.
- **Decorator / middleware.** Go: `func(http.Handler) http.Handler` (go.http.middleware),
  or any function from `T` to `T`. TS: a higher-order function, and for class members the
  TS 5.0 `@` syntax. SQL has no clean equivalent: a view over a view adds columns, a
  `BEFORE` trigger rewrites a row on its way in.
- **Observer / pub-sub.** Observer is in-process and synchronous: the subject calls its
  listeners. Pub-sub puts something between them. Go: a slice of callbacks, or channels
  fanned out to goroutines. TS: `subscribe` returning an unsubscribe function, or
  `EventTarget`. SQL: a trigger is an observer inside the transaction, `LISTEN/NOTIFY` is
  pub-sub at commit, and an **outbox table** is pub-sub that survives a crash.

What disappears is the scaffolding: interfaces declared only so a class can implement
them, one class per variant, abstract decorator base classes, `Subject` and `Observer`
registries. What remains is a type and a few functions.

## Coming from C/C++/Java/Python/JS

- **Java**: since Java 8 a lambda is the idiomatic strategy (`Comparator<T>`), and Servlet
  filters are middleware. `java.util.Observable` has been deprecated since Java 9. Note
  that `@Transactional` is an annotation, metadata that Spring reads to generate a proxy
  (a decorator) at run time. A TS decorator is not metadata: it is code that runs when
  the class is defined and returns the replacement.
- **Python**: `@functools.lru_cache` is a higher-order function applied at definition
  time, which is exactly what a TS standard decorator is. Python's `@` also works on
  plain functions; TS's works only on classes and class members.
- **C++**: strategy is `std::function` or a template parameter (policy-based design);
  a decorator is a wrapper class holding the wrapped object. Go and TS closures replace
  the functor object and its captured fields.
- **JS**: Node's `EventEmitter` and the DOM's `EventTarget` are the observer pattern
  built in. TypeScript adds typed event maps; it adds no new pattern.

## How it works

Decorators compose like function application, so `logged(cached(upper))` means "call
`logged`'s wrapper, which calls `cached`'s wrapper, which may call `upper`". The outer
layer always runs; inner layers run only if outer ones pass the call on. That single rule
explains the TS output and the Go fee table: `freeAbove` answers 0 for a 720 TL order
before `capped` or the carrier SDK are asked.

A TS standard decorator receives the original method and a context object describing it
(`kind`, `name`, `static`, `private`, `addInitializer`). Whatever it returns replaces the
method on the prototype. Because that happens once, at class definition, state created
in the decorator body is per class.

In SQL, an `AFTER ... FOR EACH ROW` trigger runs inside the statement that fired it, in
the same transaction. The outbox row and the order row therefore commit or roll back
together; a separate worker reads the outbox and publishes. `NOTIFY` is also held until
commit, but it reaches only sessions that are listening at that moment, so it is a
wake-up call, not storage (go.services.listen-notify).

## Design notes

Start with the function. Promote a Go strategy to an interface with a struct behind it,
or a TS function to an object, when it needs a second method that shares state, a
lifecycle (`Close`), or an identity you want in logs and stack traces. Promote a
decorator stack into a named chain helper when the same stack appears in more than one
place, as Go routers do with middleware.

Patterns in SQL trade differently. A rules table lets an operator change pricing without
a deploy, but a typo in a rule name is a missed join rather than a compile error, so add
a foreign key from the orders to the rules. A view is the cheapest adapter there is, and
the right way to rename columns during a migration. A trigger as observer is powerful
and hidden: use it for things that must happen on every write path (an outbox, an
audit row), and keep business decisions out of it (sql.server-side.triggers,
design.architecture.business-rules).

StackLab shows most of these in production code, with one honest gap.
`httpapi.Runner` is a port with two adapters: `runexec` runs lessons in-process and
`runclient` adapts the Docker runner's HTTP API to the same method. `withLogging` in
`httpapi/middleware.go` is the decorator, wrapping the whole mux so even 404s are
logged. The frontend's `CodeRunner` is a strategy and `learning/store.ts` an observer
(ts.architecture.patterns reviews both). There is no pub-sub, because nothing needs one:
one user, one process. If a second process ever needed to react to runs, an outbox table
plus `LISTEN/NOTIFY` as the wake-up would be the boring first step, long before a broker.
