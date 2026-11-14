---
id: go.performance.pprof
title: "pprof: CPU and heap profiling, escape analysis (-gcflags=-m)"
track: go
module: performance
order: 1
level: advanced
minutes: 15
since: go1.24
prerequisites: [go.testing.benchmarks-fuzz, go.basics.pointers]
compare_with: [sql.indexes.plans, ts.react.routing]
tags: [performance, pprof, profiling, cpu, heap, escape-analysis, pgo]
summary: >
  pprof samples where a program spends CPU time and where it allocates, down to
  the source line; go build -gcflags=-m explains why a value lives on the heap.
  Profile a benchmark or a running service, fix the top entry, measure again.
examples:
  - id: cpu-profile
    file: examples/01_cpu_profile.go
    title: "A CPU profile finds the quadratic loop nobody suspected"
    runtime: go
    check: contains
    timeout_s: 15
    annotations:
      - lines: "22-31"
        note: >
          One loop, one append: it reads as linear. slices.Contains scans out
          on every iteration, so it is O(n²), and nothing on the page looks
          expensive. This is why guessing fails.
      - lines: "33-35"
        note: >
          format uses fmt and reflection, the usual suspect in a code review.
          It doesn't appear in the profile at all.
      - lines: "49-53"
        note: >
          While profiling is on, the runtime records the running goroutine's
          stack 100 times per second of CPU time. Overhead is low enough for
          production, but it is a sample: a 200 ms run gives about 20 stacks.
      - lines: "56"
        note: >
          -top lists functions; -cum sorts by cumulative time (the function
          plus everything it calls) instead of flat time (the function's own
          instructions). cum finds the expensive path, flat the instructions
          burning the time.
      - lines: "62-64"
        note: >
          The table is real pprof output; only the header (binary path, date,
          sample count) is cut. Timings differ on every run, which is why this
          example is checked with "contains".
  - id: heap-profile
    file: examples/02_heap_profile.go
    title: "A heap profile names the function and the line that allocate"
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "15"
        note: >
          By default the runtime records one allocation per 512 KiB allocated,
          on average, and pprof scales the samples up. Rate 1 records every
          allocation so the numbers are exact and repeatable; it is far too
          slow for production.
      - lines: "26-32"
        note: >
          2,000 concatenations, each copying everything so far: about 20 MB
          allocated to build a 20 KB string. The CSV parsing that looks busier
          allocates less than 0.1 MB.
      - lines: "49-53"
        note: >
          A heap profile reports as of the last completed GC, so collect first.
          go test -memprofile does the same before writing its profile.
      - lines: "57"
        note: >
          alloc_space counts every byte allocated since start; the default,
          inuse_space, counts only what is still live, which here is almost
          nothing. Use alloc_* to find churn that keeps the GC busy, inuse_* to
          find memory that is retained or leaking.
      - lines: "64"
        note: >
          -list takes a regular expression and prints the source of matching
          functions with per-line totals: the profile points at line 29, not
          just at render.
  - id: escape-analysis
    file: examples/03_escape_analysis.go
    title: "go build -gcflags=-m: the compiler's heap decisions, one per function"
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "18-21"
        note: >
          "moved to heap: p": the address is returned, so p must outlive the
          call. One allocation per NewPoint, which is fine unless NewPoint is
          in a hot loop.
      - lines: "23-26"
        note: >
          "&Point{...} does not escape": taking an address costs nothing when
          the pointer never leaves the function; the struct stays on the
          stack.
      - lines: "28"
        note: >
          The slice is returned, so its backing array must be on the heap.
      - lines: "30-33"
        note: >
          "does not escape" is not "never allocates". The size is known only at
          run time; since Go 1.25 the compiler keeps a small buffer on the
          stack and falls back to the heap when n doesn't fit (in an
          AllocsPerRun test on Go 1.27, 32 bytes fit and 33 did not).
      - lines: "35-38"
        note: >
          Nothing escapes, yet buf is moved to the heap: a 1 MiB variable is
          too large for a goroutine stack frame.
      - lines: "40"
        note: >
          fmt.Sprint takes ...any, so p.X is converted to an interface and
          "escapes". The variadic slice itself stays on the stack ("...
          argument does not escape").
      - lines: "52"
        note: >
          -gcflags=-m applies to the packages named on the command line;
          -m=2 explains each decision, and -gcflags=all=-m includes every
          dependency. The example filters out the "can inline" lines.
      - lines: "60-63"
        note: >
          Positions are file:line:column in esc.go, whose first line is line
          12 of this file: esc.go:8:2 is the p := Point{x, y} on line 19 here.
use_cases:
  - "A slow benchmark: go test -bench=Name -cpuprofile cpu.out, then go tool pprof -top or -http=:8081 cpu.out"
  - "A service using too much CPU or memory: import net/http/pprof on a localhost-only listener and pull a 30-second profile"
  - "GC pressure: a heap profile with -sample_index=alloc_space, then -gcflags=-m on the hot package to see why values escape"
  - "Goroutine leaks: the goroutine profile, and since Go 1.27 the goroutineleak profile"
  - "Profile-guided optimization: commit a representative CPU profile as default.pgo next to main"
strengths:
  - "Built into the runtime and the go command: no agent, no separate profiler install, unlike Java's async-profiler or JFR setup"
  - "Sampling keeps CPU profiling overhead low enough to run in production, unlike Python's instrumenting cProfile"
  - "Profiles carry symbols and inlining information, so -list maps samples to source lines, inlined calls included"
  - "Escape decisions are printable on demand, where the JVM's escape analysis happens invisibly in the JIT"
weaknesses:
  - "A profile is a statistical sample: short runs and rare paths can be missing or noisy"
  - "Heap profiles show where memory was allocated, not who holds it now; finding a retention path needs reasoning or a heap dump"
  - "-gcflags=-m output changes between Go versions, so don't build tooling that parses it"
  - "The CPU profile shows on-CPU time; time spent waiting for locks, I/O or the database needs the block, mutex or trace views"
pitfalls:
  - title: "Optimizing without a profile"
    detail: "The expensive code is often the innocent-looking loop, not the reflection-heavy formatter. Profile a benchmark or production first, fix the top entry, and measure again."
  - title: "Exposing net/http/pprof on the public listener"
    detail: "The blank import registers /debug/pprof/ on http.DefaultServeMux. Serve that mux on 127.0.0.1 only, separately from the application's own mux."
    bad: "import _ \"net/http/pprof\" // and the public server uses DefaultServeMux"
    good: "go func() { log.Println(http.ListenAndServe(\"127.0.0.1:6060\", nil)) }() // app uses its own mux"
  - title: "Reading inuse_space when hunting garbage"
    detail: "The default heap view shows live memory. Short-lived garbage that drives GC cost appears only under alloc_space or alloc_objects."
  - title: "Taking 'does not escape' as 'no allocation'"
    detail: "Variable-size make and very large locals can still use the heap. Confirm with testing.AllocsPerRun or -benchmem."
  - title: "Profiling with the race detector or coverage on"
    detail: "-race and -cover change the code being measured. Profile a normal build."
quiz:
  - q: "In pprof -top, main.handle has flat 0 and cum 95%. What does that mean?"
    options: ["It is the bottleneck itself", "It does almost no work itself; functions it calls account for 95% of the samples", "The profile is corrupt"]
    answer: 1
    why: "flat counts samples where the function was executing its own code; cum counts samples where it was anywhere on the stack."
  - q: "You want the code that creates the most short-lived garbage. Which heap view?"
    options: ["inuse_space (the default)", "alloc_space or alloc_objects", "The goroutine profile"]
    answer: 1
    why: "inuse_* shows what is still live; garbage has already been collected and appears only in the alloc_* totals."
  - q: "-gcflags=-m prints 'make([]byte, n) does not escape'. Can that line still allocate on the heap?"
    options: ["No, never", "Yes: with a size known only at run time, a large n falls back to the heap", "Only with GOGC=off"]
    answer: 1
    why: "'Does not escape' means the slice doesn't outlive the function. Since Go 1.25 a small stack buffer covers small sizes; larger ones are heap allocated. Large fixed-size locals are moved to the heap too."
  - q: "How should a production service expose pprof?"
    options: ["Register net/http/pprof on the public mux behind a feature flag", "Serve http.DefaultServeMux with the pprof handlers on a localhost-only listener", "Write a CPU profile to disk continuously"]
    answer: 1
    why: "Profiles reveal internals and a CPU profile request costs CPU for its duration; keep the endpoint off the public listener and reach it through SSH or kubectl port-forward."
further_reading:
  - title: "Diagnostics (profiling, tracing, debugging)"
    url: "https://go.dev/doc/diagnostics"
  - title: "package runtime/pprof"
    url: "https://pkg.go.dev/runtime/pprof"
  - title: "package net/http/pprof"
    url: "https://pkg.go.dev/net/http/pprof"
  - title: "Profile-guided optimization"
    url: "https://go.dev/doc/pgo"
  - title: "A Guide to the Go Garbage Collector: optimization guide"
    url: "https://go.dev/doc/gc-guide#Optimization_guide"
---

## Concept

pprof is Go's built-in profiler, in three parts: the runtime collects samples,
`runtime/pprof`, `net/http/pprof` and `go test` write them as profile files, and
`go tool pprof` reads and presents them. The profiles you will use most:

- **cpu**: where on-CPU time goes, sampled 100 times per second.
- **heap**: allocations by call stack, viewable as `inuse_space`/`inuse_objects`
  (live now) or `alloc_space`/`alloc_objects` (since start).
- **goroutine**, **block**, **mutex**: what goroutines are doing and where they wait.
  Go 1.27 adds **goroutineleak**, goroutines blocked on something no runnable
  goroutine can ever reach.

The workflow is a loop: reproduce the load (a benchmark, or a real service under
traffic), collect a profile, read the top entries, change one thing, measure again.
A benchmark run from the scratch module behind the first example, on the author's
machine:

```
$ go test -bench=Dedupe -benchmem -cpuprofile cpu.out
BenchmarkDedupe-32   625   1860894 ns/op   60032 B/op   14 allocs/op
$ go tool pprof -list Dedupe cpu.out
ROUTINE ======================== example.com/report.Dedupe in .../report.go
         0      1.11s (flat, cum) 97.37% of Total
         .          .      5:func Dedupe(ids []int) []int {
         .          .      6:	var out []int
         .          .      7:	for _, id := range ids {
         .      1.11s      8:		if !slices.Contains(out, id) {
         .          .      9:			out = append(out, id)
```

Replacing the scan with a `map[int]struct{}` set brought it to 87524 ns/op, 21 times
faster, at 207802 B/op and 31 allocs/op: CPU and memory profiles can pull in
opposite directions, so measure both. `go tool pprof -http=:8081 cpu.out` opens the
same data in a browser, with a flame graph as the default view since Go 1.26.

When a heap profile points at a line, `go build -gcflags=-m` tells you why the value
there lives on the heap: it escapes (returned, stored, captured, converted to an
interface) or is too large for the stack.

## Coming from C/C++/Java/Python/JS

- **Java**: JFR and async-profiler give similar sampled CPU and allocation views; in
  Go there is nothing to attach, the profiler is in every binary. The JVM's escape
  analysis is a JIT detail you can't easily see; Go prints its decisions at compile
  time.
- **Python**: `cProfile` instruments every call and distorts timings; pprof samples,
  like py-spy. Python has no stack allocation to reason about at all.
- **JS/Node**: Chrome DevTools and `node --cpu-prof` are sampling CPU profilers too.
  V8 heap snapshots show the object graph; Go's heap profile shows allocation sites
  instead, which is better for finding churn and worse for finding who retains what.
- **C/C++**: `perf` works on Go binaries, but pprof understands goroutines, inlined
  frames and Go symbol names without extra setup. There is no `new` vs stack choice:
  escape analysis makes it for you.

## How it works

For CPU profiles the runtime interrupts threads running Go code at 100 Hz and
records the current goroutine's stack; pprof merges identical stacks and counts
them. **flat** is the samples where a function was at the top of the stack, **cum**
those where it appeared anywhere. Inlined calls are kept as separate frames marked
`(inline)`, which is why `slices.Index` shows up with all the flat time although it
was compiled into `dedupe`.

For heap profiles, `mallocgc` records a stack for one allocation per
`runtime.MemProfileRate` bytes (512 KiB by default) and pprof scales the counts
back up. Profiles are only updated at the end of a GC cycle, so a profile can lag
the program by up to two cycles; collecting first gives a current one.

To profile a service, the blank import registers handlers on
`http.DefaultServeMux`. If the application serves its own mux, as StackLab's does,
serve the default mux on a separate localhost listener:

```go
import _ "net/http/pprof"

go func() { log.Println(http.ListenAndServe("127.0.0.1:6060", nil)) }()
```

```
go tool pprof http://127.0.0.1:6060/debug/pprof/profile?seconds=30
go tool pprof -sample_index=alloc_space http://127.0.0.1:6060/debug/pprof/heap
```

## Design notes

Keep the profile loop cheap and routine. Benchmarks with `b.Loop` and `-benchmem`
make the CPU and allocation cost of hot functions visible in review; `pprof` explains
the numbers; `-gcflags=-m` explains the allocations. Fix the top entry, not the one
you expected, and keep the benchmark so the fix stays fixed.

A CPU profile from production is also a build input. Saved as `default.pgo` in the
main package's directory, it is picked up by `go build` (the default `-pgo=auto`),
which then inlines and devirtualizes more aggressively along the hot paths; the
Go 1.22 notes report 2 to 14% gains for most programs. The profile must come from
representative load, and it is worth refreshing when the code changes a lot.
