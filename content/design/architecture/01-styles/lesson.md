---
id: design.architecture.styles
title: "Layered, hexagonal/clean, modular monolith vs microservices; REST API design"
track: design
module: architecture
order: 1
level: advanced
minutes: 15
since: go1.23
prerequisites: [go.design.layers, ts.architecture.folders, go.http.handlers]
compare_with: [go.design.packages, go.services.grpc-brokers, go.services.retries, sql.transactions.row-locks, inside.codebase.at-10x]
tags: [architecture, modular-monolith, microservices, rest, http, etag, optimistic-locking]
summary: >
  Layers and ports organize code inside a deployable; the number of deployables is a
  separate choice. A module boundary that is a function call is cheaper than one that is
  a network call, which is why a solo developer starts with a modular monolith and a
  REST contract that states its concurrency rules.
examples:
  - id: boundary
    file: examples/01_boundary.go
    title: "One module boundary, as a function call and as an HTTP call"
    runtime: go-server
    check: exact
    annotations:
      - lines: "15-18"
        note: >
          The module boundary is a small interface declared by the caller
          (go.design.packages). Everything below implements it twice; placeOrders never
          changes. That is what keeps extracting a service later a wiring change.
      - lines: "20-31"
        note: >
          In-process, billing is a struct behind a mutex. The call either returns or
          panics; there is no third outcome, and in a real app it could join the
          caller's database transaction.
      - lines: "33-45"
        note: >
          The same method over HTTP gains a timeout, a request, a response body to close
          and new errors. Each is an obligation the in-process version never had
          (go.services.client).
      - lines: "53-54"
        note: >
          The new outcome a network adds: the call failed, and the caller cannot know
          whether the effect happened. Retrying risks a double invoice; not retrying
          risks none. Only an idempotency key settles it (go.services.retries).
      - lines: "66-76"
        note: >
          The billing service finished order 2 after the client gave up, so its ledger
          holds all three orders while the orders side saw a failure. Microservices turn
          this into a normal Tuesday: outboxes, idempotency and reconciliation jobs exist
          to repair it.
  - id: conditional-put
    file: examples/02_conditional_put.go
    title: "REST: ETag and If-Match turn a lost update into 412"
    runtime: go-server
    check: exact
    annotations:
      - lines: "13-18"
        note: >
          The version counts the note's states. Sent as a strong ETag ("1"), it
          lets a client say which state its edit is based on.
      - lines: "29-30"
        note: >
          The check and the write must be one step, or two PUTs can both pass
          the check. The mutex is held for one request only, never while a
          user edits; in PostgreSQL the same step is one UPDATE (example 3).
      - lines: "32-33"
        note: >
          428 Precondition Required (RFC 6585) makes the condition mandatory: a
          client that never read the note cannot blindly overwrite it.
      - lines: "34"
        note: >
          If-Match uses the strong comparison, so a weak ETag (W/"1") never
          matches. Simplified: a full parser also accepts a list of ETags and
          If-Match: *.
      - lines: "39-40"
        note: >
          A stale version is a false precondition: 412 Precondition Failed, per
          RFC 9110, and the method is not performed.
      - lines: "61-67"
        note: >
          Two tabs read version 1. A saves first; B's write would have silently
          replaced A's text, and gets 412 instead. B must re-read (version 2)
          and merge before its PUT succeeds.
  - id: version-column
    file: examples/03_version_column.sql
    title: "The version column: two concurrent UPDATEs, one winner"
    runtime: sql-multisession
    setup: |
      CREATE TABLE notes (
          lesson_id text PRIMARY KEY,
          body      text NOT NULL,
          version   int  NOT NULL DEFAULT 1
      );
      INSERT INTO notes VALUES ('go.basics.defer', 'defer runs LIFO');
    check: exact
    annotations:
      - lines: "6-7"
        note: >
          The compare-and-set in one statement: the row changes only if its
          version is still the one the client read, and the version moves
          forward in the same UPDATE. This is optimistic locking, JPA's
          @Version without the framework.
      - lines: "8-9"
        note: >
          B's UPDATE needs the row A has locked, so it waits.
      - lines: "10-11"
        note: >
          Once A commits, B's blocked UPDATE re-checks its WHERE clause against
          the newly committed row (READ COMMITTED), finds version 2 and changes
          nothing: UPDATE 0, no version returned. The handler turns zero rows
          into 412.
      - lines: "13"
        note: >
          A's text survived; B's was never written over it.
use_cases:
  - "Modular monolith: a product built by one person or one team, where modules need clean boundaries but not separate deploys"
  - "Extracting one service: when a module needs a different security boundary, runtime, scaling profile or release cadence (StackLab's sandboxed runner)"
  - "Microservices: many teams that must deploy independently, where coordination cost exceeds the cost of distributed systems"
  - "ETag + If-Match: any resource two clients (or two tabs) can edit: notes, settings, documents, admin forms"
strengths:
  - "A modular monolith keeps calls in-process: no partial failure, one transaction across modules, and a refactor across a boundary is one commit and one deploy"
  - "Go enforces module boundaries at compile time (internal/, acyclic imports, unexported names), so a monolith doesn't decay into a ball of mud as easily as in languages without package privacy"
  - "HTTP already defines the concurrency contract: ETag, If-Match, 412 and 428 are understood by clients, proxies and tools, unlike a custom version field in a JSON body"
  - "Optimistic locking needs no held locks: no transaction stays open while a user edits, unlike SELECT ... FOR UPDATE"
weaknesses:
  - "A monolith scales and deploys as one unit: a memory-hungry module, or one that must be isolated, takes the others with it"
  - "Module boundaries inside one process are a discipline; nothing stops a shortcut SQL join into another module's tables except review and lint"
  - "Microservices multiply operational work: per-service pipelines, versioned contracts, tracing across hops, and data that can no longer be joined or committed together"
  - "Optimistic locking under heavy contention fails often; hot rows need queuing or pessimistic locks (sql.transactions.row-locks)"
pitfalls:
  - title: "Microservices to get modularity"
    detail: "Splitting a codebase without clean module boundaries produces a distributed monolith: services that must deploy together and call each other synchronously, with all the network failure modes and none of the independence. Get the boundaries right in one process first."
  - title: "Sharing tables across modules"
    detail: "If billing reads orders' tables directly, the boundary is gone, whatever the packages say. Give each module its own tables (one schema per module works well in PostgreSQL) and cross the boundary through the module's API or a view it publishes."
  - title: "PUT without a precondition"
    detail: "Last write wins silently. For anything two clients can edit, return an ETag and require If-Match; answer 412 for a stale version and 428 for a missing one."
  - title: "Verbs in REST paths for everything"
    detail: "/api/createNote, /api/getNotes give up method semantics (GET is safe, PUT and DELETE are idempotent) that clients and proxies rely on. Model state as resources; keep RPC-style POST for real actions such as running code."
quiz:
  - q: "A remote call timed out. What do you know about its effect?"
    options: ["It did not happen", "It happened", "Nothing: it may or may not have happened", "It will be rolled back automatically"]
    answer: 2
    why: "The billing service invoiced order 2 after the client gave up. A timeout reports that the answer didn't arrive, not that the work didn't happen."
  - q: "What is the main reason microservices exist?"
    options: ["They are faster than in-process calls", "They let many teams deploy and scale parts independently", "They remove the need for transactions", "They make code easier to refactor"]
    answer: 1
    why: "They trade in-process simplicity for independent deployment, scaling and ownership. That trade pays for organizations with many teams, rarely for one developer."
  - q: "Tab B sends PUT with If-Match: \"1\" but the note is at version 2. Which status should it get?"
    options: ["409 Conflict", "412 Precondition Failed", "428 Precondition Required", "200 OK, last write wins"]
    answer: 1
    why: "RFC 9110: when an If-Match condition evaluates to false, the server must not perform the method and answers 412. 428 is for a request that carries no precondition at all."
  - q: "Why is UPDATE ... WHERE version = $3 RETURNING version safe against two concurrent PUTs with the same version?"
    options: ["PostgreSQL serializes all updates", "The second UPDATE waits for the first's row lock, then re-checks the WHERE clause on the new row version and matches nothing", "pgx retries the loser", "It isn't safe without SERIALIZABLE"]
    answer: 1
    why: "Under READ COMMITTED a blocked UPDATE re-evaluates its WHERE clause against the committed row; the version has moved, so it updates zero rows and the handler answers 412."
further_reading:
  - title: "MDN: HTTP conditional requests"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/Conditional_requests"
  - title: "MDN: If-Match"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/If-Match"
  - title: "MDN: 428 Precondition Required"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Status/428"
  - title: "Go: Organizing a Go module (internal/ and cmd/)"
    url: "https://go.dev/doc/modules/layout"
  - title: "PostgreSQL: Concurrency control (READ COMMITTED and UPDATE re-checks)"
    url: "https://www.postgresql.org/docs/current/transaction-iso.html#XACT-READ-COMMITTED"
---

## Concept

Three different decisions hide under "architecture":

1. **How code is organized inside one deployable.** Layered (transport, service,
   storage) or ports and adapters (a core that declares interfaces, adapters that
   implement them). go.design.layers covers this for Go and ts.architecture.folders for
   the frontend; this lesson takes it as known.
2. **How many deployables there are.** A **modular monolith** is one process, one deploy
   and usually one database, divided into modules with explicit public surfaces: Go
   packages with consumer-declared interfaces, `internal/`, each module owning its own
   tables. **Microservices** put modules in separate processes that deploy, scale and
   fail independently and talk over the network.
3. **What contract clients see.** For HTTP APIs, usually REST: resources named by URLs,
   uniform methods with defined semantics, status codes, and headers for caching and
   concurrency.

The second decision is the expensive one. Moving a boundary from a function call to a
network call adds failure modes that did not exist before: timeouts with unknown
outcomes, half-finished work on the other side, no shared transaction, contracts that
must be versioned, and traces that span processes. Microservices buy **independent
deployment and ownership** with that cost, which is why they fit organizations of many
teams (Conway's law: systems mirror the communication structure of the organization that
builds them). A solo developer has one team and one release train, so the benefit is near
zero and the cost is the same. Start with a modular monolith, keep the module boundaries
clean, and extract a service when a module needs something the process cannot give it:
a security boundary, a different runtime, separate scaling.

REST design follows from HTTP's own rules (RFC 9110). GET, HEAD and OPTIONS are **safe**;
PUT and DELETE are **idempotent**; POST is neither. Use `201 Created` with `Location` for
creation, `204` for success without a body, `404`, `409`, `412`, `422` for what they
mean, and one error format (problem details, RFC 9457) for all of them. Paginate with
cursors (go.services.pagination-streaming), evolve by adding fields, and protect
editable resources with **conditional requests**: an `ETag` on read, `If-Match` on
write. Not everything is a resource: an action such as "run this code" is an honest
`POST`.

## Coming from C/C++/Java/Python/JS

- **Java**: Spring Boot plus Spring Cloud made microservices the enterprise default for
  a decade; Spring Modulith is the same ecosystem's answer for modular monoliths. JPA's
  `@Version` is the optimistic locking the third example writes by hand, and its
  `OptimisticLockException` maps naturally to 412.
- **Python**: a Django project with several apps is a modular monolith; the discipline
  is to keep apps from importing each other's models freely.
- **JS/Node**: NestJS modules and Nx monorepos give module boundaries within one
  deployable; Express apps often have none, and splitting them into services is used
  to get boundaries the code never had.
- **C/C++**: a monolith is one binary linking several libraries; the boundary is a
  header. The difference with Go is that Go enforces package boundaries (and acyclic
  imports) in the compiler.

## How it works

The first example's numbers tell the story. In-process, three calls give three invoices
and three successes. Over HTTP, the second call takes longer than the client's 100 ms
budget, the client reports a timeout, and the server, unaware, finishes the work. The
two sides now disagree, and only extra machinery reconciles them: an idempotency key so
the client can safely retry (go.services.retries), an outbox so events are not lost
(design.patterns.classic), and a reconciliation job for what still slips through.

Conditional requests push the same problem, concurrent writers, into the protocol. The
server sends the version it has as an `ETag`. A client that wants to replace the resource
sends `If-Match` with the version it read. The server must evaluate the condition and
the write atomically: example 2 holds a mutex for the request, and with PostgreSQL it is
one `UPDATE ... WHERE version = $n` (example 3). Under READ COMMITTED, a second UPDATE
blocked on the same row re-checks its `WHERE` clause against the newly committed row
once the first commits, finds a different version and changes nothing. Zero rows means
the version moved or the note doesn't exist; both make `If-Match` false, so both become
412, and the client re-reads, merges and tries again.

## Design notes

StackLab is a modular monolith, with one deliberate extraction that was later folded
back. ADR-0007 puts all Go code in one module, split by feature under `server/internal/`;
the only interface is `httpapi.Runner`, declared by its consumer. The runner container
of Docker mode (ADR-0003) is the one service, and it exists for a security boundary, not
for scale: lesson code runs as a non-root user with memory, pid and network limits. When
local mode arrived, ADR-0008 kept the module (`runexec`) and the port, and made the
deployment boundary optional: one `if` in `main.go` picks `runexec` in-process or
`runclient` over HTTP. That is the modular monolith's promise working in reverse, and
ADR-0008 is explicit about the price: in local mode lesson code runs with the owner's
privileges.

The HTTP API is mostly resource-shaped: `PUT /api/notes/{lessonId}` replaces a note,
bookmarks have `GET`, `POST` and `DELETE`, and every error leaves through one `problem`
helper as RFC 9457 problem details. Two places are worth a critical look. Progress uses
verb sub-resources (`POST /api/progress/{lessonId}/viewed`), which is RPC in REST
clothing; the operations are idempotent (a status only moves up), and a `PUT` with the
status in the body would say so to every client. And notes carry no ETag, so two
browser tabs editing one note lose an update silently. For a single-user app that is a
fair trade; examples 2 and 3 are the fix if it ever bites. `POST /api/run` is an action,
and modelling it as one is the right call.
