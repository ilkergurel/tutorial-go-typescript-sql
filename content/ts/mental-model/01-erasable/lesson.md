---
id: ts.mental-model.erasable
title: "TS = JS + erasable static types; what tsc 7 does vs what bundlers do"
track: ts
module: mental-model
order: 1
level: basic
minutes: 12
since: ts5.8
prerequisites: []
compare_with: [ts.node.type-stripping, ts.tooling.vite, inside.codebase.ts-runner, go.tooling.go-command]
tags: [typescript, tsc, esbuild, type-erasure, erasableSyntaxOnly]
summary: >
  Types exist only for the checker: the transpiler that feeds the browser deletes
  them without looking. tsc 7 checks the whole program; bundlers strip one file at
  a time and run code with type errors in it.
examples:
  - id: types-vanish
    file: examples/01_types_vanish.ts
    title: What actually runs has no types in it
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "8-13"
        note: >
          toString() prints the function as the engine received it: esbuild deleted
          `: User` and `: string` and changed nothing else. This is the whole contract
          of erasable syntax: the output is your code minus the types.
      - lines: "15-17"
        note: >
          An interface is a compile-time name only, so there is nothing to typeof,
          iterate or reflect over. Java keeps class metadata at run time and C# keeps
          generics; TypeScript keeps nothing. Runtime type questions need runtime
          values (typeof on values, instanceof on classes, a validator).
      - lines: "19-22"
        note: >
          The annotation on parsed is an unchecked promise. JSON.parse returns any,
          which is assignable to everything, so tsc is satisfied and "41" + 1 gives
          "411". Data crossing a boundary (fetch, JSON, localStorage) must be
          validated at run time; the zod lesson does it properly.
      - lines: "24-26"
        note: >
          A real type error: delete the directive and tsc reports TS2322. The
          transpiler never asked, so the program runs and prints 80801. Vite, Node
          and this runner all behave this way; only tsc (in your editor and in CI)
          stops it.
  - id: emitted-syntax
    file: examples/02_emitted_syntax.ts
    title: The few TS constructs that are not erasable
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-6"
        note: >
          A union type alias is pure type syntax: it disappears and only the string
          "info" is left. This is why literal unions are the default replacement for
          enums (lesson ts.everyday-types.unions-enums).
      - lines: "8-15"
        note: >
          enum is TypeScript-only syntax with runtime behaviour: the transpiler has to
          generate an object, here with a reverse mapping (0 -> "Low"). Node's type
          stripping refuses to run it (ERR_UNSUPPORTED_TYPESCRIPT_SYNTAX), and
          erasableSyntaxOnly makes tsc reject it up front.
      - lines: "17-22"
        note: >
          `private readonly id` in the parameter list looks like a type annotation but
          is a field declaration plus an assignment. esbuild wrote `this.id = id` and a
          field `id;` that do not appear in your source.
      - lines: "24-31"
        note: >
          The erasable spelling produces the same class, but every line of the output
          now exists in the source. `private` on a field is just a modifier the
          checker reads and the transpiler deletes; for privacy enforced at run time
          use a JavaScript #private field instead.
use_cases:
  - "Reading a stack trace or the browser debugger: the code you see is your TS with the types removed, so line-by-line debugging works without surprises"
  - "Choosing tsconfig flags for a project transpiled by Vite/esbuild or run directly by Node 22.18+/23.6+: erasableSyntaxOnly plus verbatimModuleSyntax"
  - "Deciding where validation must happen: every value that arrives as text (fetch, JSON.parse, localStorage, URL params)"
strengths:
  - "Zero runtime cost: unlike Java or C# there is no type metadata or reflection data at run time, and the shipped JS is as small as hand-written JS"
  - "Transpiling is fast and per-file, so Vite's dev server serves a changed file in milliseconds without waiting for the checker; tsc 7 then checks the whole program several times faster than the JavaScript-based 6.0"
  - "Any JavaScript engine runs the output: browsers, Node, Deno, Bun, workers. There is no TypeScript runtime to install or version"
weaknesses:
  - "Unlike Go, where a type error means no binary, a TS type error does not stop `vite build`, the dev server or this runner. The checker is a separate step you must wire into CI"
  - "No runtime types: you cannot ask an interface whether a JSON value matches it. Validation libraries (zod) exist to rebuild that at the boundary"
  - "Annotations on any-typed values (JSON.parse, catch variables without strict) are trusted blindly, so a wrong type can travel far before something breaks"
pitfalls:
  - title: "Treating a green bundler build as a type check"
    detail: "Vite, esbuild and Node strip types and never report type errors. In this repo `npm run build` does not run tsc; `npm run typecheck` does, and CI needs both."
  - title: "Annotating parsed JSON and calling it validated"
    detail: "`const u: User = JSON.parse(text)` compiles for any text. Parse to unknown and validate, or the first wrong field surfaces far from where it entered."
    bad: "const u: User = JSON.parse(text);"
    good: "const raw: unknown = JSON.parse(text); const u = UserSchema.parse(raw);"
  - title: "Non-erasable syntax in code that Node runs directly"
    detail: "enum, namespaces with values, parameter properties and `import =` make Node's type stripping throw ERR_UNSUPPORTED_TYPESCRIPT_SYNTAX. Turn on erasableSyntaxOnly so tsc reports TS1294 in the editor instead."
quiz:
  - q: "`vite build` succeeds on a file that has a type error. Why?"
    options:
      - "Vite downgrades type errors to warnings"
      - "Vite transpiles each file by deleting the types and never runs the type checker"
      - "tsc 7 only checks .tsx files"
    answer: 1
    why: "Bundlers strip types per file without type information. Only tsc (or your editor's language service) reports type errors."
  - q: "Which of these produces JavaScript code rather than being erased?"
    options:
      - "type Level = \"debug\" | \"info\""
      - "interface User { name: string }"
      - "enum Level { Debug, Info }"
      - "function f(x: number): string"
    answer: 2
    why: "An enum compiles to an object (with a reverse mapping for numeric members). The other three disappear completely."
  - q: "`const u: User = JSON.parse(text)`: what verifies that text contains a numeric age?"
    options:
      - "tsc, at compile time"
      - "The browser, when the annotated line runs"
      - "Nothing, unless you validate the value yourself"
    answer: 2
    why: "JSON.parse returns any and the annotation is erased. Neither the checker nor the runtime looks at the actual data."
further_reading:
  - title: "TypeScript Handbook: The Basics (Erased Types, Emitting with Errors)"
    url: "https://www.typescriptlang.org/docs/handbook/2/basic-types.html"
  - title: "TypeScript 5.8 release notes: --erasableSyntaxOnly"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-5-8.html"
  - title: "TSConfig reference: isolatedModules"
    url: "https://www.typescriptlang.org/tsconfig/#isolatedModules"
  - title: "Node.js: Modules: TypeScript (type stripping)"
    url: "https://nodejs.org/api/typescript.html"
---

## Concept

TypeScript is JavaScript plus a layer of type syntax that is designed to be deleted. Two
different tools act on a `.ts` file, and they never talk to each other:

- The **checker**, `tsc`, reads the whole program (every file, every `.d.ts`), works out
  the types and reports errors. In a bundler project like this app it writes nothing:
  `npm run typecheck` is `tsc --noEmit -p tsconfig.app.json`.
- The **transpiler** (esbuild inside Vite, Node's type stripping, the esbuild-wasm worker
  behind this Run button) looks at one file at a time, removes the type syntax and hands
  JavaScript to the engine. It does not know or care whether the types are right.

So nothing about a type exists at run time: no reflection, no checks on assignment, no
`typeof SomeInterface`. A type is a claim that the checker verified, or (via `any`)
didn't.

**TypeScript 7.0** (July 2026) is the same language with the compiler ported to Go:
type-checking runs in parallel, and the 7.0 announcement measured full builds 8 to 12
times faster than the JavaScript-based compiler. tsc 7 can still emit JavaScript, and
like every tsc it emits even when there are errors unless `noEmitOnError` is set. The
bundler-plus-`noEmit` split is a project choice, not a limit of the compiler. What 7.0
does not ship is the old JavaScript compiler API, which is why this repo keeps
TypeScript 6.0 installed beside it for typescript-eslint (ADR-0006).

## Coming from C/C++/Java/Python/JS

- Python: the closest analogue. Type hints are ignored at run time and mypy/pyright is a
  separate checker. TypeScript works the same way, except that the hints are stripped
  out of the file instead of being kept in `__annotations__`.
- Java: you know generic type erasure. TypeScript erases *everything*: classes survive
  (they are JavaScript), but interfaces, type aliases, generics and annotations leave no
  trace, so there is nothing like `getClass()` or reflection over a declared type.
- C/C++: types drive code generation there (overload resolution, layout, templates).
  In TypeScript the types never change the emitted code, apart from the few
  non-erasable constructs in example 2.
- Go: `go build` refuses to produce a binary with a type error. Vite, Node and this
  runner happily run TypeScript that tsc rejects. The gate is your editor plus CI.
- JS: the runtime is exactly the one you know. Nothing new executes.

## How it works

```mermaid
flowchart LR
  src["app.ts"] --> tsc["tsc 7 (whole program)"]
  tsc --> diag["type errors, or nothing (noEmit)"]
  src --> strip["esbuild / Vite / Node (one file at a time)"]
  strip --> js["app.js: same code minus types"]
  js --> engine["browser or Node"]
```

Erasable syntax is what a per-file transpiler can delete without understanding it:
annotations, `interface`, `type`, generics, `as` casts, `import type`. A short list of
older TypeScript features generate code instead: `enum`, `namespace` with values, class
parameter properties and `import x = require(...)`. tsc also flags the old `<T>value`
assertion syntax under `erasableSyntaxOnly`. Node's built-in type stripping (on by default
since Node 23.6 and 22.18) refuses the code-generating constructs outright, and the
`erasableSyntaxOnly` flag (TypeScript 5.8) makes tsc report them as TS1294 so you learn
it in the editor rather than at startup.

`isolatedModules` (on in this repo's tsconfig) is the matching guard for the
transpiler's blindness: it makes tsc reject code that a one-file-at-a-time tool can't
transpile correctly, such as re-exporting a type without `export type`.

## Design notes

Treat the two tools as two jobs. The transpiler gives you a fast edit-run loop; the
checker is a test suite for your types and belongs in CI next to the unit tests
(`npm run typecheck` here). Write only erasable syntax (`erasableSyntaxOnly` plus
`verbatimModuleSyntax`), so the JavaScript that runs is exactly what you wrote minus
the types, and the same file works unchanged under esbuild, Node, Deno or Bun. Finally,
because types vanish, every value that enters as text (fetch bodies, `JSON.parse`,
storage, URL parameters) is `unknown` in reality, whatever its annotation says:
validate it once at the boundary, and inside the program trust the types.
