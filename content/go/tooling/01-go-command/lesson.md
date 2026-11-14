---
id: go.tooling.go-command
title: "The go command: mod, run, build, test, vet, fmt, doc"
track: go
module: tooling
order: 1
level: basic
minutes: 12
since: go1.16
prerequisites: []
compare_with: [ts.tooling.lint, ts.mental-model.erasable]
tags: [go-command, modules, packages, vet, gofmt, testing, toolchain]
summary: >
  One binary builds, runs, tests, vets, formats and documents Go code. A
  package is a directory of files compiled together; a module is a tree of
  packages under one go.mod, versioned as a unit.
examples:
  - id: modules-packages
    file: examples/01_modules_packages.go
    title: "go mod init makes a module of packages; go list, go run and go doc show it"
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "19"
        note: >
          An import path is the module path plus the package's directory:
          example.com/hello + greet. There are no relative imports inside a
          module, so moving a directory means updating its import paths.
      - lines: "48"
        note: >
          go mod init writes go.mod with the module path and a go line, the
          Go version the module needs. Every directory below it belongs to the
          module, down to the next directory with its own go.mod. go mod tidy
          would add requirements for imported modules; there are none here.
      - lines: "49-50"
        note: >
          go list -m lists modules, plain go list lists packages. ./... means
          this directory and everything below it. The output shows that the
          import path and the package name are different things: the root
          package is imported as example.com/hello but declares package main.
      - lines: "51"
        note: >
          go run . compiles the main package in the current directory into a
          temporary binary and runs it (since Go 1.24 that binary is cached).
          Use go build when you want a file to ship.
      - lines: "52"
        note: >
          go doc prints doc comments from source, exported identifiers only.
          go doc -u adds unexported ones; go doc strings.Cut works for any
          package or symbol, standard library included.
      - lines: "58-60"
        note: >
          The program drives the real go command from inside a lesson. It
          pins GOTOOLCHAIN=local and GOPROXY=off so a stray version line can
          never trigger a download.
  - id: check-before-commit
    file: examples/02_check_before_commit.go
    title: The compiler accepts it; go vet, go test and go fmt do not
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "15-22"
        note: >
          Two problems the compiler doesn't care about: a %d verb fed a string,
          and four-space indentation where gofmt uses tabs.
      - lines: "47"
        note: >
          go build with several packages, or a non-main package, compiles and
          discards the results: a pure compile check. Silence means success,
          as with most go subcommands.
      - lines: "48"
        note: >
          vet's printf analyzer checks format strings against argument types
          and exits with status 1, which is what fails a CI step.
      - lines: "49"
        note: >
          go test runs a high-confidence subset of vet (printf, errorsas,
          stdversion and a few more) before the tests, and refuses to run them
          if vet complains: "[build failed]", no test executed.
      - lines: "50"
        note: >
          With -vet=off the test runs and shows the bug: fmt never panics on a
          wrong verb, it writes %!d(string=Ayşe) into the output. Vet found it
          without running anything.
      - lines: "51"
        note: >
          go fmt runs gofmt -l -w on the packages and prints each file it
          rewrote. In CI, run gofmt -l . and fail if it prints anything.
      - lines: "54"
        note: >
          Test timings change on every run, so the demo strips them. That is
          also why the go-test runtime compares output with "contains".
  - id: build-info
    file: examples/03_build_info.go
    title: What go build records inside the binary
    runtime: go
    check: regex
    annotations:
      - lines: "9-11"
        note: >
          Every module-mode binary carries this record. go version -m ./app
          prints it without running the program, which is how scanners such as
          govulncheck find out which Go and which dependency versions built it.
      - lines: "16"
        note: >
          The toolchain's version, such as go1.27.1. It differs between
          machines, which is why this example's expected output is a regex.
      - lines: "18-19"
        note: >
          (devel) means "built from a local directory, version unknown". go
          install example.com/tool@v1.2.3 records v1.2.3, and since Go 1.24
          go build in a Git checkout records the tag or a pseudo-version.
          Dependencies are listed with versions and checksums; this program
          has none, because the standard library is not a module dependency.
      - lines: "21-28"
        note: >
          Build settings include the target platform. Inside a Git checkout go
          build also adds vcs.revision, vcs.time and vcs.modified (Go 1.18;
          -buildvcs=false turns that off). The lesson runs in a temp dir, so
          there is nothing to stamp.
use_cases:
  - "Day-to-day loop: go run . while developing, go test ./... before committing, go build -o bin/app ./cmd/app for a release"
  - "CI gate: gofmt -l . must print nothing, then go vet ./... and go test -race ./..."
  - "Dependencies: go get example.com/lib@v1.4.0 to add or upgrade, go mod tidy to drop unused ones and fill go.sum"
  - "Command-line tools: go install example.com/tool@latest puts a binary in GOBIN; go get -tool and go tool (Go 1.24) track dev tools in go.mod"
  - "Finding out which Go built a binary: go version -m ./app"
strengths:
  - "One binary replaces what Java splits across Maven or Gradle plugins and JS splits across npm scripts, ESLint, Prettier and Jest, with no config file to keep in sync"
  - "No build scripts: the import graph is the build graph, and neither fetching nor building runs dependency code, unlike npm install hooks or Gradle build scripts"
  - "Build and test caches are keyed by content (sources, flags, compiler), not timestamps, so go test ./... after a one-line change only rebuilds and reruns what depends on it"
  - "gofmt has no style options, so formatting debates and formatter configuration files don't exist"
weaknesses:
  - "There is no task runner: code generation, migrations or Docker images go into go generate directives, a Makefile or scripts"
  - "vet's default analyzers are deliberately conservative; unchecked errors and many other bugs need staticcheck or golangci-lint, separate installs"
  - "A cached ok is not a fresh run: a test that talks to a database is cached like any other unless you pass -count=1"
pitfalls:
  - title: "go run main.go in a multi-file package"
    detail: "Naming files compiles only those files, so helpers in other files of package main are undefined. Run the package instead."
    bad: "go run main.go"
    good: "go run ."
  - title: "Treating vet findings as style nits"
    detail: "The compiler doesn't check format verbs, struct tags or misuse of errors.As; vet does, and go test runs the important analyzers anyway. Keep go vet ./... green."
  - title: "Believing a cached test result"
    detail: "\"ok example.com/x (cached)\" means the test binary and its inputs were unchanged, so the old result was replayed. Use -count=1 when the test depends on something outside the module."
  - title: "A module path without a dot"
    detail: "Import paths without a dot in the first element are reserved for the standard library. Use your repository path (github.com/you/app) or example.com/...; the names example and test are reserved for local experiments."
quiz:
  - q: "go.mod says module example.com/shop. Which import path does the package in the directory tax/ have?"
    options: ["tax", "./tax", "shop/tax", "example.com/shop/tax"]
    answer: 3
    why: "An import path is the module path joined with the package's directory inside the module; Go has no relative imports in module mode."
  - q: "go build ./... in a module with three library packages prints nothing. What happened?"
    options: ["Nothing was compiled", "Everything compiled and the results were discarded: it was a compile check", "Three binaries were written next to the packages"]
    answer: 1
    why: "When building several packages or a non-main package, go build only checks that they compile. Use -o or a single main package to get a binary."
  - q: "go test fails with a fmt.Sprintf vet message and runs no tests. Why?"
    options: ["go test runs every vet analyzer", "go test runs a high-confidence subset of vet, including printf, before running tests", "The test file has a syntax error"]
    answer: 1
    why: "go help test lists the subset (atomic, bools, buildtag, directive, errorsas, ifaceassert, nilfunc, printf, stdversion, stringintconv, tests). -vet=off disables it, -vet=all runs everything."
  - q: "How do you make go test run tests again instead of printing (cached)?"
    options: ["go test -v ./...", "go test -count=1 ./...", "go test -run . ./..."]
    answer: 1
    why: "-v and -run are on the list of cacheable flags, so results can still be replayed. -count is not, and -count=1 is the documented idiom for bypassing the cache."
further_reading:
  - title: "Command go"
    url: "https://pkg.go.dev/cmd/go"
  - title: "How to Write Go Code"
    url: "https://go.dev/doc/code"
  - title: "Go Modules Reference"
    url: "https://go.dev/ref/mod"
  - title: "Command vet"
    url: "https://pkg.go.dev/cmd/vet"
  - title: "How Go Mitigates Supply Chain Attacks"
    url: "https://go.dev/blog/supply-chain"
---

## Concept

The `go` command is compiler driver, dependency manager, test runner, linter,
formatter and documentation viewer in one binary. It works on two units:

- A **package** is one directory of `.go` files that share a `package` clause and are
  compiled together. It is the unit of import and of visibility.
- A **module** is a tree of packages with a `go.mod` at its root. It is the unit of
  versioning: you require, upgrade and publish modules, never single packages.
  `go.mod` names the module (its path), the Go version it needs, and the modules it
  requires; `go.sum` records checksums of every module that contributes to the build.

A package's import path is the module path plus its directory, so the code layout on
disk *is* the import graph, and the import graph is the build graph. Commands take
package patterns: `.` is the package in the current directory, `./...` is it and
everything below, and `example.com/hello/greet` is an import path.

| You want to | Command |
|---|---|
| start a module | `go mod init example.com/app` |
| add or upgrade a dependency | `go get example.com/lib@v1.4.0` |
| make go.mod and go.sum match the imports | `go mod tidy` |
| run while developing | `go run .` |
| build a binary | `go build -o bin/app ./cmd/app` |
| test, check, format | `go test ./...`, `go vet ./...`, `go fmt ./...` |
| read docs offline | `go doc strings.Cut`, `go doc -all ./greet` |
| install a CLI tool | `go install example.com/tool@latest` |
| inspect settings | `go env`, `go env GOPATH` |

## Coming from C/C++/Java/Python/JS

- **Java**: Maven or Gradle, Checkstyle, Javadoc and the JUnit runner are all built in,
  with no plugin configuration. `go.mod` is like the dependency section of a POM, but
  versions are minimums: minimal version selection picks the highest minimum anyone
  requires, deterministically, without a separate lock file.
- **JS/Node**: `go.mod` ≈ `package.json` without `scripts`; `go.sum` holds hashes, not a
  resolved tree. There is no `node_modules`: downloaded modules live once per version
  in a shared, read-only module cache.
- **Python**: no virtualenv. Two projects needing different versions of a library
  simply get different directories in the module cache. `gofmt` is Black without the
  options.
- **C/C++**: no headers, no Makefile or CMake for the build itself. The build cache is
  keyed by content hashes, so there are no stale objects to `make clean` away.

## How it works

`go build` and `go test` hash every input (sources, flags, compiler version) and look
the result up in the build cache (`go env GOCACHE`). `go test` additionally caches
passing results: when nothing it tracks has changed, it replays the old output with
`(cached)` instead of a duration. Since Go 1.24, the binaries `go run` builds are
cached too, so a second `go run .` starts almost instantly.

## Design notes

The tool is opinionated on purpose: one layout rule (directory = package), one
formatter with no options, one test runner, and vet analyzers chosen for few false
positives. That makes every Go repository look alike, which is the point: a new
codebase builds with `go build ./...` and tests with `go test ./...`, with nothing to
read first. The cost is that anything outside that loop, such as generating code or
building images, lives in a Makefile or `go generate` directives. A reasonable CI
gate is `gofmt -l .` (must print nothing), `go vet ./...`, then `go test -race ./...`,
plus staticcheck or golangci-lint if the team agrees on one.
