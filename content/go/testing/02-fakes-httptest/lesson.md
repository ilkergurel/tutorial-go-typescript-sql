---
id: go.testing.fakes-httptest
title: "Fakes behind small interfaces; httptest; golden files"
track: go
module: testing
order: 2
level: intermediate
minutes: 13
since: go1.27
prerequisites: [go.testing.table-driven, go.interfaces.implicit, go.http.handlers]
compare_with: [ts.testing.msw-playwright, go.http.cors-cookies-httptest, go.postgres.repository-testing]
tags: [testing, fakes, interfaces, httptest, golden-files]
summary: >
  Test against a hand-written fake that satisfies a one-method interface the
  consumer declared, drive handlers through httptest, stand in for upstream
  APIs with a test server, and keep long expected output in golden files.
examples:
  - id: fake
    file: examples/01_fake_test.go
    title: "A hand-written fake behind a consumer-declared interface"
    runtime: go-test
    check: contains
    annotations:
      - lines: "13-17"
        note: >
          The handler declares what it needs: one method. The real HTTP client
          in the next example satisfies it without naming it, so there is
          nothing to generate and nothing to keep in sync.
      - lines: "37-50"
        note: >
          A fake is a small working implementation, not a script of expected
          calls: canned answers in a map, an error to inject, and a slice that
          records calls. Written once, it serves every test that needs a
          RateSource.
      - lines: "61"
        note: >
          The row that needs a spy: invalid input must be rejected before the
          upstream is called. The fake's call log answers that (calls == 0)
          without a mocking library's expect-never-called DSL.
      - lines: "65"
        note: >
          A fresh fake per subtest, so no row sees another row's calls or
          injected error.
      - lines: "66-67"
        note: >
          ResponseRecorder plus NewRequest runs the whole mux, routing and
          PathValue included, in-process: no socket, no goroutine, microseconds
          per case.
  - id: upstream
    file: examples/02_upstream_test.go
    title: "The real client against a fake upstream: httptest.NewTestServer (Go 1.27)"
    runtime: go-test
    check: contains
    annotations:
      - lines: "16-20"
        note: >
          Inject the *http.Client, not just a URL. Production passes one with
          timeouts; a test passes the test server's client. The client code
          itself is the code under test, unchanged.
      - lines: "49-61"
        note: >
          The fake upstream is an ordinary handler with canned answers per
          path. It can also assert on what it receives; here it logs the
          request, which is safe from the server's goroutine while the test
          is running.
      - lines: "62"
        note: >
          NewTestServer (Go 1.27) serves over an in-memory network, registers
          its own shutdown with t.Cleanup and fails the test if the handler
          panics. Before 1.27: srv := httptest.NewServer(h), defer srv.Close(),
          and BaseURL: srv.URL.
      - lines: "64-65"
        note: >
          The production base URL stays as it is: the server's client sends
          every host to the fake, and https:// works over the in-memory TLS.
          The log lines show the request arriving for rates.example.com.
      - lines: "67-78"
        note: >
          Each upstream answer maps to one behaviour: 200 to a value, 404 to the
          ErrUnknownCurrency sentinel callers can test with errors.Is, 503 to
          an error that names the status.
  - id: golden
    file: examples/03_golden_test.go
    title: "A golden file: compare whole output, regenerate with -update"
    runtime: go-test
    fixtures: [fixtures/testdata/TestReceipt.golden]
    check: contains
    annotations:
      - lines: "22-32"
        note: >
          Aligned text is exactly what golden files are for. tabwriter counts
          runes, not bytes, so Ç and ş align; it assumes one column per code
          point, so decomposed text (s plus a combining cedilla) would not.
      - lines: "36"
        note: >
          A custom flag in the test file. go test passes flags it doesn't know
          to the test binary, so go test -run TestReceipt -update rewrites the
          file instead of comparing.
      - lines: "45"
        note: >
          go test runs the binary in the package directory, and the go tool
          ignores directories named testdata, so testdata/ is the place for
          fixtures. Naming the file after t.Name() keeps one file per test.
      - lines: "46-50"
        note: >
          With -update the test writes the file and then compares against what
          it just wrote, so it passes. The review happens in the version
          control diff, which is the point: a human approves the new output.
      - lines: "55-58"
        note: >
          On a mismatch, print both versions and how to regenerate. The lesson
          text shows a real failure after one price changed.
use_cases:
  - "Services and handlers whose dependencies (stores, API clients, clocks, mailers) sit behind one- or two-method interfaces declared where they are used"
  - "HTTP clients for third-party APIs: run the real client code against a test server that returns 200s, 404s, 503s and malformed JSON"
  - "Golden files for output that is long or formatted: rendered templates, reports, generated code, JSON responses, CLI help text"
  - "Asserting a dependency was not called (validation before I/O) with a call log in the fake"
strengths:
  - "Implicit interfaces let the test define the seam: no mockgen or Mockito, no generated code to regenerate when an interface changes"
  - "A fake is reusable, readable Go that behaves like the real thing, so tests assert on outcomes instead of on call sequences that break on every refactor"
  - "httptest is standard library: recorder, in-memory server (Go 1.27) and loopback server cover what Java needs MockMvc and WireMock for, or JS needs supertest and MSW"
  - "Golden files make a behaviour change visible as a reviewable diff, like Jest snapshots, with the update step made explicit by a flag"
weaknesses:
  - "A fake encodes your belief about the dependency; when the belief is wrong, tests stay green (test the real store against PostgreSQL, see go.postgres.repository-testing)"
  - "Fakes are code to maintain; a large interface makes them large, which is a reason to keep interfaces small"
  - "Golden files are easy to update without reading the diff, which turns them into a record of whatever the code does"
  - "go test ./... -update passes -update to every package's test binary, and packages that don't define it fail with \"flag provided but not defined\""
pitfalls:
  - title: "Producer-side interfaces mirroring the whole type"
    detail: "A 15-method RateService interface next to its only implementation makes every fake implement 15 methods. Declare the one or two methods the consumer calls, in the consumer's package."
  - title: "Mocks that assert the call sequence"
    detail: "Expecting Rate(\"USD\") exactly once, before Save, tests the implementation's choreography. Refactor it (cache, batch) and the test breaks although the behaviour didn't. Assert outcomes; check calls only where they are the behaviour."
  - title: "Hard-coding the base URL inside the client"
    detail: "A client that builds requests against a constant URL with http.DefaultClient can only be tested by intercepting the network. Take the *http.Client (and the base URL) as fields."
  - title: "Golden files and Git line endings on Windows"
    detail: "With core.autocrlf=true, Git checks the golden file out with CRLF and the byte comparison fails on Windows only. Exempt golden files from line-ending conversion in .gitattributes, or normalize line endings before comparing."
    good: "# .gitattributes\ntestdata/*.golden -text"
quiz:
  - q: "Where should the RateSource interface be declared?"
    options: ["In the package that consumes it, with only the methods it calls", "Next to the HTTP client that implements it", "In a shared interfaces package"]
    answer: 0
    why: "Go interfaces are satisfied implicitly, so the consumer can declare the minimal set it needs and the implementation never has to know."
  - q: "What does a client from httptest.NewTestServer(t, h).Client() do with a request to https://rates.example.com/v1/rates/USD?"
    options: ["Fails: the host doesn't resolve", "Sends it to the test server over the in-memory network", "Sends it to the real rates.example.com"]
    answer: 1
    why: "The in-memory network directs HTTP and HTTPS requests to the test server regardless of host, so production base URLs can stay unchanged."
  - q: "You changed the receipt format on purpose and TestReceipt fails. What is the golden-file workflow?"
    options: ["Edit the expected string in the test", "Delete the golden file", "Run go test -run TestReceipt -update, then review the diff of testdata/TestReceipt.golden before committing"]
    answer: 2
    why: "The flag regenerates the file; the version-control diff is where a person confirms the new output is right."
  - q: "What can a hand-written fake with a calls slice do that a ResponseRecorder can't?"
    options: ["Check the status code", "Show whether the handler called its dependency, and with what", "Run the router"]
    answer: 1
    why: "The recorder captures the response; only the dependency side can tell you what the handler asked for."
further_reading:
  - title: "package net/http/httptest"
    url: "https://pkg.go.dev/net/http/httptest"
  - title: "Go 1.27 release notes: net/http/httptest"
    url: "https://go.dev/doc/go1.27"
  - title: "Effective Go: Interfaces and other types"
    url: "https://go.dev/doc/effective_go#interfaces_and_types"
  - title: "Go Code Review Comments: Interfaces"
    url: "https://go.dev/wiki/CodeReviewComments#interfaces"
  - title: "go test flags (custom flags, testdata, working directory)"
    url: "https://pkg.go.dev/cmd/go#hdr-Testing_flags"
---

## Concept

Go tests replace dependencies with **fakes**: small, hand-written implementations of an
interface that the code under test declares itself. Because Go interfaces are satisfied
implicitly, the consumer can ask for exactly the one or two methods it calls, and the
production type and the fake both fit without either naming the interface. A fake keeps
state (a map of answers, an injectable error, a log of calls), so tests check outcomes
and, where it matters, what the code asked for.

`net/http/httptest` covers both sides of HTTP:

- `httptest.NewRecorder` and `httptest.NewRequest` call a handler directly, in-process.
- `httptest.NewTestServer(t, h)` (Go 1.27) runs a real server on an in-memory network.
  Its `Client()` reaches it under any host name and it shuts down via `t.Cleanup`. That
  makes it the natural fake for an **upstream API** your client code talks to.
- `httptest.NewServer(h)` listens on a loopback port, for code that needs a real address.

**Golden files** hold the expected output of a test in `testdata/`, compared byte for
byte. A `-update` flag defined in the test regenerates them, and the version-control
diff is where you review the change.

## Coming from C/C++/Java/Python/JS

- **Java**: Mockito creates mocks from classes at runtime and verifies interactions;
  WireMock fakes upstream HTTP services. In Go both are a few lines of ordinary code: a
  struct with a method, and a handler behind a test server.
- **Python**: `unittest.mock.patch` swaps a module attribute behind the code's back. Go
  has no monkey-patching, so dependencies arrive through parameters or struct fields,
  and the seam is visible in the signature.
- **JS/TS**: `jest.mock` replaces modules; MSW intercepts `fetch` at the network layer,
  which is what `NewTestServer` plus an injected `*http.Client` does here. Golden files
  are Jest snapshots with the update step made explicit (`-update` instead of `-u`).
- **C++**: gMock needs virtual methods and a mock class per interface; Go's implicit
  interfaces need neither.

## How it works

One-method interfaces often get a function adapter, like `http.HandlerFunc`:
`type RateFunc func(context.Context, string) (int64, error)` with a `Rate` method that
calls it. A test can then pass an inline closure as the dependency, with no named fake
type. Use it for one-off behaviour; a struct fake is better when several tests share it.

The golden test in example 3 fails on its first run because the file doesn't exist; run
`go test -run TestReceipt -update` once, review the file, commit it. After a deliberate
change to one price, the real failure looks like this:

```
--- FAIL: TestReceipt (0.00s)
    03_golden_test.go:56: output differs from testdata/TestReceipt.golden; if intended, run with -update and review the diff
        got:
        Çay            x2    30,00 TL
        Simit          x3    39,00 TL
        Kaşar peyniri  x1   189,90 TL
        TOPLAM              258,90 TL

        want:
        Çay            x2    30,00 TL
        Simit          x3    37,50 TL
        Kaşar peyniri  x1   189,90 TL
        TOPLAM              257,40 TL
FAIL
```

Custom flags go after the package list (`go test ./receipt -update`) or are run from the
package directory: flags `go test` doesn't know are passed through to the test binary.

## Design notes

Mock frameworks earn their keep where interfaces are large and owned by someone else.
Go code avoids both conditions: interfaces are declared by consumers and stay small, so
a fake is ten lines and reads like the thing it replaces. Prefer fakes that behave
(return what was stored, fail when told to) over mocks that script calls; tests then
survive refactors that keep behaviour.

Know where fakes stop. A fake store tells you nothing about SQL, constraints or
collations, and a fake upstream only knows what you told it about the real API. Test the
real implementations at their boundaries too: the repository against PostgreSQL (see
go.testing.integration), and the HTTP client against real responses recorded once and
kept as fixtures.

Golden files suit output a person can judge at a glance and that would take dozens of
assertions to pin down. Keep them small and deterministic (sorted keys, fixed times),
put them under `testdata/`, and make sure Git doesn't rewrite their line endings.
