---
id: go.testing.benchmarks-fuzz
title: "Benchmarks with b.Loop, fuzzing, coverage, Example functions"
track: go
module: testing
order: 3
level: advanced
minutes: 15
since: go1.24
prerequisites: [go.testing.table-driven, go.basics.strings-runes]
compare_with: [go.performance.allocations, go.performance.pprof, ts.testing.vitest]
tags: [testing, benchmarks, fuzzing, coverage, examples, utf-8]
summary: >
  Benchmarks measure with for b.Loop() and run only under -bench; fuzz targets
  check properties on generated inputs and replay their corpus as ordinary tests;
  Example functions are documentation that go test verifies. Coverage says what
  ran, not what is right.
examples:
  - id: bench
    file: examples/01_bench_test.go
    title: "b.Loop benchmarks, and the parts plain go test can check"
    runtime: go-test
    check: contains
    annotations:
      - lines: "8-18"
        note: >
          Strings are immutable, so every += allocates a new string and copies
          everything so far: quadratic work. Eight fields cost 14 allocations
          (the first += onto "" just returns f).
      - lines: "39-44"
        note: >
          A benchmark is func BenchmarkXxx(b *testing.B). go test runs none of
          them unless you pass -bench, which is why the output below has no
          Benchmark lines. The lesson text shows real -bench output.
      - lines: "41"
        note: >
          for b.Loop() (Go 1.24) resets the timer on its first call, stops it
          when it returns false, runs the function body once per measurement
          instead of several times, and keeps the call's arguments and results
          alive so the compiler can't delete the work being measured.
      - lines: "52-60"
        note: >
          testing.AllocsPerRun returns an exact count, unlike a timing, so it
          can live in a normal test and fail the build when a change adds an
          allocation to a hot path.
      - lines: "64-67"
        note: >
          testing.Benchmark drives a b.Loop benchmark from a test or a main
          program, outside -bench. It runs for about a second; print only what
          is stable, such as allocations per operation.
  - id: fuzz
    file: examples/02_fuzz_test.go
    title: "A fuzz target: properties, seeds and a regression corpus"
    runtime: go-test
    fixtures:
      - fixtures/testdata/fuzz/FuzzTruncate/447915f5237d057f
      - fixtures/testdata/fuzz/FuzzTruncate/7c28aed13156482b
    check: contains
    annotations:
      - lines: "12-14"
        note: >
          Found by the fuzzer, not by a person: the first version sliced s[:n]
          and panicked for any negative n. The input it found is a file in the
          corpus now, so the bug can't come back unnoticed.
      - lines: "18-20"
        note: >
          The UTF-8 fix. A cut at byte n may land inside a character (ğ is two
          bytes, many emoji four); RuneStart finds the first byte of the
          character that doesn't fit, and the cut moves back to it.
      - lines: "25-27"
        note: >
          Seed inputs. Plain go test runs each one as a subtest (seed#0...);
          go test -fuzz starts from them and mutates both arguments.
      - lines: "28-36"
        note: >
          The target checks properties that hold for every input: no longer
          than n, a prefix of s, and valid UTF-8 whenever s was. Input
          validity is a condition, not a skip, so invalid UTF-8 still gets the
          other two checks.
      - lines: "37"
        note: >
          Logs show here because plain go test runs seeds and corpus files as
          ordinary subtests. The last two subtests in the output are named
          after the files in testdata/fuzz/FuzzTruncate, inputs that real fuzz
          runs found.
  - id: examples
    file: examples/03_example_test.go
    title: "Example functions: documentation that go test runs"
    runtime: go-test
    check: contains
    annotations:
      - lines: "9-12"
        note: >
          Turkish also writes â, î and û (kâr, Kâğıthane, millî). An earlier
          version without this line had 100% statement coverage and still
          turned Kâğıthane into k-githane; the lesson text shows that run.
      - lines: "23-24"
        note: >
          unicode.Mn is the category of combining marks. Dropping them handles
          text lowercased by JavaScript, where İ becomes i plus U+0307, and
          decomposed input such as a followed by a combining circumflex.
      - lines: "33-41"
        note: >
          ExampleSlug documents Slug: pkg.go.dev and go doc show it under the
          function. go test runs it and compares stdout with the Output comment,
          ignoring leading and trailing space; a mismatch fails like a test.
      - lines: "43-52"
        note: >
          Map iteration order is random, so this example uses Unordered output:
          the same lines in any order. The suffix after the underscore must
          start with a lower-case letter.
      - lines: "54-58"
        note: >
          No output comment: compiled, so it can't rot into code that doesn't
          build, but not executed. That is why the run below has no line for
          ExampleSlug_untested.
use_cases:
  - "Benchmarks: comparing two implementations of a hot function, with -benchmem and -count=10 piped into benchstat"
  - "AllocsPerRun tests: pinning zero- or one-allocation guarantees of encoders, parsers and logging paths"
  - "Fuzzing: parsers, decoders, truncation and escaping code, anything taking untrusted bytes or strings, round trips such as Parse(Format(x)) == x"
  - "Example functions: the usage snippet of every exported function in a library, kept correct by go test"
  - "Coverage: finding branches no test reaches (error paths, rare switch cases), not proving correctness"
strengths:
  - "Benchmarking, fuzzing, coverage and runnable docs ship with the toolchain; Java needs JMH, Jazzer and JaCoCo, Python pytest-benchmark, Hypothesis and coverage.py"
  - "b.Loop removes the classic benchmark mistakes (timed setup, measured code optimized away) that JMH solves with Blackhole and annotations"
  - "Fuzz failures become files in testdata/fuzz that plain go test replays forever, so a found bug is a regression test at once"
  - "Examples compile with the package and are checked by go test, unlike doc comments or README snippets in other ecosystems, which silently go stale"
weaknesses:
  - "Benchmark numbers depend on the machine, its load and its power state; single runs on a laptop mislead, and comparing needs repeated runs and statistics"
  - "go test -fuzz fuzzes one target in one package at a time and supports only basic argument types (strings, []byte, integers, floats, bool, rune)"
  - "Statement coverage counts executed lines, not checked behaviour: 100% says nothing about inputs you didn't think of"
  - "Example output is compared as text, so examples suit deterministic output only"
pitfalls:
  - title: "Benchmarks whose work the compiler deletes"
    detail: "In a b.N loop, an inlined pure call whose result is unused may be partly or fully removed. In a real run of digitSum(123456789), the b.N version reported 3.8 ns/op and the b.Loop version 7.3 ns/op: the disassembly showed the additions gone, only the division loop left."
    bad: "for i := 0; i < b.N; i++ {\n\tdigitSum(123456789)\n}"
    good: "for b.Loop() {\n\tdigitSum(123456789)\n}"
  - title: "Mixing b.Loop and b.N"
    detail: "A benchmark uses exactly one for b.Loop() loop, or a loop to b.N, never both. Setup that must run before each iteration still needs b.StopTimer and b.StartTimer inside the loop."
  - title: "Deleting the fuzz corpus"
    detail: "testdata/fuzz/FuzzXxx holds the inputs that once broke the code. Commit it: plain go test replays it on every run, which is the only thing that keeps the fix fixed."
  - title: "Expecting coverage from single-file examples"
    detail: "Coverage instruments non-test files only. With code and tests in one _test.go file, as in this app's examples, go test -cover reports \"[no statements]\"; split code and tests into x.go and x_test.go."
quiz:
  - q: "What does plain go test (no flags) do with BenchmarkX, FuzzX and ExampleX functions?"
    options: ["Runs all three fully", "Skips benchmarks, runs FuzzX on its seed corpus only, runs ExampleX if it has an output comment", "Runs only tests and examples; fuzz targets need -fuzz"]
    answer: 1
    why: "Benchmarks need -bench. Without -fuzz a fuzz target acts like a test over its F.Add seeds and testdata/fuzz files. Examples with Output comments are run and checked."
  - q: "Why is for b.Loop() preferred over for i := 0; i < b.N; i++?"
    options: ["It excludes setup from timing, runs the function once per measurement and keeps the measured work from being optimized away", "It runs more iterations", "It runs benchmarks in parallel"]
    answer: 0
    why: "Those are the three classic b.N mistakes that Loop handles automatically (Go 1.24)."
  - q: "go test -fuzz=FuzzTruncate finds a failing input. What happens to it?"
    options: ["It is printed and forgotten", "It is added as an F.Add call in your source", "It is written to testdata/fuzz/FuzzTruncate/<id> and replayed by every later go test"]
    answer: 2
    why: "The file is a seed corpus entry; commit it and it runs as a regression subtest."
  - q: "Slug has 100% statement coverage, yet Slug(\"Kâğıthane\") is wrong. How?"
    options: ["Coverage was measured wrongly", "Coverage records which statements ran, not whether results were right for inputs nobody wrote down", "Example functions don't count toward coverage"]
    answer: 1
    why: "The â went through the same statements as a space; no example or test asserted on it."
further_reading:
  - title: "package testing: benchmarks, fuzzing, examples"
    url: "https://pkg.go.dev/testing"
  - title: "More predictable benchmarking with testing.B.Loop"
    url: "https://go.dev/blog/testing-b-loop"
  - title: "Go Fuzzing"
    url: "https://go.dev/doc/security/fuzz/"
  - title: "Tutorial: Getting started with fuzzing"
    url: "https://go.dev/doc/tutorial/fuzz"
  - title: "Testable Examples in Go"
    url: "https://go.dev/blog/examples"
  - title: "benchstat"
    url: "https://pkg.go.dev/golang.org/x/perf/cmd/benchstat"
---

## Concept

The `testing` package has three more kinds of function besides `TestXxx`:

- `func BenchmarkXxx(b *testing.B)` measures time and allocations per operation. Write
  the measured code inside `for b.Loop() { ... }` (Go 1.24). Benchmarks run only with
  `go test -bench=regexp`.
- `func FuzzXxx(f *testing.F)` registers seed inputs with `f.Add` and a target with
  `f.Fuzz(func(t *testing.T, args...) { ... })`. Plain `go test` runs the target on the
  seeds and on the files in `testdata/fuzz/FuzzXxx`; `go test -fuzz=FuzzXxx` generates
  new inputs until it finds a failure or time runs out.
- `func ExampleXxx()` prints to stdout; with an `// Output:` comment, `go test` runs it
  and compares. Examples appear in `go doc` and on pkg.go.dev next to the documented
  function.

Coverage is a flag, not a function: `go test -cover` reports the percentage of statements
executed, and `-coverprofile=cover.out` writes the details for `go tool cover`.

The runner here uses plain `go test -v` (plus a `-timeout` from the lesson's time
limit), so the examples show what plain `go test` checks. The `-bench`, `-fuzz` and `-cover` runs below are real output from a scratch
module on Go 1.27.0 (Windows, i9-14900HX).

## Coming from C/C++/Java/Python/JS

- **Java**: JMH needs a separate harness, `@Benchmark` annotations and `Blackhole` to
  stop dead-code elimination; `b.Loop` does that job in the standard tool. Jazzer brings
  fuzzing to JUnit; JaCoCo measures coverage.
- **C/C++**: Google Benchmark's `benchmark::DoNotOptimize` is what `b.Loop` does to the
  arguments and results of calls in its body. Go fuzzing is coverage-guided like
  libFuzzer, built into `go test`.
- **Python**: Hypothesis generates inputs from strategies and shrinks failures; Go's
  fuzzer mutates seed inputs guided by coverage and minimizes failures. `doctest` is
  the closest relative of Example functions.
- **JS/TS**: `vitest bench` and fast-check cover benchmarks and property tests; there
  is no standard runnable-doc mechanism, so README snippets drift.

## How it works

**Benchmarks.** Run them by name or all at once; `-run='^$'` skips the tests:

```
$ go test -run='^$' -bench=. -benchmem
goos: windows
goarch: amd64
pkg: lesson
cpu: Intel(R) Core(TM) i9-14900HX
BenchmarkJoinPlus-32       	 3072550	       385.1 ns/op	     552 B/op	      14 allocs/op
BenchmarkJoinBuilder-32    	19025572	        67.15 ns/op	      64 B/op	       1 allocs/op
PASS
ok  	lesson	4.161s
```

The `-32` suffix is GOMAXPROCS. The iteration count is chosen by the framework to fill
about a second (`-benchtime`). Timings move between runs; to compare two versions, run
each with `-count=10` and feed both outputs to `benchstat`, which reports the change and
whether it is statistically significant.

**Fuzzing.** The first `Truncate` was `if len(s) <= n { return s }; return s[:n]`, with
only `f.Add("hello", 3)` as seed. `go test -fuzz=FuzzTruncate -fuzztime=60s` found a
negative `n` in under a second (`panic: runtime error: slice bounds out of range [:-90]`,
input saved to `testdata/fuzz/FuzzTruncate/`). After the `n <= 0` fix, the next run:

```
fuzz: elapsed: 0s, gathering baseline coverage: 0/2 completed
fuzz: elapsed: 1s, gathering baseline coverage: 2/2 completed, now fuzzing with 32 workers
fuzz: minimizing 38-byte failing input file
fuzz: elapsed: 2s, minimizing
--- FAIL: FuzzTruncate (1.97s)
    --- FAIL: FuzzTruncate (0.00s)
        trunc_test.go:30: Truncate("00خ", 3) = "00\xd8": split a character

    Failing input written to testdata\fuzz\FuzzTruncate\d019b15370b97745
    To re-run:
    go test -run=FuzzTruncate/d019b15370b97745
FAIL
```

The saved file is plain text that you commit:

```
go test fuzz v1
string("00خ")
int(3)
```

Fuzzing runs only in one package and for one target per command, and runs until it
fails unless `-fuzztime` limits it. With the `RuneStart` fix, 60 seconds and about
9.8 million executions found nothing more.

**Coverage.** With `Slug` in `slug.go` and only the examples in `slug_test.go`, coverage
was complete while a new example for `Kâğıthane` failed (this was before the â, î, û
line was added):

```
$ go test -cover
--- FAIL: ExampleSlug_circumflex (0.00s)
got:
k-githane
want:
kagithane
FAIL
coverage: 100.0% of statements
FAIL	example.com/slug	2.152s
```

`go test -coverprofile=cover.out` followed by `go tool cover -func=cover.out` lists the
percentage per function, and `go tool cover -html=cover.out` colours the source.

## Design notes

Measure honestly: benchmark the operation you care about with realistic input, keep
setup outside `b.Loop`, report allocations (`-benchmem` or `b.ReportAllocs()`), and
compare distributions with `benchstat` rather than single numbers. Timings stay out of
`go test`; allocation counts are exact and belong in ordinary tests via
`testing.AllocsPerRun`.

Fuzz what parses, decodes, escapes or cuts untrusted input. Write the target as
properties (length bounds, prefixes, valid UTF-8, round trips) rather than expected
values, add seeds that reach the interesting paths (multi-byte characters, empty input,
zero and negative sizes), and commit every crasher the fuzzer saves. UTF-8 is a
reliable source of findings: Go strings are bytes, and any limit expressed in bytes, a
`varchar` byte budget or a 255-byte header, can split ğ, ş or İ in half.

Use Example functions as the usage documentation of exported APIs; they are the one
kind of documentation that can't go stale. Use coverage to find untested branches, not
as a target: 100% statement coverage is easy to reach and, as `Slug` shows, says
nothing about inputs nobody wrote down.
