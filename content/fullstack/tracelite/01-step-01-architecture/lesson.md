---
id: fullstack.tracelite.step-01-architecture
title: "Step 1: Architecture and request lifecycle; contract-first API with OpenAPI"
track: fullstack
module: tracelite
order: 1
level: intermediate
minutes: 15
since: go1.27
prerequisites: [go.http.handlers, go.postgres.pgx, ts.services.openapi, ts.tooling.vite]
compare_with: [inside.codebase.wiring, inside.codebase.contract, design.architecture.styles, go.design.layers]
tags: [tracelite, architecture, request-lifecycle, vite-proxy, openapi, contract-first, json, naming]
summary: >
  TraceLite is one Go server, one PostgreSQL database and one React app, with
  openapi.yaml as the contract between them. A request goes browser, Vite proxy,
  ServeMux, handler, service, store, pgx, PostgreSQL and back; names change form at
  two well-defined places.
examples:
  - id: lifecycle
    file: examples/01_lifecycle.go
    title: "One request through every layer, printed hop by hop"
    runtime: go-server
    check: exact
    timeout_s: 15
    annotations:
      - lines: "21-25"
        note: >
          The wire type, abridged. Its json tags are where snake_case columns
          become camelCase members; nothing else in the request path renames
          anything.
      - lines: "30-31"
        note: >
          A fixed id and a timestamp written in Istanbul time (+03), so the
          output is the same on every run. Step 2 has the real schema.
      - lines: "40-41"
        note: >
          Go 1.22 patterns: the method and a named wildcard in the pattern
          itself. r.Pattern (Go 1.23) reports which pattern matched, which is
          also what TraceLite's request log records.
      - lines: "43-46"
        note: >
          The id is parsed before any SQL runs, and travels to PostgreSQL as a
          parameter: the extended protocol sends the statement and the value
          separately.
      - lines: "45"
        note: >
          r.Context() is passed down to pgx. If the browser goes away, the
          request's context is canceled and the call returns at once: by
          default pgx sets a deadline on the connection and closes it.
          Sending PostgreSQL a real cancel request is opt-in, through
          pgconn's CancelRequestContextWatcherHandler.
      - lines: "48-53"
        note: >
          The error leg: an id that doesn't parse, or parses but names nothing
          (pgx.ErrNoRows), takes the same route and gets an RFC 9457 problem
          instead of a resource. Step 4 maps every error this way, in one
          function.
      - lines: "54-55"
        note: >
          pgx returns timestamptz in the process's local zone ("Local"). The
          wire carries UTC with Z (06:30Z for 09:30+03), whatever zone the
          server runs in.
      - lines: "62-66"
        note: >
          What vite.config.ts's server.proxy does: requests under /api go to the
          Go server; everything else, the app itself, Vite serves. The browser
          only ever talks to vite.URL, so the page and the API share one origin
          and the browser never sends a CORS preflight.
  - id: contract
    file: examples/02_contract.go
    title: "The spec as the yardstick: checking a Go wire type against openapi.yaml"
    runtime: go
    check: exact
    annotations:
      - lines: "11-16"
        note: >
          The Requirement schema from the spec below, reduced to what this
          check reads. The spec, not the Go struct and not the table, defines
          what a client receives.
      - lines: "19-22"
        note: >
          The check runs on the encoded bytes, exactly what a handler would
          write, so struct tags, omitempty and custom marshalers are all part
          of what is checked.
      - lines: "23-34"
        note: >
          Three kinds of drift: a required member missing, a member the spec
          doesn't know, and a value outside the enum. Step 14 turns this into a
          test; a production suite would load openapi.yaml into a JSON Schema
          validator, which also checks types, patterns and lengths. The idea
          is the same.
      - lines: "48-56"
        note: >
          Each edit compiles, passes go vet and looks fine in a diff. Both
          change the wire, and neither changes the TypeScript client, which
          keeps reading parentId and description.
      - lines: "60"
        note: >
          A struct conversion: the field names and types match, and since Go 1.8
          tags are ignored when converting, so the same values are encoded both
          ways.
      - lines: "61"
        note: >
          The third kind of drift is in the data, not the type: a display label
          in a field the spec defines as an enum of lowercase values.
use_cases:
  - "Starting a full-stack feature: write the operation and schemas into openapi.yaml first, then the migration, the store query, the handler and the client"
  - "Reviewing a change: a diff to openapi.yaml is a change to the contract and needs both sides"
  - "Debugging: following a request through the proxy, router, handler, store and database, one layer at a time"
  - "Local development with one origin: Vite proxies /api, so cookies, relative URLs and fetch work without CORS"
strengths:
  - "The contract is a reviewed file in the repo, not whatever the Go structs happen to encode; TypeScript types are generated from it"
  - "One origin in development (Vite proxy) and production (the Go binary serves the app), so there is no CORS configuration to drift between them"
  - "Every layer has one job and one package; a bug report names a layer: decoding, rules, SQL, or rendering"
  - "Names change form in exactly two places, Go struct tags (SQL to Go to JSON) and nowhere in TypeScript, because the client uses the wire names directly"
weaknesses:
  - "The Go wire types are written by hand next to the spec; only a test notices when they drift"
  - "Contract-first means writing YAML before code, which feels slow for a one-person project until the first client and server disagree"
  - "Three representations of a requirement (row, Go model, wire JSON) mean mapping code in the store and the handler"
  - "The Vite proxy hides cross-origin problems in development that a deployment on two origins would have"
pitfalls:
  - title: "Letting the Go structs become the contract"
    detail: "If the spec is updated from the code after the fact, a tag edit silently changes the API. Change openapi.yaml first, then the Go type, and let the contract test compare them."
  - title: "Calling the Go server directly from the browser in development"
    detail: "fetch(\"http://127.0.0.1:8080/api/...\") from a page on :5173 is cross-origin and needs CORS headers that production never sends. Use relative URLs and the Vite proxy."
    bad: "fetch(\"http://127.0.0.1:8080/api/requirements/\" + id)"
    good: "fetch(\"/api/requirements/\" + id) // same origin; Vite forwards /api"
  - title: "snake_case leaking onto the wire"
    detail: "A tag copied from the column name (json:\"parent_id\") breaks every client that reads parentId. Wire names are camelCase, always written as explicit tags."
  - title: "Dropping the request context"
    detail: "context.Background() in a handler's query keeps the handler waiting for a result after the client disconnected, holding a pooled connection. Pass r.Context() all the way to pgx."
quiz:
  - q: "In development, the browser loads the app from :5173 and calls /api/requirements/{id}. Who answers?"
    options: ["Vite, from a mock", "The Go server, through Vite's proxy, on the same origin", "The Go server directly, with CORS headers"]
    answer: 1
    why: "server.proxy forwards /api to the Go server; the browser only ever talks to the dev server's origin."
  - q: "Where does parent_id become parentId?"
    options: ["In a SQL alias", "In the json tag of the Go wire type", "In a TypeScript mapping function"]
    answer: 1
    why: "SQL keeps snake_case, sqlc maps columns to Go field names, and the wire type's json tags produce camelCase; the client uses the wire names unchanged."
  - q: "Which file is the source of truth for the JSON a client receives?"
    options: ["The Go wire structs", "openapi.yaml", "The SQL schema"]
    answer: 1
    why: "Contract-first: the spec is written first and both sides conform to it. TypeScript types are generated from it; the Go types are checked against it."
  - q: "A handler adds omitempty to description. What does example 2 report?"
    options: ["Nothing: the field still exists in Go", "missing description, when the description is empty", "undocumented description"]
    answer: 1
    why: "omitempty drops the member when it is empty; the spec lists description as required."
further_reading:
  - title: "package net/http: ServeMux (patterns, PathValue)"
    url: "https://pkg.go.dev/net/http#ServeMux"
  - title: "package net/http/httputil: ReverseProxy"
    url: "https://pkg.go.dev/net/http/httputil#ReverseProxy"
  - title: "Go spec: conversions between struct types (tags ignored)"
    url: "https://go.dev/ref/spec#Conversions"
  - title: "MDN: Cross-Origin Resource Sharing (CORS)"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CORS"
  - title: "MDN: Same-origin policy"
    url: "https://developer.mozilla.org/en-US/docs/Web/Security/Same-origin_policy"
---

## Concept

TraceLite is a small requirements-traceability tool: projects contain a tree of
requirements, test cases verify requirements through trace links, and test runs record
results. It is built like StackLab itself, with one difference: the API is
**contract-first**.

- **PostgreSQL** holds the data and the rules a row can express (step 2).
- **One Go binary** (`cmd/tracelite`) serves `/api`: `httpapi` (routing, decoding,
  errors), the `tracelite` service (rules), `store` (sqlc queries, transactions). In
  production it also serves the built React app, so there is one origin (step 15).
- **A React app** built with Vite. In development Vite serves it on :5173 and forwards
  `/api` to the Go server on :8080.
- **`openapi.yaml`** in the repo root is the contract. It is written first. TypeScript
  types are generated from it; the Go wire types are written to match it and tested
  against it.

## Coming from C/C++/Java/Python/JS

- **Spring Boot + React**: the same split. springdoc usually generates the spec from the
  controllers (code-first); TraceLite writes the spec and checks the Go code against it.
- **Django + DRF**: serializers are both the wire format and the validation. Here the
  wire format is the spec, validation is the service, and the two are separate files.
- **Node full-stack (Next.js, tRPC)**: sharing TypeScript types between client and server
  makes the contract implicit. With a Go server the contract has to be a language-neutral
  file, which is also what lets any other client use the API.
- **C/C++**: a header shared by two programs, except that this one is checked at both
  ends: generated types at compile time, schemas at run time.

## How it works

The lifecycle of `GET /api/requirements/{id}`, which example 1 prints:

```mermaid
sequenceDiagram
  participant B as Browser (React, TanStack Query)
  participant V as Vite dev server :5173
  participant G as Go server :8080
  participant P as PostgreSQL
  B->>V: GET /api/requirements/{id}
  V->>G: proxied (server.proxy "/api")
  G->>G: ServeMux pattern, handler parses id
  G->>P: pgx: SELECT ... WHERE id = $1
  P-->>G: one row (snake_case columns)
  G-->>V: 200 application/json (camelCase, UTC)
  V-->>B: same response
  B->>B: zod parses it, the cache stores it, React renders
```

Where each name and type lives:

- **SQL** (`migrations/*.sql`): snake_case tables and columns, `uuid` ids, `timestamptz`.
- **Go store** (`internal/store`, generated by sqlc): structs per table, `uuid.UUID`,
  `*uuid.UUID` for nullable ids, `time.Time`.
- **Go wire types** (`internal/httpapi`): one struct per schema in `openapi.yaml`, the
  same names (`Requirement`, `RequirementCreate`, `Problem`), camelCase json tags.
- **TypeScript** (`web/src/api`): `schema.d.ts` generated from the spec, zod schemas that
  must satisfy it, branded ids (step 5).

The repository:

```text
openapi.yaml                 the contract
sqlc.yaml                    step 3
migrations/                  001_init.sql (step 2), then 002, 003, ...
queries/                     SQL for sqlc
cmd/tracelite/main.go        wiring: config, pool, migrate, store, service, router, server
internal/store/              generated queries + store.go (WithTx, errors)
internal/tracelite/          the service: rules shared by the API and the importer
internal/httpapi/            router, handlers, wire types, problems, cursors
web/                         Vite + React; vite.config.ts proxies /api in development
web/src/api/                 schema.d.ts, schemas.ts, client.ts, queries.ts (step 5)
```

The proxy, in `web/vite.config.ts`:

```ts
export default defineConfig({
  plugins: [react()],
  server: { proxy: { "/api": "http://127.0.0.1:8080" } },
});
```

The contract. This excerpt of `openapi.yaml` is canonical: later steps add their paths
and schemas next to these and follow the same patterns (step 4 lists every path).

```yaml
openapi: 3.1.0
info: { title: TraceLite, version: 0.1.0 }
paths:
  /api/projects/{projectId}/requirements:
    parameters: [{ $ref: "#/components/parameters/ProjectId" }]
    get:
      operationId: listRequirements
      parameters: [{ $ref: "#/components/parameters/Limit" }, { $ref: "#/components/parameters/Cursor" }]
      responses:
        "200":
          description: One page, ordered by code
          content: { application/json: { schema: { $ref: "#/components/schemas/RequirementPage" } } }
        "400": { $ref: "#/components/responses/Problem" }
        "404": { $ref: "#/components/responses/Problem" }
    post:
      operationId: createRequirement
      requestBody:
        required: true
        content: { application/json: { schema: { $ref: "#/components/schemas/RequirementCreate" } } }
      responses:
        "201":
          description: Created
          headers: { Location: { schema: { type: string } } }
          content: { application/json: { schema: { $ref: "#/components/schemas/Requirement" } } }
        "400": { $ref: "#/components/responses/Problem" }
        "404": { $ref: "#/components/responses/Problem" }
        "409": { $ref: "#/components/responses/Problem" }
        "422": { $ref: "#/components/responses/Problem" }
  /api/projects/{projectId}/requirement-tree:
    parameters: [{ $ref: "#/components/parameters/ProjectId" }]
    get:
      operationId: getRequirementTree
      responses:
        "200":
          description: Every requirement of the project, depth first
          content: { application/json: { schema: { $ref: "#/components/schemas/RequirementTree" } } }
        "404": { $ref: "#/components/responses/Problem" }
  /api/requirements/{requirementId}:
    parameters: [{ $ref: "#/components/parameters/RequirementId" }]
    get:
      operationId: getRequirement
      responses:
        "200": { description: The requirement, content: { application/json: { schema: { $ref: "#/components/schemas/Requirement" } } } }
        "404": { $ref: "#/components/responses/Problem" }
    patch:
      operationId: updateRequirement
      requestBody:
        required: true
        content: { application/json: { schema: { $ref: "#/components/schemas/RequirementPatch" } } }
      responses:
        "200": { description: Updated, content: { application/json: { schema: { $ref: "#/components/schemas/Requirement" } } } }
        "400": { $ref: "#/components/responses/Problem" }
        "404": { $ref: "#/components/responses/Problem" }
        "422": { $ref: "#/components/responses/Problem" }
    delete:
      operationId: deleteRequirement
      responses:
        "204": { description: Deleted }
        "404": { $ref: "#/components/responses/Problem" }
        "409": { $ref: "#/components/responses/Problem" }
  /api/requirements/{requirementId}/parent:
    parameters: [{ $ref: "#/components/parameters/RequirementId" }]
    put:
      operationId: moveRequirement
      requestBody:
        required: true
        content: { application/json: { schema: { $ref: "#/components/schemas/RequirementMove" } } }
      responses:
        "200": { description: Moved, content: { application/json: { schema: { $ref: "#/components/schemas/Requirement" } } } }
        "400": { $ref: "#/components/responses/Problem" }
        "404": { $ref: "#/components/responses/Problem" }
        "422": { $ref: "#/components/responses/Problem" }
components:
  parameters:
    ProjectId: { name: projectId, in: path, required: true, schema: { type: string, format: uuid } }
    RequirementId: { name: requirementId, in: path, required: true, schema: { type: string, format: uuid } }
    Limit: { name: limit, in: query, schema: { type: integer, minimum: 1, maximum: 200, default: 50 } }
    Cursor: { name: cursor, in: query, description: nextCursor of the previous page, schema: { type: string } }
  responses:
    Problem:
      description: An RFC 9457 problem
      content: { application/problem+json: { schema: { $ref: "#/components/schemas/Problem" } } }
  schemas:
    Project:
      type: object
      required: [id, name, createdAt]
      properties:
        id: { type: string, format: uuid }
        name: { type: string, minLength: 1, maxLength: 200 }
        createdAt: { type: string, format: date-time }
    Requirement:
      type: object
      required: [id, projectId, parentId, code, title, description, status, createdAt, updatedAt]
      properties:
        id: { type: string, format: uuid }
        projectId: { type: string, format: uuid }
        parentId: { type: [string, "null"], format: uuid }
        code: { $ref: "#/components/schemas/Code" }
        title: { type: string, minLength: 1, maxLength: 200 }
        description: { type: string }
        status: { type: string, enum: [draft, approved, obsolete] }
        createdAt: { type: string, format: date-time }
        updatedAt: { type: string, format: date-time }
    RequirementCreate:
      type: object
      required: [code, title]
      properties:
        code: { $ref: "#/components/schemas/Code" }
        title: { type: string, minLength: 1, maxLength: 200 }
        description: { type: string, default: "" }
        parentId: { type: [string, "null"], format: uuid, default: null }
    RequirementPatch:
      type: object
      minProperties: 1
      properties:
        title: { type: string, minLength: 1, maxLength: 200 }
        description: { type: string }
        status: { type: string, enum: [draft, approved, obsolete] }
    RequirementMove:
      type: object
      required: [parentId]
      properties:
        parentId: { type: [string, "null"], format: uuid }
    RequirementPage:
      type: object
      required: [items, nextCursor]
      properties:
        items: { type: array, items: { $ref: "#/components/schemas/Requirement" } }
        nextCursor: { type: [string, "null"] }
    RequirementTree:
      type: object
      required: [items]
      properties:
        items:
          type: array
          items:
            allOf:
              - { $ref: "#/components/schemas/Requirement" }
              - { type: object, required: [depth], properties: { depth: { type: integer, minimum: 0 } } }
    TestCase:
      type: object
      required: [id, projectId, code, title, description, createdAt, updatedAt]
      properties:
        id: { type: string, format: uuid }
        projectId: { type: string, format: uuid }
        code: { $ref: "#/components/schemas/Code" }
        title: { type: string, minLength: 1, maxLength: 200 }
        description: { type: string }
        createdAt: { type: string, format: date-time }
        updatedAt: { type: string, format: date-time }
    TraceLink:
      type: object
      required: [projectId, requirementId, testCaseId, createdAt]
      properties:
        projectId: { type: string, format: uuid }
        requirementId: { type: string, format: uuid }
        testCaseId: { type: string, format: uuid }
        createdAt: { type: string, format: date-time }
    TestRun:
      type: object
      required: [id, testCaseId, result, finishedAt, durationMs, notes]
      properties:
        id: { type: string, format: uuid }
        testCaseId: { type: string, format: uuid }
        result: { type: string, enum: [passed, failed, skipped] }
        finishedAt: { type: string, format: date-time }
        durationMs: { type: [integer, "null"], minimum: 0 }
        notes: { type: string }
    Code:
      type: string
      pattern: "^[A-Z][A-Z0-9]*(-[A-Z0-9]+)*$"
      maxLength: 40
    FieldError:
      type: object
      required: [pointer, detail]
      properties:
        pointer: { type: string, description: "JSON Pointer into the request body, e.g. #/code" }
        detail: { type: string }
    Problem:
      type: object
      required: [type, title, status]
      properties:
        type: { type: string, description: "about:blank or https://tracelite.example/problems/{validation|conflict}" }
        title: { type: string }
        status: { type: integer }
        detail: { type: string }
        errors: { type: array, items: { $ref: "#/components/schemas/FieldError" } }
```

Test cases, links and runs get the same operations under the paths step 4 lists, with
`TestCasePage`, `TraceLinkPage` and `TestRunPage` shaped like `RequirementPage`.

## Design notes

**Contract-first, with one generator.** The spec is written by hand and reviewed. The
TypeScript side is generated from it (openapi-typescript), because there the types are
the whole benefit. The Go side is written by hand in `internal/httpapi`, mirroring the
schema names, and a contract test checks the encoded output against the spec (example 2
shows the idea; step 14 makes it a test). The alternative, generating Go server code
with oapi-codegen, was rejected to keep sqlc the project's only Go generator and the
handlers ordinary `net/http` code that the HTTP lessons already taught.

**Naming.** SQL is snake_case because PostgreSQL folds unquoted identifiers to lower
case, and camelCase columns would need double quotes in every query. JSON is camelCase
because its main consumer is TypeScript, where camelCase members need no renaming. The
change happens once, in Go struct tags, and sqlc's column-to-field mapping
(`parent_id` to `ParentID`) is mechanical. Nothing in TypeScript renames members.

**Same origin everywhere.** Development uses the Vite proxy, production the Go binary
serving the built app; both mean relative URLs, no CORS headers and cookies that just
work. StackLab does the same with its own `vite.config.ts`.

**What this step doesn't decide.** Authentication (TraceLite is local and single-user,
like StackLab), deployment (step 15) and real-time updates (step 9) all slot into this
shape without changing it: a middleware, a build step, and one more endpoint.
