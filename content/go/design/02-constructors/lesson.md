---
id: go.design.constructors
title: "Constructor injection, functional options, useful zero values"
track: go
module: design
order: 2
level: intermediate
minutes: 13
since: go1.25
prerequisites: [go.composite.structs, go.interfaces.implicit]
compare_with: [design.patterns.constructors, ts.objects-classes.classes, go.testing.fakes-httptest]
tags: [constructors, dependency-injection, functional-options, zero-value, configuration, cmp]
summary: >
  Go has no constructors, only functions named New. Make the zero value usable
  where you can, pass every dependency to New explicitly and wire them in main,
  and pick a Config struct or functional options for optional settings.
examples:
  - id: zero-values
    file: examples/01_zero_values.go
    title: "Zero values that work, and zero values that only look safe"
    runtime: go
    check: exact
    annotations:
      - lines: "12-15"
        note: >
          Neither field needs setup: a zero sync.Mutex is unlocked, and a nil
          map reads as empty. That makes var t Tally a complete, valid value,
          the way var mu sync.Mutex is.
      - lines: "20-22"
        note: >
          Writing to a nil map panics, so the first write creates it. This lazy
          initialization is the usual price of a useful zero value, and the
          mutex makes it safe under concurrent use.
      - lines: "33-36"
        note: >
          The property composes: Report holds a Tally, so the zero Report is
          ready too. Don't copy a Tally once it is in use: the copy shares the
          map but gets its own lock (go vet's copylocks check reports this).
      - lines: "44-48"
        note: >
          bytes.Buffer, strings.Builder and sync.WaitGroup have no
          constructors either. wg.Go (Go 1.25) adds the goroutine to the group
          and starts it, replacing the Add(1)/go/defer Done() pattern.
      - lines: "51-55"
        note: >
          A zero http.Client never times out, and a zero http.Server sets no
          deadline for reading headers. Usable, but wrong for production;
          StackLab's main.go sets every server timeout for this reason.
  - id: injection
    file: examples/02_injection.go
    title: "Constructor injection: the store and the clock are parameters"
    runtime: go
    check: exact
    annotations:
      - lines: "16-19"
        note: >
          The interface sits next to its consumer and has one method, the one
          Notes calls. A pgx-backed store, an in-memory map and a failing fake
          all satisfy it without importing this package.
      - lines: "22-29"
        note: >
          Every dependency is a parameter of NewNotes, so the signature is the
          full list of what Notes needs. No package-level variables, no
          container, no reflection: wiring is ordinary code in main.
      - lines: "24"
        note: >
          Time is a dependency too. A func() time.Time field lets tests freeze
          the clock, which is why the output below is deterministic; production
          passes time.Now.
      - lines: "39-41"
        note: >
          Wrap with the context the caller can't see (which lesson) and %w, so
          errors.Is and errors.As still reach the store's error.
      - lines: "68"
        note: >
          A failing dependency is one line, with no mocking library: write a
          type with the method. That is the payoff of small consumer-declared
          interfaces.
  - id: options
    file: examples/03_options.go
    title: "Functional options vs a Config struct, and the explicit-zero problem"
    runtime: go
    check: exact
    annotations:
      - lines: "20-24"
        note: >
          An Option is a function that modifies the value under construction.
          Adding WithProxy later adds a function; no existing call breaks, which
          is why libraries such as grpc-go take ...Option.
      - lines: "27-33"
        note: >
          Defaults are set first, options applied after, in the order given.
          Options can also validate: make them func(*Client) error and have New
          return (*Client, error).
      - lines: "36-47"
        note: >
          The Config struct is the simpler design and the more common one in
          the standard library (tls.Config, slog.HandlerOptions). cmp.Or
          (Go 1.22) returns its first non-zero argument: default-if-zero in one
          expression.
      - lines: "53"
        note: >
          With options, WithRetries(0) says "zero retries" and gets it.
      - lines: "56"
        note: >
          With a Config, Retries: 0 is indistinguishable from "not set", so the
          caller silently gets 2. Fix it with a *int field, a documented
          negative sentinel, or options.
use_cases:
  - "Types whose zero value works: counters, buffers, caches with lazily created maps, anything guarded by a sync.Mutex field"
  - "Services: NewService(store Store, now func() time.Time, log *slog.Logger), constructed once in main and passed down"
  - "Application settings: one Config struct filled from the environment in one function, as StackLab's config.Load does"
  - "Library clients with many optional, evolving settings: New(addr, WithTimeout(d), WithRetries(n))"
strengths:
  - "Dependencies are ordinary parameters, so main.go reads as the dependency graph; Spring and Guice build the same graph from annotations at run time"
  - "A usable zero value removes 'forgot to initialize' bugs: var mu sync.Mutex is ready, where C needs pthread_mutex_init and Java a constructor call"
  - "Functional options let a library add settings without breaking callers, the job Java does with builders and overloaded constructors"
  - "Fakes are plain types with one method, so tests need no mocking framework"
weaknesses:
  - "No default parameter values or keyword arguments: optional settings need a Config struct or options, more ceremony than Python's def connect(timeout=10)"
  - "A struct literal bypasses the constructor: Notes{} compiles and panics on first use with a nil store"
  - "A Config struct can't tell 'unset' from an explicit zero"
  - "Functional options are harder to discover in go doc and impossible to print: the settings live inside closures"
pitfalls:
  - title: "Dependencies in package-level variables"
    detail: "A global pool set in init() hides the dependency from every signature, makes tests share state, and ties startup order to import order. Create it in main and pass it in."
    bad: "var pool *pgxpool.Pool // set in init()"
    good: "func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }"
  - title: "Zero meaning both 'unset' and 'zero'"
    detail: "Config{Retries: 0} gets the default when the caller meant none. Use options, a pointer field, or a documented sentinel such as -1 for 'none'."
  - title: "Copying a struct that contains a sync.Mutex"
    detail: "Passing Tally by value copies the lock and the map header: both copies now point at one map, guarded by two unrelated locks. Use pointer receivers and pass *Tally; go vet's copylocks analyzer reports the copy."
    bad: "func show(t Tally) { fmt.Println(t.Count(\"go\")) }"
    good: "func show(t *Tally) { fmt.Println(t.Count(\"go\")) }"
  - title: "Constructors that return interfaces by reflex"
    detail: "Return the concrete type and let each consumer declare the interface it needs. Returning an interface hides methods and forces callers to type-assert; Effective Go reserves it for types that exist only to implement one interface, like crc32.NewIEEE returning hash.Hash32."
    bad: "func NewNotes(s NoteStore) NoteService"
    good: "func NewNotes(s NoteStore) *Notes"
  - title: "Trusting the zero value of http.Client and http.Server"
    detail: "Both mean 'no timeout'. Set Client.Timeout (or per-request contexts) and the server's ReadHeaderTimeout, ReadTimeout, WriteTimeout and IdleTimeout."
quiz:
  - q: "Which zero value is not ready to use for its main purpose?"
    options: ["sync.Mutex", "bytes.Buffer", "map[string]int, for writing", "strings.Builder"]
    answer: 2
    why: "Reading a nil map returns zero values, but assigning to one panics. The other three are documented as ready to use when zero."
  - q: "A constructor computes retries := cmp.Or(cfg.Retries, 2). What does Config{Retries: 0} produce?"
    options: ["0 retries", "2 retries", "A compile error"]
    answer: 1
    why: "cmp.Or returns the first argument that isn't the zero value, so an explicit 0 is treated as 'not set'."
  - q: "Where should the database pool be created in a Go service?"
    options: ["In an init() function of the store package", "In main, or a function main calls, and passed to constructors", "Lazily inside each HTTP handler"]
    answer: 1
    why: "Creating it at the top and passing it down keeps every dependency visible in signatures, lets main decide what to build (real pool or fake), and lets tests construct their own."
  - q: "A zero-value http.Client calls a server that accepts the connection and never answers. What happens?"
    options: ["It fails after a 30-second default timeout", "It waits until the request's context is cancelled or the connection drops, which may be never", "It fails immediately"]
    answer: 1
    why: "Client.Timeout zero means no timeout. Set it, or give each request a context with a deadline."
further_reading:
  - title: "Effective Go: Allocation with new (useful zero values)"
    url: "https://go.dev/doc/effective_go#allocation_new"
  - title: "Effective Go: Constructors and composite literals"
    url: "https://go.dev/doc/effective_go#composite_literals"
  - title: "package cmp: Or"
    url: "https://pkg.go.dev/cmp#Or"
  - title: "package net/http: Client"
    url: "https://pkg.go.dev/net/http#Client"
  - title: "Go Code Review Comments: Interfaces"
    url: "https://go.dev/wiki/CodeReviewComments#interfaces"
---

## Concept

Go has no constructor syntax. A constructor is a function, by convention `New` or
`NewThing`, that returns a ready value (usually a pointer) and sometimes an error.
Three habits decide how construction looks in Go code:

- **Make the zero value useful.** If `var x T` can be a valid `T`, callers need no
  constructor at all. `sync.Mutex`, `bytes.Buffer`, `strings.Builder` and
  `sync.WaitGroup` work this way; your types can too, by treating zero fields as
  "empty" or "default" and creating maps lazily.
- **Inject dependencies through the constructor.** When a type needs a store, a
  clock, a logger or an HTTP client, `New` takes them as parameters, typed as small
  interfaces the type itself declares. `main` builds the concrete things once and
  passes them down. There is no container and no annotation; the wiring is code.
- **Choose a shape for optional settings.** Required dependencies are positional
  parameters. Optional settings go in a `Config` struct whose zero fields mean
  "default", or in functional options (`New(addr, WithTimeout(d))`) when there are
  many of them, they evolve, or an explicit zero must be expressible.

When the zero value can't be valid (a dependency is required, a goroutine must start,
input must be checked), write `New` and return `(*T, error)` if construction can fail.

## Coming from C/C++/Java/Python/JS

- **Java**: no overloading, so telescoping constructors and builders turn into a
  `Config` struct or options. No Spring or Guice: `main` is the composition root,
  written out by hand, and an unused dependency is visible in a signature rather
  than hidden in an annotation.
- **Python**: no default arguments or keyword arguments. A struct literal with named
  fields, `Config{Timeout: 45 * time.Second}`, is the closest thing to
  `Client(timeout=45)`, and omitted fields are zero.
- **JS/TS**: an options object with `??` defaults is a `Config` struct with `cmp.Or`,
  with one difference: `??` distinguishes `undefined` from `0`, `cmp.Or` does not.
- **C++**: a constructor always runs. In Go, `var x T` and `T{}` never run code, so
  either the zero value is valid or callers must be told to use `New`. Unexported
  fields stop other packages from setting them, but not from writing `T{}`.

## Design notes

StackLab follows these rules throughout `server/`. `config.Load` fills one `Config`
struct from the environment and applies every default in one place.
`runexec.New(playgroundDSN, nodeProject)`, `runclient.New(baseURL)` and
`httpapi.NewRouter(logger, db, runner)` take their dependencies as plain parameters,
and `cmd/stacklab/main.go` constructs everything in order, so reading `run()` from
top to bottom is reading the dependency graph. CLAUDE.md states it as a convention:
constructor injection, interfaces declared by the consumer, no globals, no DI framework.

StackLab uses no functional options, and that is a choice, not an omission: each
constructor has two or three parameters, all required. Options earn their keep in
libraries with many optional settings and callers you can't update, which is where
you meet them (grpc-go's dial options, for instance). A reasonable default order is:
positional parameters for what is required, a `Config` struct for a handful of
optional settings, options when the list is long or keeps growing. Use one of the
two per constructor, not both.
