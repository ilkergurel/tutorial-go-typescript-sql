---
id: go.testing.table-driven
title: "Table-driven tests, t.Run, t.Helper, t.Cleanup, t.Context"
track: go
module: testing
order: 1
level: basic
minutes: 13
since: go1.24
prerequisites: [go.errors.is-as-join, go.composite.structs]
compare_with: [ts.testing.vitest, go.testing.fakes-httptest]
tags: [testing, table-driven, subtests, helpers, cleanup]
summary: >
  A Go test is ordinary code: a table of cases, a loop, t.Run for named
  subtests, and if got != want { t.Errorf(...) }. Helpers call t.Helper and
  register their own t.Cleanup; t.Context ends with the test.
examples:
  - id: table
    file: examples/01_table_test.go
    title: "A table of cases, one subtest per row"
    runtime: go-test
    check: contains
    annotations:
      - lines: "30-43"
        note: >
          The table is a slice of an anonymous struct, one row per case, each
          with a name. Adding a case is one line; the checking code below is
          written once. An err field compared with errors.Is covers success and
          failure rows alike, because errors.Is(nil, nil) is true.
      - lines: "41"
        note: >
          "1,234" reads as one thousand to an English eye and is invalid in
          Turkish notation. Locale traps like this are cheap to pin down once
          they are a row in a table.
      - lines: "45"
        note: >
          t.Run makes each row a subtest with its own PASS/FAIL line and name.
          Spaces become underscores and ş is kept, so go test -run
          'TestParseKurus/kuruş' runs that row alone.
      - lines: "46"
        note: >
          t.Parallel pauses the subtest until the loop has finished, then runs
          the paused ones together (the PAUSE and CONT lines). Before Go 1.22
          every closure shared one tt: all seven subtests checked the last row
          and passed. Old code has tt := tt here for that reason.
      - lines: "48-53"
        note: >
          No assertion library: an if and a message in the form
          F(input) = got, want want. Fatalf ends this subtest, since comparing
          values after a wrong error is noise; Errorf records the failure and
          lets the test continue.
  - id: helpers
    file: examples/02_helpers_test.go
    title: "Helpers with t.Helper, t.Cleanup, t.TempDir and t.Context"
    runtime: go-test
    check: contains
    annotations:
      - lines: "31-32"
        note: >
          A helper takes t, fails the test itself and returns a ready value.
          t.Helper makes failures and logs inside it report the caller's line:
          the output shows "journal opened" at line 52, in TestJournal, while
          logWithoutHelper's message points at its own line 48.
      - lines: "33"
        note: >
          t.TempDir creates a fresh directory per test and registers a cleanup
          that deletes it, so nothing leaks between tests or runs.
      - lines: "37-42"
        note: >
          The helper registers the Close itself, so no caller can forget it.
          Cleanups run last-in, first-out: this one, registered after
          TempDir's, closes the file before the directory is removed. On
          Windows, deleting a directory with an open file fails the test.
      - lines: "38"
        note: >
          t.Context() (Go 1.24) is already canceled when cleanups run, which
          the log shows. Cleanup code that still needs to talk to something,
          such as a database rollback, uses context.Background().
      - lines: "54"
        note: >
          The closure in a cleanup is not a helper, so its log line points
          here. It runs before the helper's cleanup because it was registered
          later.
      - lines: "56"
        note: >
          Pass t.Context() to the code under test: it is canceled as soon as
          the test function returns, so goroutines the test started see Done
          before the cleanups wait for them.
  - id: compare
    file: examples/03_compare_test.go
    title: "Comparing without an assertion library: slices.Equal, DeepEqual, %+q"
    runtime: go-test
    check: contains
    annotations:
      - lines: "16-31"
        note: >
          Table-driven without t.Run is fine for fast, independent rows: the
          loop reports every failing row with Errorf and keeps going. Use t.Run
          when you want names, -run filtering or parallelism.
      - lines: "27-28"
        note: >
          slices.Equal (Go 1.21) compares element by element with == and
          treats nil and empty as equal. %+q prints strings with non-ASCII
          escaped, so a failure message shows exactly which code points differ.
      - lines: "33-38"
        note: >
          strings.Fields returns an empty, non-nil slice. reflect.DeepEqual
          tells nil and empty apart, so a test written with DeepEqual against
          nil would fail although the behaviour is right.
      - lines: "40-46"
        note: >
          JavaScript's toLowerCase applies Unicode's full mapping and turns İ
          into i plus U+0307 COMBINING DOT ABOVE; Go's ToLowerSpecial with
          TurkishCase gives a plain i. %q shows two strings that look alike;
          %+q shows the escaped U+0307 that makes them unequal.
use_cases:
  - "Parsers, validators and formatters: one row per input, including the locale and boundary cases that bite"
  - "Error paths: a want-error column checked with errors.Is, next to the success rows"
  - "Shared setup (temp files, servers, database connections) wrapped in a helper that calls t.Helper and registers its own t.Cleanup"
  - "Code that starts goroutines or takes a context: pass t.Context() so it stops when the test does"
strengths:
  - "No framework to learn: tests are plain Go, so the debugger, refactoring tools and the race detector work on them unchanged, unlike Jest's or pytest's runtime magic"
  - "Errorf keeps going after a failure, so one run reports every broken row, where a JUnit assertEquals throws and hides the rest of the method"
  - "Subtests give every row a name that -run can select and that CI reports separately, with no @ParameterizedTest or test.each annotation layer"
  - "t.Cleanup lets a helper own its teardown, so the setup and its undo sit together instead of in a separate @AfterEach or afterEach"
weaknesses:
  - "Messages are written by hand; without a library there is no automatic diff of two large structs (teams that want one usually allow github.com/google/go-cmp)"
  - "reflect.DeepEqual is strict about nil versus empty and compares unexported fields, so it fails on differences that don't matter"
  - "t.Fatal only works from the test's own goroutine; goroutines the test starts must report back over a channel or use t.Error"
pitfalls:
  - title: "Parallel subtests on a pre-1.22 go line"
    detail: "With go 1.21 or older in go.mod, every closure shares the loop variable: all subtests test the last row and pass. go test doesn't catch it; go vet's loopclosure check does. Raise the go line, or copy the variable."
    bad: "for _, tt := range tests { // module says go 1.21\n\tt.Run(tt.name, func(t *testing.T) {\n\t\tt.Parallel()\n\t\tcheck(t, tt) // always the last row"
    good: "for _, tt := range tests {\n\ttt := tt // needed only before Go 1.22\n\tt.Run(tt.name, func(t *testing.T) {\n\t\tt.Parallel()\n\t\tcheck(t, tt)"
  - title: "Calling t.Fatal from a goroutine"
    detail: "Fatal stops the goroutine that calls it, via runtime.Goexit. From a goroutine the test started, it ends that goroutine, not the test, and a test waiting for that goroutine's result hangs until go test's timeout. go vet reports \"call to (*testing.T).Fatal from a non-test goroutine\". Send the error back to the test goroutine, or use t.Error, which only records the failure."
  - title: "Using t.Context() in a cleanup"
    detail: "It is canceled before cleanups run, so a rollback or close that takes it fails at once. Use context.Background() there."
  - title: "Leaving files open in t.TempDir"
    detail: "On Windows the directory can't be removed while a file in it is open, and the test fails with \"TempDir RemoveAll cleanup\" even though Linux CI is green. Close files in a cleanup registered after TempDir."
quiz:
  - q: "What is the difference between t.Errorf and t.Fatalf?"
    options: ["Both fail the test; Fatalf also stops the current test or subtest", "Errorf logs, Fatalf fails", "Fatalf exits the whole test binary"]
    answer: 0
    why: "Errorf marks the test failed and continues; Fatalf marks it failed and calls runtime.Goexit, ending only that test's goroutine."
  - q: "A helper calls t.Helper() and then t.Fatalf. Which line does the failure message show?"
    options: ["The t.Fatalf line inside the helper", "The line in the test that called the helper", "No line number"]
    answer: 1
    why: "Helper marks the function as a helper, and the testing package reports the first caller that isn't one."
  - q: "In what order do t.Cleanup functions run?"
    options: ["Registration order", "Concurrently", "Last registered, first run"]
    answer: 2
    why: "They run last-in, first-out after the test and its subtests finish, so later resources are released before the ones they depend on."
  - q: "When is t.Context() canceled?"
    options: ["Never; it is context.Background()", "Just before the Cleanup functions run", "After all Cleanup functions have run"]
    answer: 1
    why: "The docs specify cancellation just before cleanups, so cleanups can wait for goroutines that stop on Done."
further_reading:
  - title: "package testing"
    url: "https://pkg.go.dev/testing"
  - title: "Go wiki: TableDrivenTests"
    url: "https://go.dev/wiki/TableDrivenTests"
  - title: "Using Subtests and Sub-benchmarks"
    url: "https://go.dev/blog/subtests"
  - title: "Go FAQ: Where is my favorite helper function for testing?"
    url: "https://go.dev/doc/faq#testing_framework"
  - title: "Go Code Review Comments: Useful Test Failures"
    url: "https://go.dev/wiki/CodeReviewComments#useful-test-failures"
---

## Concept

A test is a function `func TestXxx(t *testing.T)` in a file ending in `_test.go`.
`go test` compiles the package together with its test files into a test binary and runs
every `Test` function. There are no assertions: you compare with `==`, `errors.Is` or
`slices.Equal` and report with `t.Errorf` (record a failure, keep going) or `t.Fatalf`
(record it and stop this test).

The idiomatic shape is the **table-driven test**: a slice of cases, each a row with a
name, inputs and the expected result, and one loop that checks them all. `t.Run(name, f)`
turns each row into a **subtest** with its own name, result line and `-run` filter.

The rest of `*testing.T` keeps tests short and clean:

- `t.Helper()` marks a function as a helper, so failures point at the caller.
- `t.Cleanup(f)` registers teardown; cleanups run last-in, first-out when the test ends.
- `t.TempDir()`, `t.Setenv(k, v)` and `t.Chdir(dir)` (Go 1.24) undo themselves.
- `t.Context()` (Go 1.24) is a context canceled just before the cleanups run.

## Coming from C/C++/Java/Python/JS

- **GoogleTest**: `EXPECT_EQ` keeps going and `ASSERT_EQ` stops the test; that is exactly
  `t.Errorf` versus `t.Fatalf`. `TEST_P` parameterized tests are a table and a loop.
- **JUnit 5**: `@ParameterizedTest` with `@MethodSource` is the table; `assertEquals`
  becomes an `if`. `@AfterEach` becomes a `t.Cleanup` registered by the helper that
  created the resource.
- **pytest**: `@pytest.mark.parametrize` is the table; a fixture with `yield` is a helper
  that returns a value and registers `t.Cleanup`. Go has no assert rewriting, so the
  message you write is the message you get.
- **Jest/Vitest**: `test.each` is the table, `expect(x).toEqual(y)` becomes
  `slices.Equal` or `==` plus `t.Errorf`, and `afterEach` becomes `t.Cleanup`.

## How it works

`go test` runs top-level tests one after another in source order. A subtest that calls
`t.Parallel()` pauses until its parent's function returns, then runs alongside its
paused siblings; the parent finishes only when all of them have. That is why the
`-v` output shows `=== PAUSE` and `=== CONT` lines, and why the summary lines can come
in a different order each run.

When a test fails, the output names the test, the file and the line of the `t.Errorf`
call (or of the caller, for a helper). Real output from changing one row of the first
example to expect `1234560`:

```
--- FAIL: TestParseKurus (0.00s)
    --- FAIL: TestParseKurus/thousands (0.00s)
        01_table_test.go:52: ParseKurus("1.234,56") = 123456, want 1234560
FAIL
FAIL	lesson	1.695s
FAIL
```

`-run` takes one regular expression per subtest level, separated by `/`:
`go test -run 'TestParseKurus/kuruş' -v` runs only the `only_kuruş` row. `-v` prints
`t.Log` output and a line per test; without it, only failures are printed.

## Design notes

The Go FAQ explains the missing assertions: a test should report everything that is
wrong in one run, with a message that someone who doesn't know the code can act on. A
table amortizes the cost of that message over every row. Write it as
`F(input) = got, want want`, quote strings with `%q` (or `%+q` when non-ASCII text can
hide a difference), and compare with the narrowest tool: `==` for comparable values and
structs, `errors.Is` for errors, `slices.Equal` and `maps.Equal` for collections, and
`reflect.DeepEqual` only when nothing else fits.

Put setup in helpers that take `t`, call `t.Helper()`, fail the test themselves and
register their own cleanup. Tests then read as a list of facts instead of setup code,
and teardown can't be forgotten. Give code under test `t.Context()` rather than
`context.Background()`, so anything it starts stops when the test does; in cleanups,
which run after that cancellation, use `context.Background()`.
