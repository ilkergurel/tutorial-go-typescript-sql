---
id: go.design.packages
title: "Package design: by feature, small API surface, no util packages"
track: go
module: design
order: 1
level: intermediate
minutes: 12
since: go1.16
prerequisites: [go.tooling.project-layout, go.interfaces.implicit]
compare_with: [ts.architecture.folders, ts.objects-classes.modules, inside.codebase.wiring]
tags: [packages, design, import-cycles, naming, interfaces, dependency-inversion]
summary: >
  A package is Go's unit of naming, privacy and dependency, and the compiler
  refuses import cycles. Give each package one job and a name that says it,
  group code by feature, and when two packages need each other, move the code
  or let the consumer declare an interface.
examples:
  - id: util-cycle
    file: examples/01_util_cycle.go
    title: "A util package grows an import cycle; one job per package removes it"
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "29-31"
        note: >
          util starts innocently with Slug, which needs nothing of ours.
          OrderURL is where it breaks: a "shared helper" that takes a domain
          type has to import the domain package, and that package already
          imports util. Grab-bag packages attract exactly this kind of code.
      - lines: "58"
        note: >
          The go command names the whole loop and the file behind each edge.
          The spec makes a package importing itself, directly or indirectly,
          illegal, so this fails on the first build, not at run time as a
          circular import can in Python or with ES modules.
      - lines: "39-43"
        note: >
          slug has one job and a name that says it, and imports only strings,
          so any package can use it without risking a loop. A real slug
          function has decisions to make (should ş and ı become s and i?) and
          tests to prove them: one more reason it deserves its own package.
      - lines: "53"
        note: >
          URL needs nothing but an Order, so it is a method on Order. Most
          functions in a util package belong to the package that owns their
          main argument.
      - lines: "59"
        note: >
          The same program, rearranged, builds (a nil error). The edges now
          point one way: order imports slug, and slug imports only strings.
  - id: consumer-interface
    file: examples/02_consumer_interface.go
    title: "Two packages that need each other: the consumer declares an interface"
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "10-12"
        note: >
          A genuine two-way need. No code is misplaced here, so moving a
          function can't fix it: written the direct way, with order calling
          email.Confirm(o), it fails with example 1's import cycle error. The
          fix has to change the direction of one edge.
      - lines: "19-22"
        note: >
          order states what it needs, in its own terms, with the one method it
          calls. It no longer knows email exists: a test can pass a fake, and
          an SMS sender can be added without touching this package.
      - lines: "32-36"
        note: >
          email imports order for the Order type and satisfies Notifier without
          naming it. Implicit satisfaction is what makes consumer-side
          interfaces possible; in Java, Sender would need "implements
          order.Notifier" and the dependency would point back again.
      - lines: "45-48"
        note: >
          main is the one package that knows both sides and makes the concrete
          choice. StackLab's cmd/stacklab/main.go does the same when it hands
          httpapi either a runexec.Executor or a runclient.Client.
      - lines: "62-63"
        note: >
          go list with a template prints the import graph: order imports
          nothing from the module, email depends on order, and only the root
          package depends on both. The same command in review or CI shows when
          a change adds an edge that shouldn't exist.
use_cases:
  - "A feature in a service: one package (internal/order) holding its types, rules and storage, instead of an order file in each of models/, services/ and handlers/"
  - "A helper two features share: a small package named for what it does (slug, money, clock), or a copied three-line function"
  - "Two packages that need each other: move the misplaced code, merge the packages, or declare an interface in the package that makes the call"
  - "Reviewing structure: go list -f '{{.ImportPath}} -> {{.Imports}}' ./... shows every edge of the graph"
strengths:
  - "Import cycles are a compile error, so the package graph is acyclic by construction; Java compiles cyclic packages happily and needs ArchUnit or JDepend to catch them"
  - "Every call site carries the package name (order.New, slug.Make, http.Server), so identifiers stay short without Java's OrderServiceImpl or Python's from-import ambiguity"
  - "Interfaces are satisfied implicitly, so the consumer can declare the abstraction it needs without editing the implementing package"
  - "An unused import is a compile error, so removing a dependency removes the edge for real"
weaknesses:
  - "Splitting a package later means renaming identifiers at every call site and exporting names that used to be private"
  - "There is no visibility level between package-private and exported, apart from internal/ directories; a helper two sibling packages share must be exported from a third"
  - "The cycle error shows the loop, not the fix; choosing between moving, merging and inverting is a design decision"
  - "Consumer-declared interfaces multiply: three packages may each declare a one-method interface over the same concrete type"
pitfalls:
  - title: "A util, common or helpers package"
    detail: "The name says nothing about the contents, so everything lands there, including helpers that take domain types, and the package starts importing the packages that import it. Put each function with the type it works on, or in a package named for its one job."
    bad: "url := util.OrderURL(o)"
    good: "url := o.URL() // and slug.Make(name) for the helper with a job of its own"
  - title: "Packages per layer"
    detail: "models/, services/, handlers/ spreads one feature across three packages, makes every change touch all of them, and turns models into a package everyone depends on. Group by feature; layers can live inside a feature package."
    bad: "internal/models, internal/services, internal/handlers"
    good: "internal/order, internal/customer, internal/billing"
  - title: "Stutter"
    detail: "The package name is already part of every reference, so repeating it in the identifier is noise."
    bad: "order.NewOrder(...), order.OrderService"
    good: "order.New(...), order.Service"
  - title: "Breaking a cycle with a types package"
    detail: "Moving every struct into a shared types or models package silences the compiler but keeps the coupling: now every package depends on one that changes for every feature. Move behaviour to its owner or invert the dependency instead."
  - title: "Exporting by default"
    detail: "Every exported name is a promise to every importer. Start lower-case and export when a second package needs it; put shared implementation details under internal/."
quiz:
  - q: "Package order imports util, and someone adds a util function that takes an order.Order. What happens?"
    options: ["It builds; Go resolves the cycle lazily", "go build fails with import cycle not allowed", "It builds, but initialization order becomes undefined"]
    answer: 1
    why: "The spec forbids a package importing itself directly or indirectly, and the go command reports the loop with the file behind each edge."
  - q: "order.Service needs to send notifications that package email implements. Where should the Notifier interface live?"
    options: ["In email, next to the implementation", "In order, the package that calls it", "In a shared interfaces package"]
    answer: 1
    why: "Interfaces belong with their consumer (Go Code Review Comments). email satisfies it implicitly, so order no longer imports email and the cycle is gone; a shared interfaces package becomes one more thing everyone depends on."
  - q: "Package order's main type is Order. Which constructor reads best at call sites?"
    options: ["order.NewOrder", "order.New", "order.CreateOrderInstance"]
    answer: 1
    why: "Callers always write the package name, so order.New already says what it builds; the same reasoning gives http.Server rather than http.HTTPServer."
  - q: "Why is an internal/models package holding every struct a design smell?"
    options: ["Package names may not be plural", "Every feature depends on it, it changes for every feature, and it tends to grow helpers that import feature packages", "It makes the binary larger"]
    answer: 1
    why: "Plural names are legal and binary size is unaffected; the problem is coupling. A package with no single job becomes the hub of the graph and the usual place for cycles to start."
further_reading:
  - title: "Package names (Go blog)"
    url: "https://go.dev/blog/package-names"
  - title: "Organizing a Go module"
    url: "https://go.dev/doc/modules/layout"
  - title: "Go Code Review Comments: Package Names and Interfaces"
    url: "https://go.dev/wiki/CodeReviewComments#interfaces"
  - title: "Go at Google: Language Design in the Service of Software Engineering (dependencies)"
    url: "https://go.dev/talks/2012/splash.article"
  - title: "The Go Programming Language Specification: Import declarations"
    url: "https://go.dev/ref/spec#Import_declarations"
---

## Concept

A Go package is one directory compiled as a unit, and it is three things at once:

- the unit of **naming**: callers always write `pkg.Name`, so the package name is
  half of every identifier (`order.New`, `slug.Make`, `http.Server`);
- the unit of **privacy**: lower-case names are visible in every file of the package
  and nowhere else;
- a node in the **import graph**, which the compiler requires to be acyclic.

Good package design follows from those facts. Give each package one job and name it
for what it provides. Group code by feature (`order`, `customer`, `runexec`) rather
than by kind (`models`, `services`, `utils`). Export only what other packages need.
The Go blog puts the naming rule bluntly: "Packages named `util`, `common`, or `misc`
provide clients with no sense of what the package contains."

When two packages need each other, the compiler stops you, and the fix is one of three:

1. **Move** code that is in the wrong package (a helper that takes an `Order` belongs
   with `Order`).
2. **Merge** the packages if they are really one feature.
3. **Invert** the dependency: the package that makes the call declares a small
   interface for what it needs, and the other package satisfies it without being
   imported.

## Coming from C/C++/Java/Python/JS

- **Java**: a package is a namespace, javac compiles mutually dependent packages
  together, and "package by layer" (`controller`, `service`, `repository`) with links
  in both directions is common. A Go package is closer to a small module: many types
  and files, one purpose, one-way dependencies.
- **Python**: circular imports are legal until they aren't. Depending on import order
  you get `ImportError: ... partially initialized module` at run time. Go reports
  every cycle at build time, whatever the order.
- **JS/TS**: ES modules allow cycles through live bindings and fail at run time with a
  TDZ `ReferenceError` or an `undefined` import. Barrel files make such cycles easy
  to create; Go has no barrels, because the directory is the package.
- **C/C++**: forward declarations let two headers refer to each other. Go has no
  cross-package forward declarations; a package sees only what it imports.

## How it works

The compiler builds a package from its own sources plus the compiled export data of
its direct imports, which therefore have to be built first. A cycle would have no
first package, so the spec rules it out ("It is illegal for a package to import itself,
directly or indirectly"). The same ordering drives initialization: imported packages
are initialized before the package that imports them, and `init` functions run in
that order. An acyclic graph is also what lets the go command build independent
packages in parallel and cache each one by content.

## Design notes

StackLab's backend is the case study. ADR-0007 puts all Go code in one module, with
`server/internal/` split **by feature**: `gorun` and `sqlrun` run lessons, `runexec`
dispatches to them, `runclient` talks to the Docker runner, `learningstore` and
`runstore` own their SQL, `httpapi` owns HTTP. The layered alternative
(`handlers/`, `services/`, `models/`) was rejected because it scatters one feature
across three directories. `platform` is the closest thing to a shared package, and it
works because its direction is fixed: features import it, it imports no feature.

`httpapi` declares the `Runner` interface it calls. `runexec.Executor` and
`runclient.Client` both satisfy it without importing `httpapi` or each other, and
`cmd/stacklab/main.go`, the only package that knows all three, picks one. That is the
inversion from the second example at production scale.

Start with fewer, larger packages and split when a part has its own vocabulary, its
own reason to change and a small surface towards the rest. A package boundary is an
API: everything that crosses it must be exported, documented and kept stable, so a
split that needs twenty exported names in both directions is in the wrong place.
