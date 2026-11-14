---
id: ts.react.routing
title: "Routing and URL state; the rendering model and memoization; composition"
track: ts
module: react
order: 6
level: advanced
minutes: 15
since: ts4.1
prerequisites: [ts.react.state, ts.browser-dom.storage]
compare_with: [go.http.handlers, ts.architecture.state, go.performance.pprof]
tags: [react, react-router, url-state, search-params, rendering, memo, useCallback, composition]
summary: >
  Put navigation state (which page, which filter) in the URL so back, reload and
  shared links work. A component re-renders when its parent does; memo,
  useCallback and passing children skip that work, and the first two are worth
  it only for a problem you have measured.
examples:
  - id: url-state
    file: examples/01_url_state.tsx
    title: "The URL owns the filter: it survives navigating away and back"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "18-23"
        note: >
          Two kinds of state side by side: level lives in the URL's search
          params, sorted in the component. params.get returns string | null:
          the URL is user input, so treat it like untrusted data (validate,
          default) before relying on its type.
      - lines: "23"
        note: >
          filter returned a new array, so sorting it in place is safe. The
          Turkish collator gives defer, Sözlükler, Şablonlar. A plain sort()
          compares UTF-16 code units and gives Sözlükler, defer, Şablonlar:
          capitals before lowercase, Ş after every ASCII letter.
      - lines: "26"
        note: >
          setSearchParams navigates: it pushes a history entry with the new
          query string (pass { replace: true } to replace it instead), and every
          component reading the params re-renders.
      - lines: "28-32"
        note: >
          Link renders a real `<a href>`, so open-in-new-tab and copy-link work;
          a plain click is intercepted and becomes a client-side navigation.
      - lines: "37-41"
        note: >
          useParams gives the :id segment as string | undefined. navigate(-1) is
          the history back button.
      - lines: "50-62"
        note: >
          MemoryRouter keeps the history in memory: right for tests and for this
          iframe, which has no address bar. The app itself uses
          createBrowserRouter in src/main.tsx. useTransitions={false} is for the
          printout only: by default React Router applies navigations inside
          startTransition, which renders later than flushSync returns.
      - lines: "64-74"
        note: >
          The last line is the point: after going to a lesson and back, the
          level filter is still applied, because it is part of the history
          entry. The sort is gone: LessonList unmounted on navigation and its
          useState died with it.
  - id: render-memo
    file: examples/02_render_memo.tsx
    title: "What re-renders, and the two ways to skip it"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "6-12"
        note: >
          Counting renders by mutating a Map during render breaks render purity
          on purpose, as instrumentation. Under StrictMode, development builds
          render each component twice and these counts would double.
      - lines: "13"
        note: >
          memo(Chart) compares each prop with Object.is against the previous
          render and skips the component when all are equal. It is a
          performance hint, not a guarantee: correct code must not depend on it.
      - lines: "15-28"
        note: >
          Each keystroke re-renders Search and every child it creates: plain
          renders 4 times (mount plus 3). memo skips when the props are equal
          (1 render). An inline arrow is a new function each render, so memo
          sees unequal props and renders anyway (4). useCallback returns the
          same function while its dependencies don't change (1).
      - lines: "30-40"
        note: >
          Composition instead of memo. Pinning re-renders Panel and what Panel
          creates ("created inside": 4), but not its children: those elements
          were created by the parent, which didn't re-render, so React reuses
          them ("passed as children": 1). react.dev lists this as the first way
          to make memoization unnecessary.
      - lines: "54-64"
        note: >
          Each press is its own flushSync, so each is one commit. The table is
          the measurement; in an app the React DevTools Profiler or the
          `<Profiler>` component gives timings, and only an interaction that
          feels slow justifies adding memo.
use_cases:
  - "URL state: filters, sort order, pagination, selected tab, search query, the open item: anything a user would bookmark, share or expect Back to restore"
  - "Component state: hover, open menus, draft input not yet applied, animation state"
  - "memo/useCallback/useMemo: a measured slow interaction where an expensive child re-renders with equal props, or a value used as an effect dependency"
  - "Composition (children, slot props): stateful wrappers such as panels, modals and layouts around content that shouldn't re-render with them"
strengths:
  - "URL state is free persistence: reload, back/forward, deep links and shared links all work, with no storage code"
  - "Typed route params and search params as string | undefined and string | null force the parsing that server handlers in Go also need"
  - "React's model is simple to predict: a component renders when its own state changes (store subscriptions included), its parent renders, or a context it reads changes"
  - "Composition fixes most unnecessary renders structurally, without the dependency bookkeeping memo and useCallback need"
weaknesses:
  - "Search params are strings only: numbers, booleans and arrays need parsing and validation on every read"
  - "memo is defeated by any new object, array or function prop, so it tends to pull useMemo and useCallback through the whole call chain"
  - "Memoization costs memory and comparisons on every render; applied everywhere it can make things slower and the code harder to read"
  - "React Router's own API changed between v5, v6 and v7; much code found online targets older versions"
pitfalls:
  - title: "Mirroring URL state in useState"
    detail: "Copying a search param into state means two sources of truth, and Back updates one of them. Read the param where you render and write it with setSearchParams."
    bad: "const [level, setLevel] = useState(params.get(\"level\"));"
    good: "const level = params.get(\"level\"); // and setParams({ level }) to change it"
  - title: "Trusting the URL's type"
    detail: "`params.get(\"page\") as number` compiles and lies: it is a string or null. Parse and validate, with a default."
    bad: "const page = params.get(\"page\") as unknown as number;"
    good: "const page = Math.max(1, Number(params.get(\"page\") ?? \"1\") || 1);"
  - title: "memo with inline object or function props"
    detail: "`<MemoChart options={{ dense: true }} onSelect={() => pick(id)} />` creates new props every render, so memo never skips. Hoist constants, use useMemo/useCallback, or drop memo."
  - title: "Memoizing by default"
    detail: "react.dev: memo is only valuable when a component re-renders often with the same props and its rendering is expensive. Measure with the Profiler first; React Compiler, where adopted, applies memoization automatically."
quiz:
  - q: "A list filter is kept in useState in the list component. The user opens an item and presses Back. What does the list show?"
    options: ["The filtered list", "The unfiltered list", "It depends on the browser"]
    answer: 1
    why: "Navigating away unmounted the list and its state. A filter in the search params is part of the history entry and comes back with it."
  - q: "A parent re-renders. Its child receives no props at all and is not wrapped in memo. Does the child render?"
    options: ["No, its props didn't change", "Yes, rendering a parent renders the children it creates", "Only in development"]
    answer: 1
    why: "Without memo React doesn't compare props; a parent's render re-renders its subtree. The DOM is only touched where output differs."
  - q: "memo(Chart) still renders on every keystroke. The parent passes onSelect={() => select(id)}. Why?"
    options: ["memo only works on class components", "The arrow is a new function each render, so the props are never equal", "memo needs a key"]
    answer: 1
    why: "memo compares each prop with Object.is. Wrap the callback in useCallback, or don't rely on memo there."
  - q: "A Panel with its own state wraps expensive content. Which change stops the content re-rendering when Panel's state changes, without memo?"
    options: ["Move the content into Panel's own JSX", "Pass the content as children from Panel's parent", "Store Panel's state in a ref"]
    answer: 1
    why: "Elements passed as children are created by the parent; when only Panel re-renders, those elements are the same objects and React skips them."
further_reading:
  - title: "React: Render and commit"
    url: "https://react.dev/learn/render-and-commit"
  - title: "React: memo"
    url: "https://react.dev/reference/react/memo"
  - title: "React: useCallback"
    url: "https://react.dev/reference/react/useCallback"
  - title: "React: Profiler"
    url: "https://react.dev/reference/react/Profiler"
  - title: "React Compiler"
    url: "https://react.dev/learn/react-compiler"
  - title: "MDN: URLSearchParams"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/URLSearchParams"
  - title: "Go: net/http ServeMux patterns"
    url: "https://pkg.go.dev/net/http#ServeMux"
---

## Concept

**Routing** maps the URL to components. With React Router v7 (`react-router-dom`, this
app's version) you declare routes (`path="/lessons/:id"`), read the matched segment with
`useParams`, read and write the query string with `useSearchParams`, and navigate with
`<Link>` or `useNavigate`. The rule that makes routing pay off: **the URL owns navigation
state**. Which page, which item, which filter, which sort, which tab: if a user would
bookmark it, share it or expect Back to restore it, it belongs in the path or the search
params, not in `useState`. Component state is for what's transient: an open menu, a draft.

The **rendering model** decides how much work each change costs. A component renders when
its state changes, when a context it reads changes, or when its parent renders; React then
compares the output and touches only the DOM that differs. A parent's render re-renders the
children it creates, whether or not their props changed. Two tools skip that:

- **`memo(Component)`** skips rendering when every prop is `Object.is`-equal to last time,
  which makes `useCallback`/`useMemo` necessary for function and object props.
- **Composition**: a stateful wrapper that receives `children` from its parent re-renders
  without re-rendering those children, because they are elements the parent already created.

Composition is free and structural; memoization costs comparisons and code. Reach for memo
when a profile shows a slow interaction, not by default. React Compiler (stable, and
documented to support React 17 to 19) inserts this memoization automatically at build time.
React Router's own documentation, including the v7 modes (declarative, data, framework), is
at [reactrouter.com](https://reactrouter.com/home).

## Coming from C/C++/Java/Python/JS

- **Go:** `mux.HandleFunc("GET /lessons/{id}", h)` with `r.PathValue("id")` (Go 1.22+) is the
  server-side twin of `path="/lessons/:id"` with `useParams()`. Both hand you strings, and
  `r.URL.Query().Get("level")` returns `""` for a missing parameter where
  `searchParams.get("level")` returns `null`.
- **Java (Swing/Android):** navigation is not a stack of view objects you push; it is a URL,
  and the view is a function of it. Android's deep links are the closest analogue.
- **Python (Django/Flask):** the URL conf moves to the client; the server serves one HTML
  page and an API.
- **C++:** memo is an output cache keyed by shallow equality of the inputs; like any cache it
  helps only when hits are common and recomputation is expensive.
- **JS:** URLSearchParams percent-encodes UTF-8, so `?q=şablon` travels as
  `?q=%C5%9Fablon`; `get("q")` decodes it back. Never build query strings by concatenation.

## Design notes

This app follows the URL-first rule for the big things: `src/main.tsx` maps
`/tracks/:trackId` and `/lessons/:lessonId` with `createBrowserRouter`, and its comment
cites ADR-0005. The same comment says `?example=` and tab params would come next; in
`src/features/lessons/LessonContent.tsx` the selected tab and example are still `useState`,
so a reload or a shared link always opens the Concept tab and the first example. Moving
them to `useSearchParams` is the change this lesson argues for, and it costs a few lines of
parsing (the param can be any string, so fall back to the defaults).

For performance, think in this order: keep state as low as possible (a keystroke in a
search box should re-render the search box, not the page), pass slow content as children,
derive instead of syncing with effects, and only then memoize the specific component the
Profiler points at. When you do memoize, check what the value feeds: a `useMemo` whose
result is a dependency of a `useCallback` or an effect keeps those stable, which is a
correctness concern as much as a speed one.
