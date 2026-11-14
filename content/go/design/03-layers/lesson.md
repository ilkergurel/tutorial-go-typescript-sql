---
id: go.design.layers
title: "Layered vs hexagonal: handler/service/store trade-offs"
track: go
module: design
order: 3
level: advanced
minutes: 14
since: go1.25
prerequisites: [go.design.constructors, go.http.handlers, go.postgres.repository-testing]
compare_with: [design.architecture.styles, ts.architecture.patterns, sql.transactions.row-locks, inside.codebase.http-layer]
tags: [architecture, layers, hexagonal, ports-and-adapters, services, transactions]
summary: >
  A handler that calls a store is a fine design until rules, a second entry
  point or tests without a database appear. Then a service with ports declared
  by the core pays for itself, and its ports must keep each invariant in one
  call.
examples:
  - id: flat
    file: examples/01_flat.go
    title: "Flat: the handler talks to the store, and that's enough"
    runtime: go
    check: exact
    annotations:
      - lines: "15-33"
        note: >
          A concrete store with no interface in front of it. With one
          implementation and tests that run against a real database, an
          interface would only add a file to read.
      - lines: "38-41"
        note: >
          The handler decodes, calls the store, and encodes. There is no rule
          to put anywhere else, so a service layer here would be a
          pass-through: three files for one map write.
      - lines: "53-54"
        note: >
          PUT is idempotent here: the second request changes nothing. When the
          domain is this thin, the database (a primary key, ON CONFLICT DO
          NOTHING) holds the rules and the Go code stays flat.
  - id: ports
    file: examples/02_ports.go
    title: "Rules and two entry points: a core with a port, adapters around it"
    runtime: go
    check: exact
    annotations:
      - lines: "12-23"
        note: >
          The core owns its errors and declares the port it needs. It imports
          no net/http and no pgx, so it reads as the feature's rules and runs
          in a test with a slice for storage.
      - lines: "27-39"
        note: >
          The rules: a well-formed id, at most maxBookmarks bookmarks. They
          hold no matter who calls, which is the reason this layer exists.
          Note the shape: count, then add. The next example shows what that
          costs.
      - lines: "47-61"
        note: >
          The HTTP adapter translates: path value in, status code out. Domain
          errors become statuses in this one switch; the core never mentions
          HTTP. (StackLab's httpapi likewise sends every error response
          through one helper, problem.)
      - lines: "66-69"
        note: >
          A second driving adapter. An import job, a CLI command or a queue
          consumer gets the same rules for free. Without the core, each would
          copy the validation, and copies drift.
      - lines: "64"
        note: >
          Wiring stays in main: the storage adapter is built here and handed
          to the core, and the core to the router. Swapping memStore for a pgx
          store changes this line and nothing else.
  - id: atomic-port
    file: examples/03_atomic_port.go
    title: "A port shaped like CRUD splits an invariant; shape it like the use case"
    runtime: go
    check: exact
    annotations:
      - lines: "19-26"
        note: >
          The gate makes both requests count before either adds, the
          interleaving that two concurrent HTTP requests can produce. It is
          only there so the race happens on every run instead of
          occasionally.
      - lines: "47-56"
        note: >
          The service's rule spans two store calls. Each call is safe on its
          own, but nothing makes the pair atomic, so two requests both see 2
          and both add.
      - lines: "36-44"
        note: >
          The fix is in the port's shape, not in the service: AddWithin does
          the check and the insert as one operation. In PostgreSQL that means
          one transaction that locks the user's row first (SELECT ... FOR
          UPDATE) or SERIALIZABLE with a retry; a count inside the INSERT
          statement is still racy under READ COMMITTED.
      - lines: "73-74"
        note: >
          Same requests, same starting data: four bookmarks with the split
          port, three plus one rejection with the atomic one. Which request
          wins varies between runs; the totals don't.
use_cases:
  - "Flat (handler -> store): CRUD endpoints, internal tools, one entry point, rules enforced by database constraints"
  - "Layered (handler -> service -> store): business rules that span several store calls, transactions, more than one entry point"
  - "Ports and adapters: a core tested without a database, several drivers (HTTP, CLI, queue consumer), storage that may change"
  - "Mixed: most features flat, the two or three with real rules get a service; consistency within a feature matters more than across them"
strengths:
  - "Consumer-declared interfaces make ports nearly free in Go: the core declares Store, the pgx package satisfies it without importing the core, unlike Java's ports, which the adapter must implement by name"
  - "One place maps domain errors to HTTP statuses, so a new error can't leak as a 500 from one handler and a 400 from another"
  - "Rules in a core run in microseconds in tests, with no database or HTTP server"
  - "Package boundaries enforce the layering: a core package that imports nothing from net/http or pgx can't call them by accident, unlike a folder convention in a JS project"
weaknesses:
  - "Every layer adds types, constructors and mapping code; for CRUD that is cost without benefit"
  - "Ports shaped like tables (Get, Insert, Count) push invariants into services, where concurrent requests break them"
  - "Transactions don't fit neatly: a service that needs two store calls in one transaction needs a store method for the use case or a way to pass a transaction down"
  - "Mapping between transport, domain and storage structs multiplies when each layer gets its own type for the same data"
pitfalls:
  - title: "A pass-through service"
    detail: "A service whose methods only call the store with the same arguments is a layer of indirection with no rule in it. Call the store from the handler until a rule appears."
    bad: "func (s *Service) Get(ctx context.Context, id string) (Note, error) { return s.store.Get(ctx, id) }"
    good: "note, err := h.store.Get(r.Context(), id) // in the handler, until there is a rule"
  - title: "Check-then-act across two port calls"
    detail: "Count then Insert, Get then Update: each call is correct, the pair isn't. Give the port a method per use case that runs in one transaction, or enforce the invariant with a constraint."
    bad: "n, _ := store.Count(ctx); if n < limit { store.Add(ctx, id) }"
    good: "err := store.AddWithin(ctx, id, limit) // one transaction inside the store"
  - title: "Driver types leaking into the core"
    detail: "A core that checks pgx.ErrNoRows or *pgconn.PgError is coupled to the driver. The store translates them into the core's own errors (ErrNotFound, ErrDuplicate)."
  - title: "An interface for every struct"
    detail: "Interfaces belong where there is a second implementation or a test that needs a substitute. StackLab's stores are concrete; only httpapi.Runner, with two implementations, is an interface."
  - title: "Layers as packages named after layers"
    detail: "handlers/, services/, repositories/ packages recreate the layer cake across the whole app and invite import cycles. Keep the layers inside each feature package, or one package per feature with sub-packages only when it grows."
quiz:
  - q: "A feature has three CRUD endpoints, one entry point, and its only rule is a unique constraint in the database. Which shape fits?"
    options: ["Handler -> service -> repository with interfaces", "Handlers calling a concrete store directly", "A domain core with ports for storage and HTTP"]
    answer: 1
    why: "There is no rule for a service to hold and only one implementation of storage. Add layers when rules, a second entry point or substitutes appear."
  - q: "A service enforces 'at most 3 bookmarks' with store.Count followed by store.Add. What goes wrong under load?"
    options: ["Nothing: each store method locks", "Two concurrent requests can both see 2 and both add, ending with 4", "The second request deadlocks"]
    answer: 1
    why: "Each call is atomic, the pair isn't. Make check and insert one store operation in one transaction, or enforce the limit in the database."
  - q: "In ports and adapters, who declares the Store interface?"
    options: ["The storage adapter, next to the SQL", "The core that uses it", "A shared interfaces package"]
    answer: 1
    why: "Ports belong to the core; adapters satisfy them. In Go this is the ordinary consumer-declared interface, satisfied implicitly."
  - q: "Where should a domain error such as ErrLimit become HTTP 409?"
    options: ["In the core, by returning the status code", "In the HTTP adapter, in one mapping function or switch", "In the store"]
    answer: 1
    why: "The core stays transport-agnostic, so the CLI and the import job can use it; one mapping point keeps statuses consistent."
further_reading:
  - title: "Go Code Review Comments: Interfaces"
    url: "https://go.dev/wiki/CodeReviewComments#interfaces"
  - title: "Organizing a Go module (server project layout)"
    url: "https://go.dev/doc/modules/layout"
  - title: "PostgreSQL: Explicit locking (row-level locks)"
    url: "https://www.postgresql.org/docs/current/explicit-locking.html"
  - title: "PostgreSQL: Transaction isolation"
    url: "https://www.postgresql.org/docs/current/transaction-iso.html"
---

## Concept

Three shapes cover most Go services:

- **Flat**: HTTP handlers call a store (or run SQL) directly. Right for CRUD, admin
  tools and anything whose rules live in database constraints.
- **Layered**: a transport layer (decode, validate shape, map errors to statuses), a
  service layer (business rules, transactions, orchestration) and a storage layer
  (SQL). Each layer calls only the one below.
- **Ports and adapters** (hexagonal, Alistair Cockburn's term): a core of rules
  declares *ports*, interfaces for what it needs from the outside (storage, mail,
  clocks). *Adapters* implement them (a pgx store) or drive the core (an HTTP
  handler, a CLI, a queue consumer, a test). Dependencies point inward: the core
  imports neither `net/http` nor `pgx`.

In Go the step from layered to hexagonal is small. If the service declares the `Store`
interface it uses and the pgx package satisfies it, you already have a port and an
adapter. The real decisions are how many layers a feature gets, and what shape the
ports have.

Add structure when you can name what it buys: rules that aren't just validation, a
second entry point, tests that must run without a database, a transaction across
several statements. Without one of those, a layer is a pass-through, and a
pass-through is cost.

## Coming from C/C++/Java/Python/JS

- **Java/Spring**: `Controller -> Service -> Repository`, each with an interface and an
  `Impl`, is the default from the first endpoint. Go code usually starts flat, adds a
  service when a rule appears, and declares an interface only where a consumer needs
  a substitute.
- **Python**: Django puts rules on models (active record); FastAPI and Flask apps often
  stay flat with SQLAlchemy in the route. Both map directly onto the shapes above.
- **JS/TS**: NestJS enforces modules, controllers and providers; Express leaves it to
  you. Unlike a folder convention, a Go package boundary is enforced by the compiler:
  a core package without a `pgx` import cannot call pgx.
- **"Clean architecture"** adds more rings (entities, use cases, interface adapters),
  each with its own types. In Go that usually means mapping code nobody needs; take
  the dependency rule and skip the ceremony.

## How it works

```mermaid
flowchart LR
  http["HTTP handler"] --> core["Bookmarks (rules)"]
  job["import job / CLI"] --> core
  test["test"] --> core
  core -->|"Store port"| pg["pgx store"]
  core -->|"Store port"| mem["memory store"]
```

Arrows are calls; the Go imports point the same way except at the ports: the pgx
store imports the core (for its types and errors), the core imports nothing from
it. `main` is the only package that sees both ends and connects them.

A port's shape decides where invariants live. A port mirroring a table (`Get`,
`Insert`, `Count`) leaves every rule that touches two rows to the service, which then
runs several statements without a shared transaction, as the third example shows. A
port shaped like the use case (`AddWithin`, `ReserveSeat`, `TransferFunds`) lets the
adapter run the whole thing in one transaction with the right locks, and keeps the
core free of transaction handles.

## Design notes

StackLab's backend is mostly flat, on purpose. `httpapi` handlers call
`*learningstore.Store` and `*runstore.Store` directly: one database, no rule beyond
what the schema enforces, nothing to swap. The run path is the
exception, and it is exactly where a port pays: `httpapi.Runner` has two adapters,
`runexec` in local mode and `runclient` in Docker mode, chosen in `main`. Every error
response leaves through one function, `httpapi`'s `problem` helper, so the error shape
stays uniform.

That mix is the recommendation. Choose a shape per feature, for the program you have:
flat until a rule, a second driver or a test without a database asks for more; a
service with use-case-shaped ports when it does. Moving from flat to layered later is
a mechanical refactor in Go (extract the rule, declare the interface where it is
called); undoing premature layers is harder, because every caller has learned them.
