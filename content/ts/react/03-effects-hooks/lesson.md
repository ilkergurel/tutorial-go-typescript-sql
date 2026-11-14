---
id: ts.react.effects-hooks
title: "Effects and cleanup (and when you don't need an effect); custom hooks; context"
track: ts
module: react
order: 3
level: intermediate
minutes: 15
since: ts4.1
prerequisites: [ts.react.state, ts.async.cancellation]
compare_with: [go.concurrency.context, go.basics.defer, ts.async.cancellation]
tags: [react, useEffect, cleanup, strict-mode, custom-hooks, context, useSyncExternalStore]
summary: >
  An effect keeps a component synchronized with something outside React, and its
  cleanup undoes exactly what its setup did. Values computed from props belong in
  render, not in an effect. Custom hooks share logic; context injects values down
  the tree.
examples:
  - id: effect-lifecycle
    file: examples/01_effect_lifecycle.tsx
    title: "Setup, cleanup and Strict Mode's extra cycle"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "6-10"
        note: >
          The external system: a socket, a timer, a map widget, a subscription.
          Effects exist for exactly this; if nothing outside React is involved,
          you probably don't need one.
      - lines: "12-18"
        note: >
          Setup connects, the returned function disconnects. The dependency
          array [room] means "re-synchronize when room changes": React compares
          each dependency with Object.is and skips the effect otherwise, which
          is why the second render of #genel logs nothing.
      - lines: "22-29"
        note: >
          StrictMode is on, as in this app's src/main.tsx. In development React
          runs one extra setup and cleanup cycle before the real setup, to prove
          the cleanup mirrors the setup: connect, close, connect. Production
          connects once.
      - lines: "32-39"
        note: >
          On a dependency change the old cleanup runs first, with the old room,
          then the new setup. Unmount runs the last cleanup. The order is the
          contract: never two connections at once. In this runner the effects of
          a flushSync render run before it returns, which keeps the log in order;
          in a browser an effect not caused by an interaction generally runs
          after paint.
  - id: without-effect
    file: examples/02_without_effect.tsx
    title: "You might not need an effect: derive during render"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "6-8"
        note: >
          toLocaleUpperCase("tr") makes "ilkay ışık" İI. toUpperCase() would
          give II, a different person's initials in Turkish.
      - lines: "10-16"
        note: >
          State that copies props, synchronized by an effect. The output shows
          the cost: a committed render with an empty label, then a second
          render. The DOM briefly shows a value that was never true.
      - lines: "18-23"
        note: >
          The same value computed during render: one render, correct the first
          time, and it can't go stale when props change. If it were expensive,
          useMemo would cache it; an effect never would be the answer.
      - lines: "29-35"
        note: >
          Harness: flushSync commits synchronously, then one timer turn lets
          the update scheduled by the effect render. Printing before and after
          makes the extra render visible.
  - id: context-hooks
    file: examples/03_context_hooks.tsx
    title: Context with a guard hook, and custom hooks built on it
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "7-8"
        note: >
          A locale has no sensible default, so the context type says
          Locale | null and the default is null. A fake default like "en-US"
          would hide a missing provider until a Turkish user saw English
          sorting.
      - lines: "10-16"
        note: >
          The pattern react.dev recommends: one hook reads the context, throws
          without a provider and returns the narrowed type, so no consumer
          handles null.
      - lines: "18-27"
        note: >
          Custom hooks are functions whose names start with use and that call
          other hooks; useSorted qualifies only because it calls useLocale. They
          run on every render as part of the calling component, and each call
          has its own state: they share logic, not state.
      - lines: "19-21"
        note: >
          [...items] copies before sort, which sorts in place: never mutate data
          you received. Intl.Collator("tr-TR") puts Ç after C, Ö after O and Ş
          after S; English collation treats them as accented C, O and S.
      - lines: "35-42"
        note: >
          The same component under two providers: tr-TR gives Ceyhan, Çanakkale
          and 1.249,90; en-US gives Çanakkale, Ceyhan and 1,249.90. In React 19
          you may write `<LocaleContext value={locale}>` instead of
          `.Provider`; React 18 needs `.Provider`.
      - lines: "44-48"
        note: >
          Without a provider the guard throws during render, with a message
          that names the fix, instead of rendering wrong data.
use_cases:
  - "Effects: WebSocket or EventSource connections, timers, third-party widgets (maps, charts, editors), browser APIs such as IntersectionObserver"
  - "No effect: derived values, filtering and sorting for display, resetting state when an id changes (use a key), reacting to a click (do it in the handler)"
  - "Custom hooks: a named piece of stateful logic used by several components, such as useOnlineStatus, useLocale or useDebouncedValue"
  - "Context: values many components need and that change rarely: locale, theme, the current user, a QueryClient"
strengths:
  - "Cleanup lives next to setup in one closure, like Go's defer after acquiring a resource, instead of split across componentDidMount and componentWillUnmount"
  - "Strict Mode's development-only remount finds a missing cleanup on the first run, not in production after a route change"
  - "Custom hooks compose by plain function calls, with no mixins or higher-order component wrappers"
  - "Context removes prop drilling for cross-cutting values without a global mutable singleton, and a test can wrap a component in any provider it likes"
weaknesses:
  - "Effects are easy to overuse: an effect that only sets state from props costs an extra render and can show stale values"
  - "Dependency arrays compare by reference, so objects and functions created during render re-run the effect every time"
  - "A context value change re-renders every consumer, and memo doesn't stop it; frequently changing values belong elsewhere"
  - "Fetching in effects needs hand-written race handling, caching and retries; a data library or framework does it better"
pitfalls:
  - title: "Fetching in an effect without ignoring stale responses"
    detail: "When the input changes quickly, responses can arrive out of order and the last one to arrive wins, even if it is for an old input. Flag the old request in cleanup. Better still, let TanStack Query handle it."
    bad: "useEffect(() => {\n  fetchBio(person).then(setBio);\n}, [person]);"
    good: "useEffect(() => {\n  let ignore = false;\n  fetchBio(person).then((b) => {\n    if (!ignore) setBio(b);\n  });\n  return () => {\n    ignore = true;\n  };\n}, [person]);"
  - title: "Copying props into state with an effect"
    detail: "One render with the stale value, then another. Compute it during render; reset state on identity change with a key."
    bad: "const [full, setFull] = useState(\"\");\nuseEffect(() => setFull(`${first} ${last}`), [first, last]);"
    good: "const full = `${first} ${last}`;"
  - title: "Subscribing to an external store with useEffect and useState"
    detail: "It works, but an update between render and subscription can be missed, and every component repeats the code. React 18's useSyncExternalStore is the purpose-built hook."
    bad: "const [, force] = useState(0);\nuseEffect(() => store.subscribe(() => force((n) => n + 1)), []);\nconst value = store.get();"
    good: "const value = useSyncExternalStore(store.subscribe, store.get);"
  - title: "A new context value object on every render"
    detail: "value={{ user, login }} is a new object each time the provider renders, so every consumer re-renders. Wrap it in useMemo (and login in useCallback) when the provider re-renders often."
quiz:
  - q: "An effect with dependency [room] logged connect a. The prop changes to b. What runs, in order?"
    options: ["connect b", "close a, then connect b", "connect b, then close a"]
    answer: 1
    why: "React runs the previous cleanup with the old values before the next setup, so there is never a moment with two connections."
  - q: "In development with StrictMode, a component with a connecting effect mounts. What does the log show?"
    options: ["connect", "connect, close, connect", "connect, connect"]
    answer: 1
    why: "React 18 runs one extra setup and cleanup cycle in development to check that cleanup mirrors setup. Production shows a single connect."
  - q: "fullName is first + ' ' + last, both props. Where should it be computed?"
    options: ["In a useEffect that sets fullName state", "During render, as a plain const", "In a context provider"]
    answer: 1
    why: "Derived values belong in render. An effect would commit one render with a stale value and then render again."
  - q: "Why createContext<Locale | null>(null) plus a hook that throws, rather than createContext<Locale>(\"en-US\")?"
    options: ["null is faster", "A missing provider fails loudly instead of silently using the wrong locale", "TypeScript doesn't allow string defaults"]
    answer: 1
    why: "The default is used only when no provider is above. A plausible default hides the mistake; the guard hook turns it into an immediate error and removes null from consumers' types."
further_reading:
  - title: "React: useEffect"
    url: "https://react.dev/reference/react/useEffect"
  - title: "React: Synchronizing with effects"
    url: "https://react.dev/learn/synchronizing-with-effects"
  - title: "React: You might not need an effect"
    url: "https://react.dev/learn/you-might-not-need-an-effect"
  - title: "React: Reusing logic with custom hooks"
    url: "https://react.dev/learn/reusing-logic-with-custom-hooks"
  - title: "React: useContext"
    url: "https://react.dev/reference/react/useContext"
  - title: "React: useSyncExternalStore"
    url: "https://react.dev/reference/react/useSyncExternalStore"
  - title: "MDN: Intl.Collator"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Intl/Collator"
---

## Concept

Rendering must be pure: given props and state, return UI and touch nothing else. **Effects**
are the escape hatch for code that must run *because the component is on screen*:
connecting to a chat room, starting a timer, attaching a third-party widget.
`useEffect(setup, deps)` runs `setup` after React commits; if `setup` returns a function,
that **cleanup** runs before the next setup (when a dependency changed, compared with
`Object.is`) and once more on unmount. In development, `StrictMode` adds one extra
setup-cleanup cycle on mount to prove the cleanup undoes the setup.

Most code people put in effects doesn't need one. react.dev names the two big cases:
**transforming data for rendering** (compute it during render) and **handling user events**
(do it in the event handler, where you know what happened). Resetting state when an id
changes is a `key`, and subscribing to an external store is `useSyncExternalStore`.

A **custom hook** is a function named `useSomething` that calls other hooks. It shares
stateful *logic*; every call gets its own state. **Context** passes a value to every
component below a provider without threading props through each level: create it with
`createContext`, provide it with `<Ctx.Provider value={...}>` (React 18; React 19 also
accepts `<Ctx value={...}>`), and read it with `useContext`, ideally through one custom
hook that handles the missing-provider case.

## Coming from C/C++/Java/Python/JS

- **Go:** React context is **not** `context.Context`. Go's context carries cancellation,
  deadlines and request values explicitly as a first parameter; React context is implicit
  dependency injection through the component tree. The cleanup function is the closer
  analogy to Go: it is your `defer conn.Close()`, and the `ignore` flag in a fetching effect
  is checking `ctx.Err()` after a call returns.
- **Java/Android:** an effect is not `onCreate`/`onDestroy`. Think "keep this in sync with
  these values": it re-runs when the dependencies change, not only at mount and unmount.
  Context plays the role of a DI container scoped to a subtree, without annotations or
  reflection.
- **C++:** setup/cleanup is RAII expressed as a closure pair; the dependency array decides
  when the old object is destroyed and a new one is constructed.
- **Python:** a custom hook is like a context manager or decorator you compose freely, except
  that it must be called unconditionally, at the top level of a component or another hook.
- **JS (class components):** `componentDidMount`/`componentDidUpdate`/`componentWillUnmount`
  collapse into one effect per concern, so the setup and teardown of one subscription
  stay together.

## Design notes

Treat every `useEffect` in review as a question: which external system does it synchronize
with? If the answer is "React state", delete it and compute during render or in the handler.
Fetching is the common exception that still deserves a library: an effect-based fetch needs
race handling, caching, deduplication and retries, which is the next-but-one lesson.

This app has a real example of each side. `LessonQuiz` (`src/features/learning/LessonQuiz.tsx`)
loads answers in an effect with a `live` flag set to false in cleanup, the `ignore` pattern
above. `src/features/learning/store.ts` subscribes components to a small module-level store
with `useState` plus `useEffect` (`useProgressStatus`, `useIsBookmarked`). That is the shape
react.dev suggests replacing with `useSyncExternalStore`, calling it less error-prone; one
concrete gap is that the effect subscribes after the commit, so a store change that lands
between render and subscription is not seen until the next change.
`RunPanel` (`src/features/runner/RunPanel.tsx`) used to reset its editor state in an effect
when the example changed, while its parent already gave it a `key` per example, so the
effect only repeated what remounting did. It was removed: with the key, the initial
`useState` values are the reset.

Keep context for values that are read widely and change rarely (locale, theme, the
QueryClient). A value that changes on every keystroke re-renders every consumer; give such
state a narrower home.
