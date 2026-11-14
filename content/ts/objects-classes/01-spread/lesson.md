---
id: ts.objects-classes.spread
title: "Spread/rest and immutability"
track: ts
module: objects-classes
order: 1
level: basic
minutes: 12
since: es2023
prerequisites: [ts.everyday-types.objects]
compare_with: [go.composite.structs, design.paradigms.immutability, ts.react.state]
tags: [spread, rest, immutability, structuredClone, shallow-copy]
summary: >
  Spread copies one level, so a functional update copies every object on the path to
  the change and shares the rest. structuredClone deep-copies data; no copy, deep or
  shallow, carries a class's behaviour.
examples:
  - id: functional-update
    file: examples/01_functional_update.ts
    title: Shallow copies and the nested functional update
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "21-29"
        note: >
          { ...c } is a new object, but its address property holds the same
          object as the caller's, so writing through copy.address changes the
          original and both lines say İzmir. Declaring Address's fields readonly
          turns line 24 into error TS2540, a cheap guard for data you mean to
          update functionally.
      - lines: "31-38"
        note: >
          The functional update copies the customer and the address, the two
          objects on the path to the change, and shares everything else:
          d.tags is still c.tags. That structural sharing keeps updates cheap
          and is exactly what React's Object.is comparisons rely on. A changed
          branch gets a new reference, an unchanged one keeps its identity.
      - lines: "40-43"
        note: >
          Spread defines properties left to right, so the last one wins. Put
          defaults first. A key that is present with the value undefined also
          wins; lesson ts.mental-model.tsconfig shows how
          exactOptionalPropertyTypes catches that.
      - lines: "45-48"
        note: >
          Spread, filter and map already return new arrays. ES2023 added copying
          versions of the mutating methods: with(i, v) for arr[i] = v, plus
          toSorted, toReversed and toSpliced. The last value printed shows the
          original untouched.
  - id: rest-and-iterables
    file: examples/02_rest_and_iterables.ts
    title: Rest to drop keys, spreading class instances and iterables
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "9-17"
        note: >
          Rest collects everything not named, and tsc computes its type: the
          object type minus passwordHash, with no Omit to maintain. It is still
          a deny-list. A field added to User later, say totpSecret, flows into
          rest silently (see the pitfalls).
      - lines: "19-37"
        note: >
          Object spread copies own enumerable properties. The getter and the
          method live on Price.prototype, so the copy has only amount and is not
          a Price. tsc models this correctly: the spread type has no methods and
          no getters, so the call on line 36 does not compile.
      - lines: "39-44"
        note: >
          Array spread accepts any iterable. [...new Set(xs)] removes duplicates
          and keeps first-occurrence order; new Map([...m, entry]) is the
          copy-with-change for Maps. Object spread of a Map gives {}, because its
          entries are not properties.
      - lines: "46-50"
        note: >
          .length counts UTF-16 code units (the emoji is a surrogate pair, 2
          units), spread iterates code points (3). Neither counts what a reader
          sees: "İ".toLowerCase() without a locale is i plus U+0307 COMBINING DOT
          ABOVE, two code points for one letter. The "tr" locale gives the plain
          i. Intl.Segmenter counts user-perceived characters.
  - id: deep-copy
    file: examples/03_deep_copy.ts
    title: structuredClone vs a JSON round trip
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "11-16"
        note: >
          Three copies of one order, then one mutation of the original. The
          shallow copy shares the lines array and sees qty 99; the structured
          clone has its own and still says 2.
      - lines: "18-21"
        note: >
          A JSON round trip keeps only what JSON can express: the Date becomes an
          ISO string, the Map becomes {}, and the undefined property disappears.
          structuredClone keeps Date, Map, Set, RegExp, ArrayBuffer and typed
          arrays, Error types, undefined values and even cycles.
      - lines: "23-28"
        note: >
          JSON.parse returns any, so viaJson.createdAt.getTime() type-checks and
          then throws. A deep copy through JSON loses both the runtime types and
          the static ones.
      - lines: "30-46"
        note: >
          structuredClone does not walk the prototype chain and does not copy
          getters, setters or #private fields, so a class instance comes back as
          a plain object. Its declared signature is <T>(value: T) => T, which is
          why cloned is typed Money and format() compiles, then throws.
      - lines: "48-53"
        note: >
          Functions (and DOM nodes) cannot be cloned at all: structuredClone
          throws a DOMException named DataCloneError. Data with callbacks needs
          a hand-written copy.
use_cases:
  - "State updates in reducers and React components: { ...state, user: { ...state.user, name } }, and items.map((i) => (i.id === id ? { ...i, done: true } : i))"
  - "Merging options: { ...defaults, ...fromUser }"
  - "Removing a field for display or logging: const { passwordHash, ...rest } = user (for API responses, see the deny-list pitfall)"
  - "Snapshots of plain data: undo history, a draft copy of a form, test fixtures you are about to mutate: structuredClone"
  - "Deduplicating while keeping order: [...new Set(values)]"
strengths:
  - "A copy-with-change is one expression, and structural sharing makes it cheap: unchanged branches keep their identity. Java records have no built-in copy-with-change; you call the constructor with every field again"
  - "tsc computes the types: a spread of two objects gives the merged type, and a rest pattern gives the object type minus the named keys"
  - "structuredClone is built into browsers and Node and handles Date, Map, Set, typed arrays and cycles, which a JSON round trip mangles; Python needs copy.deepcopy, Go a hand-written copy"
weaknesses:
  - "Spread is shallow: each nested level needs its own spread, which gets noisy past two or three levels (normalized state or a library such as Immer are the usual answers)"
  - "Every update copies the whole object or array at that level; spreading an accumulator in a loop is quadratic"
  - "No copy carries behaviour: class instances become plain objects, and structuredClone's signature claims they don't"
  - "Immutability is a convention: readonly is compile-time and shallow, so a stray write through an alias still mutates the shared original"
pitfalls:
  - title: "Mutating after a shallow copy"
    detail: "The copy's nested objects are the original's. Copy each level you change, and declare the nested fields readonly so the compiler rejects the write."
    bad: "const next = { ...state }; next.user.name = name;"
    good: "const next = { ...state, user: { ...state.user, name } };"
  - title: "Rest as a deny-list for API responses"
    detail: "Removing secrets with rest leaks every field added to the type later. For data leaving the process, list the fields you send."
    bad: "const { passwordHash, ...body } = user; return Response.json(body);"
    good: "return Response.json({ id: user.id, email: user.email });"
  - title: "Spreading the accumulator in reduce"
    detail: "Each iteration copies everything collected so far: O(n²) for what should be O(n). Build the object once."
    bad: "rows.reduce((acc, r) => ({ ...acc, [r.id]: r }), {})"
    good: "Object.fromEntries(rows.map((r) => [r.id, r]))"
  - title: "JSON round trip as a deep copy"
    detail: "Dates become strings, Map and Set become {}, undefined properties vanish, and the result is typed any. Use structuredClone for data; for class instances, give the class its own copy method."
    bad: "const copy = JSON.parse(JSON.stringify(order));"
    good: "const copy = structuredClone(order);"
quiz:
  - q: "const b = { ...a }; b.address.city = \"Ankara\"; What happens to a.address.city?"
    options: ["Unchanged: b is a copy", "It becomes \"Ankara\": address is shared", "tsc rejects the assignment"]
    answer: 1
    why: "Spread copies one level. b.address and a.address are the same object, and tsc only objects if city is declared readonly."
  - q: "Which call gives an independent deep copy in which createdAt is still a Date?"
    options: ["{ ...order }", "JSON.parse(JSON.stringify(order))", "structuredClone(order)", "Object.assign({}, order)"]
    answer: 2
    why: "Spread and Object.assign share createdAt (same Date object); the JSON round trip turns it into a string."
  - q: "What do { ...defaults, ...fromUrl } and { ...fromUrl, ...defaults } differ in?"
    options: ["Nothing", "Which value wins for keys both define: the later spread", "Only the key order of the result"]
    answer: 1
    why: "Properties are defined left to right, so the last spread overwrites earlier ones. Defaults go first."
  - q: "class Money { format() { ... } } — what is structuredClone(new Money())?"
    options: ["A Money, typed Money", "A plain object, typed Money", "A plain object, typed unknown", "A DataCloneError"]
    answer: 1
    why: "The prototype is not cloned, so format() is gone, but the lib signature returns T. The method call compiles and throws."
further_reading:
  - title: "MDN: Spread syntax (...)"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/Spread_syntax"
  - title: "MDN: Destructuring (rest property)"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/Destructuring"
  - title: "MDN: structuredClone()"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Window/structuredClone"
  - title: "MDN: The structured clone algorithm (what can't be cloned)"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Web_Workers_API/Structured_clone_algorithm"
  - title: "react.dev: Updating Objects in State"
    url: "https://react.dev/learn/updating-objects-in-state"
---

## Concept

The same three dots do two jobs:

- **Spread** expands a value. In an array literal or argument list it iterates any
  iterable (`[...set]`, `Math.max(...xs)`). In an object literal it copies the source's
  own enumerable properties (`{ ...defaults, ...overrides }`, ES2018).
- **Rest** collects what is left: `const { id, ...rest } = obj`, `const [head, ...tail] = xs`,
  and rest parameters.

Every copy made this way is **shallow**. The new object has new property slots, but each
slot holds the same value as before, and for objects and arrays that value is a
reference. So a functional update of nested data copies each object on the path from the
root to the change and shares everything else. That is cheap, because untouched branches
are reused, and it is what makes `prev === next` a valid "did this change?" test in React.

TypeScript types all of it. `{ ...a, ...b }` gets the merged type with later properties
winning, a rest pattern gets the object type minus the named keys, and spreading a class
instance gives a type without its methods and accessors, which matches what happens at
run time.

For a real deep copy of **data**, use `structuredClone` (an HTML platform function,
available in browsers and Node). It copies `Date`, `Map`, `Set`, typed arrays and cycles.
It throws `DataCloneError` on functions and DOM nodes, and it ignores prototypes, getters
and `#private` fields, so class instances come back as plain objects.

## Coming from C/C++/Java/Python/JS

- C++: copying a struct copies its members, and a `std::vector` member is copied deeply.
  In JavaScript every object is reached through a reference, so `{ ...outer }` behaves
  like copying a struct of `shared_ptr`s: a new outer object, the same inner ones.
- Java: object spread is a shallow `clone()` without `Cloneable`. Java records are
  immutable, but changing one field means calling the constructor with all of them; in
  TS the "wither" is `{ ...rec, field: value }`.
- Python: `{**d, "k": v}` and `[*xs, x]` are the same idea. `copy.copy` vs `copy.deepcopy`
  maps to spread vs `structuredClone`, except that `deepcopy` keeps the class of each
  instance and `structuredClone` does not.
- Go: assigning a struct copies it, nested structs included, but slices and maps inside
  still share their backing storage: the same trap, one level further down
  (go.composite.structs).
- JS: `Object.assign({}, a, b)` is the pre-ES2018 spelling. The difference: it calls
  setters on the target and can mutate an existing object; spread only defines
  properties on the new one.

## Design notes

Treat state that other code can see as immutable and update it by copying the path to the
change. Two levels of spread are readable; when updates need four, the data is usually
nested more deeply than it should be. Normalize it (an `entities` map keyed by id instead
of arrays inside arrays) before reaching for a helper library. Annotate the shapes with
`readonly` (lesson ts.everyday-types.objects) so a mutation of shared data is a compile
error rather than a code-review finding.

Keep deep copies for the places that need a snapshot: undo stacks, a draft of a form the
user may cancel, data handed to a worker (`postMessage` uses the same structured clone
algorithm). Copying "just in case" hides ownership questions and costs time on every
call. Classes are where copying gets awkward. If instances must be copied, give the class
a method that returns a new instance (`withAmount(n)`), because neither spread nor
`structuredClone` will produce one.
