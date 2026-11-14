---
id: ts.services.zod
title: "Runtime validation with zod at the trust boundary"
track: ts
module: services
order: 2
level: intermediate
minutes: 14
since: es2022
prerequisites: [ts.services.fetch, ts.narrowing.discriminated-unions]
compare_with: [go.http.json-validation, ts.narrowing.guards, ts.react.forms-lists]
tags: [zod, validation, runtime-types, trust-boundary, json]
summary: >
  TypeScript types are erased, so nothing checks what arrives over the network.
  A zod schema is a run-time value that validates and normalizes the data, and the
  static type is derived from it: one definition, checked where data enters the app.
examples:
  - id: boundary
    file: examples/01_boundary.ts
    title: "A cast checks nothing; a schema checks and produces the type"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "13-14"
        note: >
          JSON.parse returns any, so the cast compiles. At run time id is the
          string "7" and + concatenates: the bug surfaces as "71" somewhere far
          from the response that caused it.
      - lines: "17-22"
        note: >
          The schema is an ordinary value that exists at run time. z.infer
          derives the static type from it, so there is no interface to keep in
          sync by hand.
      - lines: "24-32"
        note: >
          safeParse returns { success: true, data } or { success: false, error },
          a discriminated union, so data is typed only after the check. Every
          issue is collected, not just the first; each has a path and a
          machine-readable code, and prettifyError formats them for logs.
      - lines: "34-37"
        note: >
          By default z.object drops keys it doesn't declare (admin is gone) and
          applies normalizations such as trim(). Dropping is the tolerant choice
          for responses: a server that adds a field doesn't break old clients.
      - lines: "39-44"
        note: >
          parse() throws a ZodError instead of returning a result. strictObject
          turns unknown keys into an error, which suits data you fully own, such
          as a config file, rather than responses from a server that evolves.
  - id: input-output
    file: examples/02_input_output.ts
    title: "Input type vs output type: coerce, transform, default, refine"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "6-9"
        note: >
          overwrite() rewrites the value and keeps the schema a ZodString.
          toLocaleUpperCase("tr") maps i to İ. zod's own toUpperCase() and
          toLowerCase() are locale-free: the output shows IZMIR, and
          lowercasing İSTANBUL gives i plus a combining dot.
      - lines: "10"
        note: >
          coerce runs Number(input) before validating, so "2" becomes 2. It also
          turns "" and null into 0, which only min(1) catches. For booleans in
          query strings use z.stringbool(): z.coerce.boolean() turns "false"
          into true.
      - lines: "13"
        note: >
          transform changes the type: a string goes in, a Date comes out. JSON
          has no date type, so the schema is the natural place to convert.
      - lines: "16-19"
        note: >
          refine adds a rule across fields; path attaches the issue to "to", so a
          form can show it next to that field. It runs only when the object has
          the right types: a failed min() doesn't stop it, a wrong type does.
      - lines: "21-22"
        note: >
          One schema, two types. z.input is what arrives (guests unknown,
          createdAt string, note optional); z.output is what the app gets
          (number, Date, string with a default). z.infer is z.output.
      - lines: "24-30"
        note: >
          Go's encoding/json writes a time.Time in RFC 3339 with that value's
          own offset, +03:00 for Europe/Istanbul, and with the microseconds
          PostgreSQL stored.
      - lines: "35"
        note: >
          A Date has millisecond precision, so the microseconds are gone. If a
          timestamp is used as a key or a pagination cursor, keep the string.
      - lines: "42-43"
        note: >
          z.iso.datetime() accepts only a Z suffix by default, so a server
          writing +03:00 fails validation. Agree on UTC with Z, or pass
          { offset: true } as above.
  - id: client-edge
    file: examples/03_client_edge.ts
    title: "One validating entry point, discriminated unions, and a policy for bad items"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "4-8"
        note: >
          discriminatedUnion selects the branch by the status field, so errors
          name the bad discriminator instead of listing every branch's failures.
          The inferred type is a TypeScript discriminated union.
      - lines: "10-16"
        note: >
          A new server meets an old client, which happens with every deploy
          while users still run the previous bundle: a status this client was
          never written for.
      - lines: "18-29"
        note: >
          The single point where unknown JSON becomes typed data. T is inferred
          from the schema's output type, so callers never write a type argument
          or a cast.
      - lines: "32-37"
        note: >
          All or nothing: one unknown item fails the whole list, and the path
          [2].status says which item and which field.
      - lines: "39-46"
        note: >
          Per item: keep the valid ones and report the rest. Choose per
          endpoint: a ledger whose total must add up can't drop rows silently,
          while a notification feed can.
      - lines: "52-56"
        note: >
          Because the inferred type is a discriminated union, checking status
          narrows it: kurus exists only in the "paid" branch, and amounts stay
          integer kuruş until formatting.
use_cases:
  - "Responses from your own API: one validating getJson(url, schema) in the client module"
  - "Data the app didn't just create: localStorage, URL search params, postMessage payloads, WebSocket messages"
  - "Form input, with the same schema reused on submit; issue paths map to fields"
  - "Configuration and environment variables at startup, where a strict object catches typos"
strengths:
  - "One definition gives both the run-time check and the static type, unlike Java's DTO class plus separate Bean Validation annotations"
  - "safeParse returns a discriminated union, so a failed parse can't be used by mistake"
  - "Transforms normalize at the edge (trim, Turkish uppercase, string to Date), so the rest of the code sees clean values"
  - "Every issue comes with a path, which makes contract breaks easy to locate and form errors easy to place"
weaknesses:
  - "A library and a run-time cost on every parse; Go's encoding/json checks field types during decoding without any schema code"
  - "Schemas are a second definition when the types already come from an OpenAPI generator; keep them aligned (ts.services.openapi)"
  - "Built-in string helpers such as toUpperCase() are locale-free, which is wrong for Turkish text"
  - "Complex schemas slow down type checking and produce long editor tooltips"
pitfalls:
  - title: "Casting instead of validating"
    detail: "A cast tells the compiler to stop checking. The wrong data then fails far away, or not at all."
    bad: "const order = (await res.json()) as Order;"
    good: "const order = Order.parse(await res.json());"
  - title: "z.coerce.boolean() on query strings"
    detail: "Coercion is Boolean(input), and any non-empty string is true, \"false\" included. Use z.stringbool(), which accepts true/false, 1/0, yes/no, on/off."
  - title: "Strict schemas for server responses"
    detail: "strictObject or a closed enum turns every additive server change into a client outage. Strip unknown keys, and give enums a fallback with .catch() where an unknown value is survivable."
  - title: "Validating the same data twice, or everywhere"
    detail: "Validate once, where data crosses into the app. Inside, the types are trustworthy; re-validating internal calls costs time and suggests the boundary is in the wrong place."
quiz:
  - q: "const user = (await res.json()) as User, and the server sends \"id\": \"7\" where User says number. What happens at the cast?"
    options: ["A TypeError is thrown", "Nothing: the string flows on until something misbehaves", "The value is converted to 7"]
    answer: 1
    why: "A cast is erased at compile time; no code runs for it. The string surfaces later, for example as \"71\" from id + 1."
  - q: "z.object({ id: z.number() }).parse({ id: 1, isAdmin: true }) returns?"
    options: ["{ id: 1, isAdmin: true }", "{ id: 1 }", "It throws: unrecognized key"]
    answer: 1
    why: "Plain z.object strips unknown keys. z.strictObject would throw; z.looseObject would keep them."
  - q: "A query string carries ?archived=false. What does z.coerce.boolean() produce?"
    options: ["false", "true", "A validation error"]
    answer: 1
    why: "Coercion is Boolean(\"false\"), and every non-empty string is truthy. z.stringbool() parses \"false\" as false."
  - q: "For z.object({ at: z.iso.datetime().transform((s) => new Date(s)) }), what are z.input and z.output of at?"
    options: ["Date and Date", "string and Date", "string and string"]
    answer: 1
    why: "The input is what the schema accepts (an ISO string); the output is what transform returns. z.infer gives the output."
further_reading:
  - title: "MDN: JSON.parse()"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/JSON/parse"
  - title: "TypeScript Handbook: Narrowing (discriminated unions)"
    url: "https://www.typescriptlang.org/docs/handbook/2/narrowing.html#discriminated-unions"
  - title: "MDN: String.prototype.toLocaleUpperCase()"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/String/toLocaleUpperCase"
  - title: "MDN: Date"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Date"
  - title: "package encoding/json (Go, for comparison)"
    url: "https://pkg.go.dev/encoding/json"
---

## Concept

TypeScript types are erased: `JSON.parse` and `res.json()` return `any`, and a type
annotation or `as` cast on the result is a promise the compiler believes and nothing at run
time checks. Data from outside the program, such as an API response, a URL, localStorage or
a message from another window, crosses a **trust boundary**. It has to be checked there,
once, by code that exists at run time.

[zod](https://zod.dev) makes the check a value. A schema such as
`z.object({ id: z.number() })` validates input at run time and also *is* the type:
`z.infer<typeof Schema>` derives the static type, so the two can't drift apart.

- `parse(data)` returns the typed value or throws a `ZodError`; `safeParse(data)` returns
  `{ success: true, data }` or `{ success: false, error }`.
- `error.issues` lists every problem with its `path`, `code` and `message`;
  `z.prettifyError(error)` formats them.
- A schema can also normalize: `trim()`, `default()`, `z.coerce`, `overwrite()` and
  `transform()`. A schema therefore has an **input** type (`z.input`) and an **output** type
  (`z.output`, the same as `z.infer`).
- Objects strip unknown keys by default; `z.strictObject` rejects them, `z.looseObject`
  keeps them.

This lesson uses zod 4 (`import { z } from "zod"`). Version 3 differs in details:
`z.prettifyError`, `z.iso.*` and top-level `z.email()` are version 4 APIs.

## Coming from C/C++/Java/Python/JS

- **Go**: `json.Unmarshal` into a struct already rejects a string in an `int` field, but
  ignores unknown fields unless you call `DisallowUnknownFields`, turns missing fields
  into zero values, and leaves range checks to a hand-written `validate()` method. zod
  does the decoding checks and the validation in one place; TypeScript alone does neither.
- **Java**: Jackson maps JSON onto a class and Bean Validation annotations (`@NotNull`,
  `@Min`) check it. zod inverts the arrangement: the schema is the value, and the type is
  derived from it.
- **Python**: pydantic is the closest relative: a model validates, coerces and documents.
  pydantic's default lax mode turns `"7"` into `7`; zod coerces only where you write
  `z.coerce`.
- **JS**: hand-written type guards (`isUser(x): x is User`) do the same job for small
  shapes. They don't report which field failed, and the guard and the type can drift apart.

## Design notes

Validate at the edge and nowhere else. In a frontend the edges are few: the API client
(`getJson(url, schema)`), the code that reads storage and URL parameters, and message
handlers. Everything behind them can trust the types, which is the point of having them.

Be tolerant with what servers send and strict with what you own. Old clients outlive
deploys: a cached bundle will meet a newer server. Stripping unknown keys and giving
enums a fallback value (`z.enum(["paid", "failed", "unknown"]).catch("unknown")`; the
fallback must be one of the values) keeps additive server changes from becoming outages. Use strict objects for configuration files, where an unknown key is a
typo.

Decide what a contract break means per endpoint. For money, fail the whole response and
report it, because a broken contract is a bug and partial data would be wrong. For feeds,
skip the bad item and report it. Either way, send the issues to your error tracking: they
show the bug, while the user only needs "something went wrong".

When the types come from an OpenAPI generator (next lesson), the schema and the generated
type are two definitions. Annotate the schema with the generated type,
`const Order: z.ZodType<components["schemas"]["Order"]> = z.object({...})`, and tsc will
reject a schema whose output no longer matches the contract.

zod 4 implements the Standard Schema interface (a `~standard` property), so libraries that
accept "any Standard Schema" take a zod schema directly. For bundle-size-sensitive code
there is `zod/mini`, with a functional API that tree-shakes better.
