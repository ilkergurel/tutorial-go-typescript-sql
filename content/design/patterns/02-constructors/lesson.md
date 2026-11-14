---
id: design.patterns.constructors
title: "Repository, factory/constructor, builder vs functional options, DI without frameworks"
track: design
module: patterns
order: 2
level: intermediate
minutes: 14
since: pg14
prerequisites: [go.design.constructors, ts.react.effects-hooks]
compare_with: [go.postgres.repository-testing, ts.architecture.patterns, go.design.layers, sql.security-ops.roles, inside.codebase.wiring]
tags: [patterns, dependency-injection, builder, factory, repository, search_path]
summary: >
  Construction and wiring in Go, TypeScript and SQL: when a builder beats options, how a
  composition root replaces a DI container, and how PostgreSQL's search_path wires
  names, early or late.
examples:
  - id: query-builder
    file: examples/01_query_builder.go
    title: "Go: a builder for a query assembled over several calls, with a sticky error"
    runtime: go
    check: exact
    annotations:
      - lines: "9-16"
        note: >
          A builder is right when construction accumulates: a variable number of parts,
          added under conditions. Functional options (go.design.constructors) configure
          a value once from a fixed set of knobs, which this is not.
      - lines: "18-24"
        note: >
          The builder owns the placeholder numbering. Callers write ?, the builder
          appends the value and writes $1, $2, so a value can never become SQL text and
          no hand-counted $3 can drift out of step with the args slice.
      - lines: "26-36"
        note: >
          A chained method can't return an error without breaking the chain, so the
          first one is stored and every later call carries on. This is the errWriter
          idea from the Go blog's "Errors are values", and why bufio.Writer reports a
          failed write at Flush.
      - lines: "38-42"
        note: >
          Build is the single exit, and the only place a caller has to check. The third
          form's sort key never reaches the SQL string.
      - lines: "60-68"
        note: >
          The reason for the builder: parts are added in ifs. With a constructor or
          options, the caller would collect conditions and arguments in parallel slices
          and number placeholders by hand.
  - id: composition-root
    file: examples/02_composition_root.tsx
    title: "TypeScript: a factory, a port, and React context as the injection channel"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "6-19"
        note: >
          The port is a type the UI declares; the factory is a function that closes over
          its configuration (base URL, the fetch to use) and returns an object literal.
          No class, no new, no interface with an I prefix.
      - lines: "21-26"
        note: >
          Context is React's injection mechanism. A null default plus a throwing hook
          turns "forgot to wire it" into an error naming the fix, instead of a component
          calling methods on undefined.
      - lines: "28-35"
        note: >
          The component asks for a Progress and has no idea whether it talks HTTP or is a
          stub. Compare a component that imports a concrete client module: a test then
          has to mock the module or the network.
      - lines: "58-60"
        note: >
          The composition root: the only lines that name concrete implementations. The
          fake is an object literal that fits the type; nothing had to be designed for
          testing beyond accepting the port.
  - id: search-path-wiring
    file: examples/03_search_path_wiring.sql
    title: "SQL: search_path binds names; the binding time decides who controls it"
    runtime: sql
    check: exact
    annotations:
      - lines: "3-9"
        note: >
          Two schemas, one table name: two implementations of the same "interface".
          Unqualified names are resolved through search_path, the database's wiring.
      - lines: "13-15"
        note: >
          A function body written as a string is parsed when the function runs, so
          "lessons" is looked up in whatever search_path the caller has at that moment:
          late binding, the service-locator shape.
      - lines: "16-18"
        note: >
          The SQL-standard body (RETURN or BEGIN ATOMIC, PostgreSQL 14) is parsed at
          CREATE time. The table is bound then, and recorded as a dependency, which the
          DROP TABLE at the end runs into.
      - lines: "19-22"
        note: >
          SET search_path on the function keeps late lookup but fixes where it looks.
          PostgreSQL's docs ask for exactly this on SECURITY DEFINER functions, so a
          caller cannot slip a look-alike table or function in front.
      - lines: "26-28"
        note: >
          The caller puts catalog_fixture first. Only the late-bound function follows it
          (1 instead of 21). For a test that is the point; for a privileged function it
          is a hijack.
      - lines: "30-31"
        note: >
          The dependency recorded for minutes_bound blocks the drop (SQLSTATE 2BP01).
          The string-bodied functions recorded nothing: dropping the table would have
          succeeded and left them to fail at their next call.
use_cases:
  - "Builder: query and filter assembly, request construction with optional parts, report generation; anything built over several conditional calls"
  - "Functional options or an options object: configuring a client or server once, with defaults, from a known list of settings"
  - "Composition root plus constructor parameters (Go) or context and props (React): wiring an app with a real and a fake implementation of each port"
  - "search_path: schema-per-tenant or schema-per-test setups, and pinning it on every SECURITY DEFINER function"
strengths:
  - "Wiring written as code is read like code: Go's run() and a React root show the whole dependency graph, unlike Spring's component scan, where you find a bean's implementation by searching annotations"
  - "Go builders need no library: methods returning the receiver plus a sticky error give fluent calls with one error check"
  - "A TS options object gets required-field checking from the type system, so the telescoping-constructor problem that builders solve in Java does not arise"
  - "PostgreSQL lets you choose the binding time per function: late (string body), early (SQL-standard body) or late in a fixed place (SET search_path)"
weaknesses:
  - "A fluent Go builder hides errors until Build; code that forgets to call Build, or ignores its error, fails silently"
  - "React context is implicit: a component's dependencies are not in its props, and a missing provider surfaces only at render time"
  - "search_path is session-global state: any code that changes it changes name resolution for everything that follows, the same hazard as a mutable service locator"
  - "Hand-written wiring grows with the app; past a few dozen components, main becomes long (still readable, but long)"
pitfalls:
  - title: "A builder where a struct literal would do"
    detail: "If every part is known at the call site, a Config struct or options are clearer than a chain. Keep builders for construction that accumulates."
    bad: |
      c := NewClientBuilder().BaseURL(u).Timeout(5 * time.Second).Build()
    good: |
      c := NewClient(u, WithTimeout(5*time.Second))
  - title: "Service locator disguised as DI"
    detail: "A module-level function that returns 'the' implementation (a cached singleton) is a locator: callers fetch their dependency instead of receiving it, and tests must mock the module. Pass the dependency in through a constructor, a prop or context."
  - title: "SECURITY DEFINER without a pinned search_path"
    detail: "The function runs with its owner's rights but resolves names through the caller's search_path. A caller who can create objects in any schema on the path can substitute a table or function the owner never meant to call."
    bad: |
      CREATE FUNCTION grant_badge(uid int) RETURNS void
        LANGUAGE sql SECURITY DEFINER
        AS $$ INSERT INTO badges VALUES (uid) $$;
    good: |
      CREATE FUNCTION grant_badge(uid int) RETURNS void
        LANGUAGE sql SECURITY DEFINER
        SET search_path = app, pg_temp
        AS $$ INSERT INTO badges VALUES (uid) $$;
  - title: "A repository interface with one implementation and no consumer that needs a substitute"
    detail: "go.postgres.repository-testing: the repository is the type that owns the SQL; the interface is added by the consumer that needs a fake, not up front."
quiz:
  - q: "Which situation calls for a builder rather than functional options?"
    options: ["A client with six optional settings", "A WHERE clause whose conditions depend on which form fields were filled in", "A server that needs a logger and a database pool", "A type whose zero value is useful"]
    answer: 1
    why: "Options configure a value once from known settings. A builder accumulates parts across conditional calls, which is exactly a dynamic WHERE clause."
  - q: "The Go builder stores the first error and keeps returning the receiver. Where must the caller check for errors?"
    options: ["After every chained call", "Only on Build's return value", "Nowhere; the builder panics", "In a deferred recover"]
    answer: 1
    why: "Chained calls can't return errors without breaking the chain, so the error sticks and Build reports it, as bufio.Writer does at Flush."
  - q: "After SET search_path = catalog_fixture, catalog_live, which function still sums catalog_live.lessons?"
    options: ["Only minutes_late", "minutes_bound and minutes_pinned", "All three", "None of them"]
    answer: 1
    why: "The SQL-standard body bound the table at CREATE time and the pinned function always looks in catalog_live; only the string body resolves names at call time."
  - q: "In React, what is the injection channel that plays the role of Go's constructor parameters for deeply nested components?"
    options: ["Module-level singletons", "Context, read through a hook", "Global window properties", "Class inheritance"]
    answer: 1
    why: "A Provider at the composition root supplies the implementation; components read it with useContext, and a test wraps them in a Provider with a fake."
further_reading:
  - title: "Go blog: Errors are values"
    url: "https://go.dev/blog/errors-are-values"
  - title: "React: useContext"
    url: "https://react.dev/reference/react/useContext"
  - title: "PostgreSQL: CREATE FUNCTION (sql_body, SET, Writing SECURITY DEFINER Functions Safely)"
    url: "https://www.postgresql.org/docs/current/sql-createfunction.html"
  - title: "PostgreSQL: Schemas and the schema search path"
    url: "https://www.postgresql.org/docs/current/ddl-schemas.html#DDL-SCHEMAS-PATH"
---

## Concept

Every program constructs things and then connects them. The three languages split that
work differently.

**Construction.** Go has no constructor syntax: `New` functions return a ready value, the
zero value should be useful, and optional settings go in a `Config` struct or functional
options (go.design.constructors). TypeScript constructs with **factory functions** that
return object literals, an options object with defaults for the settings, and a class
when instances share methods or need `#private` state. In SQL the table definition is
the constructor: `DEFAULT`, identity columns and `CHECK` decide what a valid row is, and
`INSERT ... RETURNING` hands the constructed row back (sql.constraints.keys,
sql.modifying.returning-upsert).

**Builder vs options.** Options configure a value once, from a known list of knobs. A
builder **accumulates**: parts added over several calls, often inside `if`s and loops.
That makes builders rare in Go and TS application code and common in one place, query
construction. A Go builder cannot return an error mid-chain, so the first error sticks
and `Build` reports it.

**Repository.** One type owns the SQL for one part of the domain and speaks domain values
(go.postgres.repository-testing). In TS the equivalent is the API client module: the
only code that knows URLs and wire shapes. Inside the database, views and functions play
the same role: a stable interface in front of tables you want to be free to change.

**DI without frameworks.** Every dependency is created in one place, the **composition
root**, and handed to whatever needs it. Go hands it over as constructor parameters, React
as props or through context. PostgreSQL resolves unqualified names through
`search_path`, which is wiring too, with a choice of binding time.

## Coming from C/C++/Java/Python/JS

- **Java**: Spring finds implementations by component scan and injects them by
  annotation. Here `main` (Go) or `main.tsx` (React) is the container, written out.
  Lombok's `@Builder` exists to escape telescoping constructors; Go's named struct
  fields and TS's options objects don't have that problem, so a builder there must earn
  its keep by accumulating.
- **Python**: keyword arguments with defaults are the TS options object. FastAPI's
  `Depends` injects through function parameters, which is close to how a React
  component receives a dependency from context through a hook.
- **C++**: a constructor always runs, and RAII ties construction to resource ownership.
  Go's `var x T` runs no code, so a type that needs set-up must be built by `New`, and a
  builder's `Build` is where invariants get checked.
- **JS/React**: you have used DI already: `QueryClientProvider` and React Router's
  `RouterProvider` deliver objects to deep components through context. Your own ports
  can travel the same way.

## How it works

PostgreSQL parses a function body written as a string constant at execution time, so
each unqualified name in it is looked up in the `search_path` of the session that calls
it. A SQL-standard body (`RETURN expr` or `BEGIN ATOMIC ... END`, PostgreSQL 14) is
parsed at `CREATE FUNCTION`: names are resolved once, stored as references to specific
objects, and tracked as dependencies. A `SET search_path` clause on the function
replaces the caller's path for the duration of the call and restores it afterwards.
The documentation's section on writing `SECURITY DEFINER` functions safely asks for that
clause, with `pg_temp` last, because a privileged function that resolves names through
the caller's path can be steered to objects the caller created.

React context works by looking up the component tree: `useContext` returns the value of
the nearest enclosing `Provider`, or the context's default when there is none. A `null`
default plus a hook that throws makes a missing provider fail loudly at the first render.

## Design notes

StackLab's backend has a textbook composition root: `run()` in `cmd/stacklab/main.go`
builds config, logger, pool, migrations, the runner (in-process or Docker, chosen by one
`if`), the router and the server, in that order. The wiring has a second, quieter root:
`httpapi.NewRouter(logger, db, runner)` builds `learningstore` and `runstore` itself from
`pool.Pool()`. The run path is injectable, so its handler can be tested with a fake
`Runner`; the learning handlers can only be tested against a real database. That is a
defensible choice for a single-database app (go.design.layers calls it flat on purpose),
and the fix if it ever pinches is mechanical: pass the stores in.

The frontend wires differently. `LessonNotes`, `LessonQuiz` and `learning/store.ts` call
`pickApi()`, a module-level factory that probes the backend once and caches the chosen
adapter in a module variable. That is a service locator: the components fetch their
dependency rather than receive it, and a test has to mock the module or intercept the
network. Providing the chosen API through context from `main.tsx`, as in the second
example, would make the choice visible in one place.

On the SQL side, prefer SQL-standard bodies for `LANGUAGE sql` functions that should
always mean the same tables, pin `search_path` on every `SECURITY DEFINER` function, and
treat a session-wide `SET search_path` as global state. The lesson runtime itself uses
it deliberately: each run's `search_path` is set to an empty schema of its own, which
is why your examples can create tables with common names without colliding.
