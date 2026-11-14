---
id: ts.react.query
title: "TanStack Query: caching, invalidation, optimistic updates"
track: ts
module: react
order: 5
level: advanced
minutes: 15
since: ts4.1
prerequisites: [ts.react.effects-hooks, ts.services.fetch]
compare_with: [ts.architecture.state, ts.services.advanced, go.services.retries]
tags: [react, tanstack-query, react-query, caching, invalidation, optimistic-updates, server-state]
summary: >
  Server data is a cache, not component state. useQuery reads it by key with
  deduplication and stale-while-revalidate refetching; useMutation changes the
  server, updates the cache optimistically, rolls back on error and invalidates
  what it touched.
examples:
  - id: cache
    file: examples/01_cache.tsx
    title: "useQuery: one request for many components, cached by key"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "8-12"
        note: >
          The fake server is an async function with an access log: the lesson
          iframe has no network, and the log makes every request visible.
          useQuery doesn't care where the promise comes from.
      - lines: "16-21"
        note: >
          v5 accepts only the object form. queryKey is the cache identity:
          include every input the fetch depends on (here track), so
          ["titles", "go"] and ["titles", "ts"] are separate entries. staleTime
          says how long fetched data counts as fresh; the default is 0.
      - lines: "22-27"
        note: >
          status is a union, "pending" | "error" | "success" (v5 renamed
          "loading" to "pending"), and the result type is a discriminated union
          on it: before the checks, data is string[] | undefined; after them,
          string[]. isFetching is separate: true during any fetch, including a
          background refetch while old data is on screen.
      - lines: "30-33"
        note: >
          retry: false because the default retries a failed query three times
          with exponential backoff, which a lesson doesn't want. gcTime (called
          cacheTime before v5) is how long unused data stays cached, 5 minutes
          by default; Infinity here only so no timer outlives the run.
      - lines: "46-51"
        note: >
          Harness: wait until no query is fetching, then one more timer turn,
          because TanStack hands results to components in a batched 0 ms timer
          and timers fire in the order they were set.
      - lines: "53-62"
        note: >
          Three components, two keys, two requests: the two "go" components
          share one cache entry and one in-flight promise. No prop drilling, no
          global store, no request bookkeeping in components.
      - lines: "63-67"
        note: >
          Unmounting doesn't drop the data. Remounting renders it immediately
          and, because it is stale (staleTime 0), refetches in the background:
          "(yenileniyor)" and a third request. This is stale-while-revalidate.
      - lines: "68-69"
        note: >
          With staleTime 60 s the cached "ts" data still counts as fresh, so
          mounting causes no request at all. Choose staleTime per kind of data:
          how old may this be before the user would mind?
  - id: optimistic
    file: examples/02_optimistic.tsx
    title: "useMutation: optimistic update, rollback, invalidation"
    runtime: ts-browser
    check: exact
    timeout_s: 15
    annotations:
      - lines: "9-21"
        note: >
          The fake server holds each PATCH until the harness answers, so the
          output can show the screen while the request is in flight, then
          after success, then after a failure.
      - lines: "23"
        note: >
          One constant for the key. Bigger apps use query-key factories, so
          queries and the mutations that invalidate them can't drift apart.
      - lines: "27-36"
        note: >
          onMutate runs before the request. Cancel in-flight fetches of the key
          so an older response can't overwrite the optimistic data, snapshot
          the current list, write the expected result into the cache, and
          return the snapshot. Every component reading ["todos"] updates at
          once. The second parameter is the mutation context, whose client is
          the QueryClient; older v5 code calls useQueryClient() instead.
      - lines: "37-40"
        note: >
          onError receives onMutate's return value and restores the snapshot.
          onSettled runs after success or failure and invalidates the key:
          mark it stale and refetch active queries, so the cache ends up as the
          server says, not as we guessed. invalidateQueries matches by prefix,
          so ["todos"] would also cover ["todos", { done: true }].
      - lines: "42-48"
        note: >
          The component knows nothing about caching: it renders the query and
          calls mutate. toggle.isError shows the failure next to the rolled-back
          item. A button, not a clickable div, keeps it keyboard-accessible.
      - lines: "51-56"
        note: >
          Mutations are kept in a cache of their own and have a gcTime too;
          Infinity again only so no timer outlives the runner.
      - lines: "73-79"
        note: >
          First round: [x] immediately, the server agrees, the refetch confirms.
          Second round: [ ] immediately, the server fails with 503, the
          snapshot puts [x] back, and the refetch confirms that too.
use_cases:
  - "Lists and detail views backed by a REST or GraphQL API, shared by several components without a global store"
  - "Dashboards that should refresh when the user returns to the tab or reconnects"
  - "Toggles, likes, reordering and inline edits that should feel instant: optimistic updates with rollback"
  - "After a create or delete: invalidate the affected keys instead of hand-editing every cached list"
strengths:
  - "Deduplication, caching, background refetch and retries come from one library, where a useEffect fetch needs race handling and still has no cache"
  - "The result is a discriminated union on status, so TypeScript forces the pending and error branches before data is usable"
  - "Server state stays out of Redux-style global stores; client state left in React is small and easy to reason about"
  - "Invalidation by key prefix says what changed, not how to fix every view; the next fetch makes the cache right"
weaknesses:
  - "Another dependency with its own vocabulary (stale, inactive, gc, fetchStatus) and defaults that surprise: data is stale immediately and refetches on window focus"
  - "Query keys are untyped strings and arrays by default; a typo creates a separate cache entry silently"
  - "Optimistic updates are hand-written per mutation and easy to get subtly wrong (forgetting cancelQueries or the rollback)"
  - "Tests need a fresh QueryClient per test with retries off, or state leaks between tests"
pitfalls:
  - title: "Leaving inputs out of the query key"
    detail: "If the fetch depends on track but the key is just [\"titles\"], every track shares one cache entry and the UI shows the wrong data."
    bad: "useQuery({ queryKey: [\"titles\"], queryFn: () => fetchTitles(track) })"
    good: "useQuery({ queryKey: [\"titles\", track], queryFn: () => fetchTitles(track) })"
  - title: "Copying query data into useState"
    detail: "The copy stops updating when the cache refetches or a mutation invalidates it. Read data from the query result where you render it."
  - title: "Optimistic update without cancelQueries"
    detail: "A refetch already in flight can resolve after setQueryData and overwrite the optimistic value with older server data."
  - title: "A QueryClient created inside a component"
    detail: "new QueryClient() in render creates a new, empty cache on every render. Create it once at module level or in a useState initializer."
    bad: "function App() {\n  const client = new QueryClient();\n  return <QueryClientProvider client={client}>...</QueryClientProvider>;\n}"
    good: "const client = new QueryClient();\nfunction App() {\n  return <QueryClientProvider client={client}>...</QueryClientProvider>;\n}"
quiz:
  - q: "Two components call useQuery with the same queryKey while nothing is cached. How many times does queryFn run?"
    options: ["Once", "Twice", "Once per render"]
    answer: 0
    why: "Both observers subscribe to the same cache entry, and the second one joins the in-flight fetch."
  - q: "Cached data exists, staleTime is the default, and a component using it mounts. What happens?"
    options: ["It shows pending until a new fetch finishes", "It shows the cached data and refetches in the background", "It shows the cached data and never refetches"]
    answer: 1
    why: "The default staleTime is 0, so the data is stale: render it immediately and revalidate. isFetching is true meanwhile."
  - q: "Why return { previous } from onMutate?"
    options: ["TanStack requires a return value", "onError receives it, so it can restore the cache if the request fails", "It becomes the mutation's data"]
    answer: 1
    why: "The value is passed to onError, onSuccess and onSettled as onMutateResult; the snapshot is what a rollback restores."
  - q: "After a successful mutation, why invalidate instead of trusting the optimistic value?"
    options: ["The optimistic value is discarded automatically", "The server may have changed more than you predicted (timestamps, ordering, other users), so refetch the truth", "invalidateQueries is required for the mutation to finish"]
    answer: 1
    why: "The optimistic update is a guess for responsiveness; invalidation makes the cache converge to the server's state."
further_reading:
  - title: "React: You might not need an effect (fetching data)"
    url: "https://react.dev/learn/you-might-not-need-an-effect#fetching-data"
  - title: "React: useSyncExternalStore (what useQuery subscribes with)"
    url: "https://react.dev/reference/react/useSyncExternalStore"
  - title: "MDN: Cache-Control stale-while-revalidate"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Cache-Control"
  - title: "MDN: Using the Fetch API"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch"
---

## Concept

Data that lives on a server is not your component's state: other users change it, it gets
old, several components show it at once. TanStack Query (v5, formerly React Query) treats it
as a **cache** owned by a `QueryClient` that you provide once at the top of the tree.

- `useQuery({ queryKey, queryFn })` subscribes a component to one cache entry. The key is an
  array that identifies the data (`["titles", track]`); components with equal keys share one
  entry and one request. The result is a discriminated union on `status`
  (`"pending" | "error" | "success"`), plus `isFetching` for background refetches.
- Data is **stale** after `staleTime` (default 0). Stale data is still shown, and refetched
  when a new component mounts, the window regains focus or the network reconnects.
  Unused entries are garbage-collected after `gcTime` (default 5 minutes).
- `useMutation({ mutationFn, onMutate, onError, onSettled })` changes the server. After it,
  **invalidate** the keys it affected; for instant feedback, write the expected result into
  the cache in `onMutate` and roll back in `onError` (an **optimistic update**). The simpler
  alternative is optimistic *UI*: render `mutation.variables` while `isPending` and leave the
  cache alone.

The library's own documentation (guides on query keys, important defaults, invalidation
and optimistic updates) is at
[tanstack.com/query](https://tanstack.com/query/latest/docs/framework/react/overview);
v5 needs React 18 or later.

## Coming from C/C++/Java/Python/JS

- **Java/Spring:** think of `@Cacheable` keyed by method arguments plus `@CacheEvict` after a
  write, moved to the client and made reactive: evicting (invalidating) a key re-renders
  every component that reads it once the refetch lands.
- **HTTP:** `staleTime` and background revalidation are the `stale-while-revalidate` idea
  from `Cache-Control`, applied per query key instead of per URL.
- **Redux/global stores (JS):** server responses copied into a store need hand-written
  loading flags, invalidation and refetch logic. Here the store is the cache and those
  concerns are configuration.
- **Go:** like a `singleflight.Group` (concurrent callers for one key share one call) in
  front of a TTL cache, but on the client and wired to rendering. The default of three
  retries with exponential backoff is the retry loop you'd write with `go.services.retries`.

## How it works

Each `useQuery` creates an observer for its key and subscribes to it with React's
`useSyncExternalStore`, the hook from the effects lesson for external stores. The
`QueryCache` maps a stable hash of the key to a query holding `data`, `error`, timestamps and
any in-flight promise. Fetches update the query, observers compute a new result, and the
library notifies subscribed components in batches. `invalidateQueries({ queryKey })` matches
every query whose key starts with the given prefix, marks it stale and refetches the ones that
are currently rendered.

## Design notes

Split state by owner. Server state goes through queries and mutations; URL state (filters,
pagination) goes into the URL (next lesson); what remains for `useState` and context is small:
open panels, draft input. Most "we need a global store" discussions dissolve after that split.

Treat query keys as an API. Put them in one module per feature (`todoKeys.all`,
`todoKeys.list(filter)`) so a mutation invalidates exactly the keys its queries use. Choose
`staleTime` from the data's nature: a curriculum that changes on deploy can be fresh for
minutes, a chat list for seconds. Keep `retry` on in the app and off in tests. Optimistic
updates pay off for small, high-frequency, likely-to-succeed actions (toggles, reorders);
for a payment, show the pending state honestly instead.

This app doesn't use TanStack Query yet: `src/features/learning` keeps progress in a
hand-written store with optimistic updates and manual rollback (`markLesson`,
`toggleBookmark` in `store.ts`). Comparing that file with the second example shows what
the library would take over: the rollback, the subscription, and the refetch after a write.
