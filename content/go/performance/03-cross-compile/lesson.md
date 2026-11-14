---
id: go.performance.cross-compile
title: "Cross-compilation and the single binary: GOOS/GOARCH, CGO_ENABLED=0, ldflags"
track: go
module: performance
order: 3
level: intermediate
minutes: 13
since: go1.24
prerequisites: [go.tooling.go-command, go.files-io.embed]
compare_with: [ts.node.cli-server, ts.tooling.bundle, inside.codebase.compose-topology]
tags: [cross-compilation, goos, goarch, cgo, ldflags, build-constraints, buildinfo, release]
summary: >
  Any Go toolchain builds for any supported OS and CPU: set GOOS and GOARCH.
  With CGO_ENABLED=0 the result is one self-contained file per target. Stamp
  versions with -ldflags -X, or since Go 1.24 let go build take them from
  your Git tags.
examples:
  - id: file-selection
    file: examples/01_file_selection.go
    title: "GOOS decides which files are compiled: suffixes and //go:build"
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "11"
        note: >
          StackLab's gorun package has this shape: proc_unix.go uses process
          groups, proc_windows.go a Job Object. Each target compiles
          exactly one of them, and both must keep building, which is why
          CLAUDE.md asks for go vet with GOOS=linux as well as on Windows.
      - lines: "20-21"
        note: >
          A file name ending in _GOOS, _GOARCH or _GOOS_GOARCH (before .go and
          an optional _test) carries an implicit build constraint. No comment
          needed.
      - lines: "27-28"
        note: >
          unix is a build tag (Go 1.19), not a GOOS, so the _unix suffix means
          nothing: without this line proc_unix.go would be compiled on Windows
          too. StackLab's file says //go:build !windows instead, which also
          admits plan9 and wasm targets where its Unix syscalls don't exist.
      - lines: "50-55"
        note: >
          GOOS, GOARCH and CGO_ENABLED are ordinary environment variables for
          one command. go list doesn't compile anything, so asking about four
          targets takes a moment.
      - lines: "60"
        note: >
          plan9 matches neither file, so killTree would be undefined there.
          go list shows it before a release build fails on it; go tool dist
          list prints every supported target (47 in Go 1.27).
  - id: ship-binaries
    file: examples/02_ship_binaries.go
    title: "Four targets from one machine, with a version stamped in by the linker"
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "16"
        note: >
          -X works only on a package-level string variable that is
          uninitialized or initialized to a constant string. A const, or a var
          initialized by a function call, is silently left alone.
      - lines: "26"
        note: >
          -s drops the symbol table and, since Go 1.22, implies -w (no DWARF
          debug info), so "-s -w" is now redundant. Smaller files; stack
          traces still have function names and lines, which come from a
          separate table.
      - lines: "30-31"
        note: >
          No cross toolchain, no sysroot: the Go compiler and linker target
          every port. CGO_ENABLED=0 keeps it that way and makes Linux binaries
          fully static. -trimpath removes local file system paths, so the
          binary doesn't reveal the build machine's directories.
      - lines: "37-46"
        note: >
          debug/buildinfo reads the record that go version -m prints, from an
          ELF, PE or Mach-O file alike. With -trimpath the -ldflags setting is
          left out on purpose (Go issue 52372), since linker flags can contain
          local paths; without -trimpath it is recorded.
      - lines: "49-50"
        note: >
          Only the build for this machine can run here. Name Windows outputs
          with .exe yourself: -o uses the name exactly as given.
  - id: vcs-version
    file: examples/03_vcs_version.go
    title: "Since Go 1.24, go build takes the version from Git"
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "19-22"
        note: >
          The program reads its own version at run time. Main.Version used to
          be (devel) for every local build, which is why -X main.version was
          the standard trick.
      - lines: "30-34"
        note: >
          A clean checkout of a tagged commit: the tag becomes the main
          module's version, exactly what go install example.com/app@v1.4.2
          would record.
      - lines: "36-37"
        note: >
          Uncommitted changes add +dirty, so a binary built from a modified
          tree can't pass for the release.
      - lines: "39-40"
        note: >
          A commit after the tag gets a pseudo-version: the next patch number,
          the commit time and the first 12 characters of the commit hash
          (masked here). -buildvcs=false turns stamping off, and without a
          repository (a Docker build that copies only the source) there is
          nothing to stamp: Main.Version is (devel) again.
      - lines: "45-48"
        note: >
          An untracked file counts as a modification, so a binary written into
          the repository would make the next build +dirty. Release builds
          write to a dist/ directory listed in .gitignore, or outside the
          checkout.
      - lines: "56-64"
        note: >
          The demo pins git's identity and clock and ignores your git
          configuration (no signing, no line-ending conversion), so the history
          and the pseudo-version's timestamp are the same on every run. go
          build itself calls git to read the revision, so it gets the same
          environment.
use_cases:
  - "Release matrix: linux/amd64 and linux/arm64 for servers, darwin/arm64 for Macs, windows/amd64 for Windows users, all from one CI job"
  - "Container images: CGO_ENABLED=0 go build -trimpath, copied into a scratch or distroless image"
  - "Developing on Windows, deploying to Linux: GOOS=linux go vet ./... and go build catch Unix-only code before CI does"
  - "Version reporting: debug.ReadBuildInfo() in a --version flag or a /version endpoint; go version -m on a binary in the field"
strengths:
  - "Cross-compiling is two environment variables, unlike C/C++ cross toolchains and sysroots or Rust's per-target linker setup"
  - "With CGO_ENABLED=0 the output is one file with no runtime to install, unlike Java (a JVM), Python (an interpreter and site-packages) or Node (node_modules)"
  - "Build metadata (Go version, dependencies with checksums, VCS revision, build flags) is embedded in every binary and readable without running it"
  - "go:embed puts templates, migrations and static assets in the same file, so deployment is a copy"
weaknesses:
  - "cgo breaks the easy path: cross-compiling with cgo needs a C cross compiler and the target's libc"
  - "Binaries are larger than C equivalents (a hello world is about 1.5 MB stripped) because the runtime and GC are linked in"
  - "Build constraints are checked per target: code for a target you never build can rot unnoticed"
  - "-X takes strings only and fails silently on a typo in the import path or variable name"
pitfalls:
  - title: "Expecting _unix.go to be Unix-only"
    detail: "unix is a build tag, not a GOOS, so the suffix is ignored. Add a //go:build unix line (or a GOOS-specific suffix such as _linux)."
    bad: "proc_unix.go with no //go:build line"
    good: "//go:build unix"
  - title: "go env -w GOOS=linux on a development machine"
    detail: "go env -w writes to the user's Go environment file, so every later build, test and run cross-compiles. Set GOOS for one command instead."
    bad: "go env -w GOOS=linux"
    good: "PowerShell: $env:GOOS='linux'; go build ...; Remove-Item Env:GOOS (bash: GOOS=linux go build ...)"
  - title: "A dynamically linked binary in a scratch image"
    detail: "With cgo enabled (the default where a C compiler exists, as in the golang image), a program that imports net or os/user links against libc and fails to start in FROM scratch with a confusing 'no such file or directory'. Build with CGO_ENABLED=0."
  - title: "-X on a variable that isn't a plain string"
    detail: "-X ignores constants and variables initialized by function calls, without an error. Use var version = \"dev\" and check the result with go version -m or a --version flag."
    bad: "var version = defaultVersion()"
    good: "var version = \"dev\""
quiz:
  - q: "On Windows, what do you need to build a linux/arm64 binary of a pure-Go program?"
    options: ["WSL and a Linux Go toolchain", "Only GOOS=linux GOARCH=arm64 for the go build command", "A Linux C cross compiler"]
    answer: 1
    why: "The Go toolchain targets every port itself. Cgo is disabled by default when cross-compiling; a C cross compiler is needed only if you turn it back on."
  - q: "gorun/proc_unix.go has no //go:build line. When is it compiled?"
    options: ["Only on Unix-like systems", "On every target, because unix is not a GOOS value and the suffix is ignored", "Never"]
    answer: 1
    why: "Only _GOOS, _GOARCH and _GOOS_GOARCH suffixes are implicit constraints. unix exists as a build tag since Go 1.19, so write //go:build unix."
  - q: "go build -ldflags='-X main.version=v1.4.2' runs, but the program still prints dev. Which declaration explains it?"
    options: ["var version = \"dev\"", "const version = \"dev\"", "var version string"]
    answer: 1
    why: "-X sets string variables that are uninitialized or initialized to a constant string; it can't change a constant, and it doesn't report that."
  - q: "With Go 1.27, a clean checkout of the commit tagged v1.4.2 is built with plain go build. What is debug.ReadBuildInfo().Main.Version?"
    options: ["(devel)", "v1.4.2", "v1.4.3-0.<time>-<hash>"]
    answer: 1
    why: "Since Go 1.24 go build sets the main module version from VCS tags; a later commit gets a pseudo-version and uncommitted changes add +dirty."
further_reading:
  - title: "Command go: Build constraints"
    url: "https://pkg.go.dev/cmd/go#hdr-Build_constraints"
  - title: "Command link (-X, -s, -w)"
    url: "https://pkg.go.dev/cmd/link"
  - title: "Command cgo (CGO_ENABLED defaults)"
    url: "https://pkg.go.dev/cmd/cgo"
  - title: "package debug/buildinfo"
    url: "https://pkg.go.dev/debug/buildinfo"
  - title: "Go 1.24 Release Notes (version from VCS tags)"
    url: "https://go.dev/doc/go1.24"
---

## Concept

Every Go toolchain contains a compiler and linker for every supported target. Pick
one with two environment variables:

```
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags="-s" -o dist/stacklab-linux-arm64 ./cmd/stacklab
```

In PowerShell, which is what this machine runs, set them for the session and remove
them afterwards:

```
$env:GOOS = "linux"; $env:GOARCH = "arm64"; $env:CGO_ENABLED = "0"
go build -trimpath -ldflags="-s" -o dist/stacklab-linux-arm64 ./cmd/stacklab
Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED
```

The pieces:

- **GOOS/GOARCH** select the target; `go tool dist list` prints all of them. The
  same variables pick files: `_linux.go`, `_windows_amd64.go` and `//go:build` lines.
- **CGO_ENABLED=0** keeps C out of the build. Cgo is off by default when
  cross-compiling or when no C compiler is on the PATH (this Windows machine builds
  with `CGO_ENABLED=0` for that reason), and on by default for native builds where
  one exists. Pure-Go builds for Linux are fully static.
- **-trimpath** removes your directories from the binary; **-ldflags="-s"** drops the
  symbol table and DWARF.
- **Version stamps**: `-ldflags="-X main.version=v1.4.2"` sets a string variable at
  link time. Since Go 1.24, `go build` in a Git checkout records the tag (or a
  pseudo-version, with `+dirty` for uncommitted changes) as the main module's version,
  next to the `vcs.revision`, `vcs.time` and `vcs.modified` settings it has recorded
  since Go 1.18. `go version -m ./app` prints all of it.

## Coming from C/C++/Java/Python/JS

- **C/C++**: no cross toolchain, sysroot or static glibc to fight, as long as cgo is
  off. Turning cgo on brings all of that back.
- **Java**: the JAR is portable and the JVM is not. A Go binary is the native-image
  end of that trade: one file per OS and CPU, no runtime to install, start-up in
  milliseconds.
- **Python/Node**: PyInstaller and Node's single executable applications bundle an
  interpreter with your code. Go's output is compiled code plus a small runtime, and
  the dependency tree is linked in, not shipped beside it.
- **JS bundlers**: `-trimpath` and `-s` are Go's equivalent of a production build;
  there is no separate minify step, and dead code is removed by the linker.

## Design notes

StackLab's `server/Dockerfile` builds with `CGO_ENABLED=0 go build -trimpath` and
copies the result into a distroless static image; the migrations travel inside the
binary via `go:embed`, so the image needs no volume for them. On the owner's Windows
machine the same code builds natively, and gorun's `proc_*.go` files keep the
OS-specific pieces behind build constraints: `proc_windows.go` and `proc_linux.go` (the
Linux-only parent-death signal) by their suffix, `proc_unix.go` and
`proc_otherunix.go` by `//go:build` lines.

For release versions, prefer the Go 1.24 VCS stamp when you build from a tagged
checkout: it can't drift from the tag, and `+dirty` exposes local builds. Keep
`-X` for what Git doesn't know, such as a CI build number, and for builds without a
repository: a Docker build that copies only the source tree has no `.git`, so
Main.Version falls back to `(devel)`. Either way, expose the result (a `--version`
flag, a `/version` endpoint, a startup log line) so that "which build is running?"
has an answer.

Build for every target you ship on every change, even if you test on one: a
`GOOS=linux go vet ./...` step on a Windows machine, or a CI matrix, catches the
file that only compiles on one side.
