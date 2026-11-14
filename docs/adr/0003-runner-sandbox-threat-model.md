# ADR-0003: Runner sandbox and its threat model

Date: 2026-11-12 · Status: accepted — applies to Docker mode only since ADR-0008
(local mode runs lessons in the backend process, without the container).

## Context

Lessons must run arbitrary Go, SQL and TypeScript code — written by the owner, editable
freely. Something must execute that code outside the backend process.

## Decision

A dedicated `runner` container is the sandbox boundary: non-root user, read-only root
filesystem except per-run work directories, `pids_limit`, `mem_limit: 512m`, `cpus: 1`,
no internet (internal Docker network). Each run gets a fresh temp directory; compile
capped at 20 s, run at 5 s (15 s max per lesson flag); output capped at 64 KB per stream
with a visible truncation marker; on timeout the whole process group is killed; a
semaphore admits at most 2 concurrent runs.

## Threat model (plainly)

The sandbox protects the owner **from their own mistakes**: infinite loops, fork bombs,
gigantic output, deleting files, runaway memory. It does **not** protect against a
determined attacker: container escape, kernel exploits, and resource exhaustion against
the host are out of scope. That is acceptable because the app is local-only,
single-user, and the code author and the beneficiary are the same person. If this app is
ever exposed to a network, this ADR is void and the runner must be rebuilt (e.g. gVisor,
Firecracker, or a separate host).

## Alternatives

- **Run code in the backend process**: no isolation at all; a single infinite loop kills
  the app. Rejected.
- **WebAssembly isolation for Go**: Go targeting WASM changes the semantics being taught
  (no real files, no real goroutine preemption, no os/exec). Rejected.
- **gVisor/Firecracker**: real isolation, meaningful setup cost on the owner's machine,
  unjustified for single-user local use. Rejected for now.

## Consequences

- SQL runs against a dedicated `playground-db` with a non-superuser role owning only
  sandbox objects; transactional mode rolls back every run (see the SQL runner spec).
- The TS browser runtime needs no server sandbox at all: the browser iframe
  (`sandbox="allow-scripts"`, opaque origin) is its boundary.
- Runner hardening tests (timeout kills the process group, output cap, fork bomb
  contained) are part of the definition of done (Phase 3).

## Update (2026-11-14)

- The whole process tree is now killed whenever a run or a build step ends (timeout,
  cancellation or normal exit), not only on timeout; on Windows through a kill-on-close
  Job Object, which also covers a backend crash (ADR-0008's update).
- Not as stated above: `docker-compose.yml` creates the playground role through the
  postgres image's `POSTGRES_USER`, which makes it a superuser. Local mode's role is not
  (ADR-0008). Docker mode is untested since ADR-0008.
