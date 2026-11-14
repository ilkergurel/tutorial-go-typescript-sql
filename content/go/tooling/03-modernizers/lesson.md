---
id: go.tooling.modernizers
title: "go fix modernizers and the toolchain directive"
track: go
module: tooling
order: 3
level: intermediate
minutes: 14
since: go1.26
prerequisites: [go.tooling.go-command]
compare_with: [go.timeline.1-21-to-1-27, ts.tooling.lint]
tags: [go-fix, modernizers, go-directive, toolchain, gotoolchain, deprecation]
summary: >
  go fix rewrites code to newer idioms, but only as far as the module's go line
  allows. The go line is both the language version and the minimum toolchain;
  the toolchain line and GOTOOLCHAIN decide which compiler actually runs.
examples:
  - id: version-gated-fixes
    file: examples/01_version_gated_fixes.go
    title: go fix only applies what the go line allows
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "13"
        note: >
          The module declares go 1.21. Every fixer checks this version, or a
          //go:build go1.N line on the file, before suggesting anything.
      - lines: "18-26"
        note: >
          any (Go 1.18) and the min built-in (Go 1.21) are allowed at 1.21,
          so the first diff rewrites both functions.
      - lines: "28-40"
        note: >
          for i := range n and dropping n := n both need Go 1.22. Before 1.22
          the loop shared one n across iterations, so n := n was required:
          without it every deferred call would print the last name.
      - lines: "56"
        note: >
          -diff prints a unified diff and exits with status 1 when there is
          anything to fix, which makes it usable as a CI check. Squares and
          PrintAll are untouched.
      - lines: "58"
        note: >
          go get go@1.22.0 is the documented way to raise the go line. From
          now on the module compiles with Go 1.22 semantics, including
          per-iteration loop variables.
      - lines: "59"
        note: >
          The second diff holds the 1.22 fixes. Removing n := n is safe only
          now, because the go line changed what the loop means.
      - lines: "70-74"
        note: >
          strings.Lines (Go 1.24) yields each line with its newline; the
          ---/+++ diff headers hold absolute temp paths and are skipped.
  - id: go-line
    file: examples/02_go_line.go
    title: The go line is a language version and a minimum toolchain
    runtime: go
    check: regex
    timeout_s: 15
    annotations:
      - lines: "16"
        note: >
          Ranging over an int is Go 1.22 syntax. The installed compiler knows
          it; whether it accepts it depends on the module.
      - lines: "31-33"
        note: >
          With go 1.21 the compiler runs with -lang=go1.21 and names the
          version the feature needs. This is how to check a since: claim:
          lower the go line and read the error.
      - lines: "40-41"
        note: >
          Since Go 1.21 the go line is a hard requirement. With
          GOTOOLCHAIN=local the go command refuses a module that needs a
          newer Go than itself.
      - lines: "42"
        note: >
          With auto, the default, it tries to download that toolchain as the
          module golang.org/toolchain, verified against the checksum database.
          Go 1.99 doesn't exist and GOPROXY=off blocks the network, so it fails.
      - lines: "56"
        note: >
          GOTOOLCHAIN in the environment wins over go env -w settings and the
          default in $GOROOT/go.env, which is auto in standard distributions.
          The expected output is a regex because the error names the local
          Go version and platform.
  - id: fix-inline
    file: examples/03_fix_inline.go
    title: "//go:fix inline: a deprecation that migrates its callers"
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "19-24"
        note: >
          The old function is rewritten as a call to the new one and marked
          //go:fix inline. The Deprecated: paragraph makes editors and
          staticcheck flag uses; the directive lets go fix remove them.
      - lines: "37"
        note: >
          The caller, which could just as well live in another module that
          depends on this one.
      - lines: "57"
        note: >
          -inline selects one fixer (go tool fix help lists them all). Each
          call to Format is replaced by Format's body with the arguments
          substituted; the inliner keeps evaluation order and side effects.
      - lines: "58-59"
        note: >
          After applying, the program prints the same thing: the fix changes
          the code, not its behavior.
use_cases:
  - "After raising a module's go line: go fix ./... in its own commit, so the mechanical diff is reviewed separately"
  - "CI check that new code uses current idioms: go fix -diff ./... fails when it has suggestions"
  - "Renaming or replacing an API in a library: keep the old function as a //go:fix inline wrapper and let callers migrate with go fix"
  - "Making every developer and CI build with at least a given compiler: toolchain go1.27.1 in go.mod; GOTOOLCHAIN=go1.27.1 in CI for an exact version"
strengths:
  - "Fixes are gated by the file's language version, so go fix never suggests syntax or APIs the module can't use, unlike a codemod that targets the latest release"
  - "Built on the vet analysis framework and shipped with the toolchain: no plugin to install, and new modernizers arrive with each Go release"
  - "//go:fix inline turns a deprecation notice into an automatic migration, which Java's @Deprecated or JSDoc's @deprecated cannot do"
  - "Toolchain switching downloads checksum-verified compilers on demand, so go.mod alone states which Go a project needs, like .nvmrc plus nvm but built into the go command"
weaknesses:
  - "Only patterns the fixers recognize get rewritten: a hand-rolled max loop or an if/return clamp stays as it is"
  - "The toolchain line is a minimum, not a pin: a developer with a newer Go installed still uses that newer Go"
  - "GOTOOLCHAIN=auto downloads compilers from the module proxy, which locked-down networks block; there you need GOTOOLCHAIN=local and the right version installed"
pitfalls:
  - title: "Concluding that old code is already modern"
    detail: "With an old go line, go fix silently skips everything newer. Raise the go line first, then run go fix."
  - title: "Raising a library's go line casually"
    detail: "Since Go 1.21 every module that requires yours must declare at least your go version. Keep a library's go line as low as its code allows; applications can use the newest."
  - title: "Treating the toolchain line as an exact pin"
    detail: "It only takes effect when the local Go is older. For identical builds, set GOTOOLCHAIN=go1.27.1 in CI or build in an image with that exact version."
  - title: "Writing a //go:fix inline wrapper with extra logic"
    detail: "The inliner copies the body into every caller. Keep the wrapper a single call to the new API, or callers inherit logging, defaults or other surprises."
quiz:
  - q: "A module declares go 1.21 and has for i := 0; i < n; i++ loops. go fix -diff ./... shows no rangeint changes. Why?"
    options: ["rangeint is disabled by default", "Ranging over an int needs Go 1.22, and the go line says 1.21", "go fix only rewrites code in the main package"]
    answer: 1
    why: "Fixers only suggest features allowed by the file's language version. Raise the go line (go get go@1.22.0) and run go fix again."
  - q: "go.mod says go 1.25.0 and toolchain go1.27.1. The developer has Go 1.28.0 installed and GOTOOLCHAIN=auto. Which Go builds the module?"
    options: ["1.25.0", "1.27.1", "1.28.0"]
    answer: 2
    why: "The toolchain line is used only when the default toolchain is older than it. A newer local toolchain wins."
  - q: "Why does go fix remove n := n inside a range loop only when the go line is at least 1.22?"
    options: ["It is purely stylistic", "Go 1.22 gave each iteration its own variable, but only in modules declaring go 1.22 or later", "Older compilers reject the shadowing"]
    answer: 1
    why: "Loop-variable semantics follow the module's go line. Under 1.21 semantics, removing the copy would make every closure see the last value."
  - q: "What does //go:fix inline on func Format(k int) string { return Lira(k) } do?"
    options: ["The compiler inlines Format for speed", "go fix rewrites calls to Format into calls to Lira", "go vet reports every call to Format as an error"]
    answer: 1
    why: "The directive is for the inline fixer. Compiler inlining is automatic and unrelated."
further_reading:
  - title: "Go 1.26 Release Notes: go fix"
    url: "https://go.dev/doc/go1.26#go-command"
  - title: "Go 1.27 Release Notes"
    url: "https://go.dev/doc/go1.27"
  - title: "Go Toolchains"
    url: "https://go.dev/doc/toolchain"
  - title: "Go Modules Reference: go directive"
    url: "https://go.dev/ref/mod#go-mod-file-go"
  - title: "Fixing For Loops in Go 1.22"
    url: "https://go.dev/blog/loopvar-preview"
---

## Concept

**go fix** was rewritten in Go 1.26 as the home of *modernizers*: analyzers, built on
the same framework as `go vet`, whose suggested fixes move code to newer idioms
without changing behavior (`interface{}` to `any`, three-clause loops to
`range n`, hand-written loops to `slices.Contains`, `sort.Slice` to `slices.Sort`,
and a few dozen more). `go fix ./...` applies them, `go fix -diff ./...` previews,
`go tool fix help` lists them, and a flag such as `-rangeint` selects one. Go 1.27
added `atomictypes`, `embedlit`, `slicesbackward` and `unsafefuncs`, renamed
`waitgroup` to `waitgroupgo` and removed `fmtappendf`.

Every fix is limited by the **language version** of the file it edits: the `go` line
of its module, or a `//go:build go1.N` constraint on the file. That `go` line does three
jobs at once:

1. It sets the language version the compiler enforces for the module's packages.
2. Since Go 1.21 it is a hard minimum: an older toolchain refuses the module.
3. It feeds toolchain selection.

The optional **toolchain** line names a *suggested* toolchain, used only when the
installed Go is older. **GOTOOLCHAIN** controls switching: `auto` (the default,
short for `local+auto`) uses the installed Go unless `go.mod` asks for newer and then
downloads it; `local` never switches; `go1.27.1` always uses exactly that version;
`go1.27.1+auto` starts from it and may still go higher.

## Coming from C/C++/Java/Python/JS

- **Python**: `go fix` is `pyupgrade --py311-plus` built into the toolchain, with the
  target version read from the project (`go` line ≈ `requires-python`) instead of a
  flag.
- **Java**: the `go` line is like `--release 21` for the language, and the toolchain
  line like a Gradle toolchain spec that is fetched automatically. Modernizers resemble
  OpenRewrite recipes, without the plugin.
- **JS/TS**: codemods (jscodeshift) are third-party and version-blind. `engines` in
  `package.json` usually only warns; the `go` line is enforced. The toolchain line
  plus `GOTOOLCHAIN=auto` is what `.nvmrc` plus nvm or Volta does, without an extra
  tool.
- **C/C++**: `-std=c++20` is per build; the `go` line travels with the module, so
  every consumer compiles it with the same language rules.

## Design notes

Upgrade in two steps: raise the `go` line (`go get go@1.N.0`), run the tests, then run
`go fix ./...` and commit its diff separately so reviewers see a purely mechanical
change. Libraries should keep their `go` line as low as their code allows, because
it becomes a requirement for every module that depends on them; applications can
track the newest release. For reproducible builds, the toolchain line sets a floor
for everyone, and CI pins the exact compiler with `GOTOOLCHAIN=go1.27.1` or a
versioned build image. Since Go 1.27, `go test` also runs vet's `stdversion` check,
which flags standard library symbols newer than the file's `go` version, the API
counterpart of the compile error in example 2. When you deprecate an API you own,
express the old function in terms of the new one and mark it `//go:fix inline`:
the deprecation then migrates its own callers.
