---
id: fullstack.tracelite.step-04-api
title: "Step 4: The REST API: CRUD, validation, problem-details errors, keyset pagination"
track: fullstack
module: tracelite
order: 4
level: intermediate
minutes: 15
since: go1.27
prerequisites: [fullstack.tracelite.step-03-store, go.http.handlers, go.http.json-validation, go.design.layers, sql.indexes.stats-pagination]
compare_with: [go.services.pagination-streaming, ts.services.openapi, inside.codebase.http-layer, design.architecture.styles]
tags: [tracelite, rest, http, servemux, json-v2, validation, problem-details, rfc9457, keyset-pagination, cursor]
summary: >
  One vertical through TraceLite's API: the handler decodes strictly and maps errors in
  one function, the service holds the rules every entry point shares, the store speaks
  SQL; failures are RFC 9457 problems, and lists page with an opaque keyset cursor.
examples:
  - id: create
    file: examples/01_create.go
    title: "POST a requirement: strict decoding, 404 for a malformed id, 201 with Location"
    runtime: go-server
    check: exact
    annotations:
      - lines: "13-17"
        note: >
          The request body has its own type, with only what a client may send.
          id, status and timestamps aren't in it, so a client can't set them,
          and RejectUnknownMembers turns an attempt into a 400.
      - lines: "18-23"
        note: >
          The wire type, abridged: camelCase names from openapi.yaml, uuid.UUID
          encoded as a string by its MarshalText, a pointer for the nullable
          parentId so it encodes as null, and no omitempty anywhere: every field
          is always present.
      - lines: "34-37"
        note: >
          A path id that isn't a UUID can't name a resource, so it is a 404 like
          an unknown one, and the database is never asked.
      - lines: "39"
        note: >
          encoding/json/v2, on by default in Go 1.27: UnmarshalRead requires
          exactly one JSON value, rejects duplicate names and invalid UTF-8, and
          matches names case-sensitively. MaxBytesReader caps the body at 64
          KiB; the full decode helper from go.http.json-validation answers 413
          when that cap is hit.
      - lines: "44-48"
        note: >
          201 with a Location header naming the new resource, and the resource
          itself as the body, so the client needs no second request. The
          service and the store (examples 2 and 3, step 3) are left out here;
          a fixed id stands in for the one the database assigns.
      - lines: "71"
        note: >
          "titel" is not a member of RequirementCreate. With v1 and no
          DisallowUnknownFields, this would have created a requirement with an
          empty title, or failed later with a confusing message.
      - lines: "72"
        note: >
          v1 matched "Title" to the title field case-insensitively; v2 doesn't,
          so the member is unknown and the body is refused.
  - id: rules
    file: examples/02_rules.go
    title: "The service: normalize, report every broken rule, and turn a store error into a field error"
    runtime: go
    check: exact
    annotations:
      - lines: "11-19"
        note: >
          The service's answer to "what is wrong with this input": JSON
          Pointers into the request body, as in RFC 9457's own example.
          writeError (example 3) puts them into the problem's errors member.
      - lines: "28-35"
        note: >
          The service normalizes (trims the title) and validates every field
          before touching the database, collecting all problems. Titles are
          counted in runes, like the CHECK's char_length; codes are ASCII, so
          len is right there.
      - lines: "39-42"
        note: >
          The store says "parent not in this project"; the service knows that
          this is a fact about the parentId the client sent, so it becomes a
          field error, 422, not a 404.
      - lines: "57"
        note: >
          The title arrives padded and reaches the store trimmed.
      - lines: "58"
        note: >
          Both rules break, both are reported: İ (what
          "auth-login".toLocaleUpperCase("tr") produces) is not A-Z, and a
          title of spaces is empty once trimmed.
      - lines: "59"
        note: >
          200 characters in 400 bytes pass, as they pass the CHECK. A byte
          count, len(title), would have refused them.
  - id: problems
    file: examples/03_problems.go
    title: "writeError: every failure as an RFC 9457 problem, decided in one place"
    runtime: go
    check: exact
    annotations:
      - lines: "12-19"
        note: >
          RFC 9457 problem details. type identifies the kind of problem, title
          is fixed per type, errors is TraceLite's extension member.
      - lines: "35-59"
        note: >
          The one place where errors become statuses. Handlers never choose a
          status for an error themselves, so a new error kind can't come out as
          422 from one endpoint and 500 from another.
      - lines: "42-44"
        note: >
          A duplicate code is a conflict with existing state, 409, but it is
          still about one field, so the problem carries an errors entry and
          the form can mark the code input.
      - lines: "49-51"
        note: >
          Anything unexpected is a 500 with no detail: the cause, which may
          name tables or constraints, goes to the log only (stderr here, which
          the output doesn't show).
      - lines: "53-55"
        note: >
          For about:blank, RFC 9457 says the title should be the status
          code's standard phrase, so it comes from http.StatusText.
      - lines: "64"
        note: >
          errors.Is looks through wrapping, so a store error wrapped with
          context on the way up still maps to 409.
  - id: keyset
    file: examples/04_keyset.go
    title: "GET a list: limit, an opaque keyset cursor, and what a bad cursor gets"
    runtime: go-server
    check: exact
    annotations:
      - lines: "16-19"
        note: >
          Every list in TraceLite has this shape: items, always an array, and
          nextCursor, always present, null on the last page. A client loops
          until nextCursor is null and never builds a cursor itself.
      - lines: "21-30"
        note: >
          A stand-in for step 2's keyset query, so this example needs no
          database: the codes after the cursor, in byte order, at most limit of
          them. In TraceLite the store runs the SQL in the comment.
      - lines: "34-38"
        note: >
          limit defaults to 50 (cmp.Or returns its first argument that isn't
          the zero value) and is capped at 200. A query parameter isn't
          content, so a bad one is 400, not 422.
      - lines: "39-46"
        note: >
          The cursor format: the last item's sort key as a JSON array, in
          base64url without padding, so it needs no escaping in a query string.
          JSON keeps types (a timestamp keeps its microseconds); decoding into a
          Go array of the right length rejects a key of the wrong shape. No
          cursor means "", which sorts before every code, so the first page is
          the same query as every other page.
      - lines: "47"
        note: >
          One row more than the page size. In SQL the extra row costs one index
          entry and replaces a count(*) over the whole project.
      - lines: "49-54"
        note: >
          If the extra row came back, there is a next page, and the cursor is
          the sort key of the last row actually returned, not of the extra one.
      - lines: "56"
        note: >
          json/v2's MarshalWrite. Unlike v1, a nil slice encodes as [] by
          default (FormatNilSliceAsNull restores null), so the empty page after
          the last code has "items":[] as the spec says.
      - lines: "73"
        note: >
          The second and third cursors are the nextCursor of the response before
          them. The fourth, WyJaIl0, is ["Z"]: it sorts after every code, so the
          page is empty.
      - lines: "74"
        note: >
          base64url of "not json", then of ["AUTH","AUTH-LOGIN"], two values
          for a one-value key. Both get the same answer: the server doesn't
          explain what it expected, because clients aren't meant to construct
          one. The cursor is readable to anyone who decodes it; opaque means
          clients must not depend on it, not that it is secret.
use_cases:
  - "Every resource endpoint in TraceLite: projects, requirements, test cases, links and runs follow these status codes, problem types and page shapes"
  - "Form submissions that show all field errors at once, from the errors array"
  - "Lists that a user scrolls or a client iterates: keyset pages stay fast and stable while rows are added"
  - "A second entry point (the CSV import in step 7, a CLI) that reuses the service's validation unchanged"
strengths:
  - "The standard library's ServeMux (Go 1.22 patterns) routes methods and path parameters without a framework, and json/v2 makes strict decoding the default"
  - "One error-mapping function keeps statuses consistent across endpoints, unlike per-handler `http.Error` calls"
  - "Problem details let the TypeScript client handle every failure of every endpoint with one parser and one ApiError type"
  - "Keyset pages cost the same on page 1 and page 1,000 and never repeat or skip rows when others insert, unlike OFFSET"
weaknesses:
  - "The Go wire types are a hand-kept copy of the spec; nothing but a test (step 1's contract check, step 14) notices drift"
  - "Three layers mean three shapes of the same data (row, model, wire) and mapping code between them"
  - "Keyset pagination can't jump to page 37 or show 'page 3 of 12' without an extra count"
  - "Only one sort order per list: another order needs another index and another cursor shape"
pitfalls:
  - title: "Choosing statuses inside handlers"
    detail: "Each handler deciding its own status drifts: the same duplicate is 400 here and 409 there. Return errors and map them in writeError."
    bad: "http.Error(w, err.Error(), http.StatusBadRequest)"
    good: "writeError(w, err) // the one switch from error to problem"
  - title: "Matching JSON names case-insensitively by habit"
    detail: "encoding/json (v1) matched \"ID\" to a field without a tag; json/v2 doesn't. Every wire field gets an explicit json tag in camelCase."
    bad: "var created struct{ ID string } // v2 leaves it empty for {\"id\": ...}"
    good: "var created struct { ID string `json:\"id\"` }"
  - title: "Building the cursor from the extra row"
    detail: "With limit + 1, the last row fetched is not returned. Encode the cursor from items[limit-1], or the next page skips a row."
  - title: "Echoing database errors"
    detail: "err.Error() from pgx names constraints and tables. Map known errors to problems and log the rest; a 500 carries no detail."
  - title: "Times in the server's zone"
    detail: "pgx returns timestamptz in the process's local zone, so on a machine in Turkey the JSON would say +03:00. Convert to UTC when mapping to wire types; the client's z.iso.datetime() accepts only Z."
quiz:
  - q: "The body is valid JSON but the title is only spaces. What does TraceLite answer?"
    options: ["400 about:blank", "422 with an errors entry for #/title", "201, the database accepts an empty title"]
    answer: 1
    why: "Decoding succeeded, the content breaks a rule: 422 with the validation problem type. The service trims, then counts characters."
  - q: "POST /api/projects/{projectId}/requirements names a parentId that belongs to another project. Which status?"
    options: ["404: the parent doesn't exist", "409: conflict", "422: a field of the body is invalid"]
    answer: 2
    why: "The URL's project exists; the problem is the parentId field. The composite foreign key refuses it and the service reports #/parentId."
  - q: "The page size is 2 and the query returns 3 rows. Which code goes into nextCursor?"
    options: ["The third row's", "The second row's", "None: there is no next page"]
    answer: 1
    why: "The third row only proves a next page exists. The next page must start after the last row the client actually received."
  - q: "A list endpoint gets ?cursor=<base64 of [\"AUTH\",\"AUTH-LOGIN\"]> where the sort key has one value. What happens?"
    options: ["The extra value is ignored", "400: decoding into [1]string fails", "500"]
    answer: 1
    why: "json/v2 requires a JSON array to have exactly as many elements as the Go array it decodes into; the handler answers 400 for any cursor it can't decode."
further_reading:
  - title: "package net/http: ServeMux patterns"
    url: "https://pkg.go.dev/net/http#ServeMux"
  - title: "package encoding/json/v2"
    url: "https://pkg.go.dev/encoding/json/v2"
  - title: "package encoding/base64: RawURLEncoding"
    url: "https://pkg.go.dev/encoding/base64#pkg-variables"
  - title: "MDN: 422 Unprocessable Content"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Status/422"
  - title: "MDN: 409 Conflict"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Status/409"
  - title: "PostgreSQL docs: LIMIT and OFFSET"
    url: "https://www.postgresql.org/docs/current/queries-limit.html"
---

## Concept

A request to TraceLite's API passes three layers, each in its own package:

- **`internal/httpapi`**: routes (`ServeMux` patterns), decodes the body strictly into a
  wire type, calls the service, maps every error to a problem in `writeError`, encodes
  the response. The wire types live here, as a hand-kept mirror of `openapi.yaml`.
- **`internal/tracelite`**: the service. Rules every entry point shares: normalization,
  field validation, translating store errors that are really about a field. The HTTP API
  and step 7's CSV import call the same methods.
- **`internal/store`**: step 3. SQL in, store errors out.

Failures are RFC 9457 problem details, served as `application/problem+json`. Lists are
pages with an opaque keyset cursor. Both shapes are the same for every resource, so the
TypeScript client (step 5) handles them once.

The examples take one vertical apart, a piece each: the handler's decoding (example 1),
the service's rules (example 2), `writeError` (example 3) and a list endpoint (example 4).
The store is step 3's, so the examples stand in for it rather than repeat it.

## Coming from C/C++/Java/Python/JS

- **Spring**: `@RestController` + `@Valid` + `@ControllerAdvice` returning `ProblemDetail`
  is the same layering, assembled by annotations. Here the handler calls the decoder and
  the service explicitly, and `writeError` is the `@ControllerAdvice`.
- **FastAPI**: pydantic validates on decode and answers 422 automatically, with its own
  error format. TraceLite returns RFC 9457 instead, and validates rules in the service so
  the importer gets them too.
- **Express**: an error-handling middleware `(err, req, res, next)` is the equivalent of
  `writeError`; the difference is that Go handlers return after writing the error, with
  no `next()` to forget.
- **Django REST Framework**: `CursorPagination` produces the same opaque, base64-encoded
  cursors; its `next` is a full URL, TraceLite's `nextCursor` is just the token.

## How it works

TraceLite's API conventions. Every later step follows them.

**Paths.** Everything is under `/api`. Collections that belong to a project are nested
under it; a single resource has a top-level path by its id:

- `GET|POST /api/projects`, `GET /api/projects/{projectId}`
- `GET|POST /api/projects/{projectId}/requirements` (list ordered by `code`)
- `GET /api/projects/{projectId}/requirement-tree` (whole tree, depth first, with `depth`)
- `GET|PATCH|DELETE /api/requirements/{requirementId}`
- `PUT /api/requirements/{requirementId}/parent` with `{"parentId": "<uuid>" | null}` (move)
- `GET|POST /api/projects/{projectId}/test-cases`, `GET|PATCH|DELETE /api/test-cases/{testCaseId}`
- `GET /api/projects/{projectId}/trace-links` (the matrix)
- `PUT|DELETE /api/requirements/{requirementId}/test-cases/{testCaseId}` (link, unlink; idempotent)
- `GET|POST /api/test-cases/{testCaseId}/runs` (newest first)

**JSON.** camelCase member names (`parentId`, `createdAt`); SQL stays snake_case and the
mapping happens in Go struct tags. Ids are UUID strings. Times are RFC 3339 in UTC with
`Z`: pgx returns `timestamptz` in the server's local zone, so `httpapi` calls `.UTC()`
when it maps the store's model to the wire type (step 1's example shows the difference). Every member of a resource is always present: absent values are `null`, empty lists
are `[]`, empty text is `""`. Enum values are lowercase strings (`draft`, `passed`).
`PATCH` bodies contain only the members to change, and `null` is not accepted in them;
moving is its own endpoint, so "absent" and "null" never have to be told apart.

**Statuses.** 200 for reads and updates, 201 with `Location` for creates, 204 for
deletes and for link and unlink. Errors:

- 400 `about:blank`: the body can't be decoded (syntax, wrong JSON type, unknown member,
  malformed UUID inside it) or a query parameter is invalid (`limit`, `cursor`).
- 404 `about:blank`: the id in the path is malformed or names nothing, including the
  project of a nested list: an unknown project gets a 404, not an empty page, so a typo
  in a URL is noticed.
- 409 `https://tracelite.example/problems/conflict`: the request contradicts current
  state: duplicate code (with `errors` on `#/code`), a requirement that still has
  children, a test case that has runs.
- 413 `about:blank`: the body is over the endpoint's limit (64 KiB for JSON).
- 422 `https://tracelite.example/problems/validation`: well-formed content that breaks a
  rule, with every failing field in `errors`.
- 500 `about:blank`: anything else, with no `detail`; the cause is logged.

**Problems.** `{"type", "title", "status", "detail"?, "errors"?}` where each error is
`{"pointer": "#/field", "detail": "..."}`. `title` is the status text for `about:blank`,
`Validation failed` and `Conflict` for the two TraceLite types. Clients branch on
`status` and `type`, never on text.

**Pages.** `?limit=` (1 to 200, default 50) and `?cursor=`. The response is
`{"items": [...], "nextCursor": "..." | null}`. The cursor is base64url (no padding) of a
JSON array holding the last item's sort key, in `ORDER BY` order: `["AUTH-LOGIN"]` for
lists by code, `["2026-11-13T06:30:00.123456Z", "<run id>"]` for runs newest first. The
server fetches `limit + 1` rows to know whether there is a next page.

## Design notes

**Why a service layer here.** go.design.layers warns against pass-through services.
TraceLite's service earns its place with one concrete reason: the CSV import (step 7) is
a second entry point that must apply exactly the same normalization and validation as
the API. Rules that only the database can check (uniqueness, same project) stay in the
schema and arrive as store errors; the service only decides which of those are field
errors.

**Validation in Go and in SQL.** The service checks formats and lengths to report every
field at once with a useful message; the CHECK constraints catch any writer that skips
the service. Keep the two in step: both count characters, both use the same code
pattern.

**Package choice.** `encoding/json/v2` is the default for new Go 1.27 code, as the JSON
lesson recommends: strict decoding without wrapper code and `[]` for nil slices. The
standard library's `ServeMux` covers TraceLite's routes, like StackLab's own `httpapi`.
A router, a problem-details package or a validation library were not added: each would
be a dependency to explain for a few dozen lines of code.

**Cursors.** Opaque, so the server can change what's inside without breaking clients;
base64url of JSON rather than a home-grown string format, so typed values survive and a
malformed one fails to parse. Not signed: a forged cursor can only start a page at a
different place in a list the caller may read anyway. If that ever stops being true,
sign it with an HMAC.
