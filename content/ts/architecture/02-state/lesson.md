---
id: ts.architecture.state
title: "Choosing the right home for state: server, client or URL"
track: ts
module: architecture
order: 2
level: intermediate
minutes: 12
since: es2015
prerequisites: [ts.react.query, ts.react.routing, ts.browser-dom.storage]
compare_with: [ts.react.state, ts.architecture.explicit-states, sql.modeling.denormalize, go.http.handlers]
tags: [state, architecture, react, url-state, server-state, derived-state]
summary: >
  Every piece of state has one home: derived (not stored), the server cache, the URL,
  browser storage, or a component. Most state bugs are a value living in two homes
  at once, or in a home with the wrong lifetime or identity.
examples:
  - id: copy-drifts
    file: examples/01_copy_drifts.tsx
    title: "Server state copied into useState: the copy stops updating"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "6-20"
        note: >
          A minimal external store plays the server-state cache. It replaces the
          Set instead of mutating it because useSyncExternalStore compares
          snapshots with Object.is; a mutated Set would look unchanged.
      - lines: "22-27"
        note: >
          The component does re-render after markDone, but useState ignores its
          argument after the first render, so count is frozen at 1. The same bug
          appears as useState(props.x): the copy no longer tracks its source.
      - lines: "29-32"
        note: >
          No state at all: the count is derived from the one source on each
          render, so there is nothing to keep in sync.
      - lines: "47-49"
        note: >
          Same store, same moment, two answers. In an app this is the "I saved
          it but the dashboard still says 1" bug report, and no amount of effect
          code fixes it as cleanly as deleting the copy.
  - id: draft-identity
    file: examples/02_draft_identity.tsx
    title: "A draft belongs to the thing it edits: key it by that thing's id"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "11-27"
        note: >
          The same shape as LessonNotes in this app: a draft loaded into local
          state when lessonId changes. Copying is right here, because a draft is
          supposed to diverge from the server until it is saved.
      - lines: "14"
        note: >
          loaded is set once and never reset. When lessonId changes, the old
          body and loaded=true survive until the new note arrives.
      - lines: "29-32"
        note: >
          React keeps state per position in the tree. A key ties it to an
          identity instead: a different key unmounts the old Notes and mounts a
          new one with fresh useState values, no reset code needed.
      - lines: "48-49"
        note: >
          Without the key, the second lesson shows the first lesson's note,
          editable, for as long as the request takes. Anything typed in that
          window is overwritten when the load resolves; in this app's
          LessonNotes, the debounced save would also have stored it under the
          new lesson's id.
use_cases:
  - "Reviewing a component: for each useState, ask whether the value is derived, owned by the server, or needed in a link; if so, it is in the wrong home"
  - "Debugging 'shows stale data after saving': look for server data copied into useState or into a global store"
  - "Debugging 'the shared link opens the wrong view': navigation state kept in useState instead of the URL"
  - "Edit forms: seed a local draft from server data, key the form by the entity id, save explicitly"
strengths:
  - "Unlike one global store for everything (the Redux reflex), each home brings its own semantics: the query cache refetches and invalidates, the URL survives reloads and Back, useState disappears with its component"
  - "Derived values cannot drift: computing a count during render is cheaper than any synchronization code and correct by construction"
  - "Placement decisions are reviewable as a short checklist, so a team can apply them consistently instead of debating stores"
  - "Keys turn 'reset this state when X changes' into identity, replacing effects that copy props into state"
weaknesses:
  - "Several homes means several APIs: useQuery, useSearchParams, a storage wrapper and useState all live in one component"
  - "URL state is untyped strings: every read needs parsing and a fallback (lesson ts.react.routing)"
  - "localStorage is per device and per origin: a 'saved' preference doesn't follow the user to another machine; that needs the server"
  - "A key that changes too often remounts subtrees and throws away state you wanted, such as scroll position or focus"
pitfalls:
  - title: "Mirroring props or server data in state"
    detail: "useState reads its argument on the first render only. Read the source directly, or derive from it."
    bad: "const [count] = useState(progress.size);"
    good: "const count = progress.size; // derived on every render"
  - title: "Resetting state with an effect when an id changes"
    detail: "The effect runs after a render that already showed the old state. A key resets everything at once, before anything renders."
    bad: "useEffect(() => { setBody(''); setLoaded(false); }, [lessonId]);"
    good: "<LessonNotes key={lesson.id} lessonId={lesson.id} lessonTitle={lesson.title} />"
  - title: "Putting transient input in the URL"
    detail: "Writing every keystroke of a search box with a navigation that pushes history fills Back with keystrokes. Update with replace (setSearchParams(next, { replace: true })) or commit to the URL on submit."
  - title: "Keeping a filter, tab or page number in useState"
    detail: "Reload, Back and shared links lose it. If a user would bookmark it, it belongs in the path or search params."
quiz:
  - q: "A list page keeps its level filter in useState. Users report that links they share always show the unfiltered list. Where should the filter live?"
    options: ["In context, so every page can read it", "In the URL's search params", "In localStorage"]
    answer: 1
    why: "Only the URL travels with a link and survives reload and Back. localStorage would restore the sender's filter on the sender's machine only."
  - q: "`const [title, setTitle] = useState(props.title)`; the parent later passes a different title. What renders?"
    options: ["The new title", "The old title, until setTitle is called", "An error in Strict Mode"]
    answer: 1
    why: "The argument is the initial value and is ignored after the first render. Read props.title directly, or key the component if it holds an editable draft."
  - q: "An edit form for a customer loaded from the server. Where should the user's unsaved changes live?"
    options: ["Written into the server-state cache on every keystroke", "In local state seeded from the server data, with the form keyed by the customer id", "In the URL"]
    answer: 1
    why: "A draft is meant to diverge until it is saved. Writing it into the cache would show unsaved edits everywhere as if they were saved; the key resets the draft when a different customer opens."
  - q: "The sidebar shows 'completed: 12' next to a list of progress rows. Should 12 be stored?"
    options: ["Yes, in its own useState updated after each change", "No, derive it from the rows during render"]
    answer: 1
    why: "A stored count is a second source of truth that drifts when an update path forgets it. Counting is cheap; memoize only if a profile says otherwise."
further_reading:
  - title: "React: Choosing the state structure"
    url: "https://react.dev/learn/choosing-the-state-structure"
  - title: "React: Preserving and resetting state"
    url: "https://react.dev/learn/preserving-and-resetting-state"
  - title: "React: Sharing state between components"
    url: "https://react.dev/learn/sharing-state-between-components"
  - title: "React: useSyncExternalStore"
    url: "https://react.dev/reference/react/useSyncExternalStore"
  - title: "MDN: URLSearchParams"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/URLSearchParams"
---

## Concept

The earlier lessons gave each kind of state its tool: `useState` and `useReducer`, the
TanStack Query cache, the URL, Web Storage. This lesson is about choosing between them.
Ask these questions in order and stop at the first yes:

1. **Can it be computed from other state?** Then it isn't state. Derive it during render
   (`useMemo` only when a profile says so).
2. **Does the server own it?** Then it lives in the **server-state cache** and components
   read it from there. Never copy it into `useState` or a global store; the copy stops
   updating. The one deliberate copy is a **draft** being edited, seeded from the server and
   keyed by the entity's id.
3. **Should a reload, a shared link or Back restore it?** Then it lives in the **URL**: the
   path for which thing, search params for how you are looking at it (filter, sort, page,
   tab).
4. **Should it outlive the tab on this device?** Then **localStorage**, wrapped and
   validated (lesson ts.browser-dom.storage). A preference that should follow the user to
   another machine is server state again.
5. **Otherwise it is UI state**: `useState` in the lowest component that needs it, lifted to
   the nearest common parent when siblings need it, context only for values read widely and
   changed rarely.

Most "state management problems" are a value in two homes (a copy that drifts), in a home
with the wrong lifetime (a filter that dies with its component), or attached to the wrong
identity (state that belongs to lesson A shown on lesson B). React keeps component state
**per position in the tree**; a `key` makes it per identity instead.

## Coming from C/C++/Java/Python/JS

- **C++:** think ownership. Each value has exactly one owner, like a `unique_ptr`; everyone
  else holds a view. `useState(prop)` is a value copy taken once, not a reference.
- **Java (Spring MVC, Swing):** the server-side split is familiar: database, session,
  request parameters, local variables. The client has the same four with different names:
  query cache, localStorage, URL, component state. A Swing model object that every view
  mutates is the global store this checklist avoids.
- **Python/Django:** the database, `request.session` and `request.GET` again; nobody stores a
  query-string filter in the database, and the same instinct applies here.
- **Go:** handlers keep no state between requests: the database owns data, the request
  carries parameters, locals hold the rest. A frontend built the same way is easy to reason
  about.
- **SQL:** a derived value stored as a column is denormalization, and it needs a trigger to
  stay correct. A generated column or a view is the "derive during render" equivalent.
- **JS/Redux:** one store used to hold server data, route, form drafts and UI flags. The split
  gives each its proper semantics (refetch, history, reset by key) instead of reimplementing
  them in reducers.

## Design notes

ADR-0005 makes the same split for this app: server state in TanStack
Query, local UI state in `useState`/`useReducer`, navigation in the URL, including "track,
lesson, example and tab". The code is partway there, and the gaps are instructive:

- Progress and bookmarks live in a hand-written module store (`src/features/learning/store.ts`)
  rather than the query cache; lesson ts.react.query compares the two.
- The selected tab and example are `useState` in `LessonContent`, not URL parameters; lesson
  ts.react.routing covers the fix.
- `LessonNotes` and `LessonQuiz` load per-lesson data into local state with an effect, so
  without a key both would have the window from the second example. In `LessonQuiz` it
  would matter more: the previous lesson's answers would still be in state, and a click in
  that window would save them under the new lesson's id. The lesson page renders both, and
  the lesson's content around them, with `key={lesson.id}`, which closes it.

When a bug report says "sometimes shows old data", check placement before adding effects.
An effect that copies one state into another is almost always a sign that one of them
should not exist. Measure re-render cost before lifting state into context: the narrower
the home, the fewer components a change touches, which is the performance argument from
the routing lesson seen from the other side.
