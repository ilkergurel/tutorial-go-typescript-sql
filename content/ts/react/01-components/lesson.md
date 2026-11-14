---
id: ts.react.components
title: "Typed components and props: children, ComponentProps"
track: ts
module: react
order: 1
level: intermediate
minutes: 12
since: ts4.1
prerequisites: [ts.everyday-types.objects, ts.narrowing.discriminated-unions]
compare_with: [go.cli.templates, ts.type-level.utility, ts.generics.keyof]
tags: [react, components, props, children, ComponentProps, jsx]
summary: >
  A component is a function of one typed props object, and children is just
  another prop. Extend native elements with ComponentPropsWithoutRef and derive
  prop types from the types you already have instead of copying fields.
examples:
  - id: props-children
    file: examples/01_props_children.tsx
    title: Props, children and native button props
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "7-10"
        note: >
          ComponentPropsWithoutRef<"button"> is the prop type JSX itself uses
          for `<button>`, minus `ref`. The wrapper accepts onClick, disabled,
          aria-* and form without listing them, and stays in sync with
          @types/react. Why not ComponentProps: in React 18 a function component
          never receives `ref` (that needs forwardRef), so a type that allows it
          would accept a ref React then drops.
      - lines: "12-13"
        note: >
          Take what you handle, forward the rest. The default `type="button"`
          matters: a native button's default type is submit, so a reusable
          button placed in a form would otherwise submit it. Default parameters
          replace `defaultProps`, which React 19 removed for function components.
      - lines: "16-21"
        note: >
          children is an ordinary prop with a special JSX syntax. ReactNode is
          everything React can render: strings, numbers, elements, arrays,
          null, undefined and booleans. footer is a second slot: any ReactNode
          prop works like children.
      - lines: "26"
        note: >
          toLocaleUpperCase("tr") maps i to İ: SİPARİŞ. Plain toUpperCase()
          gives SIPARIŞ. For purely visual capitals, CSS `text-transform:
          uppercase` under `lang="tr"` applies the Turkish rule without
          touching the text (browser support varies).
      - lines: "27"
        note: >
          A boolean on the left of &&, so 0 warnings renders nothing. The
          output has no `<strong>`.
      - lines: "45-47"
        note: >
          The last output line is `<p>0</p>`. && returns its left operand when
          it is falsy, and React renders the number 0 (it skips only false,
          true, null and undefined). react.dev lists it as a pitfall of
          conditional rendering.
      - lines: "49-52"
        note: >
          Both mistakes fail at compile time. Without the type, the second one
          is a runtime crash: React throws "Objects are not valid as a React
          child (found: object with keys {id})".
  - id: derive-props
    file: examples/02_derive_props.tsx
    title: Derive props instead of duplicating them
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "5-11"
        note: >
          One declaration of the domain shape, ideally generated from the API
          contract. Everything below derives from it.
      - lines: "13-16"
        note: >
          Pick<User, "name" | "city"> keeps the link to User: rename city to
          town and tsc flags this component. A hand-written
          `{ name: string; city: string }` would compile and drift. Being
          structural, any object with those two fields fits, test data included.
      - lines: "18-28"
        note: >
          ComponentProps<typeof UserBadge> reads a component's props type,
          which also works for library components that don't export theirs.
          The intersection adds one prop; rest destructuring forwards the others.
      - lines: "30-42"
        note: >
          Mutually exclusive props are a discriminated union on kind. Inside
          the if, props.href exists; after it, only onClick does. No optional
          href plus optional onClick plus a runtime "exactly one" check.
      - lines: "50-51"
        note: >
          onClick never shows in the HTML: event handlers are attached by React
          on the client, not serialized as attributes.
      - lines: "56-58"
        note: >
          Mixing the variants is a compile error that names the offending prop.
use_cases:
  - "Design-system primitives (Button, Input, Link) that wrap a native element and forward everything else with `...rest`"
  - "Layout components with slots: `children` plus named ReactNode props such as `footer`, `actions`, `aside`"
  - "Components fed by API types: `Pick<User, ...>` or `Omit<User, ...>` so a contract change surfaces in every consumer"
  - "Components with mutually exclusive modes (link vs button, controlled vs uncontrolled) as discriminated-union props"
strengths:
  - "Unlike PropTypes (runtime warnings, ignored since React 19), prop types are checked at every call site at compile time, and JSX is just a typed function call"
  - "ComponentPropsWithoutRef<\"button\"> gives a wrapper the full native API in one line, so aria-* and data-* attributes keep working without a whitelist"
  - "Derived types (Pick, ComponentProps<typeof X>) turn a renamed field into compile errors at every use, where copied fields in Java DTOs drift silently"
  - "Composition through children and slot props replaces the inheritance hierarchies of Swing or Android views"
weaknesses:
  - "ReactNode is broad: you can't express 'only `<li>` children' in the type system (react.dev says so), so structural rules stay conventions"
  - "Error messages for union props and intersections with IntrinsicAttributes are long; the useful line is usually the last one"
  - "The ref story differs by version: React 18 needs forwardRef, React 19 makes ref a regular prop, and types written for one are subtly wrong for the other"
pitfalls:
  - title: "Numbers on the left of &&"
    detail: "`count && <Badge />` renders 0 when count is 0. Make the left side a boolean."
    bad: "{items.length && <List items={items} />}"
    good: "{items.length > 0 && <List items={items} />}"
  - title: "ComponentProps<\"button\"> on a React 18 function component"
    detail: "The type includes `ref`, so `<Button ref={r}>` compiles, but React 18 drops the ref with a 'Function components cannot be given refs' warning and r.current stays null. Use ComponentPropsWithoutRef, or forwardRef when callers need the DOM node."
    bad: "function Button(props: ComponentProps<\"button\">) {\n  return <button {...props} />;\n}"
    good: "const Button = forwardRef<\n  HTMLButtonElement,\n  ComponentPropsWithoutRef<\"button\">\n>(function Button(props, ref) {\n  return <button ref={ref} {...props} />;\n});"
  - title: "React.FC and implicit children"
    detail: "In @types/react 18 (this app's version), `FC<P>` does not add children. `<Title title=\"x\">text</Title>` is a compile error unless P declares `children: ReactNode`. Plain functions with a typed props parameter are the simpler default."
  - title: "Copying fields into props interfaces"
    detail: "A props interface that repeats `name: string; city: string` from a domain type will drift. Pick, Omit and ComponentProps<typeof X> keep the dependency visible to tsc."
quiz:
  - q: "React 18. `function Button(props: ComponentProps<\"button\">)` is rendered as `<Button ref={r} />`. What happens?"
    options: ["tsc accepts it and r.current is the button", "tsc accepts it, React drops the ref and warns", "tsc rejects it"]
    answer: 1
    why: "ComponentProps<\"button\"> includes ref, but a React 18 function component only gets a ref through forwardRef. ComponentPropsWithoutRef makes the same line a compile error."
  - q: "`{items.length && <List items={items} />}` with an empty array renders..."
    options: ["nothing", "0", "false"]
    answer: 1
    why: "&& returns its falsy left operand, 0, and React renders numbers. false, null, undefined and true render nothing."
  - q: "Which type accepts a string, a number, an element, an array of elements and null as children?"
    options: ["ReactElement", "ReactNode", "string | JSX.Element"]
    answer: 1
    why: "ReactNode is the union of everything renderable; ReactElement is JSX elements only."
  - q: "Why `Pick<User, \"name\" | \"city\">` instead of `{ name: string; city: string }`?"
    options: ["It is faster at runtime", "A change to User reaches the component and tsc flags what breaks", "React requires prop types to be named"]
    answer: 1
    why: "Types are erased, so there is no runtime difference. The derived type keeps the dependency on User visible to the compiler."
further_reading:
  - title: "React: Using TypeScript"
    url: "https://react.dev/learn/typescript"
  - title: "React: Passing props to a component"
    url: "https://react.dev/learn/passing-props-to-a-component"
  - title: "React: Conditional rendering"
    url: "https://react.dev/learn/conditional-rendering"
  - title: "React: forwardRef"
    url: "https://react.dev/reference/react/forwardRef"
  - title: "TypeScript Handbook: JSX"
    url: "https://www.typescriptlang.org/docs/handbook/jsx.html"
  - title: "TypeScript: Utility types (Pick, Omit)"
    url: "https://www.typescriptlang.org/docs/handbook/utility-types.html"
---

## Concept

A React component is a function that takes **one** argument, the props object, and returns
something React can render. Typing a component means typing that object: an `interface`
or `type`, destructured in the parameter list. JSX is a function call in disguise
(`<Card title="x">` compiles to `jsx(Card, { title: "x" })` with `"jsx": "react-jsx"`), so
the compiler checks every prop at every call site, including `children`, which is just a
prop that JSX fills from between the tags.

Three types do most of the work:

- `ReactNode` for anything renderable: `children`, and named slots like `footer`.
- `ComponentPropsWithoutRef<"button">` (and the other intrinsic element names) for components
  that wrap a native element and forward the rest with `...rest`.
- `ComponentProps<typeof SomeComponent>` to reuse the props of a component you already
  have, including library components that don't export a props type.

The goal is that props are **derived**: from domain types with `Pick`/`Omit`, from other
components with `ComponentProps`, from native elements with the intrinsic element types.
A field is declared once; everything else references it.

The examples use React 18.3, this app's version. React 19 changes two things here: `ref`
becomes a regular prop for function components (no `forwardRef`), and `defaultProps` and
`propTypes` for function components are gone.

## Coming from C/C++/Java/Python/JS

- **Java (Swing, Android views):** no class hierarchy, no `extends JButton`. A component is
  a function re-run with new props; `interface ButtonProps extends
  ComponentPropsWithoutRef<"button">` extends a *type*, not the DOM element. Customization
  is composition: pass elements in through `children` and slot props.
- **C++:** think of props as a `const` parameter struct. The component must not modify it;
  React relies on that when it decides what to re-render.
- **Python:** like a function with keyword-only arguments typed by a `TypedDict`;
  `children` is one more keyword that the JSX syntax fills in for you.
- **JS/React:** you know the runtime. What's new is that a missing or misspelled prop is a
  compile error instead of a PropTypes console warning, and that "objects are not valid as
  a React child" is caught before the page loads.
- **Go:** `html/template` checks fields when the template executes; a typed component is
  checked when it is compiled. Go's habit of small interfaces declared by the consumer is
  the same idea as `Pick<User, "name" | "city">`: ask for exactly what you use.

## Design notes

Props are the public API of a component, so treat them like a function signature: as
narrow as the job, derived from the source of truth, plain data and callbacks rather than
service objects. This app's `src/shared/ui/LevelBadge.tsx` is a small example: its prop is
`{ level: Level }`, where `Level` is a union the badge declares itself (the curriculum's
identical one is assignable), and its style table is a `Record<Level, string>`, so a
fourth level is a compile error at every call site until the badge handles it.
`src/features/runner/RunPanel.tsx` types its props inline
(`{ example: RunnableExample; onRan: () => void }`), which is fine for a component nobody
wraps; the example's shape is a named, exported interface because the lesson page builds
one from a lesson.

For wrappers of native elements, forwarding `...rest` beats whitelisting props: a
whitelist silently drops `aria-describedby`, `id` or `form` and breaks accessibility in
ways no test of the happy path notices. Pick `ComponentPropsWithoutRef` on React 18 and add
`forwardRef` only when callers need the DOM node (focus management, measuring). Generic
components (`List<T>` with a typed render function) are the next step; they are covered in
the generics module.
