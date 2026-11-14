---
id: go.http.json-validation
title: "JSON helpers, validation, problem-details errors (RFC 9457)"
track: go
module: http
order: 2
level: intermediate
minutes: 14
since: go1.27
prerequisites: [go.http.handlers, go.errors.is-as-join]
compare_with: [ts.services.zod, go.files-io.csv-json, inside.codebase.http-layer]
tags: [http, json, validation, problem-details, utf-8]
summary: >
  Decode request bodies strictly (size cap, unknown fields, one value), validate
  business rules at the edge, and answer failures as application/problem+json.
  Go 1.27's encoding/json/v2 makes the strict part the default.
examples:
  - id: strict-decode
    file: examples/01_strict_decode.go
    title: "A strict decode helper and what each bad body gets back"
    runtime: go-server
    check: exact
    timeout_s: 10
    annotations:
      - lines: "15"
        note: >
          MaxBytesReader caps what the server will read. Past the limit, Read
          returns *http.MaxBytesError, and a real server also closes the
          connection after the response, so a client can't stream gigabytes
          into a JSON endpoint. A plain io.LimitReader would just truncate and
          look like malformed JSON.
      - lines: "16"
        note: >
          By default encoding/json ignores keys it doesn't know, so a client's
          typo ("agee") silently becomes the zero value. Rejecting unknown
          fields turns that into a 400 the client developer sees immediately.
      - lines: "20-22"
        note: >
          Decode reads one value and stops; the rest of the body is never
          looked at. A second Decode must hit io.EOF, otherwise the body held
          more than one value. Trailing whitespace is fine.
      - lines: "27-37"
        note: >
          The one place decode errors become statuses. Messages are written
          for the client: the raw UnmarshalTypeError text names Go structs
          and types, an internal detail. A body over the limit is 413, not
          400. The remaining errors describe only the client's own input, so
          they pass through. errors.As fits a switch; errors.AsType returns
          two values and doesn't.
      - lines: "44-48"
        note: >
          The handler states intent; the helpers carry the rules. Every
          endpoint that takes JSON calls the same two functions, so the API
          rejects bad input the same way everywhere.
      - lines: "58"
        note: >
          null decodes without error and leaves the struct at its zero value
          ("hello  (0)"). Decoding proves the syntax, not the content: that is
          what validation in example 3 is for. Strict v1 also accepts
          duplicate and differently cased keys; example 2 shows why that
          matters and what v2 does instead.
      - lines: "61-62"
        note: >
          A ResponseRecorder calls the handler directly, with no server and no
          socket; the decode errors are the same ones a real server would
          see. Lesson go.http.cors-cookies-httptest covers this way of
          testing.
  - id: v1-vs-v2
    file: examples/02_v1_vs_v2.go
    title: "encoding/json/v2 (Go 1.27): strict by default"
    runtime: go
    check: exact
    annotations:
      - lines: "15-22"
        note: >
          This is the hardened v1 decoder from example 1, still the most
          common form in existing code, including StackLab's own run handler.
      - lines: "25-27"
        note: >
          UnmarshalRead consumes the whole reader and requires exactly one
          value, so the trailing-data check comes built in. RejectUnknownMembers
          is the v2 spelling of DisallowUnknownFields.
      - lines: "28-31"
        note: >
          v2 picks "cannot" or "unable to" for a SemanticError's message at
          random, once per process, so that nobody matches on the text. Branch
          on errors.Is(err, json.ErrUnknownName) or on the error's fields;
          JSONPointer says where in the body the problem is.
      - lines: "37-38"
        note: >
          A parser differential: a proxy that reads the first "name" and a Go
          service that reads the last one disagree about who the request is
          for. v2 rejects duplicate names and matches names case-sensitively.
      - lines: "39"
        note: >
          A client that sends Windows-1254 (or ISO-8859-9) instead of UTF-8
          turns ş into byte 0xFE. v1 replaces it with U+FFFD and stores
          "Ay�e" for good; v2 rejects the body, so the client learns about its
          encoding bug at once.
  - id: validate-problem
    file: examples/03_validate_problem.go
    title: "Validation at the edge, answered as problem details"
    runtime: go-server
    check: exact
    timeout_s: 10
    annotations:
      - lines: "19-31"
        note: >
          RFC 9457 defines type, title, status, detail and instance, and
          allows extension members; clients must ignore members they don't
          know. The errors array with pointer and detail follows the RFC's own
          validation example.
      - lines: "33-42"
        note: >
          Rules that types can't express live in one method. All failures are
          collected rather than returned at the first one. "Gülşen Çağlayıcı"
          is 16 characters but 22 bytes: a len(name) > 20 check would have
          rejected a valid name. PostgreSQL's varchar(20) counts characters
          too.
      - lines: "38"
        note: >
          mail.ParseAddress also accepts "Name <addr>" forms, so the parsed
          address must equal the input. Format checks stop here; whether the
          mailbox exists is a confirmation e-mail's job, not a regex's.
      - lines: "44-48"
        note: >
          The Content-Type header is set before WriteHeader, because headers
          set afterwards are dropped. Problems get application/problem+json,
          so a client can tell an error body from a resource without looking
          at the status first. Taking the status from p itself keeps the
          status member and the real response status from disagreeing.
      - lines: "53-54"
        note: >
          With type "about:blank" the problem means no more than its status
          code, and the RFC says title should then be the status phrase.
          StackLab's httpapi.problem function writes this form for every
          error.
      - lines: "57-60"
        note: >
          422: the JSON was well-formed but the content breaks the rules. 400
          stays for bodies that can't be parsed. A custom type URI names the
          kind of problem, and clients switch on type, never on title or
          detail text.
use_cases:
  - "Every JSON endpoint: one decode helper and one error writer keep a whole API consistent"
  - "Form submissions from a React client that highlights each invalid field from the errors array"
  - "Public APIs where clients in other languages need a documented, machine-readable error format"
  - "Endpoints behind a proxy or gateway, where duplicate keys and case-folding would let two parsers disagree"
strengths:
  - "Struct tags plus a decode helper replace a framework's binding layer; nothing is reflected that you can't see"
  - "Problem details is a standard format, so a client handles every error of every endpoint with one code path, unlike ad-hoc {\"error\": ...} bodies"
  - "MaxBytesReader and *MaxBytesError make the size limit a typed, testable error instead of a truncated body"
  - "Go 1.27's encoding/json/v2 rejects duplicate names, invalid UTF-8 and trailing data by default, without the wrapper code v1 needs"
weaknesses:
  - "v1's defaults are lenient (unknown keys ignored, case-insensitive matching, duplicates accepted, invalid UTF-8 replaced) and each needs explicit handling"
  - "v1's unknown-field error has no exported type; you can only match its text (v2 has json.ErrUnknownName, and scrambles its message wording so nobody matches text)"
  - "No declarative validation in the standard library: unlike zod or Bean Validation, the rules are hand-written code (or a third-party validator)"
  - "A missing field and a zero value decode to the same thing; telling them apart needs pointer fields or json/v2's options"
pitfalls:
  - title: "Reading an unbounded body"
    detail: "json.NewDecoder(r.Body) reads as much as the client sends. Wrap the body in http.MaxBytesReader with a limit per endpoint."
    bad: "json.NewDecoder(r.Body).Decode(&in)"
    good: "json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in)"
  - title: "Counting bytes instead of characters"
    detail: "len(s) is bytes. A 20-character Turkish name can be 30 bytes. Use utf8.RuneCountInString for user-facing limits, and keep byte limits for storage and transport."
  - title: "Echoing internal errors to clients"
    detail: "err.Error() from encoding/json mentions Go struct and type names, and a database error can reveal table names. Map errors to client messages in one place and log the original."
  - title: "Validating before decoding finished"
    detail: "Validate the decoded struct, not the raw map. And remember that a body of null decodes without error, so required-field checks belong in validation, not in the decoder."
  - title: "Choosing the status by habit"
    detail: "400 for bodies you can't parse, 413 for bodies over the limit, 415 for a wrong Content-Type, 422 for well-formed bodies that break the rules. The problem's status member must match the real response status."
quiz:
  - q: "A client sends {\"name\":\"Ayşe\"} {\"name\":\"Eve\"} to a handler that calls dec.Decode(&in) once. What happens with plain encoding/json (v1)?"
    options: ["It fails with a syntax error", "in.Name is \"Eve\"", "in.Name is \"Ayşe\" and the rest is ignored", "It fails with an unknown field error"]
    answer: 2
    why: "Decode reads one value from the stream and stops. Only a second Decode (expecting io.EOF), or v2's UnmarshalRead, detects the trailing value."
  - q: "The body is well-formed JSON but the e-mail is invalid. Which status fits best?"
    options: ["400 Bad Request", "415 Unsupported Media Type", "422 Unprocessable Content", "500 Internal Server Error"]
    answer: 2
    why: "The syntax was fine; the content breaks a rule. 400 is for bodies that can't be parsed at all."
  - q: "What does v1 encoding/json do with {\"name\":\"a\",\"NAME\":\"b\"} decoded into a struct with a `json:\"name\"` field?"
    options: ["Error: duplicate name", "Name is \"a\"", "Name is \"b\"", "Error: unknown field NAME"]
    answer: 2
    why: "v1 matches keys case-insensitively and lets a later duplicate overwrite an earlier one; DisallowUnknownFields doesn't help, because NAME is not unknown. v2 rejects this input."
  - q: "Why should a client branch on the problem's type rather than its title?"
    options: ["title is optional in JSON", "type is the stable identifier; title and detail are human-readable text that may change or be translated", "type is shorter", "title can only be about:blank"]
    answer: 1
    why: "RFC 9457 makes type the primary identifier of the problem kind; title is a human-readable summary."
further_reading:
  - title: "package encoding/json/v2"
    url: "https://pkg.go.dev/encoding/json/v2"
  - title: "package encoding/json: Decoder.DisallowUnknownFields and the v1/v2 differences"
    url: "https://pkg.go.dev/encoding/json"
  - title: "package net/http: MaxBytesReader"
    url: "https://pkg.go.dev/net/http#MaxBytesReader"
  - title: "MDN: 422 Unprocessable Content"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Status/422"
---

## Concept

The edge of an HTTP service is where untrusted bytes become typed values. Three steps,
always in this order:

1. **Decode strictly.** Cap the body size, reject unknown fields, accept exactly one JSON
   value. A failure here is the client's syntax problem: 400, or 413 for size.
2. **Validate.** Check the rules the type system can't express: lengths in characters,
   formats, ranges, cross-field rules. Collect all failures. A failure here is 422.
3. **Answer machine-readably.** RFC 9457 "Problem Details for HTTP APIs" (it obsoletes RFC
   7807) defines a JSON object with `type`, `title`, `status`, `detail` and `instance`,
   served as `application/problem+json`, plus any extension members you need, such as an
   `errors` array of JSON Pointers to the bad fields.

Below the edge, code works with values that are already valid. Services and repositories
don't re-check that an e-mail looks like one.

Go 1.27 ships `encoding/json/v2`. The v1 package is now implemented on top of it with
options that keep v1's historical behavior, and its documentation encourages new code to
use v2, whose defaults are the strict ones.

## Coming from C/C++/Java/Python/JS

- **TypeScript/zod**: `schema.safeParse(body)` decodes and validates in one declarative
  step. In Go, decoding is driven by struct tags and validation is ordinary code, usually a
  `validate()` method. More typing, nothing hidden (lesson ts.services.zod).
- **Java/Spring**: `@Valid @RequestBody` with Bean Validation annotations, and
  `ProblemDetail` (Spring 6) for RFC 7807/9457 responses. Go has no annotations; the
  handler calls the helpers explicitly.
- **Python/FastAPI/pydantic**: a model class validates on construction and FastAPI answers
  422 with a list of field errors automatically. In Go you build that response yourself,
  which also means you choose its format.
- **Jackson (Java)**: a plain `ObjectMapper` fails on unknown properties. v1
  `encoding/json` ignores them, and so does v2 unless you pass `RejectUnknownMembers(true)`.

## Design notes

Write the helpers once per service: a `decodeJSON` (size limit, unknown fields, single
value, error mapping), a `writeJSON` and a `writeProblem`. StackLab's backend has the last
one as `problem(w, status, detail)` in `server/internal/httpapi/router.go`, and its run
handler decodes with `MaxBytesReader` and `json.NewDecoder`, the v1 form of example 1.

For new code on Go 1.27, `json.UnmarshalRead(http.MaxBytesReader(w, r.Body, n), &v,
json.RejectUnknownMembers(true))` gives you the single-value check, duplicate-name
rejection, case-sensitive matching and UTF-8 validation without wrapper code. Moving an
existing API from v1 to v2 is a behavior change for clients, so test it against real
traffic first: a client that sends `"Email"` instead of `"email"` worked yesterday.

Keep validation next to the type it validates, as a method, and keep it free of HTTP: it
returns field errors, and the handler decides that they become a 422. The same method then
serves a CLI import or a queue consumer. Return all field errors at once; a form that
reveals one error per submit is a frustrating form. Finally, never put `err.Error()` from
a lower layer into `detail` unfiltered: log it, and send the client a message written for
the client.
