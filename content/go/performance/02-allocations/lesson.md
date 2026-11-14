---
id: go.performance.allocations
title: "Reducing allocations; sync.Pool (and when not to use it); the Green Tea GC"
track: go
module: performance
order: 2
level: advanced
minutes: 15
since: go1.26
prerequisites: [go.performance.pprof, go.basics.pointers]
compare_with: [go.testing.benchmarks-fuzz, go.concurrency.sync, design.paradigms.immutability]
tags: [performance, allocations, sync-pool, gc, gogc, gomemlimit, green-tea]
summary: >
  Every heap allocation costs once in malloc and again in GC work, and the
  collector runs more often the faster you allocate. Count allocations with
  AllocsPerRun or -benchmem, fix the hot sites with capacity and reuse, use
  sync.Pool only for churned buffers, and know what Green Tea changed.
examples:
  - id: measure
    file: examples/01_measure.go
    title: "Count allocations per operation, then remove them"
    runtime: go
    check: exact
    annotations:
      - lines: "52-54"
        note: >
          testing.AllocsPerRun runs f, then reports the average number of heap
          allocations per call, as a whole number. It pins GOMAXPROCS to 1
          while measuring, and the counts are stable for a given program and
          Go version, which makes them good test assertions: a test can fail
          when a hot path starts allocating.
      - lines: "20-26"
        note: >
          Eight ids, eight allocations: every += builds a new string holding
          everything so far, and the previous one becomes garbage. For long
          loops the copying is quadratic as well.
      - lines: "28-38"
        note: >
          strings.Builder grows by doubling: 4 allocations here. Grow with a
          good estimate makes it one, and strings.Join computes the exact size
          itself.
      - lines: "40-49"
        note: >
          The same doubling for slices: append without capacity allocates at
          1, 2, 4 and 8 elements. When the final length is known, make([]T, 0,
          n) allocates once.
      - lines: "63"
        note: >
          Two allocations: the result string, and 1234 boxed into an interface
          for the ...any parameter. Values 0 to 255 are boxed for free (the
          runtime keeps a static table), which is why small numbers can make
          this look cheaper in a test than in production.
      - lines: "64-67"
        note: >
          The strconv.Append* family writes into a caller-owned buffer.
          Reusing buf[:0] keeps the capacity, so steady state is zero
          allocations: the pattern behind fast loggers and encoders.
      - lines: "69-71"
        note: >
          m[string(key)] with a []byte key doesn't allocate: the compiler
          recognizes the conversion inside a map index. s := string(key)
          followed by m[s] does allocate when s escapes.
  - id: sync-pool
    file: examples/02_sync_pool.go
    title: "sync.Pool for buffers: Reset, cap the size, store pointers"
    runtime: go
    check: exact
    annotations:
      - lines: "30"
        note: >
          New runs when the pool is empty. A pool holds any, so it is usually
          a package-level variable with one concrete type inside, recovered by
          a type assertion on Get.
      - lines: "33-34"
        note: >
          Get may return an object some other request used. Forgetting Reset
          appends to the previous request's data, which in a server means
          leaking one user's output into another's response.
      - lines: "35-39"
        note: >
          A pool should hold objects of similar cost. One huge request would
          otherwise park a huge buffer in the pool for good; fmt drops buffers
          over 64 KiB for exactly this reason (Go issue 23199).
      - lines: "52-53"
        note: >
          Four allocations without the pool (the buffer doubling from 64 bytes,
          plus the result string), one with it: only the String() copy is
          left. A real saving only when this runs thousands of times a second.
      - lines: "55-59"
        note: >
          Put takes any. A slice header is three words, not a pointer, so
          storing it in an interface allocates on every Put, which defeats the
          pool. staticcheck reports this as SA6002.
      - lines: "60-65"
        note: >
          Pool a pointer to the slice instead (or a *bytes.Buffer): pointers
          fit in an interface without allocating.
  - id: gc-cycles
    file: examples/03_gc_cycles.go
    title: "Allocation drives GC frequency; GOGC and GOMEMLIMIT set the pace"
    runtime: go
    check: exact
    annotations:
      - lines: "12-17"
        note: >
          64 MiB of short-lived garbage, never more than 4 KiB alive. Live
          heap stays tiny, yet the collector must run repeatedly, because the
          pacer triggers on growth since the last cycle, not on total memory.
      - lines: "28-35"
        note: >
          NumGC counts completed cycles. Only booleans and zeros are printed:
          the exact number depends on heap sizes and timing, the fact that it
          is above zero does not.
      - lines: "39"
        note: >
          The same work without allocating: no cycles at all. This is the
          real payoff of reducing allocations; the cheaper malloc is a bonus.
      - lines: "41-42"
        note: >
          debug.SetGCPercent(-1) is GOGC=off. With no memory limit either,
          the heap just grows, which is fine for a short batch job and fatal
          for a server.
      - lines: "44-45"
        note: >
          SetMemoryLimit (GOMEMLIMIT, Go 1.19) is a soft limit: as the heap
          approaches it, the GC runs regardless of GOGC. Containers commonly
          set it a little below the memory limit, with GOGC at its default.
use_cases:
  - "Hot request paths: build responses with strings.Builder or append-style APIs and a known capacity"
  - "Encoders, loggers, template renderers: pooled *bytes.Buffer with Reset and a size cap"
  - "Regression tests: assert testing.AllocsPerRun(100, f) == 0 on a zero-allocation fast path"
  - "Containers: GOMEMLIMIT 5-10% under the container limit, so the GC works harder before the OOM killer acts"
strengths:
  - "Allocation counts are deterministic for a given program and Go version, so they can be asserted in tests; Java's TLAB allocation is cheap too, but per-call counts aren't a standard test assertion there"
  - "Escape analysis keeps many values on the stack automatically; there is no new vs stack choice to make as in C++"
  - "Two knobs, GOGC and GOMEMLIMIT, instead of the JVM's collector choice plus dozens of -XX flags"
  - "Green Tea arrived as a default with no code change: the same binary, rebuilt with Go 1.26 or later, spends less time marking"
weaknesses:
  - "No manual memory management: the GC is the only way memory comes back (the arena package exists only behind GOEXPERIMENT=arenas)"
  - "Allocation sites are invisible in source; you need -gcflags=-m or a profile to see what escapes"
  - "sync.Pool gives no retention guarantee: anything in it may be dropped at any GC"
  - "Interfaces box values: an any parameter can allocate where a concrete type would not"
pitfalls:
  - title: "Pooling everything"
    detail: "Small structs allocate in tens of nanoseconds; Get/Put, the type assertion and Reset can cost as much. Pool objects that are large or expensive to build and churned across many goroutines, and keep the pool only if a benchmark shows the gain."
  - title: "Forgetting Reset on a pooled object"
    detail: "A pooled buffer still holds the previous user's bytes. Reset immediately after Get, not before Put, so a path that forgets Put can't leak data."
    bad: "b := pool.Get().(*bytes.Buffer)\nb.WriteString(user)"
    good: "b := pool.Get().(*bytes.Buffer)\nb.Reset()\nb.WriteString(user)"
  - title: "Putting non-pointer values in a Pool"
    detail: "Put([]byte) allocates a box for the slice header on every call. Pool *[]byte or *bytes.Buffer."
    bad: "pool.Put(buf[:0])"
    good: "pool.Put(&buf) // pool of *[]byte"
  - title: "Returning a pooled object to the caller"
    detail: "Once Put runs, another goroutine may get and overwrite the object. Return a copy (b.String(), bytes.Clone) and Put the original."
  - title: "Optimizing without a measurement"
    detail: "A heap profile (alloc_space or alloc_objects) and -benchmem show where allocations happen. Code that runs once per request rarely matters next to the database round trip."
quiz:
  - q: "What does testing.AllocsPerRun(1000, f) return?"
    options: ["Bytes allocated per call", "The average number of heap allocations per call, as a whole number", "Nanoseconds per call"]
    answer: 1
    why: "It reads runtime.MemStats.Mallocs before and after, divides by the run count with integer division, and pins GOMAXPROCS to 1 while measuring."
  - q: "A pool is declared as sync.Pool{New: func() any { return make([]byte, 0, 512) }}. What is wrong?"
    options: ["Nothing", "Put with a []byte allocates, because a slice header is boxed into an interface", "Slices can't be stored in a Pool"]
    answer: 1
    why: "Only pointer-shaped values fit in an interface without allocating; pool *[]byte instead (staticcheck SA6002)."
  - q: "A program allocates 64 MiB of short-lived garbage while keeping 4 KiB alive. With default settings, does the GC run?"
    options: ["No, the live heap is tiny", "Yes, repeatedly: the next cycle starts when the heap grows by GOGC percent over the last live heap", "Only when the OS runs low on memory"]
    answer: 1
    why: "The pacer's trigger is growth since the last collection. Allocation rate, not live size, decides how often it runs."
  - q: "Which statement about Green Tea is correct for Go 1.27?"
    options: ["It must be enabled with GOEXPERIMENT=greenteagc", "It is the default collector since Go 1.26 and changes how marking scans small objects; programs need no changes", "It replaces GOGC with a new tuning variable"]
    answer: 1
    why: "Go 1.25 shipped it as an experiment, Go 1.26 made it the default. GOGC and GOMEMLIMIT work as before."
further_reading:
  - title: "A Guide to the Go Garbage Collector"
    url: "https://go.dev/doc/gc-guide"
  - title: "The Green Tea Garbage Collector (Go blog)"
    url: "https://go.dev/blog/greenteagc"
  - title: "Go 1.26 Release Notes: Green Tea garbage collector"
    url: "https://go.dev/doc/go1.26"
  - title: "package sync: Pool"
    url: "https://pkg.go.dev/sync#Pool"
  - title: "package testing: AllocsPerRun"
    url: "https://pkg.go.dev/testing#AllocsPerRun"
---

## Concept

A heap allocation costs twice: once in `mallocgc`, which is fast, and again in the
garbage collector, which has to mark what is live and sweep what isn't. The second
cost depends on *how often* the collector runs, and that depends on allocation rate:
with the default `GOGC=100`, a new cycle starts when the heap has grown by roughly
100% over what was live after the previous one (the GC guide has the exact formula).
Fewer allocations mean fewer cycles.

The workflow is measure, locate, fix:

1. **Measure**: `go test -bench . -benchmem` (or `b.ReportAllocs()`) reports
   allocations per operation; `testing.AllocsPerRun` does the same in a plain test.
2. **Locate**: a heap profile with `-sample_index=alloc_space` or `alloc_objects`
   ranks allocation sites; `go build -gcflags=-m` explains why a value escapes.
3. **Fix** with a few standard moves: give slices and builders their capacity up
   front, append into reused buffers (`buf[:0]`, `strconv.AppendInt`), keep hot paths
   free of `any` parameters, and stop returning pointers to small values that could
   be returned by value.

`sync.Pool` is the last of these tools, not the first. It keeps a per-processor set of
reusable objects that the runtime may drop at any GC. It pays off for buffers and
encoders that many goroutines create and discard at a high rate: `fmt` uses one for
its printers.

## Coming from C/C++/Java/Python/JS

- **C/C++**: no `free`, no arenas in the standard library, no custom allocators.
  The equivalent of "allocate less" is the same; the equivalent of a free list is
  `sync.Pool`, which the GC may empty.
- **Java**: generational collectors make short-lived garbage almost free. Go's
  collector is not generational and doesn't move objects, so short-lived garbage still
  costs mark and sweep work; escape analysis (which Java also has) and reuse matter
  more. Tuning is `GOGC` and `GOMEMLIMIT`, not a collector zoo and `-XX` flags.
- **Python/JS**: reference counting (CPython) or a generational GC (V8) with no
  control over layout. Go gives you value types, slices with capacity and stack
  allocation, so you can often avoid the allocation instead of hoping the collector
  is fast.

## How it works

Go's collector is concurrent, non-moving mark-and-sweep. The **pacer** decides when to
start a cycle from `GOGC` and the memory limit, and mutator goroutines help with
marking when they allocate quickly ("assists"), which is how allocation-heavy code
pays for GC in its own latency. `GODEBUG=gctrace=1` prints one line per cycle; from
the third example on the author's machine:

```
gc 1 @0.066s 0%: 0+1.1+0 ms clock, 0+0.59/2.3/0+0 ms cpu, 3->4->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 32 P
```

The `3->4->1 MB, 4 MB goal` part reads: heap at start, heap at end, live heap after
marking, and the target that triggered the cycle.

**Green Tea** changes the mark phase. The classic collector follows pointers object by
object, jumping around memory. Green Tea tracks *pages* of small objects instead: the
work list is a queue of pages, and scanning a page handles every marked object on it
in one pass, which uses CPU caches better and contends less across cores. It shipped
as an experiment in Go 1.25 (`GOEXPERIMENT=greenteagc`) and became the default in
Go 1.26, where the release notes expect a 10 to 40% reduction in GC overhead for
GC-heavy programs, and about 10% more on amd64 CPUs from Intel Ice Lake or AMD Zen 4
on, which get vector instructions for scanning. Some workloads gain little, and
some (one object per page to scan at a time) can be slower.

The opt-out is a build-time flag, and the binary records it. The Go 1.26 notes say
the opt-out "is expected to be removed in Go 1.27"; the 1.27 notes don't mention it,
and the Go 1.27.0 toolchain still accepts it:

```
$ GOEXPERIMENT=nogreenteagc go build -o app .
$ go version -m app
app: go1.27.0-X:nogreenteagc
	...
	build	GOEXPERIMENT=nogreenteagc
```

Go 1.27 also made small allocations cheaper on the malloc side: the compiler now calls
size-specialized allocation routines, up to 30% faster for some objects under 80
bytes (opt-out `GOEXPERIMENT=nosizespecializedmalloc`).

## Design notes

Neither Green Tea nor faster malloc changes the advice. They lower the price of each
allocation and each cycle; avoiding the allocation still removes both. Start with
the profile, fix the top few sites with capacity and reuse, and assert the result
with `AllocsPerRun` in a test so it stays fixed.

Treat `sync.Pool` as an optimization with a maintenance cost: every pooled type needs
Reset discipline, a size cap and a benchmark that justifies it. In a container that
runs only your program, set `GOMEMLIMIT` 5 to 10% below the container's memory limit
(the GC guide's rule of thumb, leaving room for memory the runtime doesn't track) and
leave `GOGC` at its default. `GOGC=off` with only a limit lets the heap run up to the
limit and then collect hard; the guide advises against it when the memory is shared
with other programs.
