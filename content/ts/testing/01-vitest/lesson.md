---
id: ts.testing.vitest
title: "Vitest: it.each tables, Testing Library (behavior, not implementation)"
track: ts
module: testing
order: 1
level: intermediate
minutes: 14
since: ts4.1
prerequisites: [ts.react.forms-lists, ts.async.promises]
compare_with: [go.testing.table-driven, ts.testing.msw-playwright, ts.react.state]
tags: [testing, vitest, testing-library, user-event, it.each, jsdom, react]
summary: >
  Vitest runs tests the way Vite runs code, and it.each turns a table of cases
  into one named test per row, the TS spelling of Go's table-driven tests.
  Testing Library drives components through roles, labels and clicks, so the
  tests survive a rewrite of the component's internals.
examples:
  - id: each
    file: examples/01_each.test.ts
    title: "it.each: a table of Turkish edge cases, one named test per row"
    runtime: ts-test
    check: contains
    annotations:
      - lines: "3-14"
        note: >
          The code under test. NFD splits ç, ğ, ş, ö and ü into a base letter
          plus a combining mark, and \p{M} removes the marks. Dotless ı has no
          decomposition, so it is mapped to i by hand. Lower-casing with "tr"
          turns I into ı and İ into i; plain toLowerCase() would turn İ into
          "i" plus U+0307, which only works here because marks are stripped
          afterwards.
      - lines: "20-30"
        note: >
          it.each takes the table and returns a test function. Each row
          becomes its own test, and $title and $want write the row into the
          name, so a failure points at the row. The row type is inferred
          from the table, so a misspelled field in the callback is a type
          error in tsc (Vitest itself doesn't check types).
      - lines: "24"
        note: >
          This row is the reason the table exists. Remove the replaceAll("ı",
          "i") line and it fails with "gd-r-kalesi": ı is not in [a-z], so it
          becomes a separator and the leading I-turned-ı is cut off. ASCII-only
          test data would never notice.
      - lines: "32-41"
        note: >
          Array rows are spread into the callback's parameters, and %s
          formats them into the name in order. toLocaleUpperCase("tr") turns
          i into İ and ı into I; toUpperCase() would give "Istanbul".
  - id: testing-library
    file: examples/02_testing_library.test.tsx
    title: "Testing Library: find by role and label, act like a user, wait for async UI"
    runtime: ts-test
    check: contains
    annotations:
      - lines: "8-28"
        note: >
          The component under test. It lives in the test file only because
          the runner takes one file; in a project, CityForm.tsx and
          CityForm.test.tsx sit side by side.
      - lines: "30"
        note: >
          Testing Library unmounts after each test by itself only when the
          runner provides a global afterEach. Vitest's globals option is off
          by default, so without this line the second test finds two forms
          and fails with: Found multiple elements with the role "button" and
          name "Save".
      - lines: "34-37"
        note: >
          Create the user before rendering. user.click fires the whole event
          sequence a real click produces (pointer, mouse, focus, click), and
          clicking a submit button submits the form. getByRole with a name
          finds the element the way a screen reader announces it; that is the
          top of Testing Library's query priority.
      - lines: "38-39"
        note: >
          getBy throws when nothing matches, and its error prints the DOM. The
          onSave prop is the component's boundary, so checking it with vi.fn()
          tests behavior, not internals.
      - lines: "44-45"
        note: >
          Promise.withResolvers (ES2024) gives the test a promise it settles
          when it wants. Until then the component sits in its "saving" state,
          which the test can observe.
      - lines: "48-51"
        note: >
          getByLabelText finds the input through its label. user.type types
          character by character, so React's onChange runs per keystroke as in
          a browser. The saved value is trimmed and upper-cased the Turkish
          way: İZMİR, not IZMIR.
      - lines: "52-54"
        note: >
          The state update happens after the promise resolves, so a plain
          getBy here would race it. findBy retries until the element appears
          (default 1000 ms), and Testing Library takes care of act() while
          it waits.
          queryBy returns null instead of throwing, the way to assert that
          something is absent.
  - id: refactor
    file: examples/03_refactor.test.tsx
    title: "describe.each over two implementations: tests that survive a refactor"
    runtime: ts-test
    check: contains
    annotations:
      - lines: "14-33"
        note: >
          Version 1 keeps the draft in state; the input is controlled and
          renders that state on every keystroke.
      - lines: "35-55"
        note: >
          Version 2 is a different design: the list lives in a reducer, and
          the input is uncontrolled. The DOM holds the draft and the component
          reads it through a ref on submit.
      - lines: "59-63"
        note: >
          describe.each works like it.each, one level up: the whole suite runs
          once per row, and %s puts the version into every test name.
      - lines: "64-73"
        note: >
          The test types, presses Enter ({Enter} submits the form), clicks
          Add, and reads list items and the field's value. It never touches
          state, hooks, refs or class names, which is why it passes for both
          versions. A test that spied on setDraft or called the reducer would
          have to be rewritten for the refactor.
use_cases:
  - "Pure functions with many edge cases (parsers, formatters, slug and case rules for Turkish text): one it.each table, one row per case"
  - "Component tests for forms, lists, dialogs and error states, written in terms of what the user sees and does"
  - "Pinning behavior before a refactor: the same suite run against the old and new implementation with describe.each"
  - "Fast feedback in watch mode while editing, with the same transform pipeline as the Vite dev server"
strengths:
  - "it.each gives Go-style table tests with less ceremony: each row is a named test, and a failing row doesn't stop the others"
  - "Queries by role and label double as an accessibility check: a button without an accessible name cannot be found by getByRole(\"button\", { name })"
  - "Tests written against roles and text survive refactors (useState to useReducer, controlled to uncontrolled), unlike tests that inspect state or CSS classes"
  - "Vitest uses Vite's transform, so ESM, TypeScript and JSX need no Babel or ts-jest setup"
weaknesses:
  - "jsdom is not a browser: no layout (sizes are zero), no real navigation or CSS media queries, and user-event only simulates input. Some behavior needs a real browser (Playwright, next lesson)"
  - "Vitest strips types like Vite and never type-checks a runtime test; unlike `go test`, which compiles the package first, a type error in a test file passes unless tsc runs too"
  - "Async UI needs findBy or waitFor at the right places; a getBy that races a state update gives flaky failures"
  - "More setup than Go's built-in testing package: an environment (jsdom), user-event, and cleanup when globals are off"
pitfalls:
  - title: "No cleanup between tests"
    detail: "With Vitest's default globals: false, Testing Library can't register its afterEach, so every render stays in the document and queries start finding duplicates. Add afterEach(cleanup), or enable globals in the Vitest config."
  - title: "Testing implementation details"
    detail: "Selecting by class name, counting renders or asserting on internal state ties the test to today's code. Ask what the user perceives: a role, a label, a text."
    bad: "expect(container.querySelector(\".error\")).not.toBeNull();"
    good: "expect(screen.getByRole(\"alert\").textContent).toBe(\"Enter a city\");"
  - title: "getBy to assert absence"
    detail: "getBy throws when nothing matches, so `expect(getBy...).toBeNull()` can never pass. Use queryBy for absence and findBy for elements that appear later."
    bad: "expect(screen.getByRole(\"alert\")).toBeNull();"
    good: "expect(screen.queryByRole(\"alert\")).toBeNull();"
  - title: "fireEvent where a user would type"
    detail: "fireEvent.change sets a value and fires one event. userEvent.type focuses, types each key and fires the events in a browser's order, so it catches handlers that a single change event skips. Prefer user-event; keep fireEvent for events users can't produce."
  - title: "Trusting jsdom with browser behavior"
    detail: "In this setup (jsdom 31, user-event 14.6) typing after form.reset() continued from the old text, although the field's value was already empty. In a browser the field is empty. When a test depends on real browser behavior (layout, focus order, form reset, navigation), test it in Playwright."
quiz:
  - q: "Which query does Testing Library recommend trying first?"
    options:
      - "getByTestId"
      - "getByRole, with the accessible name"
      - "container.querySelector"
      - "getByText"
    answer: 1
    why: "Roles and accessible names are what users and assistive technology perceive; test ids are the last resort because users can't see them."
  - q: "You want to assert that no error message is shown. Which query?"
    options:
      - "getByRole(\"alert\")"
      - "queryByRole(\"alert\")"
      - "findByRole(\"alert\")"
    answer: 1
    why: "queryBy returns null when nothing matches. getBy throws, and findBy waits and then rejects."
  - q: "A status message appears after an awaited save. `expect(screen.getByRole(\"status\"))` right after the click fails now and then. The fix?"
    options:
      - "await screen.findByRole(\"status\")"
      - "Add a setTimeout of 100 ms before the assertion"
      - "Use fireEvent.click instead of user.click"
    answer: 0
    why: "findBy retries until the element appears or the timeout passes; fixed sleeps are slow and still flaky."
  - q: "Why use it.each over a loop of expect calls inside one test?"
    options:
      - "It runs faster"
      - "Each row is reported as its own named test, and one failing row doesn't hide the others"
      - "Loops are not allowed in Vitest tests"
    answer: 1
    why: "Like t.Run per row in Go, each row has its own name and result, and you can run one row with a name filter (-t)."
further_reading:
  - title: "MDN: WAI-ARIA roles"
    url: "https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Reference/Roles"
  - title: "React: act()"
    url: "https://react.dev/reference/react/act"
  - title: "MDN: String.prototype.toLocaleUpperCase()"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/String/toLocaleUpperCase"
  - title: "Go wiki: table-driven tests"
    url: "https://go.dev/wiki/TableDrivenTests"
---

## Concept

**Vitest** runs test files through Vite's transform: TypeScript and JSX are stripped
and compiled, ESM works as it does in the app, and `expect` follows Jest's API.
`it.each(table)` (also spelled `test.each`) turns a table into one test per row, and
`describe.each` does the same for a whole suite. The [Vitest API
docs](https://vitest.dev/api/) list the name placeholders: `$field` for object rows,
`%s`, `%i`, `%#` for array rows.

**Testing Library** renders a component into jsdom and gives you queries that find
elements the way a user does: by role and accessible name (`getByRole`), by label
(`getByLabelText`), by visible text. **user-event** produces the input a user would
(click, type, Enter). The [guiding
principle](https://testing-library.com/docs/guiding-principles): the more your tests
resemble the way your software is used, the more confidence they give you. Query
variants encode timing:

- `getBy…` returns the element or throws (zero or several matches);
- `queryBy…` returns `null` when nothing matches: use it to assert absence;
- `findBy…` returns a promise that retries until the element appears (1000 ms by
  default): use it after anything asynchronous.

## Coming from C/C++/Java/Python/JS

- Go: `it.each` is the table-driven test. `for _, tc := range tests { t.Run(tc.name,
  ...) }` becomes a table plus a name template, and `go test -run 'TestSlug/Iğdır'` is
  `vitest -t "Iğdır"`. Unlike `go test`, Vitest never compiles your types: a type
  error in a test file still runs.
- Java: JUnit 5's `@ParameterizedTest` with `@MethodSource`. Testing Library has no
  close JUnit analogue; Selenium-style page objects are the nearest relative, built
  on roles instead of CSS selectors.
- Python: `pytest.mark.parametrize` for tables; Testing Library plays the role of a
  user-level driver that pytest's unit tests usually lack.
- JS: Jest's API almost one to one (`vi.fn` for `jest.fn`). Enzyme's shallow
  rendering and `wrapper.state()` are the implementation-detail style that Testing
  Library deliberately leaves out.

## How it works

`render()` mounts the component into `document.body` of a jsdom window that Vitest
creates per test file (the `environment: 'jsdom'` setting). Testing Library runs
`render` and user-event actions inside React's `act()`, and its async helpers
(`findBy`, `waitFor`) take care of act while they wait, so state updates and effects
are flushed before your next line runs. It also sets `IS_REACT_ACT_ENVIRONMENT`,
which is why these tests print no act() warnings. Queries
read the DOM through the same accessibility rules a browser uses: a `<label>` names
its input, `role="alert"` is an alert, a `<button>`'s text is its name. A component
that is hard to query is usually hard to use with a screen reader too.

## Design notes

Test at the component's boundary. That boundary is its props and callbacks on one
side and the rendered DOM on the other, so assert on what the user sees and on what
the component reports outward (`onSave` called with "İZMİR"), never on how it got
there. Example 3 is the payoff: the state model and the input style both changed, and
the suite didn't. Keep pure logic (slugs, validation, Turkish case rules) out of
components and cover it with `it.each` tables; they are cheap to extend when a bug
report brings a new edge case. Component tests then only need to check the wiring.
