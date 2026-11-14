---
id: go.performance.docker-vulns
title: "Multi-stage Docker builds, govulncheck, a release checklist"
track: go
module: performance
order: 4
level: intermediate
minutes: 14
since: go1.25
prerequisites: [go.performance.cross-compile]
compare_with: [inside.codebase.compose-topology, sql.security-ops.backup-migrations, ts.tooling.vite]
tags: [docker, containers, distroless, govulncheck, security, reproducible-builds, release]
summary: >
  Build in a golang stage, ship only the static binary in a scratch or
  distroless stage, and the image holds little more than your code. govulncheck
  reports the known vulnerabilities your code actually reaches, and -trimpath
  makes the binary the same wherever it is built.
examples:
  - id: static-binary
    file: examples/01_static_binary.go
    title: "What the build stage produces: a Linux binary with nothing to load"
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "11"
        note: >
          With cgo enabled, the default in the golang image because it ships a
          C compiler, a program importing net is linked against the system
          libc. That binary then fails in FROM scratch with "no such file or
          directory": the missing file is the dynamic loader, not your binary.
      - lines: "24"
        note: >
          Inside a container, listen on all interfaces (":8080"); restricting
          access is the job of the port mapping, such as 127.0.0.1:8080:8080
          in Compose, which is how StackLab publishes its ports.
      - lines: "39-42"
        note: >
          The command a Dockerfile's build stage runs. CGO_ENABLED=0 selects
          Go's own DNS resolver and user lookup, so nothing from libc is
          needed; -trimpath and -s keep paths and symbols out of the image.
      - lines: "52-58"
        note: >
          A dynamically linked ELF has a PT_INTERP header naming its loader
          (/lib64/ld-linux-x86-64.so.2 on amd64 glibc systems) and a list of
          needed shared libraries. This one has neither.
      - lines: "65"
        note: >
          "Can run" is about loading. A scratch image still lacks CA
          certificates, the time zone database and /etc/passwd; distroless
          static has all three, which is why StackLab uses it.
  - id: reproducible
    file: examples/02_reproducible.go
    title: "-trimpath makes two checkouts build the same bytes"
    runtime: go
    check: exact
    timeout_s: 15
    annotations:
      - lines: "26-28"
        note: >
          Same source, same Go version, same flags; only the directory
          differs, as it does between a laptop and a CI runner.
      - lines: "30-38"
        note: >
          Without -trimpath, absolute source paths are compiled into the
          binary (for stack traces and debug info), so the hashes differ and
          the binary tells anyone who reads it where you keep your code.
      - lines: "36-37"
        note: >
          With -trimpath the paths become module paths and the two builds are
          byte-for-byte identical. Anyone can rebuild a tagged release and
          compare checksums; the Go project verifies its own toolchain
          releases this way.
      - lines: "56-57"
        note: >
          Reproducibility also needs the same toolchain and flags: pin the Go
          version (the go and toolchain lines in go.mod, the image tag or
          digest) and keep the build command in one script.
use_cases:
  - "Service images: a golang build stage, a distroless/static or scratch final stage, a non-root USER"
  - "CI gate: govulncheck ./... fails the build when reachable code calls a vulnerable function"
  - "Auditing a binary you didn't build: govulncheck -mode=binary ./app and go version -m ./app"
  - "Verifiable releases: -trimpath builds from a tag, with published SHA-256 checksums anyone can reproduce"
strengths:
  - "A static Go binary makes the final image tiny and nearly empty: no shell, no package manager, no interpreter for an attacker to use, unlike a JRE or Node base image"
  - "govulncheck reports only vulnerabilities in functions your code can reach, where npm audit, pip-audit and most dependency scanners report every vulnerable version in the tree"
  - "Standard-library vulnerabilities are part of the report and are fixed by a Go upgrade, because the binary records which Go built it"
  - "Since Go 1.25 the runtime sets GOMAXPROCS from the container's CPU limit on Linux, a job that used to need a third-party package"
weaknesses:
  - "scratch has no CA certificates, time zone data or users: TLS, time.LoadLocation and user lookups fail until you add them"
  - "Distroless images have no shell, so debugging means a debug image variant or an ephemeral debug container"
  - "govulncheck needs the vulnerability database (vuln.go.dev, or a mirror via -db), so offline CI needs a local copy"
  - "Reachability analysis is static: calls through reflection are invisible, and interface calls are approximated"
pitfalls:
  - title: "Building with cgo for a scratch image"
    detail: "The golang image has gcc, so cgo is on by default and anything importing net links libc dynamically. Set CGO_ENABLED=0 in the build stage."
    bad: "RUN go build -o /out/app ./cmd/app"
    good: "RUN CGO_ENABLED=0 go build -trimpath -o /out/app ./cmd/app"
  - title: "Time zones in scratch"
    detail: "time.LoadLocation(\"Europe/Istanbul\") returns 'unknown time zone' in an image without tzdata. Use distroless/static, or embed the database with a blank import of time/tzdata (about 450 KB) or -tags timetzdata."
    bad: "FROM scratch + time.LoadLocation(\"Europe/Istanbul\")"
    good: "import _ \"time/tzdata\" // in package main"
  - title: "Copying the whole context before go mod download"
    detail: "COPY . . first invalidates the dependency layer on every source change. Copy go.mod and go.sum, download, then copy the source; or use BuildKit cache mounts for the module and build caches."
  - title: "Running as root in the final image"
    detail: "Add USER nonroot:nonroot (distroless) or a numeric USER for scratch. StackLab's server image switches to distroless's nonroot user, and its Compose file runs the runner container as uid 65532."
  - title: "Scanning once at release time"
    detail: "New advisories appear for code you already shipped. Run govulncheck in CI on every change and on a schedule against the main branch and released binaries."
quiz:
  - q: "A Go service built with plain go build in the golang image fails in FROM scratch with 'no such file or directory'. Why?"
    options: ["The binary wasn't copied", "Cgo was enabled, so the binary is dynamically linked and its loader and libc are missing from the image", "scratch can't run ELF files"]
    answer: 1
    why: "The golang image includes a C compiler, so cgo is on by default; net then links libc. CGO_ENABLED=0 produces a static binary."
  - q: "govulncheck ./... lists a vulnerability in its Informational section, with no call stack. What does that mean?"
    options: ["The scan failed", "A vulnerable package is imported, but none of your code reaches the vulnerable function", "The vulnerability is fixed"]
    answer: 1
    why: "Source mode traces calls to vulnerable functions. Without a call path, the vulnerable code is imported but not reachable from yours; govulncheck's own wording is that you may not need to take any action. Upgrading is still the tidy fix."
  - q: "Two CI runners build the same tag in different directories and get different SHA-256 sums. What is the usual cause?"
    options: ["Go builds are randomized", "Absolute source paths are embedded; build with -trimpath", "The linker embeds the build time"]
    answer: 1
    why: "Go records no timestamps; with the same toolchain, flags and -trimpath, builds are byte-for-byte reproducible."
  - q: "How does govulncheck know which standard-library vulnerabilities apply to a compiled binary?"
    options: ["It can't; it only checks modules", "From the Go version recorded in the binary's build info", "From the image's OS packages"]
    answer: 1
    why: "Every binary records the Go version that built it (go version -m prints it), and the standard library is versioned with Go."
further_reading:
  - title: "Go vulnerability management"
    url: "https://go.dev/doc/security/vuln/"
  - title: "Tutorial: Find and fix vulnerable dependencies with govulncheck"
    url: "https://go.dev/doc/tutorial/govulncheck"
  - title: "govulncheck command documentation"
    url: "https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck"
  - title: "Perfectly Reproducible, Verified Go Toolchains (Go blog)"
    url: "https://go.dev/blog/rebuild"
  - title: "Go 1.25 Release Notes: container-aware GOMAXPROCS"
    url: "https://go.dev/doc/go1.25"
---

## Concept

A Go service image is usually two stages: one with the full toolchain that builds a
static binary, and one with almost nothing that runs it. The final stage doesn't need
Go, a shell or a package manager, so it is small and offers little to an attacker.

```dockerfile
# syntax=docker/dockerfile:1
FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s" -o /out/app ./cmd/app

FROM gcr.io/distroless/static-debian13
COPY --from=build /out/app /app
USER nonroot:nonroot
ENTRYPOINT ["/app"]
```

`AS build` names the stage and `COPY --from=build` takes only the binary from it.
Copying `go.mod` and `go.sum` before the source keeps the dependency layer cached
until they change, and the BuildKit cache mounts keep the module cache and build
cache between builds (the paths are those of the official golang image, as in
Docker's own Go example). `FROM scratch` works too, if you add what the binary needs
from the file system: CA certificates, time zone data, a non-root user.

**govulncheck** checks code against the Go vulnerability database:

```
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...                # source: call stacks from your code to vulnerable functions
govulncheck -mode=binary ./app   # a built binary: Go version and module versions it records
```

In source mode it reports a vulnerability only when your code can reach the vulnerable
function, and it exits non-zero when it finds one, which is what makes it a CI gate.
The fix is an upgrade: `go get example.com/mod@fixed-version` for a module, a newer Go
for the standard library.

## Coming from C/C++/Java/Python/JS

- **Java**: the image needs a JRE (or a `jlink` runtime) plus the JAR; a Go image is
  the binary and, with distroless, CA certificates and tzdata. OWASP Dependency-Check
  matches versions; govulncheck also checks whether you call the affected code.
- **Node**: multi-stage builds exist to drop devDependencies and build tools, but
  `node_modules` and the runtime still ship. `npm audit` reports every vulnerable
  version in the tree, reachable or not.
- **Python**: slim images still carry an interpreter and site-packages; `pip-audit`
  works at the package-version level like `npm audit`.
- **C/C++**: a static musl build gets close to the Go result, with more effort; Go
  gets there with `CGO_ENABLED=0`.

## How it works

govulncheck reads the vulnerability database (vuln.go.dev), where every entry names
the affected module, versions and, where known, the vulnerable functions. In source
mode it builds a call graph of your packages and reports entries whose functions are
reachable from your code, with an example call stack; entries for code you import but
don't reach are listed separately as informational. In binary mode it reads the build
information every Go binary carries (the same record `go version -m` prints) and the
binary's symbols, so it works on artifacts you didn't build, but it can't show call
stacks. Requests to the database contain only module paths, not your code.

Reproducibility comes from what Go leaves out: no build timestamps, and with
`-trimpath` no host paths. The Go blog's recipe for programs that don't need cgo is
exactly `CGO_ENABLED=0 go build -trimpath`, and the second example confirms it. The Go
project holds its own toolchain to the same standard: since Go 1.21, release builds
are rebuilt from source every night and must match the published archives bit for
bit.

## Design notes

StackLab's `server/Dockerfile` follows this shape: a `golang:${GO_VERSION}` build
stage with `go.mod`/`go.sum` copied first, `CGO_ENABLED=0 go build -trimpath`, a
`gcr.io/distroless/static-debian13` final stage, `USER nonroot:nonroot`, and
migrations embedded with `go:embed` so the image needs no extra files. One Dockerfile
builds two programs through a `SERVICE` build argument.

A release checklist that fits on one screen:

1. `go mod tidy` leaves no diff; `go mod verify` passes.
2. `gofmt -l .` prints nothing; `go vet ./...` passes, with `GOOS=linux` too when
   you develop on Windows.
3. `go test -race ./...` passes. The race detector needs cgo, so run it on Linux CI
   (or with a C compiler installed); on this Windows machine without one, `-race`
   refuses to start.
4. `govulncheck ./...` passes.
5. Build every target with `-trimpath`, `CGO_ENABLED=0` and a version stamp from the
   tag; publish SHA-256 checksums beside the files.
6. Check the artifacts: `go version -m` shows the expected Go version, module
   versions and VCS revision; `govulncheck -mode=binary` passes.
7. Build the image from a pinned base (a version tag, ideally a digest), run as
   non-root, tag it with the version rather than only `latest`.
8. In the deployment, set `GOMEMLIMIT` below the container's memory limit. Since
   Go 1.25, `GOMAXPROCS` follows the CPU limit by itself.
