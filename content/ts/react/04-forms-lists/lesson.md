---
id: ts.react.forms-lists
title: "Controlled forms, lists and keys, accessibility basics"
track: ts
module: react
order: 4
level: intermediate
minutes: 14
since: ts4.1
prerequisites: [ts.react.state, ts.browser-dom.dom]
compare_with: [ts.browser-dom.dom, ts.services.zod, go.http.json-validation]
tags: [react, forms, controlled-inputs, keys, lists, accessibility, useId, aria]
summary: >
  A controlled input renders React state and reports every keystroke through
  onChange; the form validates on submit. useId wires labels and error messages
  for assistive technology, and list keys come from the data, never from the index.
examples:
  - id: controlled-form
    file: examples/01_controlled_form.tsx
    title: A controlled input with typed events and validation on submit
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "10-12"
        note: >
          ChangeEvent<HTMLInputElement> types e.target as an input, so .value is
          a string without a cast. onChange fires on every keystroke, like the
          DOM input event, not like the DOM change event, which waits for blur.
          Store exactly e.target.value: react.dev warns that storing anything
          else, or storing it asynchronously, makes the caret jump.
      - lines: "13-18"
        note: >
          FormEvent<HTMLFormElement> for the submit handler. preventDefault
          stops the browser's navigation; the data stays in state. Trimming and
          Turkish upper-casing happen once, here, not in onChange: "izmir" is
          saved as İZMİR, where toUpperCase() would give IZMIR.
      - lines: "20-26"
        note: >
          Wrapping the input in its label associates them without ids. The
          error paragraph has role="alert", so screen readers announce it when
          it appears. type="submit" states the default explicitly, so Enter in
          the field submits the form.
      - lines: "36-45"
        note: >
          Test plumbing. Assigning input.value directly does not reach React's
          onChange; setting it through the prototype's setter and firing a
          bubbling input event is what a typing user (and Testing Library)
          produces.
      - lines: "46-53"
        note: >
          This runner's iframe is sandboxed without allow-forms, and a browser
          then skips form submission entirely, submit event included. So the
          harness dispatches the event React listens to. In a normal page,
          clicking the button or pressing Enter does it.
      - lines: "55-58"
        note: >
          The empty submit shows the error; the input keeps exactly what was
          typed, spaces included; the second submit saves the normalized value
          and clears the error.
  - id: accessible-field
    file: examples/02_accessible_field.tsx
    title: An accessible Field component with useId
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "6-9"
        note: >
          The native input's props plus a label and an optional error: callers
          keep name, type, defaultValue and every aria-* attribute.
      - lines: "11-27"
        note: >
          useId returns an id unique to this component instance and stable
          across renders, and it matches between server and client rendering,
          which a module-level counter can't promise. Derive related ids from
          it (`${id}-error`) rather than calling useId twice.
      - lines: "20-21"
        note: >
          aria-invalid only once validation found a problem (MDN: not on empty
          fields before the user tried to submit). aria-describedby makes the
          message part of the input's description, read when it gets focus.
          ARIA's dedicated aria-errormessage is the alternative MDN points to.
      - lines: "40-48"
        note: >
          input.labels is the browser's own label association, so this checks
          the wiring the way assistive technology sees it. The ids :r0: and
          :r1: are React 18's format; treat them as opaque.
  - id: keys
    file: examples/03_keys.tsx
    title: "Keys: an index is not an identity"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "6-12"
        note: >
          Turkish plate codes make natural keys: unique and stable. Database
          ids work the same way; for items created in the browser, generate an
          id when the item is created (crypto.randomUUID()), never while
          rendering.
      - lines: "14-23"
        note: >
          Each Row holds state React keeps per position in the tree. The key
          decides which state belongs to which row after the list changes.
      - lines: "29-33"
        note: >
          Removing the first city shifts every index. With key={i}, React sees
          rows 0 and 1 survive and row 2 disappear, so Ankara's note moves to
          İzmir: the first output line. With key={city.plate}, Ankara keeps its
          row and its note. No key at all behaves like the index, plus a
          console warning.
      - lines: "41-55"
        note: >
          The harness types into Ankara's input, removes İstanbul and prints
          every row with its input's value, once per key strategy.
use_cases:
  - "Controlled inputs: live validation, inputs that depend on each other, formatting previews, disabling submit until valid"
  - "Uncontrolled inputs (defaultValue + FormData on submit): simple forms where nothing reacts to each keystroke"
  - "A shared Field component that owns label, id and error wiring for every form in the app"
  - "Keys from data ids: tables, todo lists, chat messages, anything that can be inserted, removed or re-sorted"
strengths:
  - "The input shows exactly the state, so validation, formatting and conditional fields are plain functions of state, not DOM reads"
  - "Typed events (ChangeEvent<HTMLInputElement>, FormEvent<HTMLFormElement>) give e.target the right element type, where plain DOM handlers in TypeScript need a cast"
  - "useId produces ids that survive server rendering and hydration, unlike Math.random() or a global counter"
  - "Keys let React move existing DOM nodes and state instead of recreating them, so focus, scroll position and typed text survive a re-sort"
weaknesses:
  - "Every keystroke re-renders the form component; a large form with expensive children may need splitting or uncontrolled inputs"
  - "Validation, touched-state and error wiring are hand-written in React 18; libraries or React 19's form actions take some of it over"
  - "Accessibility is not enforced by the types: a label-less input compiles fine"
pitfalls:
  - title: "Index as key in a list that changes"
    detail: "Insertions, deletions and re-sorts attach state and DOM to the wrong items. Use an id from the data."
    bad: "{items.map((item, i) => <Row key={i} item={item} />)}"
    good: "{items.map((item) => <Row key={item.id} item={item} />)}"
  - title: "Generating keys while rendering"
    detail: "key={Math.random()} or key={crypto.randomUUID()} in render gives every row a new identity each time: all rows remount and typed input is lost. Create ids when the data is created."
  - title: "value without onChange, or value that becomes undefined"
    detail: "value without onChange renders a read-only field (React warns). Switching value from undefined to a string flips the input from uncontrolled to controlled (another warning). Initialize controlled values to \"\" and use value={x ?? \"\"}."
  - title: "Transforming the value inside onChange"
    detail: "Upper-casing or formatting in onChange stores something other than what the DOM holds; react.dev's troubleshooting lists it as a cause of the caret jumping. Normalize on submit or blur."
    bad: "onChange={(e) => setCity(e.target.value.toLocaleUpperCase(\"tr\"))}"
    good: "onChange={(e) => setCity(e.target.value)}\n// on submit: city.trim().toLocaleUpperCase(\"tr\")"
  - title: "A button without a type inside a form"
    detail: "A button's default type is submit, so an 'Add row' button in a form submits it. Give non-submit buttons type=\"button\"."
quiz:
  - q: "React's onChange on a text input fires..."
    options: ["When the input loses focus, like the DOM change event", "On every edit, like the DOM input event", "Only on form submit"]
    answer: 1
    why: "react.dev: it fires immediately when the value is changed by the user and behaves like the browser input event."
  - q: "A list keyed by index has a text input in every row. You delete the first row. What happens to the text typed into the second row?"
    options: ["It stays with the second item", "It appears next to the item that used to be third", "It is cleared"]
    answer: 1
    why: "The rows at indexes 0 and 1 survive with their state and the last row is removed. The item that was third now renders at index 1 and inherits the second row's input."
  - q: "Why derive the error message id from useId instead of hard-coding id=\"city-error\"?"
    options: ["Hard-coded ids are invalid HTML", "Two instances of the component would produce duplicate ids", "useId ids are shorter"]
    answer: 1
    why: "ids must be unique in the document; useId is unique per component instance and also matches between server and client rendering."
  - q: "Where should a form trim and upper-case a city name?"
    options: ["In onChange, so the input shows the normalized value", "On submit (or blur), leaving the input exactly as typed", "In a useEffect watching the value"]
    answer: 1
    why: "A controlled input must store e.target.value; changing it in onChange can make the caret jump. An effect would add a render for no reason."
further_reading:
  - title: "React: <input>"
    url: "https://react.dev/reference/react-dom/components/input"
  - title: "React: <form>"
    url: "https://react.dev/reference/react-dom/components/form"
  - title: "React: Rendering lists (keys)"
    url: "https://react.dev/learn/rendering-lists"
  - title: "React: useId"
    url: "https://react.dev/reference/react/useId"
  - title: "MDN: aria-invalid"
    url: "https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Reference/Attributes/aria-invalid"
  - title: "MDN: <label>"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/label"
---

## Concept

An input is **controlled** when React decides what it shows: `value={city}` plus an
`onChange` that stores `e.target.value`. Every keystroke goes through state, so the form can
validate, format previews or enable buttons from plain state. An input is **uncontrolled**
when the DOM keeps the value (`defaultValue`) and you read it on submit with
`new FormData(e.currentTarget)`; that's simpler when nothing reacts to each keystroke.
Either way the form's `onSubmit` handler calls `e.preventDefault()` and owns what happens next.

Lists are rendered with `map`, and each item gets a **key**: a string or number, unique
among its siblings and stable over time. React matches elements between renders by key,
so the key decides which component state and which DOM node belong to which item after an
insert, delete or re-sort. Keys come from the data (a database id, a plate code), not from
the array index and never from `Math.random()` during render.

Accessibility basics are part of "a form that behaves": every input has a label (wrap it,
or `htmlFor` + `id`), ids come from `useId` so two instances never collide, and a failed
validation sets `aria-invalid` and points `aria-describedby` at the message.

React 19 adds form **actions** (`<form action={fn}>`, `useActionState`, `useFormStatus`).
This app runs React 18.3, where `onSubmit` is the API; the concepts above carry over.

## Coming from C/C++/Java/Python/JS

- **JS (DOM):** React's `onChange` is the DOM `input` event, not the DOM `change` event that
  fires on blur. In TypeScript, React's event types (`ChangeEvent<HTMLInputElement>`) also
  save you the `e.target as HTMLInputElement` cast that plain DOM handlers need.
- **Java (Swing/JavaFX):** a controlled input is a text field bound one way to a model,
  with every edit routed through your handler; there is no two-way binding magic.
- **Python (Django forms, WTForms):** validation is a function from the submitted values to
  an errors object, as here; the difference is that it runs in the browser first and
  must be repeated on the server anyway.
- **Go:** the server still validates (`go.http.json-validation`); client validation is a
  convenience, never a security boundary. A shared schema such as zod on the client and
  the same rules on the server keeps the two honest.

## Design notes

Put the wiring in one component. A `Field` that takes `label` and `error` and forwards the
native input props makes the accessible version the easy version: nobody hand-writes
`aria-describedby` in each form, and changing how errors are announced is one edit. Keep
validation a pure function of the form values, called on submit (and later on blur if you
want earlier feedback); it is then trivially testable and reusable with a schema library.
MDN's advice on `aria-invalid` matches that timing: don't flag empty required fields before
the user has tried to submit.

For lists, decide identity when data is created, not when it is rendered. Server data
already has ids; client-created items get one at creation (`crypto.randomUUID()`), and
that id is the key. If a list never changes order and items have no state, index keys
happen to work, but nothing in the code says so, which is why reviewers flag them.
