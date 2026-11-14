---
id: fullstack.tracelite.step-14-testing
title: "Step 14: Testing the stack: Go integration tests, Vitest + MSW, one Playwright e2e test"
track: fullstack
module: tracelite
order: 14
level: advanced
minutes: 15
since: go1.27
prerequisites: [fullstack.tracelite.step-13-audit, go.testing.integration, ts.testing.vitest, ts.testing.msw-playwright]
compare_with: [go.testing.fakes-httptest, go.postgres.repository-testing, go.testing.table-driven, ts.services.zod]
tags: [tracelite, testing, integration-tests, pgx, httptest, contract-test, vitest, msw, testing-library, playwright, e2e, downloads]
summary: >
  Each layer gets the test that can see its bugs: Go integration tests over HTTP against
  real PostgreSQL for rules, constraints and the contract; Vitest with MSW for the
  client's handling of real responses; one Playwright test for what only a browser
  decides.
examples:
  - id: api-integration
    file: examples/01_api_integration_test.go
    title: "A Go integration test: the real router and handler against PostgreSQL, one project per test"
    runtime: go-test
    check: contains
    timeout_s: 15
    annotations:
      - lines: "19-38"
        note: >
          The code under test, cut down to one rule. Whether a parent belongs
          to the same project is decided by the composite foreign key; the Go
          code can't know it without a query, and a fake store would simply
          answer 201.
      - lines: "25"
        note: >
          The real decoder settings, as in step 4: an unknown member is a 400,
          which a test that calls the store directly never sees.
      - lines: "31-32"
        note: >
          The store maps constraint names to answers. Whether that name exists,
          and fires for the rows the test sends, only a real database can tell:
          a migration that renames the constraint breaks this mapping without
          a compile error, and this test with it.
      - lines: "43-56"
        note: >
          TestMain runs once per test binary: the pool, the migrations, the
          routes. In TraceLite it applies the real migrations with goose, so a
          migration that fails on PostgreSQL 18 fails the suite before any
          test runs.
      - lines: "46-49"
        note: >
          This runner hands the test DATABASE_URL, an empty schema of its own.
          TraceLite's suite reads TEST_DATABASE_URL behind the integration build
          tag. pgxpool.New only parses the URL; the first query connects.
      - lines: "58-65"
        note: >
          Isolation by data, not by transaction: the handler commits, so each
          test makes a project of its own (here a fresh project id; TraceLite
          inserts the projects row first). Tests never see each other's rows,
          and a failed test leaves its rows to inspect.
      - lines: "67-71"
        note: >
          The request goes through the router, so the path pattern and
          PathValue are exercised too. TraceLite's suite serves the router with
          httptest.NewServer, which adds a real connection; the recorder runs
          the same routing and handler code without one.
      - lines: "74"
        note: >
          t.Parallel is safe because nothing in the test depends on rows it
          didn't create, and nothing needs cleaning up afterwards.
      - lines: "76-77"
        note: >
          The bug only this layer catches: AUTH exists, but in another
          project. The same request with the parent from the test's own
          project is the control: 201 there, 422 here.
  - id: form-msw
    file: examples/02_client_msw.test.ts
    title: "Vitest + MSW: step 5's client against the answers it will really get"
    runtime: ts-test
    check: contains
    timeout_s: 15
    annotations:
      - lines: "26-40"
        note: >
          Step 5's two boundaries: a failure becomes an ApiError holding a
          problem (made up when the body isn't one), and a success must parse.
          The tests below hit both through real fetch calls.
      - lines: "32"
        note: >
          The one adaptation for Node: jsdom has a location, but Node's fetch
          doesn't resolve relative URLs against it. The browser build keeps
          fetch(path).
      - lines: "37"
        note: >
          safeParse's data is undefined when the body isn't a problem, so ??
          picks the made-up one. Either way the caller gets one shape to show.
      - lines: "50-55"
        note: >
          No default handlers: each test installs the answer it needs, and
          resetHandlers removes it afterwards. MSW intercepts fetch at the
          network layer, so the request is built, serialized and sent exactly
          as in the browser.
      - lines: "51"
        note: >
          A request no handler matches fails the test instead of leaving the
          machine (MSW 3's option name; MSW 2 called it onUnhandledRequest).
      - lines: "57-66"
        note: >
          The server's 409 problem, as step 4 writes it, reaches the caller
          intact, pointer included; the form uses #/code to put the message
          next to its field.
      - lines: "68-73"
        note: >
          What the browser gets when the Go server is down in development: the
          Vite proxy's 502 page, HTML, not a problem. request() makes an
          about:blank problem from the status line instead of crashing on
          JSON.parse.
      - lines: "75-79"
        note: >
          Contract drift in a response: the Go side renamed a member. zod
          refuses it at the boundary, so the bug surfaces as a parse error
          naming parentId, not as an undefined deep in a component. The Go
          contract check (step 1) should have caught it first; this is the
          client's own line of defense.
  - id: download
    file: examples/03_download.spec.ts
    title: "One Playwright test in Edge: the CSV export downloads under its Turkish name, with its BOM"
    runtime: ts-test
    check: contains
    timeout_s: 15
    annotations:
      - lines: "9-12"
        note: >
          A stand-in for the project page. In TraceLite, playwright.config's
          webServer starts cmd/tracelite against a test database and the test
          opens the real page; this runner has no app server, so the test
          serves the page itself.
      - lines: "22-39"
        note: >
          A real HTTP server on a free port, rather than ts.testing's page.route.
          While writing this lesson, the page.route version failed: in
          Chromium-based browsers a download started by a link with the
          download attribute never reaches page.route's handler (Playwright
          issue 22650), so the request went to the network and the file was
          named after the URL. A real server answers with the real
          Content-Disposition, so the test measures the browser, not the
          interception.
      - lines: "47-50"
        note: >
          The download event and its file name are decided by the browser:
          the download attribute, Content-Disposition, RFC 6266's filename*
          with percent-encoded UTF-8. jsdom can't download at all, and no Go
          test can see what Edge does with the header.
      - lines: "52-56"
        note: >
          The saved bytes: the BOM is still there (Excel on Turkish Windows
          needs it, step 8), and so are ';' and CRLF.
use_cases:
  - "Deciding where a new test goes: the layer whose bug it is, not the one that is easiest to test"
  - "Integration tests that hit the real database through the real HTTP handler, in parallel, without cleanup code"
  - "Testing the browser client's error handling (problems, proxy errors, drift) without a running backend"
  - "One end-to-end test for the behavior only a browser has: downloads, focus, real rendering, navigation"
strengths:
  - "The Go test exercises decoding, rules, SQL, constraint names and encoding together, against the same PostgreSQL major version that ships"
  - "Per-test projects make database tests parallel and order-independent, with no truncation between tests"
  - "MSW keeps the client code unmodified, so fetch, headers, JSON parsing and zod run exactly as in the browser"
  - "A single, targeted Playwright test stays fast and stable, unlike an e2e suite that re-tests what lower layers already cover"
weaknesses:
  - "Integration tests need PostgreSQL running; they live behind the integration build tag so go test ./... stays fast without one"
  - "Per-test data accumulates in the test database; drop and recreate it per run (TestMain or the CI job)"
  - "MSW handlers are written by hand and can drift from the server; the Go contract test is what keeps both sides honest"
  - "The e2e test here uses a stand-in page; the real one needs a build, a server and a database in CI, which is minutes, not seconds"
pitfalls:
  - title: "Testing database rules with a fake store"
    detail: "An in-memory fake accepts the cross-project parent, the duplicate code in another casing, the 201-character title: the rules live in constraints it doesn't have. Fakes are for the layers above the store; the store and the handlers that map its errors are tested against PostgreSQL."
  - title: "Rolling back a transaction around a handler test"
    detail: "The handler takes a pool and commits; wrapping the test in a transaction would need the handler to accept that transaction, which production code never does. Give each test its own project instead."
  - title: "Relative URLs in Node tests"
    detail: "Node's fetch throws on fetch('/api/...'), even under jsdom. Resolve against location.href in the test's fetch (or inject a base URL); don't change production code to absolute URLs, which would bring back CORS (step 1)."
  - title: "page.route as a stand-in for download responses"
    detail: "In Chromium-based browsers, Edge included, a download started by a link with the download attribute bypasses page.route: the handler never runs and the request goes to the network (Playwright issue 22650, closed without a fix). Serve downloads from a real HTTP server in the test (or the real app via webServer)."
  - title: "An e2e test per feature"
    detail: "Browser tests are slow and break on timing and markup; a suite that re-checks validation messages and coverage numbers through the browser duplicates the Go and Vitest tests and fails more often. Keep e2e for what only the browser decides."
quiz:
  - q: "A requirement is created with a parent from another project and the API answers 201. Which test should have caught it?"
    options: ["A Vitest test with MSW", "The Go integration test against PostgreSQL", "The Playwright test"]
    answer: 1
    why: "The rule is a composite foreign key and its mapping in the store; only a test that runs the real handler against the real schema sees both."
  - q: "The Go side changes a tag to json:\"parentID\". Which tests fail?"
    options: ["None: the code compiles", "The Go contract check, and the client's zod parse when it meets such a response", "Only the Playwright test"]
    answer: 1
    why: "The contract check compares encoded responses with the spec; zod is the client's own check at the boundary. Nothing at compile time sees a tag."
  - q: "Why does each Go test create its own project?"
    options: ["Projects are cheap to delete", "The handler commits, so tests isolate by data; separate projects let them run in parallel without seeing each other", "pgx requires it"]
    answer: 1
    why: "Rolled-back transactions don't work for code that commits; per-test data does."
  - q: "What does the Playwright test check that the other two can't?"
    options: ["That the CSV has the right columns", "What the browser does with the response: a download named by Content-Disposition's filename*, with the BOM intact", "That the export endpoint returns 200"]
    answer: 1
    why: "Columns and status are Go-testable; the download decision and file name are made by the browser."
further_reading:
  - title: "package testing: TestMain, Parallel, Context"
    url: "https://pkg.go.dev/testing"
  - title: "package net/http/httptest"
    url: "https://pkg.go.dev/net/http/httptest"
  - title: "Go: Build constraints (//go:build)"
    url: "https://pkg.go.dev/cmd/go#hdr-Build_constraints"
  - title: "MDN: Content-Disposition"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Content-Disposition"
  - title: "MDN: <a> download attribute"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/a#download"
---

## Concept

A bug lives in one layer, and the cheapest test that can see it belongs to that layer.
TraceLite has three kinds of tests:

1. **Go integration tests** (`internal/httpapi`, build tag `integration`): the real
   router, handlers, service and store, over HTTP with `httptest`, against PostgreSQL 18
   with the real migrations. Every response is checked against `openapi.yaml`'s
   schemas. Each test creates its own project, so tests run in parallel and need no
   cleanup.
2. **Vitest + Testing Library + MSW** (`web/src/**/*.test.tsx`): components and the API
   client against MSW handlers that answer like the Go server does, including problem
   documents, proxy error pages and drifted responses. Example 2 tests the client;
   components add Testing Library on top, as step 6's matrix test does.
3. **One Playwright test** (`web/e2e/`), in Microsoft Edge, for what only a browser
   decides. TraceLite's is the export download.

Below them, plain unit tests stay where they already are (the tree's `nest` and
`tallies` from step 6, the CSV parser from step 7, the queue's `Finish` with a fake
clock from step 11). Which layer catches which bug:

| Bug | Caught by |
|---|---|
| a rule in Go (code format, title length in characters) | Go unit test of the service, and the integration test |
| a constraint, or its name in the store's mapping (cross-project parent, duplicate code) | Go integration test |
| a migration that doesn't apply, a query that doesn't plan | Go integration test (TestMain runs the migrations) |
| response drift from the spec (omitempty, a tag typo, local time) | Go contract check; then the client's zod parse |
| the client mishandles a 409 problem, a 502 HTML page, a drifted body | Vitest + MSW |
| the matrix shows the wrong state after a failed toggle (step 6) | Vitest + Testing Library |
| tree nesting, coverage roll-up, CSV quoting, retry schedule | Go or Vitest unit tests |
| the export doesn't download, or downloads under a garbled name | Playwright |

## Coming from C/C++/Java/Python/JS

- **Java/Spring**: `@SpringBootTest` with Testcontainers and MockMvc is example 1;
  here the database is the one you run, `httptest` plays MockMvc's part, and `TestMain`
  sets both up, replacing `@BeforeAll` at the suite level. Spring's
  `@Transactional` test rollback is the pattern this lesson avoids for code that
  commits.
- **Python/pytest**: a session-scoped fixture for the database, a function-scoped one
  for the project; `responses` or `respx` mock HTTP the way MSW does for the browser.
- **JS**: Jest + MSW is the same idea as Vitest + MSW; Cypress or Playwright for e2e.
  Playwright's own test runner, not Vitest, drives the browser.
- **C/C++**: a test binary per layer with a fixture process; Go's `TestMain` is the
  fixture, and `go test` builds and runs it.

## How it works

The files:

```text
internal/httpapi/*_integration_test.go   //go:build integration; TEST_DATABASE_URL
internal/httpapi/contract_test.go        loads openapi.yaml, checks every recorded response
internal/tracelite/*_test.go             the service's rules, with a fake store
web/src/**/*.test.tsx                    Vitest, jsdom, Testing Library, MSW handlers in web/src/test/
web/e2e/export.spec.ts                   Playwright, Edge (channel "msedge")
web/playwright.config.ts                 webServer: builds and starts cmd/tracelite on a test database
```

Commands, from the repository root:

```text
go test ./...                                         unit tests, no database needed
go test -tags integration -count=1 ./...              + integration tests (TEST_DATABASE_URL)
npm --prefix web test                                 vitest run
npm --prefix web run e2e                              playwright test
```

Example 1 keeps one handler and one rule; the suite has the same shape with every
route. It serves the router with `httptest.NewServer`, so requests cross a real
connection; the example's `httptest.NewRecorder` runs the same routing and handler code
without one. `contract_test.go` is step 1's contract check turned into a test: every
response the integration tests record is compared with its schema in `openapi.yaml`
(required members present, no undocumented ones, enum values, times ending in `Z`). A
`json:"parentID"` tag or an `omitempty` on `description` fails there, in Go, before any
client sees the response.

`-count=1` turns off Go's test cache, which would otherwise skip integration tests
whose Go code didn't change even though the database did. The integration suite
creates a fresh database per run (`CREATE DATABASE tracelite_test_<pid>` from
TestMain, dropped at the end), so per-test projects never pile up across runs.

The Playwright configuration in the project (not runnable here):

```ts
export default defineConfig({
  testDir: "e2e",
  use: { channel: "msedge", baseURL: "http://127.0.0.1:8081" },
  webServer: {
    // The binary built by the CI step, not `go run`: on Windows, killing go run
    // leaves its child server running (StackLab's scripts/dev.mjs has the story).
    command: process.platform === "win32" ? "..\\bin\\tracelite.exe" : "../bin/tracelite",
    url: "http://127.0.0.1:8081/api/projects",
    env: { TRACELITE_ADDR: "127.0.0.1:8081", DATABASE_URL: process.env.TEST_DATABASE_URL ?? "" },
  },
});
```

## Design notes

**The pyramid, by bug.** The table above decides where a test goes. Most tests are Go
tests because most of TraceLite's rules are in Go and SQL; the client's tests are
about turning answers into screens; the browser test covers the one behavior nobody
else can observe. Adding a second e2e test needs a reason of the same kind: something
the browser decides.

**Real database, isolated by data.** Rolling each test back would be faster, but
TraceLite's handlers commit, and the tests should run the code that ships. A project
per test gives isolation for free, because every table hangs off a project and every
query filters by one (step 2's same-project rule). It also documents that the API
really is project-scoped: a query that forgot its project filter would make parallel
tests interfere and fail.

**Contract checks on both sides.** The Go check compares responses with the spec; zod
compares them again in the browser; MSW handlers are written from the spec too. None of
them is generated from the others, which is the point: a mistake has to be made three
times to slip through.

**What the e2e test found.** The first version of example 3 answered the download with
`page.route`, as ts.testing does for JSON. For the `download`-attribute link the route
handler never ran: the request went past it to the network, Edge named the file after
the URL, and the test failed. Served by a real HTTP server, the same headers gave the
right name. The lesson for test design: an interception layer is part of the system
under test, and for browser-decided behavior it can change the answer.

This is a known Chromium-side gap, not an Edge quirk. Playwright issue
[#22650](https://github.com/microsoft/playwright/issues/22650) reports that downloads
using the anchor's `download` attribute "cannot be mocked"; a maintainer reproduced it
and reported it upstream to Chromium, and it was closed for low activity without a fix;
[#38585](https://github.com/microsoft/playwright/issues/38585) (2025) found the same in
Chromium and WebKit but not Firefox. A probe run through this app's ts-test runtime on
2026-11-14 (Playwright 1.65.0, Edge 157.0.4341.46) counted the route handler's calls
for the export URL: 0 for the `<a download>` link, whose request went to the network
instead, and 1 for the same link without the attribute, which then downloaded as
`Ödeme gereksinimleri.csv`.
