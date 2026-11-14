---
id: ts.node.type-stripping
title: "Node running .ts via type stripping; erasable syntax only"
track: ts
module: node
order: 1
level: intermediate
minutes: 12
since: ts5.8
prerequisites: [ts.mental-model.erasable, ts.objects-classes.modules]
compare_with: [go.tooling.go-command, ts.mental-model.tsconfig, ts.everyday-types.unions-enums]
tags: [node, type-stripping, erasableSyntaxOnly, verbatimModuleSyntax, esm]
summary: >
  Node 22.18+ and 23.6+ run a .ts file directly by blanking out its type syntax
  with spaces: no build step, no tsconfig, no type check. Only erasable syntax
  survives that, which is exactly what erasableSyntaxOnly makes tsc enforce.
examples:
  - id: strip
    file: examples/01_strip.ts
    title: What Node does to a .ts file before V8 runs it
    runtime: ts-node
    check: exact
    annotations:
      - lines: "3-11"
        note: >
          This file runs as it is, with no compile step. The interface and the
          `: City` and `: string` annotations are replaced by spaces, and the
          default value `upper = false` stays because it is JavaScript.
          toLocaleUpperCase("tr") maps i to İ and ı to I, giving DİYARBAKIR;
          plain toUpperCase() would give DIYARBAKIR.
      - lines: "16-23"
        note: >
          stripTypeScriptTypes from node:module (22.13 and 23.2, a release
          candidate API) applies the same transformation to a string. Each
          output line is exactly as long as its input: the non-null `!`, the
          `as number` and the whole interface become spaces. Node's docs warn
          that its output may change between Node versions.
      - lines: "25-27"
        note: >
          The stack frame names line 26 of this file. Because nothing moved,
          Node needs no source maps for stack traces or breakpoints. esbuild
          and tsc rewrite the code, so they need source maps to get the same
          result.
      - lines: "29-31"
        note: >
          tsc reports TS2741 (property 'plate' is missing). Node never asks, so
          the call runs and prints "undefined Ankara". The type check is a
          separate step: your editor, and `tsc --noEmit` in CI.
  - id: erasable-only
    file: examples/02_erasable_only.ts
    title: What Node refuses, and the erasable spelling of each
    runtime: ts-node
    check: exact
    annotations:
      - lines: "4-16"
        note: >
          Each entry becomes a real file in the run's temp directory. Loading
          them with a dynamic import() makes every failure an exception for
          that file alone. A static import of enum.ts would stop the whole
          program before its first line ran, because modules are parsed and
          linked before any of them executes.
      - lines: "5-7"
        note: >
          An enum makes the compiler emit an object, so Node refuses it. The
          as-const object plus a type with the same name gives the same
          `Level.Warn` at the call site. Only the value shows up in the
          exports below. The type was erased.
      - lines: "8-11"
        note: >
          `public x` in a constructor's parameter list declares a field and
          assigns it, which is code the type stripper would have to write. The
          explicit field is erasable. A namespace with values compiles to an
          IIFE; a plain module is the modern namespace. A namespace holding
          only types is still allowed.
      - lines: "12-14"
        note: >
          types.ts exports nothing at run time. `import { User }` is therefore
          a value import of a name that doesn't exist, and module linking
          fails. Node can't tell a type from a value without type information.
          `import type` is deleted entirely, and verbatimModuleSyntax makes tsc
          insist on it (TS1484).
      - lines: "15"
        note: >
          Node resolves the specifier exactly as written: no .ts, .js or
          /index guessing, and no tsconfig paths. Write "./plain.ts". tsc
          accepts that with rewriteRelativeImportExtensions or
          allowImportingTsExtensions, and otherwise reports TS5097.
      - lines: "36-39"
        note: >
          Plumbing for stable output: Node 22 wraps the parser's message in a
          code frame and Node 24 does not. Messages of other errors (here
          ERR_MODULE_NOT_FOUND) contain absolute paths, so only their code is
          printed.
use_cases:
  - "Scripts and small tools (migrations, data fixes, code generators, CI helpers) written in TS and run with `node script.ts`, no build step and no dev dependency"
  - "Small services and CLIs where the deployed code is the source tree, checked by `tsc --noEmit` in CI"
  - "The Node test runner (`node --test`) over .ts test files"
  - "Choosing tsconfig flags for a codebase that must also run under Vite/esbuild: erasableSyntaxOnly plus verbatimModuleSyntax keep one file valid for all of them"
strengths:
  - "No build step, and unlike ts-node or tsx no package to install: Node is the only dependency, like `go run` needs only the Go toolchain"
  - "Stack traces and breakpoints point at the .ts source with no source maps, because types become spaces and nothing moves"
  - "Startup stays fast: the parser only finds and blanks out types. It doesn't check them and doesn't read tsconfig, so it never loads the whole program the way tsc does"
  - "The rules are the same erasable subset that esbuild, Vite and Deno handle without surprises, so the same file runs everywhere"
weaknesses:
  - "No type check at all. Unlike `go run`, which refuses to build code with a type error, `node app.ts` runs code that tsc rejects"
  - "tsconfig.json is ignored: no paths aliases, no `target` downleveling, no JSX (.tsx files are refused)"
  - "No TypeScript inside node_modules: libraries still have to ship compiled JavaScript plus .d.ts files"
  - "Older TS codebases full of enums, namespaces and parameter properties need refactoring (or a full transpiler like tsx) before Node runs them"
pitfalls:
  - title: "Treating `node app.ts` as a type check"
    detail: "Node strips types without reading them. Run `tsc --noEmit` in CI and before commits; in this repo that is `npm run typecheck`."
  - title: "Importing a type without `type`"
    detail: "The import survives stripping, so Node looks for a runtime export that was never there and linking fails with a SyntaxError. Turn on verbatimModuleSyntax so tsc reports TS1484 at the import."
    bad: "import { User, loadUser } from \"./users.ts\";"
    good: "import { type User, loadUser } from \"./users.ts\";"
  - title: "Extensionless imports and tsconfig paths"
    detail: "`import \"./db\"` and `import \"@/db\"` both fail with ERR_MODULE_NOT_FOUND. Write the real file name, and for aliases use package.json subpath imports, which must start with #."
    bad: "import { pool } from \"@/db\";"
    good: "import { pool } from \"#db\"; // package.json: \"imports\": { \"#db\": \"./src/db.ts\" }"
  - title: "Expecting tsconfig's target to downlevel new syntax"
    detail: "Node runs the syntax you wrote. `await using f = await open(path)` works on Node 24 but is a SyntaxError on Node 22, whatever tsconfig says. Check the oldest Node version you deploy to."
  - title: "Non-erasable syntax that still compiles with tsc"
    detail: "enum, namespaces with values, parameter properties, `import x = require()` and `<T>value` assertions all compile with plain tsc and fail in Node with ERR_UNSUPPORTED_TYPESCRIPT_SYNTAX. Set erasableSyntaxOnly so tsc reports TS1294 instead."
    bad: "class Repo { constructor(private db: Pool) {} }"
    good: "class Repo {\n  #db: Pool;\n  constructor(db: Pool) {\n    this.#db = db;\n  }\n}"
quiz:
  - q: "Which declaration runs under `node app.ts` unchanged?"
    options:
      - "enum Level { Info, Warn }"
      - "class Repo { constructor(private db: Pool) {} }"
      - "const Level = { Info: \"info\", Warn: \"warn\" } as const"
      - "namespace Geo { export const R = 6371 }"
    answer: 2
    why: "The as-const object is plain JavaScript plus an erasable `as const`. The other three make the compiler generate code, so Node throws ERR_UNSUPPORTED_TYPESCRIPT_SYNTAX."
  - q: "types.ts contains only `export interface User {}`. What does `import { User } from \"./types.ts\"` do under Node?"
    options:
      - "Nothing: Node deletes the import"
      - "Linking fails with a SyntaxError: the module does not provide an export named 'User'"
      - "User is undefined at run time"
    answer: 1
    why: "Without `type`, the import is a value import. Node has no type information, so it keeps the import, and the interface left no runtime export behind."
  - q: "Why do stack traces from a stripped .ts file point at the right line and column without source maps?"
    options:
      - "Node generates source maps in memory"
      - "Type syntax is replaced by spaces, so every remaining token keeps its position"
      - "V8 understands TypeScript natively"
    answer: 1
    why: "Stripping blanks out types in place instead of rewriting code, so positions in the JavaScript V8 runs equal positions in your file."
  - q: "A script has a type error that tsc reports. What does `node script.ts` do?"
    options:
      - "Refuses to start, like `go run`"
      - "Prints the type error as a warning and runs"
      - "Runs it without reporting anything"
    answer: 2
    why: "Node performs no type checking. Only tsc (or your editor) reports type errors."
further_reading:
  - title: "Node.js: Modules: TypeScript"
    url: "https://nodejs.org/api/typescript.html"
  - title: "Node.js: module.stripTypeScriptTypes()"
    url: "https://nodejs.org/api/module.html#modulestriptypescripttypescode-options"
  - title: "Node.js: subpath imports"
    url: "https://nodejs.org/api/packages.html#subpath-imports"
  - title: "TypeScript 5.8 release notes: --erasableSyntaxOnly"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-5-8.html"
  - title: "TSConfig reference: verbatimModuleSyntax"
    url: "https://www.typescriptlang.org/tsconfig/#verbatimModuleSyntax"
  - title: "TSConfig reference: rewriteRelativeImportExtensions"
    url: "https://www.typescriptlang.org/tsconfig/#rewriteRelativeImportExtensions"
---

## Concept

`node app.ts` works. Since Node 23.6 and 22.18 type stripping is on by default, since
24.3 it no longer prints an experimental warning, and since 24.12 and 25.2 it is
marked stable. (It arrived behind `--experimental-strip-types` in 22.6.) Node parses
the file with a built-in TypeScript parser, replaces every piece of type syntax with
spaces, and hands the result to V8. That is all it does:

- **no type checking**: a file that tsc rejects still runs;
- **no tsconfig.json**: no `paths`, no `target` downleveling, no JSX;
- **no code generation**: only *erasable* syntax is accepted, meaning syntax that
  leaves valid JavaScript behind once it is blanked out.

The constructs that make the compiler generate code (`enum`, namespaces with values,
parameter properties, `import x = require()`) are refused with
`ERR_UNSUPPORTED_TYPESCRIPT_SYNTAX`, and so is the old `<T>value` assertion syntax. tsc
5.8's `erasableSyntaxOnly` flag rejects the same set at edit time (TS1294), so your
editor tells you before Node does.

Module rules follow from "no type information, no tsconfig": a `.ts` file is ESM or
CommonJS according to the nearest package.json `"type"` (`.mts` is always ESM, `.cts`
always CommonJS), import specifiers need their real extension, type-only imports need
`import type`, `.tsx` is unsupported, and Node refuses TypeScript files anywhere under
`node_modules`.

## Coming from C/C++/Java/Python/JS

- Go: `go run main.go` compiles and type-checks; one type error and nothing runs. Node is
  the opposite: it runs first, and the check is a separate `tsc --noEmit` you have to
  wire into CI.
- Python: the closest match. CPython ignores type hints and mypy checks them separately.
  The difference is that Python keeps annotations in `__annotations__` at run time, while
  Node deletes them.
- Java: `java Main.java` (the source launcher, Java 11) compiles in memory with full type
  checking. Node does no checking and keeps no class metadata.
- JS/Node: before this, running TS meant a build step or a loader such as ts-node or
  tsx. Those are still the answer for JSX, `paths` aliases or enums; for erasable code,
  plain `node` is enough.
- C/C++: think of a preprocessor that only blanks out text, never expands anything.

## How it works

```mermaid
flowchart LR
  file["app.ts"] --> resolve["resolve the specifier exactly as written"]
  resolve --> nm{"under node_modules?"}
  nm -- yes --> refuse["refused"]
  nm -- no --> parse["TS parser (WebAssembly)"]
  parse --> erasable{"only erasable syntax?"}
  erasable -- no --> err["ERR_UNSUPPORTED_TYPESCRIPT_SYNTAX"]
  erasable -- yes --> blank["types replaced by spaces"]
  blank --> v8["V8 runs the JavaScript"]
```

Node's own docs recommend TypeScript 5.8 or newer with this tsconfig, and every flag
in it mirrors one runtime rule:

```json
{
  "compilerOptions": {
    "noEmit": true,
    "target": "esnext",
    "module": "nodenext",
    "rewriteRelativeImportExtensions": true,
    "erasableSyntaxOnly": true,
    "verbatimModuleSyntax": true
  }
}
```

- `erasableSyntaxOnly`: TS1294 on enum, value namespaces, parameter properties and `<T>x`.
- `verbatimModuleSyntax`: TS1484 when a type is imported without `type`.
- `module: nodenext`: TS2835 when a relative import has no extension.
- `rewriteRelativeImportExtensions` (TS 5.7): allows `./db.ts` specifiers, and rewrites
  them to `.js` if you ever compile instead of strip.

tsc 7.0.2 reports each of these codes for the snippets in example 2.

## Design notes

Running TypeScript directly splits the toolchain into two independent jobs. Node
executes, and tsc is your test suite for types. Keep `tsc --noEmit` in CI; without it
nothing ever checks the types. Write only erasable syntax even where a bundler would
accept more. The rule costs little (literal unions instead of enums, explicit fields
instead of parameter properties, modules instead of namespaces), and in return the same
file runs unchanged under Node, esbuild, Vite, Deno and Bun, and the JavaScript that
executes is your code minus its types. The boundary is publishing: a package on npm
still ships compiled `.js` plus `.d.ts`, because Node won't strip types inside
`node_modules` and consumers may run older Node versions.
