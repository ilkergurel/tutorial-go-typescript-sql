# StackLab

A personal, local-only learning app for **Go**, **TypeScript** and **PostgreSQL**.
Every topic is a lesson with runnable examples; the app's own source code is part of
the curriculum ("Inside StackLab" track).

Local-only and single-user: no authentication, all ports bound to 127.0.0.1.

## Status

- **All eight phases shipped, curriculum complete.** 229 lessons, every example
  verified by running it (`npm run verify:lessons`, ADR-0009). Runtimes: go, go-test,
  go-server, sql, sql-nontx, sql-multisession, ts-browser, ts-node and ts-test.
  Learning features: progress, notes, bookmarks and quizzes (with a localStorage
  fallback), learning paths, and the `/unverified` list of claims still to confirm.
- **Local mode** (no Docker) is the default way to run it (ADR-0008). Docker Compose is
  optional.

Historical phase log: Phase 1 skeleton · Phase 2 content pipeline · Phase 3 runners ·
Phase 4 learning features · Phase 5 content batch · Phase 6 advanced runner modes
(go-test, sql-multisession) · Phase 7 learning paths · Phase 8 polish.

## Run locally (Windows, no Docker)

Needs Go 1.27.x and Node 22 or newer on PATH, PostgreSQL 18, and Microsoft Edge (the
Playwright lessons drive it; Windows 11 ships it).

One-time setup, in PowerShell from the repo root:

1. Install PostgreSQL 18 (the installer asks for a password for the `postgres`
   superuser; Stack Builder is not needed):
   `winget install --id PostgreSQL.PostgreSQL.18 -e --interactive`
2. Create StackLab's roles and databases (asks for that password):
   `& "C:\Program Files\PostgreSQL\18\bin\psql.exe" -U postgres -f scripts/setup-local-db.sql`
   It is safe to re-run (existing roles and databases are left alone). It also limits
   PostgreSQL to connections from this machine, from the next service restart.
   If your `playground` database was created by an older version of the script (with
   `LC_CTYPE` `C`), run `scripts/fix-playground-ctype.sql` the same way once: it
   recreates only that database.
3. `npm install`

Every day:

```
npm start        # builds + runs the Go backend and the Vite dev server
```

Open <http://127.0.0.1:5173>. Ctrl+C stops both. The first start also bundles the
libraries that browser lessons import. PostgreSQL keeps running as the Windows service
`postgresql-x64-18`; set it to *Manual* in `services.msc` if you prefer to start it
yourself.

Lesson code runs as normal processes on your PC: timeouts, output caps and
process-tree kills apply, but there are no memory, process-count or network limits
(ADR-0008). SQL lessons run as the non-superuser `playground` role in throwaway schemas.

## Run with Docker (optional)

Docker Desktop with the WSL2 backend; the Makefile targets run in WSL2.

1. Install WSL2: `wsl --install` (PowerShell as admin), then restart.
2. Install [Docker Desktop](https://www.docker.com/products/docker-desktop/) with
   *Use WSL 2 based engine* enabled, and enable WSL integration for your distro.
3. Install Make inside WSL if missing: `sudo apt install make`.
4. From the repo root inside WSL: `make up`, then open <http://127.0.0.1:5173>.

Keep the repo inside the WSL2 filesystem (e.g. `~/code/stacklab`), not under
`/mnt/c/...`: Vite's file watching is an order of magnitude faster there. Don't
delete and re-copy the folder while Docker knows it (copy over it instead):
Docker Desktop can keep a stale mount of the old folder.

Docker mode hasn't been re-tested since local mode became the default, and lessons
that use pgx, `golang.org/x/sync`, `golang.org/x/text` or the ts-test runtime run in
local mode only (ADR-0010).

## Services

| Address | Local mode | Docker mode |
|---|---|---|
| http://127.0.0.1:5173 | Frontend (Vite dev server) | same |
| http://127.0.0.1:8080/healthz | Backend liveness | same |
| http://127.0.0.1:8080/readyz | Backend readiness (checks the app database) | same |
| 127.0.0.1:5432 | PostgreSQL: databases `stacklab` and `playground` | not reachable from the host (internal network) |

In Docker mode the `runner` and `mocksvc` containers sit on internal networks with no
internet access (ADR-0003).

## Common commands

```
npm start                      # local mode: run everything
npm run verify:lessons         # run every lesson example and check its output (needs npm start)
npm test                       # Vitest over src/
npm run typecheck              # TypeScript 7 type check
npm run lint                   # ESLint with typescript-eslint (which loads TypeScript 6.0)
node scripts/build-lessons.mjs # validate content/ and regenerate the lesson data
cd server; go vet ./...; go test ./...

make up / make down            # Docker mode (from WSL)
```

Go: `go test -tags integration ./...` also compiles and runs real lesson programs; the
database tests in it need `TEST_DATABASE_URL` set to the playground database
(`postgres://playground:playground@127.0.0.1:5432/playground`). Run `go vet` with
`$env:GOOS = "linux"` too, so the Unix process code keeps building.

## Adding a lesson

Lessons live under `content/<track>/<module>/<NN-slug>/` as `lesson.md` plus examples,
expected outputs and fixtures. After editing `content/curriculum.yaml`, run
`npm run gen:curriculum` to regenerate the sidebar data. After editing a lesson, run
`node scripts/build-lessons.mjs`: it validates frontmatter against the schema, checks
that example, expected and fixture files exist, and regenerates the lesson data the
frontend imports (`npm start` doesn't run it; Vite picks up the result).

With the app running (`npm start`), `npm run verify:lessons -- --filter <lesson-id>`
runs every example the way the Run button does and checks it against its expected
output (twice, to catch nondeterminism). `--print`, `--write-expected` and
`--annotations` support writing a lesson; docs/STYLE-GUIDE-CONTENT.md has the workflow.

## Docs

- `docs/ARCHITECTURE.md` — how the parts fit together, the Run button, the data model
- `docs/STYLE-GUIDE-CONTENT.md` — lesson writing rules
- `docs/TRACELITE-CONVENTIONS.md` — cross-step decisions of the TraceLite project lessons
- `docs/adr/` — one short ADR per significant decision (0008: local mode, 0009: verified
  runtimes, 0010: lesson libraries, 0011: request guards)
- `CLAUDE.md` — working rules for agent sessions
