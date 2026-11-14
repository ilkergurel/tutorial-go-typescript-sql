# ADR-0008: Local mode — no containers, lessons run in the backend

Date: 2026-11-13 · Status: accepted (narrows ADR-0003 to Docker mode)

## Context

StackLab is for one person on one Windows 11 machine. The Docker setup (six
containers, WSL2, bind mounts) caused more trouble than it prevented: stale
Docker Desktop mounts, internal-network port publishing, and a second copy of the
repo inside WSL to keep in sync. Go and Node were already installed on Windows.

## Decision

Local mode is the default way to run StackLab, started with `npm start`:

- One PostgreSQL 18 (Windows service) holds both databases: `stacklab` (app data)
  and `playground` (SQL lessons, a **non-superuser** role).
- The Go backend executes lessons **in-process** through `internal/runexec` when
  `STACKLAB_RUNNER_URL` is unset. Docker mode sets it and keeps using the runner
  container; both modes share runexec, so they cannot drift.
- gorun kills the whole process tree on timeout on both OSes: process group on
  Unix, `taskkill /T` on Windows (`proc_*.go`).
- Config defaults target local mode, so no environment or `.env` is needed.
- mocksvc is not part of local mode (nothing calls it yet).

Docker Compose stays as an optional mode; ADR-0003 still describes it.

## Alternatives

- **WSL without Docker**: Linux semantics and no gorun change, but Go, Node and
  PostgreSQL would need installing inside Ubuntu, and the two-copy problem stays.
- **Keep the runner as a separate local process**: keeps the HTTP hop, gains no
  isolation (no container around it), and adds a process to start and stop.
- **Job Objects for the Windows tree kill**: exact, but need `golang.org/x/sys`
  (outside the allow-list). `taskkill /T` is boring and ships with Windows.
- **SQLite for app data**: fewer installs, but PostgreSQL is needed anyway for the
  SQL lessons, so two engines would only add surface.

## Consequences

- Lesson code runs as the owner's user with no memory/pids/network limits; what
  remains is per-run hygiene: fresh temp dir, compile (20 s) and run (5–15 s)
  timeouts, process-tree kill, 64 KB output caps, at most 2 concurrent runs.
  Acceptable under ADR-0003's threat model (protect the owner from their own
  mistakes); not acceptable if the app is ever exposed beyond 127.0.0.1.
- Lesson SQL runs as a non-superuser, because a superuser could run shell
  commands (`COPY … TO PROGRAM`) on the owner's machine.
- Go lessons run on Windows: OS-specific behaviour (signals, permissions, paths)
  differs from the Linux runner image; lessons on those topics must say so.
- One-time setup: install PostgreSQL 18, apply `scripts/setup-local-db.sql`,
  `npm install`.

## Update (2026-11-14)

The Windows tree kill now uses a Job Object per run after all. The objection above
was the `golang.org/x/sys` dependency; `proc_windows.go` loads the three kernel32
calls with the standard library's `syscall.NewLazyDLL` instead, so nothing joins
go.mod. The reason to switch: `taskkill /T` walks parent links, so it can't find the
children of a program that has already exited, and leftovers outlived their run.
Every child of the lesson process joins its job, and `TerminateJobObject` ends them
all, whether the run timed out, was cancelled or exited normally; `taskkill /T` stays
as the fallback when no job can be created. On Unix the process group is now killed
in the same three cases, not only on timeout. A child can still escape through WMI
(`Win32_Process.Create`); the job has no kill-on-close flag, so a backend crash leaves
lesson processes running, as before.
