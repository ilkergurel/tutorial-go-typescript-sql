---
id: inside.codebase.testing-pyramid
title: "This repo's test pyramid"
track: inside
module: codebase
order: 8
level: intermediate
minutes: 14
since: go1.22
prerequisites: [go.testing.table-driven, go.testing.fakes-httptest, go.testing.integration]
compare_with: [ts.testing.vitest, ts.testing.msw-playwright, inside.codebase.contract, fullstack.tracelite.step-14-testing]
tags: [testing, unit-tests, httptest, integration-tests, build-tags, vitest, verification]
summary: >
  This repo's tests are shaped less like a pyramid than a top-heavy vase: a thin base of
  unit tests, a thick layer of integration tests against the real toolchains and
  PostgreSQL, and a lesson verifier that runs every example. Handlers, learningstore
  and the UI have no tests; the frontend has three small Vitest files.
examples:
  - id: unit
    file: examples/01_unit_test.go
    title: "The base: a table-driven unit test, after sqlrun/split_test.go"
    runtime: go-test
    check: contains
    annotations:
      - lines: "13-38"
        note: >
          A pure function: string in, slice out, no I/O. That is what makes it
          cheap to test in hundreds of cases. sqlrun's real splitter keeps the
          same property while handling dollar quotes and nested comments.
      - lines: "17"
        note: >
          Ranging over a string yields runes. The Turkish case below is there
          because a byte-wise splitter can cut a multi-byte character in half;
          the real table has an "İ in an identifier before $" row for the same
          reason.
      - lines: "41-50"
        note: >
          Each row is one behavior with a name. Adding a regression is one line,
          which is why the real file has twenty rows for the splitter alone.
      - lines: "51-57"
        note: >
          t.Run gives every row its own name in the output and its own failure.
          Since Go 1.22 each iteration has its own tc, so the closure is safe
          even if the subtest ran in parallel.
  - id: handler
    file: examples/02_handler_test.go
    title: "The missing layer: handleRun tested through httptest with a fake Runner"
    runtime: go-test
    check: contains
    annotations:
      - lines: "20-22"
        note: >
          The seam already exists in the app: httpapi.Runner is declared by its
          consumer, and router.go's comment says handlers stay "testable with
          fakes". Only run_test.go uses it, with a slow fake that checks the
          reply outlives the server's WriteTimeout.
      - lines: "24-44"
        note: >
          A miniature of handleRun: the same order of checks, the same status
          mapping (machinery error 502, refusal passed through) and the same
          no-history header that verify-lessons sends.
      - lines: "46-54"
        note: >
          The fake returns whatever a row puts in it. There is no mocking
          library and no generated code, only a type that has the method.
      - lines: "71-74"
        note: >
          httptest.NewRequest and NewRecorder run the handler in-process with no
          listener and no network. The whole table runs in microseconds, which is
          what the base of a pyramid should cost.
  - id: judge
    file: examples/03_judge.test.ts
    title: "The next frontend unit test: judge.ts in Vitest"
    runtime: ts-test
    check: contains
    annotations:
      - lines: "4-20"
        note: >
          A copy, because a lesson can't import repo files. The real
          src/features/runner/judge.ts is a pure function too, shared by the run
          panel and scripts/verify-lessons.mjs, so a test of it guards both the
          badge and the verifier.
      - lines: "23-31"
        note: >
          it.each is Vitest's table-driven test. The second row documents a
          surprise: judge.ts's comment says trailing whitespace never decides a
          verdict, but only whitespace at the very end of the output is trimmed.
      - lines: "33-35"
        note: >
          A non-zero exit fails every check, including contains. This is the
          rule that makes a crashing example red even when its output happens to
          match.
use_cases:
  - "Deciding which test to write for a change: a splitter rule (unit), a status code (handler), a lock wait (integration), a lesson (verify)"
  - "Running the right suite before committing: go test ./... is seconds; -tags integration needs Go, Node, Edge and TEST_DATABASE_URL"
  - "Planning where tests would pay off most in this codebase: handlers, stores and the frontend"
strengths:
  - "The runners are tested against the real things they drive: the go toolchain, Node, Microsoft Edge and PostgreSQL. A mocked exec.Command would test nothing that matters about a process-tree kill or a deadlock"
  - "verify-lessons judges with the UI's own judge.ts and the real BrowserTsRunner, and runs every example twice, so a flickering 'matches expected' badge is a failure"
  - "Integration tests sit behind //go:build integration, so go test ./... stays fast and needs no database, like a Maven surefire/failsafe split but enforced by the compiler"
  - "Tests name the bug they guard (the goose embed path, the ResponseController unwrap), which makes them read like a changelog of real incidents"
weaknesses:
  - "One handler test, for the run route's write deadline: the 400/502/503 mapping, the history header and every learning endpoint are tested only by hand"
  - "One store test: run history's trim to the newest rows. learningstore's 'progress only moves forward' upsert has none"
  - "Three small Vitest files cover the lesson-markdown parser, the lesson data and the backend probe; components, judge.ts, the runners and the rest of the learning API have none, and no end-to-end test drives the app. Testing Library, MSW and Playwright are used by ts-test lessons only"
  - "No CI: every suite runs only when someone runs it, and the integration suite needs a fully set-up machine"
pitfalls:
  - title: "Pure tests trapped behind the integration tag"
    detail: "A build tag applies to the whole file. gorun_test.go's TestCappedBufferStopsButDoesNotBlock, TestSeedRejectsPathEscape and TestNodeEntry need no toolchain, but go test ./... never runs them. Move such tests to an untagged file."
  - title: "Tagged files that no longer compile"
    detail: "go test ./... and go vet ./... skip files behind a build tag, so a refactor can break the integration suite silently. Run go vet -tags integration ./... as well."
  - title: "Reading verify-lessons as an end-to-end test of the app"
    detail: "It drives verify.html and the backend API, not the lesson page. RunPanel, navigation, progress and notes are outside it."
  - title: "A green go test ./... after a runner change"
    detail: "The runner's behavior lives in integration tests and in verify-lessons. A change to gorun or sqlrun needs go test -tags integration ./... and npm run verify:lessons before it is known to work."
quiz:
  - q: "You change how handleRun maps a refusal to a status code. Which existing test tells you if you broke it?"
    options: ["httpapi/middleware_test.go", "None: no test checks how handleRun maps errors to statuses", "runexec_test.go"]
    answer: 1
    why: "httpapi's tests cover the middleware (logging, recovery) and the request guards around stub handlers, and the real router only for requests no route matches. handleRun's one test (run_test.go) checks its write deadline with a fake Runner that answers 200; the status mapping and the learning handlers have no tests."
  - q: "Why are gorun's timeout tests integration tests rather than unit tests with a mocked exec.Command?"
    options: ["Go cannot mock exec.Command", "What they guard is how real processes and process trees die on Windows and Linux; a mock would only check that a function was called", "Integration tests run faster"]
    answer: 1
    why: "The behavior under test lives in the OS and the toolchain. Mocking it would remove the thing being tested."
  - q: "Which command runs TestCappedBufferStopsButDoesNotBlock?"
    options: ["go test ./...", "go test -tags integration ./internal/gorun/", "npm run verify:lessons"]
    answer: 1
    why: "It lives in gorun_test.go, which starts with //go:build integration, so only a build with that tag compiles it."
  - q: "What does npm run verify:lessons check that the Go tests don't?"
    options: ["That every lesson example runs, twice, through the real runners and matches its expected output", "That the lesson page renders", "That progress is saved"]
    answer: 0
    why: "It is a content-level end-to-end check of the runners. The UI itself has no automated test."
further_reading:
  - title: "package testing (subtests, t.Run)"
    url: "https://pkg.go.dev/testing"
  - title: "package net/http/httptest"
    url: "https://pkg.go.dev/net/http/httptest"
  - title: "go command: build constraints"
    url: "https://pkg.go.dev/cmd/go#hdr-Build_constraints"
  - title: "Go blog: Using Subtests and Sub-benchmarks"
    url: "https://go.dev/blog/subtests"
---

## Concept

The textbook pyramid has many fast unit tests at the bottom, fewer integration tests in
the middle and a handful of end-to-end tests on top. Here is what this repo actually
has, layer by layer.

**Static checks.** `npm run typecheck` (tsc 7), `npm run lint` (ESLint with type-aware
rules), `go vet ./...` (CLAUDE.md asks for it with `GOOS=linux` as well, because gorun
has per-OS files) and contentcheck (`scripts/build-lessons.mjs`: lesson schema, ids,
prerequisites, LEARN markers, official-docs links).

**Go unit tests**, no build tag, `go test ./...`: thirty tests in eleven files.

- `sqlrun/split_test.go`: the statement splitter (twenty named cases, among them
  SQL-standard `BEGIN ATOMIC` bodies that must stay one statement while a plain `BEGIN;`
  still splits), the A:/B: script parser, and the SQL text renderer, including column
  alignment with `İstanbul` and `Ağrı`.
- `sqlrun/guard_test.go`: SQL lessons may not end the runner's transaction
  (`COMMIT`, `ROLLBACK`, `END`) or start `COPY ... FROM STDIN`.
- `gorun/scaffold_test.go`: the pgx lesson module pins the same versions as
  `server/go.mod`; `UsesPgx` detection.
- `gorun/unit_test.go`: the output cap cuts on a character boundary (a `ğ` or `İ` split
  across two writes included), ts-test's time limit, and the color variables it drops
  from the inherited environment.
- `runexec/runexec_test.go`: an unknown runtime is a 501 refusal, not a 502.
- `httpapi/middleware_test.go`: the logging and recovery middleware keep
  `http.ResponseController` working (an `httptest` server with a short `WriteTimeout`);
  a panic becomes a 500 problem reply, or an abort once the handler has written; the
  real router, built with a nil pool, answers unmatched requests with problem 404 and
  405 and keeps `Allow`.
- `runapi/runapi_test.go`: no list in a `Result` encodes as `null`, SQL NULL cells stay
  `null`, and marshaling leaves the value alone.
- `runclient/runclient_test.go`: against an `httptest` runner, refusals keep their status
  and detail ("server busy"), and a runner 500 becomes an error.
- `httpapi/run_test.go`: a run that takes longer than the server's `WriteTimeout` still
  gets its reply, through a fake `Runner` and the real logging middleware.
- `httpapi/security_test.go`: the request guards (ADR-0011), one case per attack next to
  the callers that must keep working (the browser through Vite, verify-lessons, health
  probes), plus `isLoopbackHost`.
- `cmd/stacklab/main_test.go`: goose sees every embedded migration, each has a
  `-- +goose Up` line, and logged connection strings hide their passwords.

**Go integration tests**, `//go:build integration`, `go test -tags integration ./...`:
41 tests in six files, against real tools. `gorun` builds and runs real programs
(the whole process tree dies on timeout, on a cancelled request and after a normal exit
that left a child behind, 64 KB output caps that keep UTF-8 whole, path escapes,
`go test` mode, Node type stripping, offline `x/sync`); its tree tests start the test
binary itself as the lesson program, so they need no compiler and run on Linux too;
`nodetest_test.go` runs Vitest, MSW, a typecheck and Playwright in Edge, without colors
and within `timeout_s`; `sqlrun` and `multisession` run against PostgreSQL through
`TEST_DATABASE_URL` (rollback, NULL vs `''`, `BEGIN ATOMIC` functions, `VACUUM` in
non-transactional mode, the stale-schema janitor, lock waits, a deterministic
deadlock); `runexec` checks
that a pgx lesson's schema is dropped afterwards; `runstore` checks, in a schema of its
own built from the real migration, that run history keeps only the newest rows. The
database tests skip when
`TEST_DATABASE_URL` is unset.

**Lesson verification**, `npm run verify:lessons` (needs `npm start`): every example
of every lesson, run twice, through the running backend's `POST /api/run`, and for
ts-browser through the real `BrowserTsRunner` in headless Edge. Verdicts come from the
UI's own `judge.ts`; annotation line ranges are checked too.

**Frontend unit tests**, `npm test` (`vitest run --dir src`): three files.
`shared/ui/markdownBlocks.test.ts` covers the lesson-markdown parser (pipe tables, HTML
comments); `features/lessons/unverified.test.ts` checks that every lesson's unconfirmed-claim
markers become warning boxes in the right tab, and that the generated lesson index matches
the lesson files; `features/learning/api.test.ts` checks the `/healthz` probe that picks
the backend or the localStorage fallback.

**Missing**: tests for HTTP handlers beyond `run_test.go`'s write deadline (`handleRun`'s
status mapping, the twelve learning routes),
for `learningstore` (`runstore` has one, for the trim), for `config`, for the frontend's components,
runners, `judge.ts` and the rest of the learning API, and an end-to-end test that drives the app's UI.
There is no CI.

## Coming from C/C++/Java/Python/JS

- **Java**: no Mockito. Go tests use hand-written fakes behind small interfaces declared
  by the consumer (go.testing.fakes-httptest); `httptest.NewRecorder` replaces
  MockMvc. The surefire/failsafe split becomes a build tag.
- **Python**: `@pytest.mark.integration` is close to a build tag, with one difference: a
  pytest marker deselects tests that were still imported, while a tagged Go file isn't
  compiled at all in the default run. A compile error in it goes unnoticed until someone
  runs with the tag.
- **JS**: Vitest + Testing Library + MSW + Playwright is the standard stack, and it's
  all installed here, used only by ts-test lessons. Many JS teams prefer a "trophy"
  with integration tests as the widest layer; this repo has that shape on the Go side
  by necessity, and barely a base on the frontend.
- **C/C++**: think GoogleTest for the pure parts and a ctest label for the slow ones;
  the slow ones here start real compilers and a real browser.

## How it works

```sh
cd server
go test ./...                      # unit: seconds, needs only Go
go test -tags integration ./...    # needs go and node on PATH, npm install,
                                   # Microsoft Edge, and TEST_DATABASE_URL
                                   # (the playground database) for sqlrun/runexec
cd ..
npm run typecheck
npm run lint
npm test                           # Vitest over src/
npm run verify:lessons             # needs npm start; add --filter <id prefix>
```

The Makefile's `test` target runs the unit tests plus the integration tests for
`internal/gorun` only. The README mentions `-tags integration` for gorun and sqlrun.

Why the middle layer is the thickest: what gorun and sqlrun promise lives outside Go
code. A process tree dies (or doesn't) on Windows because of a Job Object; a deadlock
forms because PostgreSQL's detector runs after `deadlock_timeout`. A unit test with a
mocked process or a mocked connection would check that the code calls the mock, not
that the promise holds. ADR-0009 makes the same argument for verify-lessons: it runs
the real runners, because re-implementing them for the check "would check a copy of the
runtime rules, not the code learners use".

## Design notes

The shape is right where it is thick and wrong where it is empty. Handlers, stores and
the frontend are exactly the parts where unit and handler tests are cheapest, and
bugs there are found only by clicking. The drift in inside.codebase.contract
(`"files": null`, a lost error detail in Docker mode) was the kind a handler-level test
catches the first time it runs; both were found by reading, and the fixes came with the
tests that would have caught them (`runapi_test.go`, `runclient_test.go`).

What I'd add, cheapest first:

1. **A handler test for `POST /api/run`'s status mapping** with a fake `Runner`
   (example 2). The interface already exists for this, and `run_test.go` already uses
   one for the write deadline. `runstore.Store` is a concrete type in
   `runService`, so recording needs a small consumer-declared interface too.
2. **Store tests behind the integration tag**, on `TEST_DATABASE_URL`, for the
   progress upsert that never downgrades. Run history's trim has one already
   (`runstore/store_integration_test.go`); it shows the pattern: a schema of its own,
   built from the real migration file, dropped in `t.Cleanup`.
3. **Move the remaining pure tests out of `gorun_test.go`** into the untagged
   `unit_test.go`, so the default run covers the cap's write-and-discard behavior and the
   path-escape check too.
4. **More Vitest for the pure frontend functions**, next to the three that exist:
   `judge.ts` (example 3), `bareImports` in `browserTs.ts`, the learning API's
   localStorage adapter. Then `RunPanel` with MSW standing in for `/api/run`, as ADR-0005 already
   promises ("component tests mock the API layer with MSW").
5. **One Playwright smoke test of the app**: open a lesson, press Run, see "matches
   expected". The same headless Edge verify-lessons uses would do.

Two documents promise tests that don't exist: ADR-0005 (component tests with MSW, e2e
tests that drive the URL) and ADR-0003 (a "fork bomb contained" runner test; gorun has
the process-tree kill test, not a fork bomb).
