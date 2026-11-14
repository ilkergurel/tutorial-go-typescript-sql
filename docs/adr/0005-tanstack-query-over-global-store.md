# ADR-0005: TanStack Query instead of a global state library

Date: 2026-11-12 · Status: accepted

## Context

The frontend needs server state (lessons, curriculum, progress, run results), local UI
state, and navigation state. A global store (Redux, Zustand, MobX) is the reflex for
"app state".

## Decision

- **Server state**: TanStack Query (cache, invalidation, retries, optimistic updates).
- **Local UI state**: `useState` / `useReducer`.
- **Navigation state**: the URL (React Router) — track, lesson, example and tab are all
  URL parameters, making every view deep-linkable.
- No global state library.

## Alternatives

- **Redux Toolkit / Zustand**: a global cache would re-implement what TanStack Query
  already solves better (refetching, staleness, race conditions on slow responses,
  invalidation after a run or a progress change).

## Consequences

- State is colocated with the feature that owns it; no cross-feature store reads, which
  keeps feature folders independent (the frontend architecture lesson uses this).
- Component tests mock the API layer with MSW rather than a store; e2e tests drive the URL.

## Update (2026-11-14)

"No global state library" holds; TanStack Query was never adopted for the app.
Learning state (progress, notes, bookmarks, quiz answers) goes through fetch wrappers
(`src/features/learning/api.ts`) and a small module-level store with subscribers
(`learning/store.ts`), falling back to localStorage when `/healthz` fails or answers
with an error status (Vite's proxy answers 500 when the backend is down). Lessons and
the curriculum are generated files, not server state: a small eager index plus one JSON
file per lesson, loaded with `import()` when the lesson opens. Run results are the run
panel's component state. `@tanstack/react-query` is installed for
ts-browser lessons only (ADR-0010). The URL holds the path, track and lesson; the
selected example and tab are component state, so they aren't deep-linkable. At the time
of this update `npm test` runs Vitest unit tests over `src/`, with no MSW component
tests or URL-driven e2e tests. No feature folder imports another any more: shared
pieces live in `src/shared/` (`API_BASE` in `shared/api.ts`), `app/` composes the
features, and `eslint.config.js` enforces that layering.
