# CLAUDE.md — StackLab working rules

StackLab is a personal, local-only learning app for Go, TypeScript and PostgreSQL.
This file is the working contract for every agent session. If a convention changes,
update this file in the same change.

## Status

- **All eight phases shipped** — skeleton, content pipeline, runners, learning features,
  content batch, advanced runner modes (go-test, sql-multisession), learning paths, polish.
- Local mode (ADR-0008) is the default way to run: `npm start` on Windows, one
  PostgreSQL 18 service, lessons run in the backend process. Docker Compose is kept
  as an optional mode and must keep working.
- Note: BUILD_PROMPT.md is no longer in the repo (referenced by rule 3 below); the
  phase plan it defined is summarized in README's Status section.
- After any version restore, expect to re-restore: Phase 5 content dirs, the
  app.log fixture, go/errors `expected/wrap.txt`, sqlrun/multisession.go,
  scripts/copy-wasm.mjs + public/esbuild.wasm, and the Go toolchain at ~/gotool/bin/go.
  `node scripts/build-lessons.mjs` reports missing fixture/expected files.
- Every runtime is implemented and every example is verified by running it
  (ADR-0009): `npm run verify:lessons` (needs `npm start`). A lesson is done only when
  it passes there; expected files are generated with `--write-expected` and reviewed.
- Lessons may use real libraries in local mode (ADR-0010): pgx / x/sync / x/text in Go,
  react / react-router-dom / @tanstack/react-query / zod in ts-browser, and Vitest,
  Testing Library, MSW and Playwright (installed Edge) through the `ts-test` runtime.
- Curriculum complete (2026-11-14): all 229 lessons written, all 642 examples passing
  `npm run verify:lessons` (each run twice); contentcheck, `npm test`, typecheck, lint,
  `go vet` (Windows and GOOS=linux) and `go test ./...` (plus `-tags integration`) clean,
  last checked after the backlog cleanup below.
  Every example is at most 80 lines (the style guide's hard limit since 2026-11-14,
  when the 66 longer ones were shrunk or split). TraceLite's cross-step decisions
  (schema, migrations 001-006, API, events, jobs, search, audit, packaging) are in
  docs/TRACELITE-CONVENTIONS.md; a change to TraceLite starts there.
- Request guards (ADR-0011) closed a cross-site code-execution hole in `/api/run`, and the
  startup log no longer prints the database password.
- Unconfirmed claims are visible: `<!-- VERIFY: ... -->` markers render as warnings in
  the lesson, the lesson header names the tabs that hold them, and `/unverified` lists
  them all (linked from the dashboard). Removing a marker is how a claim is signed off.
  `npm test` (Vitest over `src/`) checks that every marker and table in every lesson
  renders.
- Backlog cleanup done on 2026-11-14. Fixed: the sql sandbox's COMMIT/ROLLBACK hole
  and COPY FROM STDIN hang (sqlrun/guard.go); `BEGIN ATOMIC` splitting; the database
  permissions (setup-local-db.sql revokes ALL on the stacklab database from PUBLIC and
  gives the playground's public schema to postgres; applied on the owner's machine);
  `null` lists on the wire (`runapi.Result.MarshalJSON`); panic recovery and
  problem+json 404/405; runclient's refusal detail; the process tree killed whenever a
  run ends (Job Objects on Windows, ADR-0008's update); UTF-8-safe output caps; ts-test
  colours and `timeout_s`; the run write deadline vs queue wait; run history bounded to
  1000 rows; goose direct in go.mod; Monaco bundled locally; per-lesson lazy loading;
  no cross-feature imports (eslint enforces it); example `setup` shown; ts-browser
  error-boundary and unhandled-rejection reporting; docs drift (ARCHITECTURE, ADRs
  0002-0008, README) and stale lesson facts. The owner accepted the TraceLite decisions
  in docs/TRACELITE-CONVENTIONS.md (2026-11-14). Still open:
  - Docker mode is untested since ADR-0008..0011: the playground role is a superuser
    there, the runner sits on backend-net, the runner image lacks the Go toolchain,
    runclient's 45 s timeout and cmd/runner's 45 s WriteTimeout don't count the 10 s
    queue wait, and pgx/ts-test lessons are local-only by design.
  - Runners: go-test ignores `timeout_s` (fixed 35 s); goBuild/goVet kill only the `go`
    command on timeout; the Windows job has no kill-on-close flag, so a backend crash
    leaves lesson processes running; the go runtime builds without `-ldflags` (versions
    print `dev`); goBuild and goVet return only the captured output and drop the exec
    error, so a missing `go` gives exit 1 with empty stderr (runner/Dockerfile's comment
    also claims a Go toolchain the image doesn't have); sqlrun's splitter still splits at a `;` inside parentheses (psql
    doesn't); under Playwright `page.route`, Edge doesn't intercept `<a download>`
    downloads (step 14 uses a real `node:http` server).
  - ts-browser runner: an error thrown in a timer callback or event listener ends the
    program, so output after a non-timer await can be cut; the shim's script sits in
    `<body>`, so `document.body.textContent` includes it.
  - Small: `vite build` emits Monaco's css/html/json workers that are never loaded
    (~2.2 MB of dist); no screen reads the run history; gorun.go's discriminated-union
    comment and the never-called `runapi.Decode` (inside.codebase.contract cites both);
    `npm test` from Git Bash can load Vitest twice (C:/ vs c:/): run it from PowerShell.
  - Intermittent, seen once (2026-11-14, the first full run after the Go 1.27.4
    upgrade): `sql.transactions.deadlocks-locks` / deadlock returned a transcript that
    ended after "[6] B: waiting". It passed every other full run and 6 of 6 reruns;
    cause not found. In the same run `go.performance.cross-compile` / ship-binaries
    timed out once while Go rebuilt the standard library for four targets after the
    upgrade (a cold build cache); expect that after any Go upgrade.
  - Unconfirmed claims: on 2026-11-14, at the owner's request, Claude settled 28 of the
    30 (official docs, the tools' source, runs on this machine incl. Excel 16.0 by COM
    and sqlc 1.31.1 once); 14 claims were partly wrong and their lessons are fixed. Verdicts and
    evidence are on https://claude.ai/artifact/NWnThkvahWsKyWacyMdRUc (`verdicts`
    collection). Two warnings remain: step 15's REG_MULTI_SZ `Environment` value (no
    Microsoft source; testing needs admin) and inside.codebase.compose-topology's Docker
    internal-network ports (no Docker tests, owner's instruction).
  - Owner's option: a tab-prefix guard against CSV formula injection in TraceLite's
    export (step 8; step 7's import already trims it). A TraceLite convention change.
- Next: backlog items the owner picks. No PostgreSQL 19 bump (owner's decision,
  2026-11-14). No pending phases, no unwritten lessons.

## Versions (single source of truth)

Pin versions in exactly one place. When you bump one, update this table and the
version comment in `docker-compose.yml`.

| Tool | Version | Pinned in |
|---|---|---|
| Go | 1.27.x (1.27.4 known good, 2026-11-14: installed on the owner's machine; Go and npm checks clean, 640/642 examples in the first full run and the two misses passed on rerun, see the backlog) | `docker-compose.yml` build args, `.env.example`, `runner/Dockerfile`, `server/Dockerfile` + `server/go.mod` `toolchain` |
| TypeScript | 7.0.x (GA 2026-07-08; 7.0.2 known good, 2026-11-13) for `npm run typecheck`, plus 6.0.x for typescript-eslint (ADR-0006) | root `package.json`: `typescript7` alias (7.0.x) + `typescript` (6.0.x) |
| PostgreSQL | 18.x (the owner first chose "always latest", then on 2026-11-14 decided to stay on 18) | `docker-compose.yml` `PG_MAJOR` build arg + README local-mode winget id and psql path (`PostgreSQL.PostgreSQL.18`) — the owner decided against the bump to 19 (2026-11-14) and dropped the planned "What's new in 19" lesson (2026-11-13) |
| Node | 22 LTS in the Docker images; local mode uses the `node` on PATH (owner's machine: 24.22.0 via nvm-windows, 22.15.1 also installed; ts-node lessons verified on both, 2026-11-13) | `NODE_VERSION` in `docker-compose.yml` + `runner/Dockerfile` |

## Environment

- Target machine: owner's Windows 11 machine, all ports on 127.0.0.1. Default is local
  mode natively on Windows (Go 1.27.4 + Node on PATH, PostgreSQL 18 service). Docker
  Desktop + WSL2 is optional; Makefile targets must still work from WSL2.
- gorun has per-OS process handling (`proc_unix.go`, `proc_windows.go`): keep both
  building — run `go vet ./...` with GOOS=linux as well as on Windows.
- Where to verify: on the owner's machine, verify natively (`go vet`, `go test` incl.
  `-tags integration`, `npm run typecheck`, `npm run lint`, `npm start`). In a sandbox
  without Go/Docker/psql (Debian 13), use static checks only (`tsc`, tests, contentcheck).
- The owner works in Turkey: mind İ/ı case mapping, the `turkish` FTS configuration,
  UTF-8 bytes vs characters. Relevant lessons must cover these.

## Working rules

1. Small, reviewable steps. Conventional Commits per meaningful step.
2. Never weaken a test or remove a check to make it pass. Fix the cause or ask.
3. If the spec (BUILD_PROMPT.md) is wrong/outdated/contradictory, say so and propose
   an alternative; never silently deviate.
4. Ask before adding a dependency outside the §4 allow-list, changing the stack, or
   changing the lesson format.
5. Prefer the standard library and boring solutions. Explain any cleverness.
6. Secrets stay out of the repo; only `.env.example` is committed.
7. Language facts must be verified against official docs. Unverifiable claims get a
   `<!-- VERIFY: ... -->` marker and are reported to the owner.
8. Keep this file current whenever a convention changes.

## Conventions

- Comments explain **why**, never what. Doc comments on every package/feature folder:
  what it does, why this shape, which alternative was rejected.
- `// LEARN(<lesson-id>): ...` markers on illustrative code; `contentcheck` verifies them.
- ADRs in `docs/adr/`, one page max: context · decision · alternatives · consequences.
- Errors: wrap with context, map to HTTP status in one place (httpapi).
- The API runs code, so every route sits behind request guards (ADR-0011): cross-origin
  protection, JSON-only bodies, loopback Host on `/api/`. Don't add a route outside
  `NewRouter`, and don't loosen a guard to make a client work.
- `context.Context` first parameter on anything doing I/O.
- Interfaces declared by the consumer; constructor injection; no globals, no DI framework.
- Frontend: feature folders, no `any`, discriminated unions with exhaustive switches.
- Stop at every phase checkpoint and wait for the owner's "go".
