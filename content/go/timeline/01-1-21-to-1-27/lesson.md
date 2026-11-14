---
id: go.timeline.1-21-to-1-27
title: "What changed from Go 1.21 to 1.27 and how to spot old code"
track: go
module: timeline
order: 1
level: intermediate
minutes: 14
since: go1.27
prerequisites: [go.iterators.range, go.concurrency.sync]
compare_with: [go.tooling.modernizers, sql.modern.16-to-18]
tags: [go-versions, loopvar, iterators, modernization, uuid, json-v2]
summary: >
  Seven releases changed everyday Go: per-iteration loop variables, range over
  ints and functions, wg.Go, os.Root, errors.AsType, new(expr), json/v2 and a
  standard uuid package. Knowing which version brought what tells you how old a
  piece of code, or a blog post, is.
examples:
  - id: loopvar-old
    file: examples/01_loopvar_old.go
    title: "Go 1.21 loop semantics, reproduced on Go 1.27"
    runtime: go
    check: exact
    annotations:
      - lines: "1"
        note: >
          A //go:build line naming a Go release sets the language version of
          this one file (in modules at go 1.21 or later). Here it brings back
          the pre-1.22 loop semantics on a 1.27 toolchain. Delete the line and
          run again: all three outputs become Adana Bursa Çorum / 0 1 2.
      - lines: "16-19"
        note: >
          Before Go 1.22, one variable c served every iteration. Each closure
          captured that variable, not its value, and saw its last value after
          the loop: Çorum three times.
      - lines: "23"
        note: >
          The fix everyone wrote for ten years. In code whose go line is 1.22
          or later it is redundant, and go fix (the forvar analyzer) removes it.
          Seeing it is a reliable sign the code predates 1.22.
      - lines: "27-30"
        note: >
          The same applies to three-clause loops: all three pointers point at
          one i, which ended at 3. Since Go 1.22, each iteration gets a copy of
          i, and the increment applies to the next copy.
  - id: collections
    file: examples/02_collections.go
    title: "Go 1.21 to 1.24: built-ins, sorted keys, iterators, range over int"
    runtime: go
    check: exact
    annotations:
      - lines: "15-23"
        note: >
          The old way to iterate a map in order: copy the keys, sort.Strings,
          loop. Still correct, just longer than it needs to be.
      - lines: "27-35"
        note: >
          An iterator is a plain function of type iter.Seq[int]: it calls yield
          for each value and must return when yield returns false. No channel,
          no goroutine, no interface.
      - lines: "29"
        note: >
          Range over an int (Go 1.22) replaces for i := 0; i < limit; i++.
      - lines: "38-39"
        note: >
          Byte order, both ways: İ is U+0130, encoded as C4 B0, so İzmir sorts
          after Muğla. Turkish alphabetical order needs a collator such as
          golang.org/x/text/collate, whatever the Go version.
      - lines: "40-41"
        note: >
          min and max are built-ins since Go 1.21 and take any number of
          ordered arguments; cmp.Or (Go 1.22) returns its first non-zero
          argument, handy for defaults from configuration.
      - lines: "44-49"
        note: >
          break inside a range-over-func loop makes yield return false, and
          Evens stops. A million-element range is never materialized.
      - lines: "53"
        note: >
          strings.SplitSeq (Go 1.24) yields the parts without allocating the
          slice that strings.Split builds. strings.Lines and FieldsSeq are its
          siblings.
  - id: library
    file: examples/03_library.go
    title: "Go 1.24 to 1.27: wg.Go, new(expr), os.Root, errors.AsType, json/v2, uuid"
    runtime: go
    check: exact
    annotations:
      - lines: "21-27"
        note: >
          wg.Go (Go 1.25) does Add(1), starts the goroutine and calls Done when
          it returns. The closure can use i directly because loop variables are
          per iteration since 1.22.
      - lines: "30-31"
        note: >
          new(true) (Go 1.26) allocates a variable with that initial value.
          Old code has helpers such as func ptr[T any](v T) *T for optional
          fields like this one.
      - lines: "36-47"
        note: >
          os.Root (Go 1.24) resolves every name inside its directory and
          refuses ../.env. errors.AsType (Go 1.26) returns the typed error and
          a bool, instead of declaring a target and passing its address to
          errors.As.
      - lines: "49-53"
        note: >
          In Go 1.27 encoding/json runs on the v2 implementation but keeps v1
          behaviour: the duplicate key is accepted and the last value wins.
          encoding/json/v2 rejects it by default.
      - lines: "55-58"
        note: >
          The standard library's uuid package (Go 1.27; import path "uuid").
          New is currently NewV4; NewV7 starts with a millisecond timestamp, so
          ids sort by creation time and index well as primary keys. Parse also
          accepts braces, urn:uuid: and upper case.
use_cases:
  - "Reading code or a blog post and dating it: n := n in loops, sort.Strings on map keys, wg.Add(1) before go func"
  - "Reviewing a pull request in a module at go 1.27 that still uses pre-1.22 idioms copied from older code"
  - "Upgrading a service: raise the go line, run the tests, run go fix ./... until its diff is empty"
  - "Choosing dependencies: uuid v4/v7, JSON strictness and errgroup-free fan-out no longer need third-party modules"
strengths:
  - "Language changes are gated per module by the go line (and per file by //go:build go1.N), so old code keeps its meaning after a toolchain upgrade; Java and C++ change semantics only through compiler flags for the whole build"
  - "The compiler names the release a feature needs (\"requires go1.22 or later\"), and vet's stdversion check does the same for library APIs"
  - "go fix (rewritten in Go 1.26) applies most modernizations mechanically: rangeint, forvar, minmax, waitgroupgo, errorsastype, newexpr, stringsseq, omitzero"
  - "The Go 1 compatibility promise means old idioms still compile and run; modernizing is a choice, not a migration deadline"
weaknesses:
  - "Old idioms stay valid, so codebases mix styles for years; reviewers have to know both"
  - "Library APIs aren't gated by the compiler: a module at go 1.21 can call errors.AsType and only vet (stdversion, run by go test since 1.27) notices"
  - "The uuid package covers versions 4 and 7 only; name-based v5 or other versions still need a module such as github.com/google/uuid"
  - "Behaviour changes that aren't syntax (timer channels in 1.23, encoding/json on the v2 engine in 1.27) need release-note reading; GODEBUG settings and GOEXPERIMENT=nojsonv2 roll some back, but only for a few releases (asynctimerchan is gone in 1.27)"
pitfalls:
  - title: "Removing x := x without raising the go line"
    detail: "The copy is only redundant when the module (or file) is at go 1.22 or later. In a go 1.21 module, removing it brings the shared-variable bug back."
  - title: "Concluding from the toolchain version"
    detail: "Building with Go 1.27 doesn't give a module 1.27 semantics; its go line does. A module at go 1.21 built with Go 1.27 still shares loop variables."
  - title: "Hand-written ptr helpers and Add/Done pairs in new code"
    detail: "They still work, but each has a one-call replacement with fewer ways to go wrong (a forgotten Done, an Add inside the goroutine)."
    bad: "wg.Add(1)\ngo func() {\n\tdefer wg.Done()\n\twork(i)\n}()"
    good: "wg.Go(func() { work(i) })"
  - title: "Assuming encoding/json became strict in 1.27"
    detail: "The v1 package now runs on the v2 engine, but with v1 semantics: duplicate names, case-insensitive matching and invalid UTF-8 are accepted as before. Strict defaults come from importing encoding/json/v2."
quiz:
  - q: "A closure inside for _, v := range items captures v. When does every closure see the last item?"
    options: ["Always", "When the module's go line (or the file's //go:build go1.N) is below 1.22", "Only with goroutines"]
    answer: 1
    why: "Per-iteration variables are a language change gated by the language version, not by the installed toolchain."
  - q: "Which release added each? wg.Go, os.Root, errors.AsType, package uuid."
    options: ["1.25, 1.24, 1.26, 1.27", "1.24, 1.24, 1.25, 1.26", "1.23, 1.25, 1.26, 1.27"]
    answer: 0
    why: "The API files of the Go 1.27 distribution list WaitGroup.Go in go1.25, os.OpenRoot in go1.24, errors.AsType in go1.26 and package uuid in go1.27."
  - q: "In Go 1.27, what does encoding/json (v1) do with {\"name\":\"a\",\"name\":\"b\"}?"
    options: ["Rejects it: v1 now uses v2 rules", "Keeps the first value", "Accepts it; the last value wins"]
    answer: 2
    why: "v1 is backed by the v2 implementation but preserves v1 behaviour; only encoding/json/v2 rejects duplicate names by default."
  - q: "Your module says go 1.21 and calls errors.AsType. What happens on Go 1.27?"
    options: ["Compile error naming go1.26", "It builds; go vet and go test report errors.AsType requires go1.26 or later", "It panics at run time"]
    answer: 1
    why: "The compiler gates syntax by language version; standard-library symbols are checked by vet's stdversion analyzer."
further_reading:
  - title: "Go release history"
    url: "https://go.dev/doc/devel/release"
  - title: "Go 1.22 release notes: language changes"
    url: "https://go.dev/doc/go1.22#language"
  - title: "Fixing For Loops in Go 1.22"
    url: "https://go.dev/blog/loopvar-preview"
  - title: "Range Over Function Types"
    url: "https://go.dev/blog/range-functions"
  - title: "Go 1.27 release notes: package uuid"
    url: "https://go.dev/doc/go1.27#uuid"
  - title: "package uuid"
    url: "https://pkg.go.dev/uuid"
  - title: "Go, Backwards Compatibility, and GODEBUG"
    url: "https://go.dev/doc/godebug"
---

## Concept

Go changes slowly and keeps old code compiling, so code written for Go 1.20 still runs
on Go 1.27. What changes is what idiomatic code looks like. The releases from 1.21
(August 2023) to 1.27 (August 2026), by what you meet daily:

- **1.21**: built-ins `min`, `max`, `clear`; packages `slices`, `maps`, `cmp`,
  `log/slog`; `sync.OnceFunc`/`OnceValue`; the `go` line in `go.mod` becomes a
  minimum requirement, and toolchains can switch themselves.
- **1.22**: each loop iteration gets its own variables; `for i := range n`;
  `math/rand/v2`; `net/http` patterns with methods and wildcards (`"GET /users/{id}"`,
  `r.PathValue`); `cmp.Or`.
- **1.23**: range over functions (`iter.Seq`, `iter.Seq2`), with iterator helpers in
  `slices` and `maps` (`slices.Sorted(maps.Keys(m))`); timers whose channels need no
  draining; package `unique`.
- **1.24**: generic type aliases; `os.Root`; `testing.B.Loop` and `T.Context`;
  `strings.Lines`, `SplitSeq`, `FieldsSeq`; the `omitzero` JSON tag option; `tool`
  directives in `go.mod`; a Swiss-table map implementation.
- **1.25**: `sync.WaitGroup.Go`; `testing/synctest`; `GOMAXPROCS` that respects
  container CPU limits on Linux; experimental Green Tea GC and `encoding/json/v2`.
- **1.26**: `new(expr)`; `errors.AsType`; generic types that refer to themselves in
  their constraints; Green Tea GC on by default; `go fix` rebuilt around modernizers.
- **1.27**: generic methods; struct literal keys that name promoted fields;
  `encoding/json/v2` and `jsontext` without an experiment flag, with `encoding/json`
  running on the v2 engine; package `uuid`; `httptest.NewTestServer`.

The stdlib UUID the curriculum mentions is real: `import "uuid"` (no domain in the path),
with `New` (currently version 4), `NewV4`, `NewV7`, `Parse`, `MustParse`, `Nil`, `Max`
and `Compare`.

## Coming from C/C++/Java/Python/JS

- **Java**: comparable to the jump from Java 8 to 21 (records, `var`, switch
  expressions, virtual threads), except Go gates language changes per module through
  `go.mod` rather than with `--release` for the whole build.
- **Python**: the loop-variable change fixes Go's version of the late-binding closure
  trap (`lambda: i` in a loop), which Python still has.
- **JS**: the same story as `var` versus `let` in `for` loops: ES2015 gave each
  iteration its own binding, and Go 1.22 did the same almost nine years later.
  `iter.Seq` plays the role of generator functions, without `function*`.
- **C++**: like moving between `-std=c++17` and `-std=c++20`, chosen by each module
  instead of each build.

## How it works

Every file is compiled at a **language version**: the `go` line of its module, or a
`//go:build go1.N` constraint on the file. The compiler rejects newer syntax and names
the release it needs, as these real errors for a module at `go 1.21` show:

```
cannot range over 3 (untyped int constant): requires go1.22 or later (-lang was set to go1.21; check go.mod)
new(42) requires go1.26 or later (-lang was set to go1.21; check go.mod)
generic method requires go1.27 or later (-lang was set to go1.21; check go.mod)
```

Standard-library additions are not compile errors. They are reported by vet's
`stdversion` analyzer, which `go test` runs by default since Go 1.27:

```
main.go:23:5: sync.(*WaitGroup).Go requires go1.25 or later (module is go1.21)
main.go:25:16: errors.AsType requires go1.26 or later (module is go1.21)
main.go:26:11: uuid.NewV7 requires go1.27 or later (module is go1.21)
```

Behaviour changes in the runtime and library follow the `go` line too, through
**GODEBUG** defaults. Built with Go 1.27, `panic(nil)` in a module at `go 1.20` still
recovers as `nil`; at `go 1.21` it recovers as `runtime error: panic called with nil
argument`. These settings are temporary: Go 1.27 removed `asynctimerchan`, so pre-1.23
timer channels are gone whatever the go line says.

### Spotting old code

Signs that code predates a release, with the `go fix` analyzer that rewrites it:

- `x := x` inside a loop: before 1.22 (`forvar`).
- `for i := 0; i < n; i++` with `i` only counting: before 1.22 (`rangeint`).
- if/else to clamp values, `sort.Slice` on basic types: before 1.21 (`minmax`, `slicessort`).
- `for _, s := range strings.Split(...)`: before 1.24 (`stringsseq`).
- `wg.Add(1)` with `defer wg.Done()` in the goroutine: before 1.25 (`waitgroupgo`).
- `var pe *fs.PathError; errors.As(err, &pe)`: before 1.26 (`errorsastype`).
- `func ptr[T any](v T) *T { return &v }`: before 1.26 (`newexpr`).
- `json:",omitempty"` on a `time.Time` or struct field: before 1.24 (`omitzero`).
- `context.WithCancel(context.Background())` in tests: before 1.24 (`testingcontext`).
- `for i := 0; i < b.N; i++` in benchmarks, `github.com/google/uuid` for v4 or v7 ids,
  channels used as generators, `filepath.Join` plus prefix checks on user paths:
  no fixer; change by hand.

A real `go fix -diff` on such code (module at go 1.27) rewrote the loop, the WaitGroup,
the `errors.As` and the pointer helper in one pass, and removed `i := i` on the second:

```
-	for i := 0; i < n; i++ {
+	for i := range n {
 		i := i
-		wg.Add(1)
-		go func() {
-			defer wg.Done()
+		wg.Go(func() {
 			out[i] = i * i
-		}()
+		})
```

## Design notes

Upgrade in this order: raise the `go` line (`go get go@1.27.0`), run the tests, then
run `go fix ./...` until it reports nothing, and commit that mechanical diff
separately. The language-version gate means the first step can change behaviour (loop
variables, GODEBUG defaults), which is why tests come before the cleanup.

When you read Go elsewhere, date it before copying it. Advice to write `v := v`, to
drain a timer before `Reset`, to call `rand.Seed`, or to pick a third-party router for
path parameters was correct once and is noise now. The opposite also holds: code
using `wg.Go` or `new(expr)` needs at least the release that introduced them, which
matters for libraries whose users may be on older toolchains.

Some decisions don't change with versions. Sorting strings still compares bytes, so
`İzmir` sorts after `Muğla`; Turkish order still needs a collator. And a UUID in the
standard library doesn't change the database side: PostgreSQL 18 has its own
`uuidv7()`, so decide which side generates ids and keep it to one.
