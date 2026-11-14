---
id: go.design.copying
title: "A little copying is better than a little dependency; Go proverbs, examined"
track: go
module: design
order: 4
level: intermediate
minutes: 13
since: go1.25
prerequisites: [go.design.packages, go.concurrency.sync]
compare_with: [ts.tooling.bundle, design.paradigms.side-by-side, go.timeline.1-21-to-1-27]
tags: [proverbs, dependencies, modules, stdlib, channels, mutex, culture]
summary: >
  The Go proverbs are design advice, not law. Some are enforced by tools, some
  hold almost always, and some bend: copy a five-line helper rather than import
  a module, but depend on maintained code for protocols and crypto, and use a
  mutex when a channel only adds ceremony.
examples:
  - id: dependency-cost
    file: examples/01_dependency_cost.go
    title: "What one helper from a utility module brings with it"
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "18-27"
        note: >
          The function we want needs no imports; its neighbour needs net/http.
          Go imports whole packages, so every importer of strutil compiles
          net/http and runs its package initialization, whichever function it
          calls.
      - lines: "29"
        note: >
          A require line, and a replace pointing at a local directory so the
          demo works offline. With a real module this line also brings a go.sum
          entry, a version your users must accept as a minimum, and every
          advisory govulncheck will report against it.
      - lines: "45-51"
        note: >
          Five lines with a source and license note. It counts runes, not
          bytes: s[:12] on "Merhaba dünya" would cut ü's two UTF-8 bytes in
          half, while this returns "Merhaba düny…". Small, pure, stable and
          tested code is what the proverb is about.
      - lines: "65-69"
        note: >
          go list -deps prints every package the build compiles; GOOS and
          GOARCH are fixed so the count is the same on Windows and Linux. 190
          packages against 60, and crypto/tls among them, for one function.
      - lines: "74-76"
        note: >
          The linker drops functions nothing calls, so the binary grows less
          than the package count. The rest of the cost stays: compile time,
          init code, upgrades and advisories.
  - id: stdlib-first
    file: examples/02_stdlib_first.go
    title: "Before copying or depending, check the standard library"
    runtime: go
    check: exact
    annotations:
      - lines: "23-29"
        note: >
          Grouping needs no helper. slices.Sorted(maps.Keys(m)) (Go 1.23)
          replaces the "keys, sorted" function that used to be copied into
          every project, or imported from a lodash-style library.
      - lines: "32-34"
        note: >
          A two-key sort: cmp.Compare gives -1/0/+1, and cmp.Or (Go 1.22)
          returns the first non-zero, so the second key only decides ties.
          Swapping a and b in the first Compare sorts descending.
      - lines: "42-43"
        note: >
          slices.Compact removes adjacent duplicates, so sort first. The pair is
          the standard spelling of "unique".
      - lines: "47-49"
        note: >
          slices.Chunk returns an iterator over sub-slices that share the
          original's backing array: no copies, and range-over-func reads it
          like any loop.
      - lines: "52-53"
        note: >
          strings.Cut (Go 1.18) replaced Index-and-slice code; min and max are
          built-ins since Go 1.21. Most "util" functions in older Go codebases
          now have a standard-library name.
  - id: mutex-or-channel
    file: examples/03_mutex_or_channel.go
    title: "Share memory by communicating, or guard it with a mutex? The same counter twice"
    runtime: go
    check: exact
    annotations:
      - lines: "16-30"
        note: >
          The channel version: one goroutine owns the map, and nobody else
          touches it, so no lock is needed. It runs forever: a real version
          needs a Close method or a context to stop it, which the mutex
          version doesn't.
      - lines: "34-38"
        note: >
          A read is a request with a reply channel, two channel operations and
          a goroutine hop for what is a field access in the other version.
      - lines: "41-59"
        note: >
          The mutex version: state with a guard. Effective Go itself says the
          channel approach "can be taken too far" and that a reference count may
          be best done with a mutex around an integer.
      - lines: "58"
        note: >
          maps.Clone returns a shallow copy. Returning c.counts itself would
          hand callers the guarded map, and their reads would race with Inc.
      - lines: "66-73"
        note: >
          Both pass the same concurrent workload with the same result. Pick by
          clarity: channels to hand off work and ownership, a mutex for shared
          state such as caches and counters.
use_cases:
  - "Copy: a short, pure, stable function (truncate, clamp, a retry loop) when the alternative is a module dependency"
  - "Depend: crypto, protocols, database drivers, parsers of formats you don't control, anything that receives security fixes"
  - "Look in the standard library first: slices, maps, cmp, strings and iterators replace most helper libraries since Go 1.21-1.23"
  - "Mutex for state (caches, counters, registries); channels for handing off work, ownership and results"
strengths:
  - "Go's standard library covers HTTP, JSON, crypto, SQL interfaces, templates and testing, so many services need a database driver and little else, unlike a typical npm or Maven tree"
  - "Modules make a dependency's cost visible: one require line, checksums in go.sum, go list -deps and go mod graph to see what came with it"
  - "The proverbs give a shared vocabulary for review comments, the way the Zen of Python does"
weaknesses:
  - "Copied code doesn't get upstream fixes, and govulncheck can't see it, because it scans module versions and their symbols"
  - "The proverbs predate generics and modules (2015); some, such as interface{} says nothing, read differently now"
  - "Slogans get applied without their context: 'share memory by communicating' has produced many channel-based counters that a mutex would do better"
pitfalls:
  - title: "Copying security-sensitive code"
    detail: "A hand-copied JWT check or a copied parser loses every future fix and is invisible to govulncheck. Depend on maintained code for crypto, protocols and formats from outside."
  - title: "Copying without the license notice"
    detail: "MIT and BSD licenses require keeping the copyright and permission notice with substantial copies. Put it in a comment next to the code, with the source and version."
  - title: "Re-implementing what the standard library has"
    detail: "Hand-written contains, keys, unique and chunk helpers are common in pre-1.21 code. Since Go 1.26, go fix rewrites many such loops (its slicescontains, mapsloop and minmax analyzers)."
    bad: "func contains(xs []string, x string) bool { for _, v := range xs { if v == x { return true } }; return false }"
    good: "slices.Contains(xs, x)"
  - title: "A goroutine and two channels for a counter"
    detail: "An owner goroutine needs a lifecycle (start, stop, leak checks) and makes every read a round trip. For plain shared state a mutex is shorter and faster."
  - title: "Returning a guarded map or slice"
    detail: "The lock protects the field, not values handed out: callers keep reading the same map while Inc writes to it."
    bad: "func (c *MutexCounter) Snapshot() map[string]int { c.mu.Lock(); defer c.mu.Unlock(); return c.counts }"
    good: "return maps.Clone(c.counts)"
quiz:
  - q: "Which is the strongest case for adding a dependency rather than copying code?"
    options: ["A six-line rune-aware Truncate", "Verifying JWT signatures", "A clamp function"]
    answer: 1
    why: "Signature verification is security-sensitive and receives fixes; a copy would miss them, and govulncheck couldn't flag it. The other two are small, pure and stable."
  - q: "Your app imports example.com/strutil only to call Truncate, but strutil also imports net/http. What does your build include?"
    options: ["Only Truncate", "net/http and everything it imports, compiled and initialized, although the linker drops unused functions", "Nothing extra until FetchJSON is called"]
    answer: 1
    why: "Go imports whole packages: every imported package is compiled and initialized at startup. Dead-code elimination happens later, at function level, in the linker."
  - q: "What does Effective Go say about 'share memory by communicating'?"
    options: ["Always prefer channels to mutexes", "The approach can be taken too far; a reference count may be best done with a mutex around an integer", "Mutexes are deprecated"]
    answer: 1
    why: "Effective Go introduces the slogan and immediately warns against overdoing it; the Go wiki's advice is to use whichever is most expressive or simple."
  - q: "You copy 20 lines from an MIT-licensed library. What do you keep with them?"
    options: ["Nothing; 20 lines are too few to matter", "The copyright and permission notice, plus a note of the source", "Only the author's name"]
    answer: 1
    why: "The MIT license requires the notice in all copies or substantial portions; recording the source and version also tells future readers where fixes would come from."
further_reading:
  - title: "Go at Google: Language Design in the Service of Software Engineering (dependency hygiene)"
    url: "https://go.dev/talks/2012/splash.article"
  - title: "Effective Go: Share by communicating"
    url: "https://go.dev/doc/effective_go#sharing"
  - title: "Go wiki: Use a sync.Mutex or a channel?"
    url: "https://go.dev/wiki/MutexOrChannel"
  - title: "How Go Mitigates Supply Chain Attacks"
    url: "https://go.dev/blog/supply-chain"
  - title: "Errors are values"
    url: "https://go.dev/blog/errors-are-values"
---

## Concept

The Go proverbs are nineteen one-liners from Rob Pike's talk at Gopherfest SV in 2015
(collected at go-proverbs.github.io). They compress design advice, and like all
slogans they hold in the context they came from. Read them in three groups:

- **Enforced or near-universal.** "Gofmt's style is no one's favorite, yet gofmt is
  everyone's favorite", "Errors are values", "Cgo is not Go", "With the unsafe package
  there are no guarantees", "Clear is better than clever". Tools enforce the first,
  and arguing with the others rarely ends well.
- **Strong defaults that bend.** "A little copying is better than a little dependency"
  holds for short, pure, stable code and fails for crypto, protocols and parsers.
  "Don't communicate by sharing memory, share memory by communicating" is followed in
  the list by "Channels orchestrate; mutexes serialize", which already admits the
  bend. "Make the zero value useful" works until a type needs a dependency or a limit.
  "Don't panic" allows `regexp.MustCompile` at start-up.
- **Changed by later Go.** "interface{} says nothing" predates generics: with type
  parameters (Go 1.18), code that is truly generic no longer needs `interface{}`,
  though `any` in an API still says nothing. The copying proverb predates modules
  (Go 1.11): a dependency then was an unversioned GOPATH checkout, now it is a
  checksummed, versioned requirement, cheaper but not free.

The copying proverb comes from how Go itself was built. The 2012 "Go at Google" talk
says "Dependency hygiene trumps code reuse" and cites the `net` package carrying its
own integer-to-decimal conversion to avoid depending on the formatted I/O package.
The habit persists: in Go 1.27, `go list -f '{{.Imports}}' net` shows neither `fmt`
nor `strconv`; `net` uses a small internal package instead.

## Coming from C/C++/Java/Python/JS

- **JS/npm**: the opposite culture. Micro-packages such as left-pad, and dependency
  trees of hundreds of packages, are normal; Go teams treat each new module as a
  decision. Go also runs no install scripts when fetching modules.
- **Java**: Guava and Apache Commons are reflexive imports. Go's `slices`, `maps`,
  `strings` and `cmp` packages cover much of what they are imported for.
- **Python**: the proverbs play the role of the Zen of Python (`import this`), and
  "batteries included" matches Go's standard library.
- **C/C++**: copying a header-only library into the tree is common because package
  management is hard. Go's module cache and minimal version selection make
  versioned dependencies easy, so copying is a choice, not a workaround.

## Design notes

StackLab applies the proverb as policy. CLAUDE.md asks before any dependency outside
an allow-list. ADR-0008 rejected Windows Job Objects because they need
`golang.org/x/sys`; `taskkill /T` "is boring and ships with Windows". When the runner
later also had to kill what a program leaves running after it exits, which `taskkill`
can't find, gorun took Job Objects after all, without the dependency: four kernel32
functions loaded through the standard library's `syscall.NewLazyDLL` and one structure
declared field by field from Microsoft's documentation, a little copying in place of
`golang.org/x/sys/windows`. ADR-0001 chose
the standard library router over a framework, and the config package reads
environment variables without a config library. The counter-examples are just as
deliberate: StackLab depends on pgx and goose, because nobody should copy a
PostgreSQL wire-protocol implementation or a migration engine.

A workable rule: copy when the code is short, has no state, won't need security
fixes and you can test it; keep the license notice and the source next to it.
Depend when the code implements a protocol, a format you don't control, or
cryptography. Before either, search the standard library.

Copying values has its own version of the trade-off, seen in the third example:
`maps.Clone` and `slices.Clone` are shallow. Handing out a copy of the map protects
the counter's internals; handing out the map itself shares them, lock or no lock.
