---
id: ts.react.state
title: "useState and useReducer with discriminated unions"
track: ts
module: react
order: 2
level: intermediate
minutes: 14
since: ts4.1
prerequisites: [ts.react.components, ts.narrowing.discriminated-unions]
compare_with: [ts.architecture.explicit-states, design.paradigms.immutability, sql.transactions.acid-mvcc]
tags: [react, useState, useReducer, state-machine, discriminated-unions, immutability]
summary: >
  State is a snapshot per render that you replace, never mutate. Model it as one
  discriminated union instead of independent flags, and move the transitions into
  a reducer when they need rules: impossible states then fail to compile.
examples:
  - id: snapshot
    file: examples/01_snapshot.tsx
    title: "State is a snapshot: updater functions, and why mutation is invisible"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "7-8"
        note: >
          useState infers number from 0. For an empty array it would infer
          never[], so the type argument string[] is needed there; with a union
          state, write the union as the type argument.
      - lines: "10-14"
        note: >
          count is a constant of this render: the handler closed over 0. All
          three calls request "set to 1", so the first output line says count=1.
          Calling a setter never changes the variable in running code.
      - lines: "15-19"
        note: >
          An updater function receives the pending state, so the three updates
          chain: count=4. Use the updater form whenever the next state is
          computed from the previous one, and always in async code.
      - lines: "20-23"
        note: >
          push changes the array React already holds, then setTags passes the
          same reference. Object.is sees no change and React skips the
          re-render: the "mutate" line shows tags=go.
      - lines: "24-26"
        note: >
          A new array re-renders, and the "sql" pushed earlier appears too: the
          mutation was real, only invisible. Mutation bugs surface later, in an
          unrelated update, which is what makes them expensive to find.
      - lines: "41-46"
        note: >
          The harness: flushSync applies each click's updates before the next
          line reads the DOM. In application code you don't need it; React
          batches all updates of one event and re-renders once afterwards.
  - id: union-state
    file: examples/02_union_state.tsx
    title: One union-typed state instead of independent flags
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "6-12"
        note: >
          A boolean and two optionals allow eight combinations, most of them
          nonsense. The union allows exactly four, and each one carries only
          the data that exists in it.
      - lines: "16-25"
        note: >
          The guard reads this render's snapshot. The second click of a double
          click runs the handler of the re-rendered component, which sees
          "sending" and returns, so the server gets one request, not two.
      - lines: "27-36"
        note: >
          Rendering switches on the tag. Inside case "failed" TypeScript knows
          state.error is a string; in any other case accessing it is a compile
          error, so no `error && ...` checks are needed.
      - lines: "47-53"
        note: >
          Harness plumbing. The server answers asynchronously, so show() waits
          until the expected text is on screen, like Testing Library's
          findByText, instead of guessing how many event-loop turns React
          needs.
      - lines: "55-62"
        note: >
          Three clicks, two requests: the double click was absorbed while
          sending, the retry after the failure was allowed.
  - id: reducer
    file: examples/03_reducer.tsx
    title: "useReducer: every transition in one pure, testable function"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "6-15"
        note: >
          State and actions are both discriminated unions. attempt lives only in
          the variants where it means something, and travels from failed to the
          next sending.
      - lines: "17-32"
        note: >
          Each case first checks that the transition is legal from the current
          state, otherwise it returns the same object. The declared return type
          Submit doubles as an exhaustiveness check: add an action variant
          without a case and tsc reports TS2366 "Function lacks ending return
          statement".
      - lines: "26-30"
        note: >
          After the guard, s is the sending variant and a is the succeed
          action: orderNo comes from the action because the state doesn't have
          one yet, and the compiler knows it.
      - lines: "34-49"
        note: >
          No React needed to test the machine: fold a script of actions through
          the function. Returning the same object for an ignored action matters
          in React too: Object.is equality means no re-render.
      - lines: "51-63"
        note: >
          useReducer infers state and action types from the reducer. dispatch
          has a stable identity, and a malformed action is a compile error.
          Two clicks still show "sending": the reducer, not the button,
          enforces the rule.
use_cases:
  - "Request lifecycles (editing, sending, failed, done) where each phase carries different data"
  - "Wizards and multi-step flows: the step and the data collected so far in one value"
  - "Components whose handlers all need the same rules (no submit while sending, no retry after success): a reducer keeps those rules in one place"
  - "Logic you want to unit-test without rendering: a reducer is a pure function of (state, action)"
strengths:
  - "Unlike a bag of booleans, a union makes 'sending with an error' unrepresentable, and narrowing gives each branch exactly its fields"
  - "A reducer is a pure function: test it with plain calls, no DOM, no mocks, unlike a Java controller whose state lives in fields mutated by handlers"
  - "Replacing state instead of mutating it makes change detection a reference comparison (Object.is), which is what lets React skip work cheaply"
  - "dispatch has a stable identity, so passing it down or listing it in effect dependencies costs nothing"
weaknesses:
  - "Updating nested immutable data by hand is verbose (spreads several levels deep); flatten the state or use a helper library if it hurts"
  - "A reducer adds indirection: for one or two independent values useState reads better"
  - "The snapshot model surprises newcomers: logging state right after setState prints the old value"
  - "Strict Mode calls reducers and updater functions twice in development, so any side effect inside them shows up doubled"
pitfalls:
  - title: "Mutating state and passing the same reference"
    detail: "React compares with Object.is and skips the update. Create a new array or object."
    bad: "tags.push(\"ts\");\nsetTags(tags);"
    good: "setTags([...tags, \"ts\"]);"
  - title: "Reading state right after setting it"
    detail: "The variable belongs to the current render. Compute the next value into a local and use that."
    bad: "setCount(count + 1);\nsave(count); // old value"
    good: "const next = count + 1;\nsetCount(next);\nsave(next);"
  - title: "Boolean flags that can contradict each other"
    detail: "isLoading, isError and data as three useState calls allow states your UI can't render sensibly. One union-typed state cannot contradict itself."
    bad: "const [loading, setLoading] = useState(false);\nconst [error, setError] = useState<string>();\nconst [order, setOrder] = useState<string>();"
    good: "const [state, setState] = useState<Submit>({ status: \"editing\" });"
  - title: "Side effects in reducers or updater functions"
    detail: "They must be pure. Strict Mode calls them twice in development and keeps one result, so a request sent from a reducer goes out twice. Do I/O in the event handler, then dispatch the outcome."
quiz:
  - q: "count is 5. A click handler calls setCount(count + 1) twice. What does the next render show?"
    options: ["5", "6", "7"]
    answer: 1
    why: "Both calls compute 5 + 1 from the same snapshot. setCount(c => c + 1) twice would give 7."
  - q: "A handler does items.push(x); setItems(items). What happens?"
    options: ["React re-renders with the new item", "React skips the re-render because the reference is the same", "React throws an error"]
    answer: 1
    why: "Object.is(items, items) is true, so React treats it as no change. The pushed item appears only when something else re-renders."
  - q: "Why does the reducer return the same state object for an illegal transition instead of a copy?"
    options: ["Copies are slower to create", "The same reference tells React nothing changed, so it can skip the re-render", "TypeScript requires it"]
    answer: 1
    why: "A copy would be a new reference and trigger a render with identical content. Returning state unchanged is the reducer's 'no-op'."
  - q: "A reducer is typed (s: Submit, a: Action): Submit and switches on a.type with a return in every case. You add { type: \"cancel\" } to Action. What happens?"
    options: ["Nothing until a cancel action is dispatched", "tsc reports that the function lacks an ending return statement", "The reducer returns undefined silently in production only"]
    answer: 1
    why: "The unhandled variant makes the end of the function reachable, and undefined isn't a Submit. A `const x: never = a` default branch gives the same guarantee with a clearer message."
further_reading:
  - title: "React: useState"
    url: "https://react.dev/reference/react/useState"
  - title: "React: useReducer"
    url: "https://react.dev/reference/react/useReducer"
  - title: "React: State as a snapshot"
    url: "https://react.dev/learn/state-as-a-snapshot"
  - title: "React: Choosing the state structure"
    url: "https://react.dev/learn/choosing-the-state-structure"
  - title: "React: Extracting state logic into a reducer"
    url: "https://react.dev/learn/extracting-state-logic-into-a-reducer"
  - title: "TypeScript Handbook: Narrowing (discriminated unions)"
    url: "https://www.typescriptlang.org/docs/handbook/2/narrowing.html"
---

## Concept

`useState` gives a component a value that survives re-renders and a setter that
**requests** a new render with a new value. Three rules follow from that:

- The value is a **snapshot**: every render has its own constant `count`, and handlers created
  in that render see that constant forever. Calling the setter doesn't change it.
- Setters can take an **updater function** (`c => c + 1`); React queues them and applies them
  in order to the pending state.
- React decides whether anything changed with `Object.is`. **Replace** objects and arrays;
  mutating them and passing the same reference is ignored.

For anything with phases, type the state as **one discriminated union** rather than several
independent `useState` calls. `{ status: "failed"; error: string }` cannot also be
"sending", and only the failed branch can read `error`. When the transitions start to have
rules ("send is allowed from editing or failed, nothing else"), move them into a **reducer**:
`useReducer(reducer, initial)` keeps the state, and `reducer(state, action)` is a pure
function that returns the next state, or the same object when the action doesn't apply.
Actions are a discriminated union too, so a misspelled or incomplete action doesn't compile.

## Coming from C/C++/Java/Python/JS

- **Java/C++:** there are no fields to assign. State lives in React, the component reads a
  copy per render. Think of an immutable value object you replace wholesale, not a bean you
  call setters on.
- **C:** the reducer is the classic `switch (state) { case ...: }` state machine, with the
  tagged union checked by the compiler instead of by convention.
- **Python:** like a frozen dataclass you rebuild with `dataclasses.replace`; mutating it in
  place is the bug.
- **JS/Redux:** a `useReducer` reducer is a Redux reducer scoped to one component, without the
  store, middleware or action-creator boilerplate. The typed action union replaces string
  constants.
- **PostgreSQL:** the snapshot idea is close to MVCC: a handler sees the state of the render
  it came from, as a REPEATABLE READ transaction sees its snapshot, and a new render is the
  next snapshot.

## Design notes

Ask "which combinations of my state variables are legal?" If the answer is "fewer than all
of them", the variables belong in one union. Derive everything else during render instead
of storing it: this app's `LessonQuiz` (`src/features/learning/LessonQuiz.tsx`) stores only
the chosen answers and computes the score on every render, so the two can never disagree.
The same file keeps `answers` and a separate `loaded` flag; a union such as
`{ phase: "loading" } | { phase: "ready"; answers: Answers }` would make "answers shown
before they loaded" unrepresentable.

`useState` versus `useReducer` is not about size but about where the rules live. Keep
`useState` while every update is "set this value". Switch to a reducer once several handlers
must agree on what is allowed; the reducer is then the single place to read the state
machine and the obvious unit to test. Keep I/O out of it: the handler calls the server and
dispatches what happened. Server data itself (lists, caches, refetching) is a different
kind of state with a better tool, covered in the TanStack Query lesson.
