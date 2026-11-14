---
id: ts.services.openapi
title: "Generating types from OpenAPI; a typed API client"
track: ts
module: services
order: 3
level: intermediate
minutes: 15
since: es2022
prerequisites: [ts.services.zod, ts.type-level.conditionals]
compare_with: [go.postgres.migrations-sqlc, inside.codebase.contract, go.http.json-validation]
tags: [openapi, openapi-typescript, codegen, api-client, contract]
summary: >
  An OpenAPI document is the contract between server and client. openapi-typescript
  turns it into types with no run-time code, and a small generic client checks paths,
  parameters and per-status bodies at compile time. Nothing checks the server at run
  time, so drift needs a validator or a test.
examples:
  - id: typed-client
    file: examples/01_typed_client.ts
    title: "A generic client over generated paths: wrong paths and params don't compile"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-22"
        note: >
          The shape openapi-typescript emits, abbreviated: paths keyed by the
          literal path template, then the method, then parameters and responses
          keyed by status code, each with content keyed by media type. The
          unabridged generated file works with the same client unchanged.
      - lines: "25-33"
        note: >
          The client is pure indexing into the generated types. Reply maps every
          documented status S to { status: S; body } and takes the union of
          them, a discriminated union on status built from the contract.
      - lines: "35-39"
        note: >
          P is inferred from the path literal, so path is checked against the
          contract and params gets that path's parameter type. Only the fetch
          function is injected, as in the earlier lessons.
      - lines: "45-46"
        note: >
          Types are erased, so the client can't know at run time which statuses
          the spec lists, and this cast is unavoidable. A status the spec
          doesn't mention, such as a proxy's 502, is cast all the same. Check
          the status first, or validate (example 2).
      - lines: "59-67"
        note: >
          Before the status check, body is Order | Problem; after
          reply.status === 200 it is Order. The error path is as typed as the
          happy path.
      - lines: "69-74"
        note: >
          Mistakes that a hand-written fetch(`/order/${id}`) would send to the
          server fail the build instead. Renaming a path in the spec and
          regenerating turns every stale call site into a compile error.
  - id: drift
    file: examples/02_drift.ts
    title: "When the server drifts from the spec: nil slices and int64 ids"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "4-15"
        note: >
          This is the generator's output for type: integer, format: int64. JSON
          numbers are doubles in JavaScript, so the format survives only as a
          comment.
      - lines: "17-19"
        note: >
          Two classic Go-side deviations. encoding/json writes a nil slice as
          null, and an int64 as a number, however large. Both are legal JSON,
          and neither matches the spec.
      - lines: "21-27"
        note: >
          The cast succeeds, as every generated client's does. The id is
          silently rounded to the nearest double, and items.length throws a
          TypeError far from the response that caused it.
      - lines: "29-37"
        note: >
          Annotating the schema with the generated type ties the two
          definitions: if the spec changes and the schema no longer produces an
          Order, tsc rejects it. At run time the schema reports both problems
          with their paths. int() accepts only safe integers, so the rounded id
          fails instead of passing as the wrong number.
      - lines: "39-50"
        note: >
          Fix the contract, not the client. Go initializes the slice
          (Items: []string{}), and large ids travel as strings (the ,string tag
          option, type: string in the spec). BigInt does exact arithmetic when
          you need it.
use_cases:
  - "A frontend and a Go backend developed in parallel: the spec in the repo is what both sides compile against"
  - "Calling a third-party API that publishes an OpenAPI document, without writing its types by hand"
  - "Typed error handling: per-status response types for 404, 409 and 422 problem bodies"
  - "CI checks: regenerate types and fail on a diff when the spec changed but the client didn't"
strengths:
  - "One spec drives server stubs, client types, documentation and mock servers, instead of each side retyping the shapes"
  - "Renamed fields and removed endpoints become compile errors after regenerating, not production bugs"
  - "openapi-typescript emits types only, so the client costs no bundle size beyond your fetch wrapper"
  - "Per-status response types make the error paths as typed as the happy path"
weaknesses:
  - "Generated types check nothing at run time: a server that deviates from its own spec goes unnoticed without validation"
  - "A spec written by hand next to the code drifts like any other copy: generate one side from the other, or test against it"
  - "JSON semantics leak through: int64 ids, nullable vs optional, Go's nil slices and omitempty"
  - "The generated types are deeply nested, and helpers that index into them are hard to read without a library such as openapi-fetch"
pitfalls:
  - title: "Treating generated types as validation"
    detail: "The types describe what the spec promises. If the server's real behaviour matters, validate responses at the client edge, at least in development and tests."
  - title: "int64 ids as JSON numbers"
    detail: "Above 2^53 - 1 (Number.MAX_SAFE_INTEGER), JSON.parse rounds silently. Send such ids as strings."
    bad: "ID int64 `json:\"id\"`   // spec: type: integer, format: int64"
    good: "ID int64 `json:\"id,string\"`   // spec: type: string"
  - title: "Go nil slices and omitempty"
    detail: "A nil slice marshals as null, and omitempty drops false, 0, \"\" and empty slices entirely, so a field the spec marks required can be null or missing. Initialize slices, and use omitempty only on optional fields."
  - title: "Editing the generated file"
    detail: "The next regeneration erases the edit. Change the spec, or wrap the generated types in your own module."
quiz:
  - q: "The client uses types generated by openapi-typescript. What checks at run time that a response matches components[\"schemas\"][\"Order\"]?"
    options: ["The generated code validates each response", "TypeScript, through the paths type", "Nothing, unless you add a validator"]
    answer: 2
    why: "openapi-typescript emits types only, and types are erased. The run-time check has to be code, such as a zod schema."
  - q: "A Go handler returns ID int64 = 9007199254740993 as a JSON number. What does JSON.parse produce?"
    options: ["9007199254740993", "9007199254740992", "A SyntaxError"]
    answer: 1
    why: "JSON numbers become doubles, and above 2^53 - 1 not every integer is representable: the value rounds to the nearest double, silently."
  - q: "A Go struct has Items []string `json:\"items\"` and the handler never sets it. What does the client receive?"
    options: ["\"items\": []", "\"items\": null", "No items key at all"]
    answer: 1
    why: "encoding/json writes a nil slice as null. The key would be missing only with omitempty, and [] needs an initialized empty slice."
  - q: "In example 1, why does reply.body.customer fail to compile before checking reply.status?"
    options: ["body is unknown until it is validated", "body is the union Order | Problem until the status narrows it", "customer is optional in the spec"]
    answer: 1
    why: "Reply has one member per documented status. Comparing status with 200 selects the member whose body is Order."
further_reading:
  - title: "TypeScript Handbook: Indexed Access Types"
    url: "https://www.typescriptlang.org/docs/handbook/2/indexed-access-types.html"
  - title: "TypeScript Handbook: Mapped Types"
    url: "https://www.typescriptlang.org/docs/handbook/2/mapped-types.html"
  - title: "TypeScript Handbook: Conditional Types (infer)"
    url: "https://www.typescriptlang.org/docs/handbook/2/conditional-types.html"
  - title: "MDN: Number.MAX_SAFE_INTEGER"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Number/MAX_SAFE_INTEGER"
  - title: "package encoding/json (Go): nil slices, omitempty, the string option"
    url: "https://pkg.go.dev/encoding/json"
---

## Concept

An [OpenAPI](https://spec.openapis.org/oas/v3.1.0) document describes an HTTP API as data:
every path, method, parameter, request body and response, keyed by status code and media
type, with the payloads as JSON Schema. It is the contract between server and client.

[openapi-typescript](https://openapi-ts.dev) reads the document and emits three
interfaces: `paths`, `components` (the schemas) and `operations`. It emits types only, no
run-time code, so it costs nothing in the bundle and checks nothing at run time:

```sh
npx openapi-typescript ./api/orders.yaml -o ./src/api/schema.d.ts
```

You index into the result: `components["schemas"]["Order"]` is the order type, and
`paths["/orders/{id}"]["get"]["responses"][200]["content"]["application/json"]` is the
body of a successful GET. A client built on those types checks the path template, the
parameters and each status's body at compile time. You can write a small one (example 1)
or use openapi-fetch from the same project:

```ts
import createClient from "openapi-fetch";
import type { paths } from "./api/schema";

const client = createClient<paths>({ baseUrl: "https://api.example.com" });
const { data, error } = await client.GET("/orders/{id}", {
  params: { path: { id: 7 } },
});
if (error) throw new Error(error.title); // typed as the spec's 404 body
console.log(data.customer); // typed as Order
```

The types promise that the server follows the spec. Whether it does is a separate
question, and only a run-time check or a test answers it (example 2).

## Coming from C/C++/Java/Python/JS

- **C/C++**: a shared header describing a wire struct. Both programs compile against it,
  but a stale build of one side still puts different bytes on the wire, and nothing
  checks them.
- **Java**: OpenAPI Generator produces model classes and a client, and deserializing
  builds real instances of those classes, failing on values it can't map (an object where
  a number belongs). TypeScript's generated types vanish at run time, so not even that
  check exists unless you add it.
- **Go**: `.proto` files with gRPC, or an OpenAPI spec with a generator such as
  oapi-codegen, produce structs whose decoding checks wire types. sqlc
  (go.postgres.migrations-sqlc) applies the same idea to SQL: the schema is the source,
  and the code is generated.
- **Python**: FastAPI is code-first: the spec is derived from pydantic models, which also
  validate. TypeScript clients usually consume such a spec rather than produce it.
- **JS**: the usual alternative is a hand-written mirror of the server's types plus
  `as` casts. That works until someone changes one side.

## How it works

A spec, and what openapi-typescript 7 generates from it, abridged (the generator writes
every member on its own line). The examples use parts of it.

```yaml
openapi: 3.1.0
info: { title: Orders, version: 1.0.0 }
paths:
  /orders/{id}:
    get:
      operationId: getOrder
      parameters:
        - { name: id, in: path, required: true, schema: { type: integer, format: int64 } }
      responses:
        "200":
          description: The order
          content:
            application/json:
              schema: { $ref: "#/components/schemas/Order" }
        "404":
          description: No such order
          content:
            application/problem+json:
              schema: { $ref: "#/components/schemas/Problem" }
components:
  schemas:
    Order:
      type: object
      required: [id, customer, status, items]
      properties:
        id: { type: integer, format: int64 }
        customer: { type: string }
        status: { type: string, enum: [open, shipped] }
        items: { type: array, items: { type: string } }
        note: { type: [string, "null"] }
    Problem:
      type: object
      required: [title, status]
      properties:
        title: { type: string }
        status: { type: integer }
        detail: { type: string }
```

```ts
export interface paths {
    "/orders/{id}": {
        parameters: { query?: never; header?: never; path?: never; cookie?: never };
        get: operations["getOrder"];
        put?: never;
        post?: never;
        // delete, options, head, patch, trace: all ?: never
    };
}
export interface components {
    schemas: {
        Order: {
            /** Format: int64 */
            id: number;
            customer: string;
            /** @enum {string} */
            status: "open" | "shipped";
            items: string[];
            note?: string | null;
        };
        Problem: { title: string; status: number; detail?: string };
    };
    // responses, parameters, requestBodies, headers, pathItems: never
}
export interface operations {
    getOrder: {
        parameters: {
            query?: never;
            header?: never;
            path: { id: number };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description The order */
            200: {
                headers: { [name: string]: unknown };
                content: { "application/json": components["schemas"]["Order"] };
            };
            /** @description No such order */
            404: {
                headers: { [name: string]: unknown };
                content: { "application/problem+json": components["schemas"]["Problem"] };
            };
        };
    };
}
```

The mapping is direct: a `required` property becomes a required field, any other property
an optional one (`?`), `type: [string, "null"]` becomes `string | null`, and an `enum`
becomes a union of literals. `format` survives only as a comment.

## Design notes

Pick one direction and generate the other side. **Contract-first**: the spec lives in the
repo, and both the Go server (oapi-codegen or similar) and the TypeScript client are
generated from it. **Code-first**: the spec is generated from the Go handlers and types,
and the client is generated from that. Either way, regenerate in CI and fail on a diff
(`git diff --exit-code`), so a change on one side can't merge without the other.

Generated types still trust the server. Close the gap where it is cheap: validate
responses in development and tests with schemas annotated with the generated types
(example 2), or use a generator that also emits zod schemas. And settle the JSON
representation questions in the spec, where both sides see them: large integers as
strings, timestamps in UTC with `Z`, money as integer minor units or decimal strings
(PostgreSQL `numeric` doesn't fit a double), nullable vs optional, and empty lists as `[]`.

This app shows the risk on a small scale. `server/internal/runapi` defines the wire types
in Go, `src/features/runner/types.ts` mirrors them by hand, and `CodeRunner.ts` casts the
response with `as RunResult`. When this lesson was written, a SQL run's response carried
`"diagnostics": null` and `"files": null` (nil slices in Go), while the mirror declares
`Diagnostic[]` and `WrittenFile[]`. Nothing read those fields for SQL runs, so nothing
broke: that is how drift stays invisible until someone writes `result.files.length`. It
was found by people reading the code, not by any check, and fixed on the Go side:
`runapi.Result` now writes every empty list as `[]`, and a Go test fails on `null`. The
TypeScript side still trusts the response with a cast, so it would not notice the next
drift, or a different producer that still sends `null`. A generated contract, or one test
that decodes a real response with a schema, is what would catch it.
