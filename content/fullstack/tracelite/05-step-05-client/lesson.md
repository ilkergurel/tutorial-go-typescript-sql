---
id: fullstack.tracelite.step-05-client
title: "Step 5: Typed TS client from OpenAPI + zod, wrapped in TanStack Query hooks"
track: fullstack
module: tracelite
order: 5
level: intermediate
minutes: 15
since: ts4.9
prerequisites: [fullstack.tracelite.step-04-api, ts.services.openapi, ts.services.zod, ts.type-level.branded, ts.react.query]
compare_with: [go.http.json-validation, go.services.pagination-streaming, ts.architecture.state, ts.errors.result]
tags: [tracelite, openapi, openapi-typescript, zod, branded-types, api-client, problem-details, tanstack-query, infinite-query]
summary: >
  The client side of TraceLite's contract: types generated from openapi.yaml, zod schemas
  tied to them with satisfies and checked on every response, problems turned into one
  ApiError, and TanStack Query hooks that page with nextCursor and invalidate by project.
examples:
  - id: schemas
    file: examples/01_schemas.ts
    title: "Generated types and the zod schema tied to them: brands, satisfies, UTC times"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "5-20"
        note: >
          What openapi-typescript writes into src/api/schema.d.ts, abridged:
          types only, no run-time code. format: uuid and date-time survive only
          as comments, a nullable member becomes T | null, and an enum becomes
          a union of literals.
      - lines: "23-25"
        note: >
          Branded ids. z.uuid() validates the RFC 9562 layout, and .brand() adds
          a type-only tag, the technique of ts.type-level.branded with zod's
          symbol. A ProjectId can only come out of a parse, and a RequirementId
          can't be passed where a ProjectId is expected (line 49).
      - lines: "26-33"
        note: >
          satisfies checks the schema against the generated type without
          widening it, so the inferred type keeps its brands. If the spec gains
          a required member, or the schema drops one, tsc fails here with
          TS1360. It doesn't catch a schema that is stricter than the spec, such
          as an enum with a value missing: a narrower type is still assignable.
      - lines: "32"
        note: >
          z.iso.datetime() without options accepts only a Z suffix. That turns
          step 4's convention (times in UTC) into a check on every response.
      - lines: "53-56"
        note: >
          The same instant with +03:00: a Go handler that forgot .UTC(). The
          value is right, the contract isn't, and the parse says where.
  - id: client
    file: examples/02_client.ts
    title: "request(): every response parsed, every failure one ApiError"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "15-21"
        note: >
          One error type for every failed request, carrying the parsed problem.
          Callers branch on problem.status and problem.type, and forms read
          problem.errors. The field is declared, not a parameter property,
          which keeps the class within erasable syntax.
      - lines: "30"
        note: >
          A body that isn't JSON (an HTML error page) becomes null instead of
          throwing a SyntaxError from inside the client.
      - lines: "31-38"
        note: >
          Not every failure comes from TraceLite: a proxy, a gateway or the
          Vite dev server can answer 502 with HTML. The client builds an
          about:blank problem from the status, so callers still see one
          ApiError shape.
      - lines: "39"
        note: >
          The success path validates too. A server that drifts from its own
          spec fails here, with a path to the bad member, instead of crashing
          a component three renders later.
      - lines: "42-46"
        note: >
          One function per operation, with a path relative to the page: in
          development the Vite proxy forwards /api, in production the same Go
          binary serves both, so there is one origin.
      - lines: "49"
        note: >
          The lesson iframe has no network, so this example replaces fetch
          itself and the client code stays as it is in the app. Step 14's tests
          intercept the same fetch calls with MSW instead.
      - lines: "60-62"
        note: >
          A member that came out as CreatedAt instead of createdAt. The request
          succeeded on the server, and the client still refuses the response,
          because it no longer matches the contract.
      - lines: "66"
        note: >
          "auth-login".toLocaleUpperCase("tr") is "AUTH-LOGİN", with a dotted
          capital İ. The server's 422 names the member, #/code, and the client
          passes the errors array through untouched.
  - id: hooks
    file: examples/03_hooks.tsx
    title: "Query keys and an infinite query that follows nextCursor"
    runtime: ts-browser
    check: exact
    timeout_s: 15
    annotations:
      - lines: "12"
        note: >
          The fake's cursor is readable to keep it short. TraceLite's is
          base64url of the last sort key, and the client never looks inside:
          it hands back what the server sent.
      - lines: "19-22"
        note: >
          TraceLite's key scheme: keys start with the path of the resource they
          cache, ["projects", id, ...] for everything inside a project, and
          ["requirements", id] for a single requirement. A prefix then names a
          whole family of cached queries.
      - lines: "24-31"
        note: >
          useInfiniteQuery stores pages, not items. initialPageParam null means
          "no cursor", and getNextPageParam hands the server's nextCursor back
          unchanged; null or undefined there means there is no next page, so
          hasNextPage becomes false and the button disappears.
      - lines: "40"
        note: >
          The pages are flattened for display. Each page keeps its own cursor in
          the cache, which is what fetchNextPage continues from.
      - lines: "73-76"
        note: >
          Invalidating ["projects", id] matches the list's key by prefix. An
          infinite query refetches every page it has loaded, in order from the
          first, so AUTH-2FA appears where the server's code order puts it; the
          client never guesses where a new item sorts.
  - id: form
    file: examples/04_form.tsx
    title: "A create form: a mutation that invalidates, server errors under their field"
    runtime: ts-browser
    check: exact
    timeout_s: 15
    annotations:
      - lines: "27-31"
        note: >
          After a successful create, the project prefix is invalidated: the
          list of example 3 and the requirement tree of step 6 both refetch.
          In the app this mutation is the hook useCreateRequirement in
          queries.ts, next to useRequirements.
      - lines: "32-35"
        note: >
          The form's errors come straight from the problem: each errors entry
          has a JSON Pointer, and the field whose name it ends with shows it.
          Server rules stay on the server; the client renders them.
      - lines: "41-42"
        note: >
          Codes are normalized with toUpperCase(), which is locale-independent.
          toLocaleUpperCase() without an argument uses the browser's locale, and
          in Turkish turns i into İ, which the server rejects (example 2).
      - lines: "71-73"
        note: >
          Harness: the lesson's sandboxed iframe doesn't allow form submission,
          and the HTML spec returns before firing the submit event in that
          case. Dispatching the event is what a click on the Ekle button would
          trigger.
      - lines: "79-80"
        note: >
          "auth-login" becomes AUTH-LOGIN, an existing code: 409, shown under
          the code input. The second code is new, and its i stays a plain I.
use_cases:
  - "Every TraceLite screen: lists, the tree, forms and the trace matrix read data through these hooks only"
  - "Forms that show server-side validation per field, from the problem's errors array"
  - "Infinite scroll and 'load more' over keyset-paginated endpoints"
  - "Catching contract drift in development and tests, before it reaches a component"
strengths:
  - "One spec, two compile-time checks: the Go wire types are tested against it (step 1), the zod schemas must satisfy the generated types"
  - "Branded ids make passing a RequirementId where a ProjectId belongs a compile error, at no run-time cost"
  - "Every failure arrives as an ApiError with a typed problem, including failures from proxies that don't speak problem+json"
  - "TanStack Query turns cursors, pages, refetching and invalidation into a few lines, where a hand-written useEffect would also need race handling"
weaknesses:
  - "The zod schemas are written by hand next to the generated types; satisfies catches missing and mistyped members but not a schema stricter than the spec"
  - "Validating every response costs CPU on large lists; a few thousand items per page is fine, a 50 MB export is not (step 8 streams instead)"
  - "Invalidating a whole project prefix refetches more than strictly changed, and an infinite query refetches every loaded page"
  - "Query keys are untyped arrays; the keys object is the only thing keeping them consistent"
pitfalls:
  - title: "Annotating instead of satisfying"
    detail: "const R: z.ZodType<Generated> = z.object(...) widens the schema to the generated type and drops the brands. satisfies checks the same thing and keeps the precise type."
    bad: "const Requirement: z.ZodType<components[\"schemas\"][\"Requirement\"]> = z.object({ ... })"
    good: "const Requirement = z.object({ ... }) satisfies z.ZodType<components[\"schemas\"][\"Requirement\"]>"
  - title: "Throwing the response body as the error"
    detail: "A 502 from a proxy has an HTML body, and a 204 has none. Parse a problem when there is one, synthesize an about:blank problem from the status when there isn't, and keep one ApiError type."
  - title: "Building cursors on the client"
    detail: "nextCursor is opaque. Passing back the last code, or decoding the base64, couples the client to the server's internals; pass the string back unchanged."
  - title: "Invalidating only the exact key"
    detail: "Creating a requirement changes the list and the tree. Invalidate the project prefix, or the tree shows stale data until it happens to refetch."
    bad: "client.invalidateQueries({ queryKey: keys.requirements(projectId), exact: true })"
    good: "client.invalidateQueries({ queryKey: keys.project(projectId) })"
  - title: "Locale-aware upper-casing of identifiers"
    detail: "toLocaleUpperCase() follows the user's locale; in Turkish, i becomes İ. Use toUpperCase() for codes, and keep locale rules for text people read."
quiz:
  - q: "The spec adds a required member `owner` to Requirement and the types are regenerated. What fails first?"
    options: ["Nothing until a response is parsed", "tsc, on the satisfies of the Requirement schema", "The Go server"]
    answer: 1
    why: "The zod schema's output no longer has every member of the generated type, so the satisfies check is TS1360 at compile time."
  - q: "The server answers 502 with an HTML page. What does request() throw?"
    options: ["A SyntaxError from res.json()", "A ZodError", "An ApiError with an about:blank problem and status 502"]
    answer: 2
    why: "The body is read with a catch that yields null; the problem schema doesn't match null, so the client builds a problem from the status."
  - q: "getNextPageParam returns null. What happens?"
    options: ["fetchNextPage requests page 2 with cursor=null", "hasNextPage is false and no request is made", "The query errors"]
    answer: 1
    why: "TanStack treats null and undefined from getNextPageParam as 'no next page'."
  - q: "Why does the mutation invalidate [\"projects\", id] instead of [\"projects\", id, \"requirements\"]?"
    options: ["Prefix matching is required for infinite queries", "So the requirement tree and any other project view refetch too", "It is faster"]
    answer: 1
    why: "invalidateQueries matches by prefix; the project key covers the flat list, the tree and future project-level queries."
further_reading:
  - title: "TypeScript Handbook: the satisfies operator (TS 4.9 release notes)"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-4-9.html"
  - title: "TypeScript Handbook: Indexed Access Types"
    url: "https://www.typescriptlang.org/docs/handbook/2/indexed-access-types.html"
  - title: "MDN: Response.json() (static)"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Response/json_static"
  - title: "MDN: Response.statusText"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Response/statusText"
  - title: "MDN: String.prototype.toLocaleUpperCase()"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/String/toLocaleUpperCase"
  - title: "MDN: URLSearchParams"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/URLSearchParams"
---

## Concept

The TypeScript side of the contract lives in `src/api/`, four files with one job each:

- **`schema.d.ts`**: generated by [openapi-typescript](https://openapi-ts.dev) from
  `openapi.yaml`. Types only; never edited by hand.
- **`schemas.ts`**: one zod schema per component the client receives, each followed by
  `satisfies z.ZodType<components["schemas"]["X"]>`. The schemas add what the generated
  types can't: run-time checks and branded ids.
- **`client.ts`**: `ApiError` and `request()`, which parses every success with a schema
  and turns every failure into an `ApiError` carrying a problem; then one function per
  operation.
- **`queries.ts`**: the query-key scheme and one hook per operation (`useRequirements`,
  `useCreateRequirement`), the only way components touch the API.

Components never call `fetch`, never see a raw response and never build a URL or a
cursor. The examples follow the files: example 1 is `schema.d.ts` and `schemas.ts`,
example 2 is `client.ts`, and examples 3 and 4 are the hooks of `queries.ts` in a list
and in a create form.

## Coming from C/C++/Java/Python/JS

- **Java**: OpenAPI Generator's client deserializes into classes and fails on wrong JSON
  types. Here the generated types disappear at run time, so zod does the checking that
  Jackson would have done, plus checks Jackson wouldn't (UUID layout, UTC timestamps).
- **Python**: a pydantic model per response, validated on construction, is the same
  pattern as `schema.parse`; openapi-typescript plays the role of a code generator that
  only writes type hints.
- **Go (step 4)**: the Go side decodes requests strictly and trusts its own responses.
  The client is the mirror image: it builds requests from types and distrusts every
  response.
- **JS**: `fetch(url).then(r => r.json())` cast to an interface is the version this
  replaces. It works until the server changes, and then fails somewhere else.

## How it works

Regenerating the types after the spec changes:

```sh
npx openapi-typescript ../openapi.yaml -o src/api/schema.d.ts
```

TraceLite's TypeScript conventions, used by every later step:

- Ids: `ProjectId`, `RequirementId`, `TestCaseId`, `TestRunId`, each
  `z.uuid().brand<"...">()`; the type is `z.infer` of the schema. A raw string becomes an
  id only by parsing: from a response, or from a route parameter with `.parse()`.
- Resource types come from the zod schemas (`type Requirement = z.infer<typeof
  Requirement>`), so they carry the brands; the generated types are only the yardstick.
- Times stay ISO strings in `Requirement` (`z.iso.datetime()`); components format them
  for display with `Intl.DateTimeFormat("tr-TR", ...)`.
- Every request goes through `request()`. Failures are `ApiError` with `problem:
  { type, title, status, detail?, errors? }`. A form shows `errors` entries by pointer
  (`#/code` belongs to the `code` input).
- Query keys: `["projects", projectId]` for anything inside a project, extended per view:
  `[..., "requirements"]`, `[..., "requirement-tree"]`, `[..., "test-cases"]`,
  `[..., "trace-links"]`; `["requirements", requirementId]` and
  `["test-cases", testCaseId]` for single resources. Mutations invalidate the project
  prefix, plus the single resource they changed.
- Lists use `useInfiniteQuery` with `initialPageParam: null` and `getNextPageParam: (p)
  => p.nextCursor`.

Example 3 pages through an in-memory stand-in. The real list call in `client.ts` sends
the cursor back as a query parameter, untouched, and parses the page like any other
response:

```ts
const RequirementPage = z.object({
  items: z.array(Requirement),
  nextCursor: z.string().nullable(),
});

function listRequirements(projectId: ProjectId, cursor: string | null) {
  const query = new URLSearchParams(cursor ? { cursor } : {});
  const path = `/api/projects/${projectId}/requirements?${query}`;
  return request(RequirementPage, path);
}
```

## Design notes

**Generated types plus hand-written schemas.** A generator that emits zod schemas from
the spec would remove the hand-written half. TraceLite keeps openapi-typescript (types
only, the tool the services module teaches) and writes the schemas, because the schemas
carry decisions the spec doesn't: brands, and the strictness of `z.iso.datetime()`.
`satisfies` keeps the two from drifting apart in the direction that breaks code.

**Validate responses, not just requests.** The server validates what clients send; the
client validates what the server sends. With both, a contract violation shows up at the
boundary where it happened, with a path, on whichever side broke it. The cost is parsing
time, which matters only for very large payloads, and those (exports, step 8) are
streamed files, not JSON pages.

**Server state stays in the cache.** Lists, trees and forms read through hooks; nothing
copies query data into `useState`. After a mutation the client invalidates and lets the
server's answer win, rather than splicing the new item into cached pages; step 9 adds
live updates by invalidating the same keys when a change notification arrives.
