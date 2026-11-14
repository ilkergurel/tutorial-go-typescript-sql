---
id: ts.browser-dom.dom
title: "Typed DOM queries and events; forms and validation"
track: ts
module: browser-dom
order: 1
level: basic
minutes: 13
since: es2015
prerequisites: [ts.narrowing.basic, ts.everyday-types.null]
compare_with: [ts.react.forms-lists, ts.services.zod, go.http.json-validation]
tags: [dom, events, forms, validation, narrowing]
summary: >
  TypeScript's DOM types come from strings: tag names and event names map to
  precise types. Everything else is Element or EventTarget, which you narrow
  with instanceof, not as. Forms validate natively; you add rules and messages.
examples:
  - id: typed-queries
    file: examples/01_typed_queries.ts
    title: "Typed queries: tag names, instanceof, and the lying type argument"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "8-10"
        note: >
          querySelector has one overload per tag name (HTMLElementTagNameMap in
          lib.dom.d.ts), so "form" returns HTMLFormElement | null and .elements
          is available once null is ruled out.
      - lines: "12-16"
        note: >
          An id or class selector is just a string to tsc, so the result is
          Element | null: two problems, null and a type without value or
          valueAsNumber.
      - lines: "17-19"
        note: >
          instanceof narrows Element to HTMLInputElement with a real runtime
          check. If someone changes the markup, the branch is skipped instead of
          failing somewhere later.
      - lines: "21-23"
        note: >
          The type argument is a cast in disguise: tsc doesn't check it, and
          neither does the browser. #title is an h1, so value is undefined while
          the type says string. `as HTMLInputElement` lies the same way.
      - lines: "25-35"
        note: >
          Pass the constructor and let instanceof do the check: el instanceof
          type narrows to T. The error says what was expected and what was
          found, at startup rather than in a later TypeError.
      - lines: "37-41"
        note: >
          querySelectorAll("li") is NodeListOf<HTMLLIElement>: iterable and
          indexable, with forEach but no map, so spread it into an array.
          toLocaleUpperCase("tr") gives İZMİR, not IZMIR.
  - id: events
    file: examples/02_events.ts
    title: "Event maps, delegation, custom events, and removal by signal"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "10-12"
        note: >
          One guard at the top removes null for the rest of the module: list is
          HTMLUListElement and filter is HTMLInputElement, inside callbacks too.
      - lines: "14-21"
        note: >
          Event delegation: one listener on the list handles clicks on any row,
          including rows added later. e.target is the innermost node clicked and
          is typed EventTarget | null, so narrow it before calling closest().
      - lines: "23-28"
        note: >
          Assigning .value fires no input event; only user actions do. Code and
          tests that set values must dispatch the event themselves, which is what
          testing libraries' user-event helpers do for you.
      - lines: "30-39"
        note: >
          Declaration merging into HTMLElementEventMap types the custom event at
          every addEventListener: e.detail.plate is a string. dispatchEvent is
          not checked against the map, so keep the detail shape in one place.
      - lines: "41-50"
        note: >
          The signal option removes the listener when the controller aborts. One
          controller can remove a whole group of listeners in a cleanup function.
          "keydown" maps to KeyboardEvent, so e.key is typed.
  - id: form-validation
    file: examples/03_form_validation.ts
    title: "Constraint validation, a custom rule, and FormData"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-8"
        note: >
          The rules live in the markup: required, type="email", min and max.
          The browser checks them; no library needed for the basics.
      - lines: "12"
        note: >
          Destructuring the typed NodeList gives three HTMLInputElements. tsc
          can't know how many inputs the markup has, so this is the line that
          breaks silently if the form changes.
      - lines: "14-23"
        note: >
          ValidityState has a flag per rule. validationMessage comes in the
          browser's language, so a Turkish UI on an English browser shows
          English text: map the flags to your own messages.
      - lines: "25-30"
        note: >
          setCustomValidity("message") makes the field invalid; "" makes it valid
          again, and nothing clears it for you. "izmir" upper-cased with the tr
          locale is İZMİR and matches; toUpperCase() would give IZMIR.
      - lines: "32-44"
        note: >
          preventDefault stops the browser from submitting and navigating.
          checkValidity() runs every rule and fires invalid events without
          showing the browser's bubbles (reportValidity() shows them).
          FormData.get returns string | File | null: narrow before use.
      - lines: "46-52"
        note: >
          On a real page, form.requestSubmit() validates and fires submit like a
          click on the submit button. The lesson sandbox lacks allow-forms, so
          the example dispatches the event itself.
use_cases:
  - "Pages and widgets without a framework: admin tools, embeds, browser extensions"
  - "Long or dynamic lists: one delegated listener instead of one per row"
  - "Forms that need required, type, min/max or pattern checks plus a few business rules"
  - "Inside React, refs give you these same typed elements and native APIs"
strengths:
  - "lib.dom.d.ts types the DOM for free: tag names map to element types and event names to event types, so most handlers need no annotations"
  - "instanceof is a checked narrowing; a Java cast is checked at run time too, but TypeScript's as and type arguments are erased and check nothing"
  - "Native constraint validation and FormData handle most forms without a dependency, with messages already localized by the browser"
weaknesses:
  - "Only tag-name selectors are typed; ids, classes and attribute selectors return Element, and the generic overload is an unchecked cast"
  - "Markup and code are checked separately: renaming an id breaks at run time, not at compile time (JSX closes that gap)"
  - "Native validation bubbles are hard to style, and their language is the browser's, not your app's"
pitfalls:
  - title: "Casting query results"
    detail: "as and querySelector<T> silence the compiler and check nothing. Narrow with instanceof and fail with a clear message."
    bad: "const email = document.querySelector(\"#email\") as HTMLInputElement;"
    good: "const email = document.querySelector(\"#email\");\nif (!(email instanceof HTMLInputElement)) throw new Error(\"#email missing\");"
  - title: "Treating e.target as the element with the listener"
    detail: "target is the innermost node that received the event, such as a span inside the button. Use closest() for delegation, or the variable you attached the listener to."
  - title: "Expecting value assignment to fire events"
    detail: "Setting .value or .checked from code fires neither input nor change. Dispatch the event if listeners must react."
  - title: "Forgetting to clear setCustomValidity"
    detail: "A non-empty custom message keeps the field invalid forever. Recompute it on every input and set \"\" when the value is fine."
  - title: "Trusting client-side validation"
    detail: "Anyone can skip it with devtools or curl. It is for the user's convenience; the server validates again."
quiz:
  - q: "What is the type of document.querySelector(\".price\")?"
    options: ["HTMLElement | null", "Element | null", "HTMLInputElement", "any"]
    answer: 1
    why: "Only bare tag names select a typed overload. A class selector returns Element | null."
  - q: "const el = document.querySelector<HTMLInputElement>(\"#title\"), and #title is an h1. What happens?"
    options: ["tsc reports an error", "querySelector throws", "It compiles, and el.value is undefined at run time"]
    answer: 2
    why: "The type argument is unchecked; the browser returns the h1, which has no value property."
  - q: "Code sets input.value = \"x\". Does an input listener run?"
    options: ["Yes, any change fires input", "No; only user-initiated changes fire input, so dispatch the event yourself"]
    answer: 1
    why: "MDN: setting value programmatically does not fire input. Testing tools simulate typing to get real events."
  - q: "How do you make a field fail checkValidity() because of your own rule?"
    options: ["Throw from an input listener", "input.setCustomValidity(\"message\")", "Set input.validity.customError = true"]
    answer: 1
    why: "validity is read-only; setCustomValidity sets the customError flag and the message, and \"\" clears them."
further_reading:
  - title: "TypeScript Handbook: DOM Manipulation"
    url: "https://www.typescriptlang.org/docs/handbook/dom-manipulation.html"
  - title: "MDN: EventTarget.addEventListener()"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/EventTarget/addEventListener"
  - title: "MDN: Client-side form validation"
    url: "https://developer.mozilla.org/en-US/docs/Learn_web_development/Extensions/Forms/Form_validation"
  - title: "MDN: HTMLInputElement.setCustomValidity()"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/HTMLInputElement/setCustomValidity"
---

## Concept

TypeScript ships the DOM's types in `lib.dom.d.ts`, and they are driven by strings. A
tag name passed to `querySelector`, `querySelectorAll`, `createElement` or `closest` picks
a precise element type through `HTMLElementTagNameMap`; an event name passed to
`addEventListener` picks the event type through the event maps (`"keydown"` gives a
`KeyboardEvent`). Anything the compiler cannot see, such as ids, classes or the actual
markup, comes back as `Element` or `EventTarget`. You then **narrow** with `instanceof`, a
real runtime check, instead of asserting with `as`.

Forms add a second built-in system: **constraint validation**. Attributes declare rules,
`validity` reports which ones fail, `setCustomValidity` adds your own, and `FormData`
collects the values as `string | File`.

## Coming from C/C++/Java/Python/JS

- **JS**: the same API. TypeScript adds the `null` checks and the narrowing that
  jQuery-era code skipped.
- **Java/Android**: `findViewById<T>` and `(EditText) view` are checked at run time and
  throw on a mismatch. TypeScript's `as` and `querySelector<T>` are erased and check
  nothing; `instanceof` is the checked form.
- **Go**: server-rendered pages (`html/template`) validate on the server only. Client-side
  validation is a convenience on top; the Go handler must still validate (see the JSON
  validation lesson).
- **Python**: BeautifulSoup's `select()` returns untyped tags. Here the type follows the
  selector when it is a tag name, and stops at `Element` otherwise.

## How it works

An event travels in three phases: **capture** from the window down to the target's
parent, **target**, then **bubble** back up. Listeners run in the bubble phase unless
registered with `{ capture: true }`. Delegation depends on bubbling, so it only works for
events that bubble: `click` and `input` do, `focus` doesn't (use `focusin`), and a
`CustomEvent` only bubbles when created with `bubbles: true`.

`e.target` is where the event started; `e.currentTarget` is the element whose listener is
running. TypeScript types both as `EventTarget | null`, because one listener function can
be attached to anything.

## Design notes

Query and narrow once, at the edge of the program, then pass typed elements inward:
a helper like `must(selector, HTMLInputElement)` turns a markup mistake into one clear
error at startup. The rest of the code never sees `null` or `Element`.

Use native validation for the rules HTML already knows, `setCustomValidity` for business
rules, and your own message table so the language matches your UI rather than the
browser's. Treat it as user experience, never as security: the server repeats every check.
When the UI grows, JSX (the React lessons) closes the remaining gap, because markup and
code are type-checked together.
