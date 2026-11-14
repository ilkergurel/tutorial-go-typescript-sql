# ADR-0007: One Go module, `cmd/` + `internal/` by feature

Date: 2026-11-12 · Status: accepted

## Context

The repo contains several Go programs: the backend, the runner, mocksvc, and the
content checker. They could be one module or one module per program.

## Decision

One Go module (`server/go.mod`). Programs live under `server/cmd/stacklab`,
`server/cmd/runner`, `server/cmd/mocksvc`, `server/cmd/contentcheck`; shared code under
`server/internal/` organized **by feature** (lessons, progress, runclient, gorun,
sqlrun, tsrun, store, config, platform/httpapi), not by layer type.

## Alternatives

- **A module per program**: isolated dependency graphs (the runner would not carry the
  backend's deps), but the runner template already isolates lesson-code dependencies;
  multi-module adds replace-directive friction and cross-module refactors for no
  teaching benefit.
- **`internal/` by layer (handlers/, services/, models/)**: the classic layout scatters
  one feature across three directories; organizing by feature keeps the
  handler→service→store vertical of each lesson topic in one place.

## Consequences

- `cmd/*` mains only wire dependencies; no business logic there.
- Interfaces declared by the consumer, constructor injection, no globals — the app is
  the reference example for the "Design & architecture in Go" module.
- A separate `runner-template/` module (pre-resolved go.mod/go.sum + baked module cache)
  covers the one case that genuinely needs isolation: lesson code must build offline
  with `GOPROXY=off`.

## Update (2026-11-14)

The one-module decision holds; two planned pieces never appeared. There is no
`cmd/contentcheck`: the content check is `scripts/build-lessons.mjs`, a Node script,
because the lesson data ends up in the frontend (ADR-0004's update). There is no
`runner-template/` module either: `internal/gorun` embeds the scaffold instead, a bare
`go.mod` for standard-library programs and a pinned `go.mod`/`go.sum` for lessons that
import pgx, x/sync or x/text (ADR-0010), built with `GOPROXY=off` against the module
cache the backend already uses. The programs are `cmd/stacklab`, `cmd/runner` and
`cmd/mocksvc` (the last two for Docker mode), and the features under `internal/` are
config, gorun, httpapi, learningstore, platform, runapi, runclient, runexec, runstore
and sqlrun; there is no `tsrun` (TS runs in the browser or through `gorun`'s Node path).
