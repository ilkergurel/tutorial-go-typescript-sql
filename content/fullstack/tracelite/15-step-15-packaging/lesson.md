---
id: fullstack.tracelite.step-15-packaging
title: "Step 15: Packaging: one binary embedding frontend + migrations; cross-compiling; customer-machine setup"
track: fullstack
module: tracelite
order: 15
level: advanced
minutes: 15
since: go1.27
prerequisites: [fullstack.tracelite.step-14-testing, go.files-io.embed, go.performance.cross-compile, go.http.server-lifecycle]
compare_with: [go.performance.docker-vulns, go.postgres.migrations-sqlc, inside.codebase.wiring, ts.tooling.vite]
tags: [tracelite, packaging, embed, spa, cache-control, migrations, config, ldflags, buildinfo, cross-compile, windows-service, systemd, deployment]
summary: >
  TraceLite ships as one executable per platform: the built React app and the migrations
  are embedded, configuration comes from the environment, and the binary migrates,
  serves and shuts down in a fixed order. A customer installs PostgreSQL and one file.
examples:
  - id: embed-spa
    file: examples/01_embed_spa.go
    title: "web/dist inside the binary: files, client-side routes and the API on one origin"
    runtime: go-server
    check: exact
    fixtures: [fixtures/web/dist/index.html, fixtures/web/dist/assets/index-B2xk91fa.js, fixtures/web/dist/assets/_commonjsHelpers-Dk2Y9xQb.js, fixtures/web/dist/assets/index-Cq3ZsT0e.css]
    annotations:
      - lines: "16-19"
        note: >
          A //go:embed pattern is relative to the Go file's directory and can't
          contain "..", so cmd/tracelite can't reach web/dist. TraceLite has a
          tiny package web (web/web.go) next to dist; this one-file example
          keeps the directive in main.
      - lines: "19"
        note: >
          all: keeps names starting with _ or ., which a plain directory pattern
          skips (go.files-io.embed). The build output here has one: when two
          lazy chunks import CommonJS packages, a Vite build puts the shared
          interop helper in _commonjsHelpers-hash.js (Rollup writes the helper
          module's leading NUL as _). Without all:, the binary would ship an
          app that can't load it.
      - lines: "26-29"
        note: >
          Read once at start: a binary built before npm run build fails here,
          at startup, with a message that says what to do, instead of serving
          404s.
      - lines: "32-37"
        note: >
          A real file is served by http.FileServerFS (Go 1.22): it sets
          Content-Type, Last-Modified (the embed time is zero, so none here) and
          handles ranges. Files under assets/ carry a content hash in their
          names, so they may be cached for a year: a new build has new names.
      - lines: "39-42"
        note: >
          After a deploy, a tab opened before it asks for the previous build's
          chunk. Answering with index.html would make the browser parse HTML as
          JavaScript; a 404 lets the app notice and reload. Anything with a
          file extension, or under assets/, is a file or nothing.
      - lines: "43-45"
        note: >
          Every other path is a client-side route (step 6), answered with the
          app. index.html is revalidated on every load, because it is the one
          file that names the current assets.
      - lines: "50"
        note: >
          Embedded names keep their directory (web/dist/index.html); fs.Sub
          makes dist the root. The migrations travel the same way (example 3),
          because goose reads the root of the FS it is given.
      - lines: "52-56"
        note: >
          Unknown API paths stay problems: a typo in a fetch URL must produce a
          404 problem, not an HTML page that zod then fails to parse with a
          confusing message. ServeMux picks the most specific pattern, so
          /api/ only gets what no real API route matched, and / only what isn't
          under /api/.
      - lines: "63"
        note: >
          FileServerFS redirects /index.html to /, and / is answered by the
          route handler, with no-cache. The output shows where the redirect
          ended.
  - id: config-version
    file: examples/02_config_version.go
    title: "Configuration from the environment, refused all at once; the version a binary reports"
    runtime: go
    check: exact
    annotations:
      - lines: "11-12"
        note: >
          The linker sets the variable at release time with -X (shown running in
          go.performance.cross-compile). This runner builds without -ldflags
          and outside Git, so the output says dev.
      - lines: "14-24"
        note: >
          debug.ReadBuildInfo reads what go build recorded, such as the commit
          as vcs.revision. A support call starts with tracelite --version, and
          this is what it prints. TraceLite's Version also reports vcs.modified
          and the module version from a Git tag (How it works).
      - lines: "26-30"
        note: >
          Three of the five settings; the other two have safe defaults (How it
          works). TRACELITE_OWNER is step 13's actor for an installation
          without logins: the name recorded in the audit history.
      - lines: "34-35"
        note: >
          Load takes getenv instead of calling os.Getenv, so a test passes a map,
          as main does here twice. The default address is the safe one:
          loopback only.
      - lines: "40-47"
        note: >
          TraceLite has no authentication (step 1), so listening on all
          interfaces would publish every project to the network. The binary
          refuses unless the operator says, in a variable named for what it
          does, that something in front of it checks users. ":8080" has an
          empty host, which means every interface.
      - lines: "54"
        note: >
          errors.Join (Go 1.20) keeps every problem; printing it gives one per
          line. A service that stops at the first error makes the customer
          restart it once per mistake.
      - lines: "68-69"
        note: >
          url.URL.Redacted replaces the password with xxxxx, so the startup log
          line can show which database it connected to without leaking how.
          StackLab's maskDSN does the same.
  - id: lifecycle
    file: examples/03_lifecycle.go
    title: "The binary's life: migrate before listening, then stop HTTP, jobs, listener and pool, in that order"
    runtime: go-server
    check: exact
    timeout_s: 15
    fixtures: [fixtures/migrations/001_init.sql, fixtures/migrations/002_change_events.sql, fixtures/migrations/003_webhooks.sql, fixtures/migrations/004_jobs.sql, fixtures/migrations/005_search.sql, fixtures/migrations/006_audit.sql]
    annotations:
      - lines: "17-18"
        note: >
          In TraceLite this directive lives in migrations/migrations.go, next to
          the files, for the same reason as web/web.go (example 1).
      - lines: "27-32"
        note: >
          A stand-in for goose, which isn't in this runner's module set. goose
          also records each applied version in goose_db_version, runs each file
          in a transaction, and with a session locker lets two starting
          processes migrate one at a time (go.postgres.migrations-sqlc).
      - lines: "31"
        note: >
          Migrate before listening: a request never meets a schema older than
          the code. If a migration fails, the process exits and the service
          manager shows the error. goose runs each file in a transaction, so
          the failed one leaves nothing behind, while the ones before it stay
          applied; the old binary still runs if they were additive (design
          notes).
      - lines: "35-37"
        note: >
          Step 11's pool gets its own context, derived without cancellation,
          because it must outlive the signal until HTTP has stopped.
      - lines: "40-46"
        note: >
          A request that is still running when the signal arrives, so the
          output can show what happens to it.
      - lines: "63"
        note: >
          The signal has canceled ctx, so the shutdown deadline is derived
          without its cancellation; 10 s for HTTP plus step 11's 20 s drain
          stays inside systemd's default 90 s stop timeout.
      - lines: "65-72"
        note: >
          The reverse of the start, in dependency order. HTTP first: Shutdown
          closes the listener and waits for requests in flight, which may still
          enqueue jobs or write. Then the jobs drain (step 11), then the step 9
          listener, then the pool that everything used.
      - lines: "66"
        note: >
          The request that was running at the signal was answered with 200
          before the server stopped: a deploy or a reboot doesn't turn into
          errors in someone's browser.
use_cases:
  - "Shipping an internal tool or a small product to machines you don't control: one file per OS, no runtime to install besides the database"
  - "Serving a single-page app and its API from one origin in production, as the Vite proxy does in development"
  - "Upgrades by replacing a file: the new binary migrates the database at startup"
  - "Support: a version command and a startup log line that say exactly which build and configuration is running"
strengths:
  - "One artifact per platform: no Node, no web server, no copying of static files or migration scripts that can drift from the code"
  - "Cross-compiling is two environment variables: pgx is pure Go, so CGO_ENABLED=0 builds Windows and Linux binaries from one machine"
  - "Same origin in production: no CORS, cookies just work, and the router decides in one place what is API and what is app"
  - "Configuration is validated as a whole before anything starts, and the binary refuses unsafe settings instead of logging a warning nobody reads"
weaknesses:
  - "The binary must be rebuilt for a UI change; there is no hot-swapping of static files (by design)"
  - "go build depends on npm run build having run first; a CI pipeline or a Makefile must order them"
  - "Self-migration at startup suits one instance; several instances need goose's session locker or a separate migration step"
  - "Windows services need code (golang.org/x/sys/windows/svc) or a wrapper; a plain console program isn't one"
pitfalls:
  - title: "//go:embed dist without all:"
    detail: "Files whose names start with _ or . are silently left out. Build tools generate such names; the app then fails in production only. Use all:dist, and keep secrets out of dist."
    bad: "//go:embed dist"
    good: "//go:embed all:dist"
  - title: "Answering every unknown path with index.html"
    detail: "The SPA fallback must not cover /api/ or missing assets. Otherwise a mistyped endpoint returns HTML with 200, and after a deploy an old tab gets HTML for a JavaScript chunk."
  - title: "Caching index.html"
    detail: "index.html names the hashed assets of the current build. With a long cache lifetime, browsers keep loading the previous build after an upgrade. Cache assets for a year, revalidate index.html always."
  - title: "Stopping the database pool before the HTTP server"
    detail: "Closing in the order things were started (or in no order, from deferred calls in random places) lets requests in flight fail with 'closed pool'. Stop in reverse dependency order: HTTP, jobs, listener, pool."
  - title: "Listening on 0.0.0.0 by default"
    detail: "Docker images often listen on all interfaces because the container network is the boundary. A binary on a customer's Windows machine has no such boundary; default to 127.0.0.1 and make wider exposure an explicit decision."
quiz:
  - q: "Why does TraceLite answer GET /assets/index-A1old000.js with 404 instead of index.html?"
    options: ["Go's FileServer requires it", "A tab from before a deploy asks for an old chunk; HTML would be parsed as JavaScript, while a 404 lets the app reload", "To save bandwidth"]
    answer: 1
    why: "The fallback is for client-side routes only; anything that looks like a file is a file or a 404."
  - q: "Where does the //go:embed directive for web/dist live?"
    options: ["In cmd/tracelite/main.go, with ../../web/dist", "In a small package inside web/, because patterns can't contain ..", "In go.mod"]
    answer: 1
    why: "Embed patterns are relative to the source file's directory and may not climb out of it."
  - q: "SIGTERM arrives. Why is the HTTP server stopped before the job pool?"
    options: ["HTTP is faster to stop", "Requests in flight may still enqueue jobs; the pool must keep running until they are done", "The job pool needs the HTTP server"]
    answer: 1
    why: "Stop consumers after their producers: HTTP produces jobs, the pool consumes them, and both use the database pool, which closes last."
  - q: "Which command builds a Linux binary on the Windows development machine?"
    options: ["It can't be done without WSL", "Setting GOOS=linux GOARCH=amd64 CGO_ENABLED=0 for go build", "go build -target linux"]
    answer: 1
    why: "Go cross-compiles by environment variables; with no cgo dependency, the result is a static Linux executable."
further_reading:
  - title: "package embed (patterns, all:, hidden files)"
    url: "https://pkg.go.dev/embed"
  - title: "package net/http: FileServerFS"
    url: "https://pkg.go.dev/net/http#FileServerFS"
  - title: "package runtime/debug: ReadBuildInfo"
    url: "https://pkg.go.dev/runtime/debug#ReadBuildInfo"
  - title: "Command link: -X, -s, -w"
    url: "https://pkg.go.dev/cmd/link"
  - title: "Go: Environment variables GOOS, GOARCH, CGO_ENABLED"
    url: "https://pkg.go.dev/cmd/go#hdr-Environment_variables"
  - title: "MDN: Cache-Control (immutable, no-cache)"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Cache-Control"
  - title: "PostgreSQL docs: CREATE DATABASE (locale options)"
    url: "https://www.postgresql.org/docs/current/sql-createdatabase.html"
---

## Concept

The product TraceLite set out to be is **one executable plus PostgreSQL**. Everything
else is inside the executable:

- **The web app**: `npm run build` writes `web/dist`; `//go:embed all:dist` in package
  `web` puts it into the binary; a handler serves files, answers client-side routes
  with `index.html`, and keeps `/api/` for the API. One origin, as in development.
- **The migrations**: `//go:embed *.sql` in package `migrations`; at startup goose
  applies whatever the database doesn't have yet, before the server listens.
- **Configuration**: environment variables, validated together at startup, with safe
  defaults (loopback address) and a refusal for unsafe ones.
- **Its identity**: a version set with `-ldflags -X` at release, plus the Git revision
  that `go build` records, printed by `tracelite --version` and in the first log line.

Because pgx is pure Go, `CGO_ENABLED=0` cross-compiles the same source into a Windows
`.exe` and a static Linux binary from one machine. On the customer's machine: install
PostgreSQL 18, create a role and a database, put the binary somewhere, register it as a
service with its environment, start it.

## Coming from C/C++/Java/Python/JS

- **Java/Spring Boot**: the fat JAR is the same idea (app, static resources from
  `src/main/resources/static`, Flyway migrations), but it needs a JVM on the target. A
  Go binary needs nothing; jlink or GraalVM native-image get Java close, with more
  build machinery.
- **Python/Django**: `collectstatic` plus a web server for static files, a WSGI server,
  a virtualenv, and `manage.py migrate` as a separate step. PyInstaller can bundle an
  interpreter, with platform-specific builds and a large result.
- **Node/Express**: `express.static` plus a catch-all route is this handler; shipping
  means Node plus `node_modules` on the target, or a single-executable build (Node SEA),
  which is still experimental.
- **C/C++**: a static binary with resources compiled in (`xxd -i`, or `#embed` in C23),
  and a cross-toolchain per target. Go's cross-compiler ships with the language.

## How it works

The repository, as it ends after fifteen steps:

```text
openapi.yaml  sqlc.yaml  go.mod
cmd/tracelite/main.go        config, logger, pool, migrate, store, service, jobs, listener, router, server
migrations/                  001_init … 006_audit (.sql) + migrations.go: //go:embed *.sql
internal/store/              sqlc output + store.go (WithTx sets the actor, step 13)
internal/tracelite/          rules, CSV import (7), report generation (8, 11)
internal/httpapi/            router, handlers, problems, cursors, SSE (9), SPA fallback (15)
internal/events/             LISTEN connection and hub (9)
internal/jobs/               queue, pool, report and webhook handlers (10, 11)
web/                         Vite + React; web.go: //go:embed all:dist
web/src/api/  web/src/features/  web/e2e/
```

The migrations package has one unit test of its own: the embedded names, in order, must
start with `001_`, `002_` and so on, so a duplicated or skipped number fails CI rather
than a customer's upgrade.

Configuration is five environment variables, read once at startup (example 2 shows
three of them):

| Variable | Meaning |
|---|---|
| `DATABASE_URL` | required; `postgres://` or `postgresql://`; logged with the password redacted |
| `TRACELITE_OWNER` | required; the actor of a single-user install (step 13) |
| `TRACELITE_ADDR` | default `127.0.0.1:8080`; a non-loopback address needs `TRACELITE_ALLOW_REMOTE=yes` |
| `TRACELITE_WORKERS` | job workers, 1-32, default 4 (step 11) |
| `TRACELITE_STANDARDS_URL` | the standards catalogue (step 10); optional |

`tracelite --version` prints `main.version` when the release set it with `-X`;
otherwise the module version that `go build` records from Git (since Go 1.24: the tag, or
a pseudo-version from the commit). Then comes the commit, `vcs.revision`, with
`, modified` when `vcs.modified` says the tree had uncommitted changes.

One more endpoint, for service managers and monitoring:

```yaml
  /api/health:
    get:
      operationId: getHealth
      responses:
        "200": { description: "Up: {status: ok, version, database: ok}" }
        "503": { description: "The database doesn't answer" }
```

A release, from the Windows development machine (PowerShell):

```text
npm --prefix web ci
npm --prefix web run build                      # web/dist, must exist before go build
go test ./... ; go test -tags integration -count=1 ./...
$env:CGO_ENABLED = "0"
$env:GOOS = "windows"; $env:GOARCH = "amd64"
go build -trimpath -ldflags "-s -w -X main.version=1.0.0" -o dist/tracelite-1.0.0-windows-amd64.exe ./cmd/tracelite
$env:GOOS = "linux"
go build -trimpath -ldflags "-s -w -X main.version=1.0.0" -o dist/tracelite-1.0.0-linux-amd64 ./cmd/tracelite
```

`-trimpath` removes local paths (reproducible builds, go.performance.docker-vulns);
`-s -w` leave out the symbol table and DWARF data, which makes the file smaller and
debuggers less useful, while panics still print function names and lines.

Setting up a customer's Windows machine:

```text
1. winget install PostgreSQL.PostgreSQL.18        (the installer creates the service)
2. psql -U postgres:
     CREATE ROLE tracelite LOGIN PASSWORD '<generated>';
     CREATE DATABASE tracelite OWNER tracelite TEMPLATE template0 ENCODING 'UTF8';
   then, connected to tracelite:  SELECT show_trgm('ığdır');  -- six entries, or step 12's
   typo search treats Turkish letters as separators (see sql.search.trigram)
3. Copy tracelite.exe to C:\Program Files\TraceLite\
4. Register it as a service, with DATABASE_URL, TRACELITE_OWNER (and TRACELITE_ADDR if
   not 127.0.0.1:8080) in the service's environment; start it; open http://127.0.0.1:8080
5. Back up: a scheduled task running pg_dump -Fc tracelite every night, kept off the machine
6. Upgrade: stop the service, replace the .exe, start it; the new binary migrates first
```

The role owns its database, so its migrations may create tables and the trusted
extensions of step 12. On Linux the same binary runs under systemd:

```ini
[Service]
ExecStart=/opt/tracelite/tracelite
EnvironmentFile=/etc/tracelite/env
User=tracelite
Restart=on-failure
# SIGTERM, then SIGKILL after TimeoutStopSec (90 s by default)
```

Windows is the odd one out: the Service Control Manager talks to a service through a
control handler, and a plain console program registered with `sc.exe create` doesn't
answer, so the start fails after a timeout. TraceLite's `main` asks
`svc.IsWindowsService()` (from `golang.org/x/sys/windows/svc`) and, when true, runs under
`svc.Run`, whose Stop request cancels the same context that Ctrl+C cancels on a
console. The alternative without code is a service wrapper such as WinSW or NSSM.

The service details are documented behavior, not run here. The SCM waits for a new
service process to call `StartServiceCtrlDispatcher`, which
[its reference page](https://learn.microsoft.com/en-us/windows/win32/api/winsvc/nf-winsvc-startservicectrldispatcherw)
says to do "within 30 seconds"; a console program never calls it, so the start fails
with error 1053, `ERROR_SERVICE_REQUEST_TIMEOUT`: "The service did not respond to the
start or control request in a timely fashion"
([system error codes](https://learn.microsoft.com/en-us/windows/win32/debug/system-error-codes--1000-1299-)).
[golang.org/x/sys/windows/svc](https://pkg.go.dev/golang.org/x/sys/windows/svc)
(v0.50.0) has `IsWindowsService() (bool, error)` and `Run(name, handler)`, which makes
that call for you.

<!-- VERIFY: how step 4 puts variables into the service's environment. NSSM's and Elastic's docs use a REG_MULTI_SZ value Environment (one NAME=value per line) under HKLM\SYSTEM\CurrentControlSet\Services\<name>; no Microsoft documentation of that value was found (2026-11-14), and it can't be tried here without creating a service (admin rights). -->

StackLab, the app running these lessons, is packaged differently on purpose:

| | StackLab, Docker mode | StackLab, local mode (ADR-0008) | TraceLite |
|---|---|---|---|
| Frontend | Vite dev server in a `node` container | Vite dev server, started by `npm start` | built `web/dist`, embedded, served by the binary |
| Migrations | embedded in `cmd/stacklab` | embedded | embedded |
| Backend | `CGO_ENABLED=0 go build -trimpath` in a distroless image | `go build` to `.bin/` by `scripts/dev.mjs` | one cross-compiled file per OS |
| Database | PostgreSQL containers | PostgreSQL 18 Windows service | PostgreSQL 18 on the customer's machine |
| Why | a development tool for one person: the UI changes daily, Vite's hot reload matters more than an artifact | the same, without containers | a product: one file to install, upgrade and support |

## Design notes

**Embed, don't deploy files next to the binary.** A directory of static files beside the
executable can be edited, half-copied or left from the previous version. Embedding makes
the UI and the migrations part of the build that the version names, and a customer's
"which version are you running?" has one answer.

**Self-migration, with a way back.** Migrating at startup is right for one instance per
database, which is every TraceLite installation. Each migration is additive where it can
be (step 11's jobs table, step 12's columns, step 13's log), so the previous binary still
runs against the new schema if an upgrade has to be undone; a migration that breaks that
rule says so in its release notes. Several instances would need goose's session locker,
or a separate `tracelite migrate` step run once per release.

**What the fifteen steps built.**

| Step | Added |
|---|---|
| 1 | The architecture, the request lifecycle, `openapi.yaml` as the contract |
| 2 | The schema: projects, a requirement tree, test cases, trace links, test runs; migration 001 |
| 3 | The store: sqlc queries, `WithTx`, constraint names turned into errors |
| 4 | The REST API: strict JSON, RFC 9457 problems, keyset cursors |
| 5 | The typed client: generated types, zod at the boundary, TanStack Query hooks |
| 6 | The UI: the requirement tree, the detail form, the trace matrix with coverage |
| 7 | CSV import: Turkish Excel files, all or nothing, errors by line |
| 8 | Streaming exports: CSV for Excel and the traceability report |
| 9 | Live updates: triggers, LISTEN/NOTIFY, a hub, server-sent events |
| 10 | The outside world: a cached standards catalogue and signed webhooks |
| 11 | Background jobs: a job table, SKIP LOCKED, leases, retries, graceful shutdown |
| 12 | Search: codes, Turkish words and typos, ranked and paged |
| 13 | Audit history: a trigger, the user through `set_config`, and its trade-offs |
| 14 | Tests at every layer, each catching what the others can't |
| 15 | One binary with the app and migrations inside, and a way onto a customer's machine |
