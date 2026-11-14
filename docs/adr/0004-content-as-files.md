# ADR-0004: Lesson content as repo files, not database rows

Date: 2026-11-12 · Status: accepted

## Context

Lesson content must be reviewable, versioned and editable by the owner. It could be
stored in Postgres (app-db) or as files in the repository.

## Decision

Content lives as files under `content/` (one directory per lesson: `lesson.md` with
frontmatter, `examples/`, `expected/`, `fixtures/`, `exercises/`). The backend loads and
indexes it into app-db at startup (upsert on content hash, delete removed lessons) so
search and the lessons table still work. `cmd/contentcheck` validates and runs every
example in CI-style checks. No LLM calls at runtime — all content is pre-written and
human-reviewed.

## Alternatives

- **Content in Postgres**: runtime editing from the UI, but content becomes invisible to
  diff/review tools, unversioned beside the code that teaches it, and examples can't be
  run by a checker without round-tripping through the DB.
- **Content in the frontend bundle**: no backend round-trip, but large-bundle cost and
  the Inside StackLab `/api/source` path would still need server-side file access.

## Consequences

- `content/` is the source of truth; the DB is a derived index and can be rebuilt at
  startup (`content_hash` upserts make this idempotent).
- `verify-content` (schema validation, unique ids, prerequisite graph without cycles,
  every example runs and matches expected output, LEARN markers exist) gates every
  content batch.
- Dev-mode only: watch `content/`, re-index on change, push an SSE `content-changed`
  event so open lessons refresh.

## Update (2026-11-14)

Content as files holds; the database side was never built. The backend doesn't read
`content/`, and the app database has no lessons table. `scripts/build-lessons.mjs`
(contentcheck, a Node script rather than `cmd/contentcheck`) validates every lesson and
generates the lesson data the frontend imports, which is in effect the "content in the
frontend bundle" alternative. There is no `/api/source`, no search, no watcher and no
SSE event: after editing content, run `node scripts/build-lessons.mjs` and Vite picks up
the regenerated files. contentcheck checks that prerequisites exist, not that the graph
is acyclic; running every example against its expected output is
`npm run verify:lessons` (ADR-0009), which needs the app running.
