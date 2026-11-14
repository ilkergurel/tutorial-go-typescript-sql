---
id: inside.codebase.contract
title: "The API contract and type generation"
track: inside
module: codebase
order: 7
level: intermediate
minutes: 14
since: go1.18
prerequisites: [ts.services.zod, ts.services.openapi, go.http.json-validation]
compare_with: [ts.services.openapi, ts.narrowing.guards, inside.codebase.ts-runner, inside.codebase.http-layer]
tags: [contract, json, encoding-json, zod, type-guards, problem-details, drift]
summary: >
  There is no openapi.yaml and no generated code here: the run API's contract is the Go
  package runapi, mirrored by hand in types.ts and trusted with a cast. This lesson reads
  where that mirror drifted, how two of the drifts were found and fixed, and what would
  catch the next one.
examples:
  - id: wire
    file: examples/01_wire.go
    title: "The Go half: nil slices become null, and unknown fields pass silently"
    runtime: go
    check: exact
    annotations:
      - lines: "16-21"
        note: >
          The tags are the contract. A plain slice field is always present but
          can be null; a pointer with omitempty is absent when nil, which the
          TypeScript mirror spells sql?: ... The mirror has to get both rules
          right by hand.
      - lines: "39-46"
        note: >
          gorun initializes its slices to []; this copy loop, the shape of
          runexec.fromGorun, starts from a nil slice and appends nothing when
          there is nothing to copy. The producer was right, and the adapter in
          between undid it.
      - lines: "54-57"
        note: >
          Until the fix, a clean Go run and every SQL run said "diagnostics":
          null on the wire, where types.ts promises Diagnostic[]. An empty slice
          is all the fix takes (the "fixed" line). The backend now swaps one in
          inside runapi.Result's MarshalJSON, the one type every producer
          shares; the learning handlers do the same with if rows == nil.
      - lines: "59-62"
        note: >
          This is how httpapi's handleRun decodes: a misspelled field is
          dropped without a word, and the run gets the default timeout.
      - lines: "64-66"
        note: >
          runapi.Decode does exactly this, with a comment saying drift should
          fail loudly in development. Nothing calls it: the backend, cmd/runner
          and runclient all decode leniently.
  - id: boundary
    file: examples/02_boundary.ts
    title: "The TypeScript half: a cast, a type guard and two zod schemas on the real response"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "14-17"
        note: >
          Copied from a real POST /api/run response in local mode (a Go hello
          world), from before the fix. Every server run without compile errors
          or written files carried these two nulls; the same run now sends []
          for both. Any other producer (an older backend, a proxy, a test
          double) can still send this.
      - lines: "19-25"
        note: >
          This is ServerRunner's (await res.json()) as RunResult. The cast
          compiles and checks nothing; the TypeError surfaces wherever the
          field is first used. Nothing in the UI reads files or diagnostics,
          which is why nobody noticed the nulls while the backend sent them.
      - lines: "27-34"
        note: >
          The type guard that runapi.go's package comment used to say the
          frontend has (it was never written; the comment now describes the
          cast). It catches the drift but only answers false: no field name
          and no path, and it is a third hand-written copy of the shape to
          keep in sync.
      - lines: "36-46"
        note: >
          Annotating the schema with the mirror type ties them together: if
          types.ts and the schema disagree, tsc rejects this line. At run time
          zod reports each failing field with its path. zod is already an app
          dependency (ADR-0010), so this costs no new package.
      - lines: "48-56"
        note: >
          The tolerant alternative: accept null and hand the app []. It would
          have kept the UI working against the old backend, but it also hides
          the drift. The fix went into the producer instead; a strict schema in
          development is what would catch the next drift.
  - id: problem
    file: examples/03_problem.go
    title: "The error half: how runclient lost the runner's problem detail"
    runtime: go-server
    check: exact
    annotations:
      - lines: "22-32"
        note: >
          runclient.Run's order of operations before the fix: decode into
          Result first, then look at the status. An RFC 9457 problem body
          decodes into Result without error, because unknown fields are
          ignored, and the detail was never read.
      - lines: "34-46"
        note: >
          Choosing the shape by status first keeps the detail. The runner's
          problem body and runapi.Error have the same fields, so one Decode
          fills it. runclient.Run does this now, with one exception: a 500 is
          the runner itself failing, so it becomes an error and the backend
          answers 502, as it does in local mode.
      - lines: "67-70"
        note: >
          The last hop is ServerRunner in the browser: it shows the problem's
          detail, or the status text when the detail is empty. Before the fix,
          Docker mode turned "ts-test lessons run only in local mode" into "Not
          Implemented", and "server busy" into "Service Unavailable".
use_cases:
  - "Adding a field to RunResult: change runapi.go and types.ts in the same commit, and decide null vs [] vs absent explicitly"
  - "Deciding whether a two-language project needs OpenAPI or code generation, or a contract test is enough"
  - "Reviewing any Go JSON API for nil slices, omitempty on required fields, and lenient decoding"
strengths:
  - "One Go package (runapi) is shared by the backend, cmd/runner, runclient and runexec, so the Go side cannot drift from itself"
  - "Problem details (application/problem+json) are written in one place, httpapi.problem, so every error has the same shape"
  - "The mirror is small (about a dozen types) and readable; there is no generator to install, configure or upgrade"
  - "Empty lists are [] on the wire, normalized in one place (runapi.Result's MarshalJSON), and runapi_test.go fails if any list in a Result encodes as null"
  - "SQL cells travel as *string in Go and string | null in TS, so NULL never looks like an empty string; that part of the mirror is exact"
weaknesses:
  - "Nothing checks types.ts against runapi.go: not tsc, not a test, not the browser at run time"
  - "Go's encoding/json writes nil slices as null and ignores unknown fields unless asked, so drift is silent in both directions. Result's MarshalJSON covers the nulls for the run API only; a new response type has to remember the rule again"
  - "Both fixed drifts shipped first and were found by people reading the code while writing lessons, not by a test or a type check"
  - "Comments describe checks that don't exist (a discriminated union in gorun.go, a strict Decode that nothing calls), which is worse than no comment. runapi.go's promised type guard was the third, corrected since"
pitfalls:
  - title: "Appending into a nil slice and marshaling it"
    detail: "A struct field that starts nil and receives no elements is null in JSON. Initialize it, or set it to an empty slice just before encoding."
    bad: "out := Result{}; for _, d := range in { out.Diagnostics = append(out.Diagnostics, d) }"
    good: "out := Result{Diagnostics: make([]Diagnostic, 0, len(in))}"
  - title: "A MarshalJSON on an embedded type"
    detail: "Methods of an embedded struct are promoted, MarshalJSON included. Give StatementResult a MarshalJSON and StepResult, which embeds it, encodes as the statement alone, losing step, session and the rest. runapi fills in the embedded lists from Result's MarshalJSON instead."
  - title: "Decoding an error body into the success type"
    detail: "With unknown fields ignored, any JSON object decodes into any struct without error. Check the status first, then decode into the shape that status promises."
  - title: "Treating `as` as validation"
    detail: "JSON.parse(...) as RunResult tells the compiler to stop checking. Validate at the boundary (zod, or a guard) at least in development, or accept that the first symptom will appear far from its cause."
  - title: "Believing the comment"
    detail: "runapi.go said the frontend validates with a type guard (corrected since); gorun.go says the TS side mirrors Result as a discriminated union. Neither was true. Check claims about other files against those files."
quiz:
  - q: "A Go struct with no MarshalJSON method has a field Diagnostics []Diagnostic `json:\"diagnostics\"` that is never assigned. What does json.Marshal write?"
    options: ["\"diagnostics\":[]", "\"diagnostics\":null", "Nothing: the key is omitted"]
    answer: 1
    why: "A nil slice encodes as null. [] needs an empty, non-nil slice (runapi.Result's MarshalJSON swaps one in just before encoding); omitting the key needs omitempty."
  - q: "A client posts {\"runtime\":\"go\",\"timeoutSecs\":15} to /api/run. What happens?"
    options: ["400: unknown field", "The field is ignored and the default timeout applies", "The backend maps timeoutSecs to timeoutS"]
    answer: 1
    why: "handleRun uses a plain json.Decoder. Only DisallowUnknownFields, which runapi.Decode sets but nobody calls, would reject it."
  - q: "Which of these catches \"files\": null before the UI uses it?"
    options: ["tsc, because RunResult declares files: WrittenFile[]", "The as RunResult cast in ServerRunner", "A zod schema or type guard applied to the parsed body"]
    answer: 2
    why: "Types are erased at run time, and a cast is an instruction to the compiler, not a check. Only code that runs on the value can see null."
  - q: "Where is this app's openapi.yaml?"
    options: ["server/api/openapi.yaml", "Generated by make gen", "There is none: runapi.go is the source and types.ts a hand-written mirror"]
    answer: 2
    why: "The curriculum, ADR-0001 and docs/ARCHITECTURE.md describe an OpenAPI contract and a generated client, but none was ever written."
further_reading:
  - title: "package encoding/json: nil slices, omitempty, DisallowUnknownFields"
    url: "https://pkg.go.dev/encoding/json"
  - title: "TypeScript Handbook: Narrowing (type predicates)"
    url: "https://www.typescriptlang.org/docs/handbook/2/narrowing.html#using-type-predicates"
  - title: "MDN: Response.json()"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Response/json"
  - title: "MDN: HTTP 501 Not Implemented"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Status/501"
---

## Concept

The curriculum's goal for this lesson is "one openapi.yaml, two languages". That file
does not exist, and neither does any generated code. The contract for the run API, the
one every Run button press goes through, is:

- **`server/internal/runapi/runapi.go`**: Go structs with JSON tags (`Request`, `Result`,
  `StatementResult`, `StepResult`, `SQLStmtError`, `Error`). The backend's handler, the
  Docker runner binary, `runclient` and `runexec` all import it, so the Go side cannot
  disagree with itself.
- **`src/features/runner/types.ts`**: a hand-written TypeScript mirror (`RunRequest`,
  `RunResult`, the `Runtime` union and the rest). Nothing compares it with the Go file.
  `ServerRunner` trusts it with `(await res.json()) as RunResult`.
- **Errors**: RFC 9457 problem details, written by `httpapi.problem` as
  `{ type, title, status, detail }` with `application/problem+json`. `ServerRunner`
  reads `detail`.

The learning endpoints (progress, notes, bookmarks, quiz) follow the same pattern:
structs in `learningstore`, a mirror in `src/features/learning/api.ts`, a generic
`req<T>()` that casts.

Drift found while writing the lessons (the first and the last are fixed now, the second
in part):

1. **`null` where arrays are declared.** A clean Go run, and every SQL run, returned
   `"diagnostics": null, "files": null`; `types.ts` says `Diagnostic[]` and
   `WrittenFile[]`. `gorun` initializes both slices, and `runexec.fromGorun` rebuilds them
   by appending onto nil (example 1). The learning handlers knew this rule and turned
   nil into `[]` before encoding; the run path didn't. The fix is a `MarshalJSON` on
   `runapi.Result` that writes every empty list as `[]` (SQL statements, columns, rows
   and notices too), so every producer is covered, plus `runapi_test.go`, which fails if
   any list in a `Result` encodes as `null`.
2. **A validator that isn't there.** `runapi.go`'s package comment said the frontend
   checks the shape "with a zod-free type guard in Phase 3". No guard was ever written;
   the comment now says the frontend trusts the response with a cast. `gorun.go` still
   says "the TS side mirrors it as a discriminated union on TimedOut/ExitCode"; the TS
   `RunResult` is a plain interface (the union in `types.ts` is `RunState`, a UI state).
3. **A strict decoder nobody calls.** `runapi.Decode` sets `DisallowUnknownFields` "so wire
   drift fails loudly in dev", but every decoder on the path is lenient.
4. **A lost error detail in Docker mode** (example 3). `runclient` now reads the problem
   body whenever the status isn't 200, and its tests pin "server busy" and the ts-test
   refusal.

## Coming from C/C++/Java/Python/JS

- **C/C++**: two programs sharing a header compile against one definition. Here each
  language has its own copy, so this is closer to two teams agreeing a struct layout by
  email.
- **Java**: Jackson fails on unknown properties by default
  (`FAIL_ON_UNKNOWN_PROPERTIES`), and deserialization builds real objects. Go's
  `encoding/json` ignores unknown fields unless you call `DisallowUnknownFields`, and
  TypeScript's `JSON.parse` builds plain objects whose declared type is never checked.
- **Python**: pydantic models validate on parse. A TS interface is the type hint without
  the validator: zod is what brings the pydantic half back.
- **Go**: the encoder rules to remember are nil slice or map → `null`; `omitempty` drops
  `false`, `0`, `""`, nil and empty slices; a pointer with `omitempty` is absent when nil.
  Each one changes the TypeScript type you must write.
- **JS**: `as` is the everyday way to get a typed object out of `fetch`. It works until
  the server changes.

## How it works

How one `runapi` field maps to its mirror, and where each rule shows up in this API:

- `string`, `bool`, `int`/`int64` map to `string`, `boolean`, `number`. An `int64` is
  exact in JavaScript only up to 2^53 - 1; `durationMs` and `rowsAffected` never get
  close.
- `*SQLResult` with `omitempty` maps to `sql?: {...}`: absent for non-SQL runs, never
  `null`.
- `[]StepResult` with `omitempty` maps to `steps?: StepResult[]`: absent when empty.
- `[]Diagnostic` without `omitempty` maps to `Diagnostic[] | null` under encoding/json's
  default rule, and the mirror says `Diagnostic[]`. That gap was the drift.
  `Result.MarshalJSON` closes it on the Go side by never writing `null` for a list, which
  makes `Diagnostic[]` exact.
- `[][]*string` maps to `(string | null)[][]`: a nil pointer is SQL NULL. This one is
  exact, and the SQL lessons depend on it.
- `SQLStmtError.Pos int32` with a comment "0 when it didn't say" maps to `pos: number`:
  the "absent" case is a sentinel value, not `undefined`.

The two mirrors are compiled separately: `go build` sees only Go, `tsc` sees only
TypeScript. Agreement between them is a property of the running system, so the only
things that can check it run: a test that decodes real responses, or validation at the
boundary (example 2).

## Design notes

For one endpoint and about a dozen types, generation (OpenAPI with oapi-codegen and
openapi-typescript, as ts.services.openapi describes) would cost more than it saves. The
boring fixes, in order of value:

1. **Fix the producer.** Done. The empty lists are normalized in `runapi.Result`'s
   `MarshalJSON` rather than in `runexec.fromGorun` and the SQL paths: one place covers
   every producer, including the next one. `runclient` decodes the problem body when the
   status isn't 200. Two small changes, and the mirror became true.
2. **One contract test.** Half done: `runapi_test.go` encodes a `Result` whose every list
   is nil and fails on any `null`. The other half, golden JSON responses that a
   TypeScript type test checks with `satisfies RunResult`, would catch the next drift in
   the other direction.
3. **Validate at the boundary in development.** A zod schema annotated with
   `z.ZodType<RunResult>` in `ServerRunner`, strict in development. It also gives the
   learner a readable error instead of a `TypeError` three components away.
4. **Make the comments true.** Done for `runapi.go`, whose package comment now describes
   the cast instead of a type guard. Still to do: the discriminated-union sentence in
   `gorun.go` (delete it, or write the code it describes), and `runapi.Decode` (call it
   in the handlers, or delete it).

The docs share the drift. ADR-0001 says "the OpenAPI contract stays hand-maintained for
the TypeScript side"; `docs/ARCHITECTURE.md` describes an `openapi-fetch` client, zod
validation at the trust boundary and a multipart `POST /run`; the Makefile's `gen`
target promises "(Phase 3+) sqlc/OpenAPI types". None of this exists. The request body is
JSON, the client is `fetch` plus a cast, and there is no `sqlc` either. When the docs
and the code disagree, the code is what runs; this lesson describes the code.
