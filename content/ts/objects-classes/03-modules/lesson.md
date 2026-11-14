---
id: ts.objects-classes.modules
title: "ES modules: import type, verbatimModuleSyntax, barrels, path aliases, .d.ts"
track: ts
module: objects-classes
order: 3
level: intermediate
minutes: 15
since: ts5.0
prerequisites: [ts.mental-model.erasable, ts.mental-model.tsconfig]
compare_with: [go.design.packages, go.tooling.project-layout, ts.architecture.folders]
tags: [modules, import-type, verbatimModuleSyntax, barrels, paths, declaration-files]
summary: >
  ES modules run once, share one instance per specifier and export live, read-only
  bindings. Mark type-only imports with `type` so every tool emits the same code, import
  from the defining file instead of a barrel, and configure path aliases for both tsc
  and the runtime.
examples:
  - id: type-only-imports
    file: examples/01_type_only_imports.ts
    title: import type is erased before the code runs
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-5"
        note: >
          ./billing.ts does not exist in this sandbox, yet the program runs:
          every transpiler (esbuild here, Node, tsc) deletes an `import type`.
          Under verbatimModuleSyntax an import without `type` stays in the
          output exactly as written, and here it would fail to load.
      - lines: "7-11"
        note: >
          `import type` may name a value. CURRENCIES is a const in billing.ts,
          and `typeof CURRENCIES` derives a union from it without loading the
          module; reading it at run time is TS1361. This is how one module can
          depend on another's types without creating a runtime import cycle.
      - lines: "13-21"
        note: >
          The imported types work as usual. The tr-TR currency format puts the
          symbol first and swaps the separators: ₺1.250,50, and €99,00 for euros.
      - lines: "23-25"
        note: >
          A per-file transpiler can't tell whether Invoice is a type or a value
          in billing.ts, so verbatimModuleSyntax (like isolatedModules) makes the
          re-export say it. `export type` is erased with the rest.
  - id: live-bindings
    file: examples/02_live_bindings.ts
    title: Live bindings, one instance per specifier, read-only namespaces
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-9"
        note: >
          The sandbox runs one file, but import() accepts any URL its CSP allows,
          including data: URLs, so a second module can be written inline. Real
          code imports files; the semantics are the same.
      - lines: "11-19"
        note: >
          import() of a computed string is typed Promise<any>, so the interface
          is a description you assert, exactly like a hand-written .d.ts file:
          tsc trusts it and never compares it with the JavaScript. If the module
          changes, nothing warns you.
      - lines: "20-23"
        note: >
          counter.count reads the exporting module's variable, which increment()
          changed: a live binding. A named import (import { count }) is live in
          the same way. Destructuring copied the value at that moment, which is
          also what CommonJS's const { count } = require() and Python's
          from m import count give you.
      - lines: "25-27"
        note: >
          A module is evaluated once per resolved specifier, and every importer
          gets the same instance: one "evaluated" line, same object. Module-level
          state is therefore a singleton, handy for caches and a trap for tests
          that expect a fresh copy.
      - lines: "29-34"
        note: >
          Only the exporting module can assign its bindings. The namespace object
          rejects writes, and modules are strict, so the write throws. Without
          the cast tsc refuses the line (TS2540), because the interface declares
          count readonly to mirror that rule.
  - id: barrels
    file: examples/03_barrels.ts
    title: What a barrel costs
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "5-22"
        note: >
          Three feature modules that log when their body runs, and a barrel that
          re-exports all of them with export *, which is what a typical
          src/shared/index.ts looks like.
      - lines: "29-31"
        note: >
          Importing the defining module evaluates that module and its own
          dependencies, nothing else.
      - lines: "33-36"
        note: >
          Importing the same function through the barrel evaluates every module
          the barrel re-exports, in import order, before the import resolves.
          In a dev server each is a separate request; Vite's performance guide
          recommends direct imports for this reason. money is not evaluated
          again, and the function is the same object.
      - lines: "38-39"
        note: >
          money and charts both export VERSION, so the name is ambiguous and
          export * exports neither. A namespace import just lacks it; a named
          import { VERSION } from the barrel fails at link time with a
          SyntaxError. tsc reports TS2308 on the barrel. Namespace keys are
          listed in sorted order.
use_cases:
  - "import type for everything used only in annotations, including types from API clients imported into components"
  - "Direct imports from the defining file in application code; a barrel only as the public entry point of a package"
  - "Path aliases (@/... in tsconfig paths plus the bundler's alias, or #/... in package.json imports) instead of ../../../ chains"
  - ".d.ts files for untyped JavaScript, bundler-injected globals such as import.meta.env, and non-code imports such as *.svg"
strengths:
  - "verbatimModuleSyntax makes emit predictable: an import stays or goes based on what it says, not on what another file declares, so esbuild, Node and tsc produce the same JavaScript"
  - "Imports are static and linked before any code runs, so bundlers can tree-shake and editors can rename across files; CommonJS require() can appear anywhere, with any computed string"
  - "Bindings are live and read-only: importers always see the current value and can't overwrite another module's variables, unlike Python's from m import x copy"
  - "Declaration files type existing JavaScript without touching it, the way C headers describe a compiled library"
weaknesses:
  - "Aliases need two configurations that must agree: tsconfig paths for tsc and the bundler or Node mapping for the runtime; tsc never rewrites import paths"
  - "Barrels make each importer pay for every sibling module, hide name clashes from namespace imports, and make import cycles more likely"
  - "Module state is a singleton per specifier, shared by every feature and every test in the same process"
  - "A .d.ts is trusted, not verified against the JavaScript it describes; a wrong declaration is a type error waiting at run time"
pitfalls:
  - title: "Inline type modifiers still load the module"
    detail: "Under verbatimModuleSyntax, import { type User } from \"./user\" becomes import {} from \"./user\": the module is still fetched and evaluated. When every name is a type, put type on the import itself."
    bad: "import { type User, type Role } from \"./user.ts\";"
    good: "import type { User, Role } from \"./user.ts\";"
  - title: "Circular imports through a barrel"
    detail: "Feature A imports from \"@/shared\", whose barrel re-exports a module that imports A. One of them runs first and reads a const or class that isn't initialized yet: ReferenceError: Cannot access 'X' before initialization. Import from the defining file, and keep dependencies pointing one way."
    bad: "import { formatMoney } from \"@/shared\";"
    good: "import { formatMoney } from \"@/shared/money.ts\";"
  - title: "baseUrl in a TypeScript 7 tsconfig"
    detail: "baseUrl is deprecated in 6.0 (TS5101) and removed in 7.0 (TS5102). paths entries resolve relative to the tsconfig file and must start with ./ (TS5090)."
    bad: "\"baseUrl\": \".\", \"paths\": { \"@/*\": [\"src/*\"] }"
    good: "\"paths\": { \"@/*\": [\"./src/*\"] }"
  - title: "paths without a runtime mapping"
    detail: "tsc resolves @/x, the editor is happy, and the build or Node then fails to resolve the import. Add the same alias to the bundler (Vite resolve.alias) or use package.json imports, which Node, Vite and TypeScript all understand."
quiz:
  - q: "With verbatimModuleSyntax, what does import { type User } from \"./user\" compile to?"
    options: ["Nothing: it is erased", "import {} from \"./user\": the module still loads", "import { User } from \"./user\""]
    answer: 1
    why: "Only the type-marked name is dropped. To drop the whole import, write import type { User }."
  - q: "c.ts exports let count = 0 and inc(). main.ts: import { count, inc } from \"./c\"; inc(); console.log(count). What prints?"
    options: ["0", "1", "A TypeError"]
    answer: 1
    why: "Named imports are live bindings to the exporter's variable, not copies."
  - q: "tsconfig has \"paths\": { \"@/*\": [\"./src/*\"] } and tsc emits JavaScript. What does import { x } from \"@/util\" become?"
    options: ["A relative path such as ../util", "It stays \"@/util\"; the runtime or bundler must resolve it", "A compile error"]
    answer: 1
    why: "paths only tells tsc how another tool will resolve the specifier; tsc does not rewrite it."
  - q: "A barrel has export * from \"./money\" and export * from \"./charts\", and both export VERSION. What do importers get?"
    options: ["money's VERSION (first wins)", "charts's VERSION (last wins)", "Neither: the name is ambiguous, and tsc reports TS2308"]
    answer: 2
    why: "Ambiguous star exports are excluded. A namespace import lacks the name; a named import of it is a link-time SyntaxError."
further_reading:
  - title: "MDN: JavaScript modules"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Modules"
  - title: "MDN: The module graph (evaluation order, caching, cycles, export * ambiguity)"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Modules/Module_graph"
  - title: "TypeScript 5.0 release notes: --verbatimModuleSyntax"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-5-0.html"
  - title: "TypeScript 6.0 release notes: baseUrl deprecation, #/ subpath imports"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-6-0.html"
  - title: "TSConfig reference: paths"
    url: "https://www.typescriptlang.org/tsconfig/#paths"
  - title: "TypeScript Handbook: Declaration files"
    url: "https://www.typescriptlang.org/docs/handbook/declaration-files/introduction.html"
  - title: "Node.js: Subpath imports"
    url: "https://nodejs.org/api/packages.html#subpath-imports"
---

## Concept

An ES module is a file with its own scope. Its `import` declarations are hoisted and
linked before any of its code runs; its dependencies are evaluated first, depth-first in
import order; and it is evaluated **once** per resolved specifier, so every importer
shares one instance. What it exports are **live, read-only bindings**: importers see the
exporter's variables, not copies, and only the exporter can assign them.

TypeScript adds imports that carry only types. Mark them, and choose the rule with
`verbatimModuleSyntax` (TS 5.0): anything with `type` is erased, anything without it is
kept exactly as written.

```ts
import type { Invoice } from "./billing.ts";       // erased entirely
import { formatInvoice, type Invoice } from "./billing.ts"; // becomes import { formatInvoice } ...
import { type Invoice } from "./billing.ts";       // becomes import {} from "./billing.ts": still loads
export type { Invoice } from "./billing.ts";       // erased
```

Before that flag, tsc *elided* imports it decided were only used as types, which needs
knowledge of other files. esbuild, Vite and Node see one file at a time, so the source has
to say it.

Three conventions shape how modules are laid out:

- **Barrels**: an `index.ts` that re-exports a folder (`export * from "./money.ts"`).
  Shorter imports, at a price: every importer evaluates every re-exported module, clashing
  names silently disappear, and import cycles get easier to create.
- **Path aliases**: `paths` in tsconfig tells *tsc* that `@/shared/money.ts` means
  `./src/shared/money.ts`. tsc never rewrites specifiers, so the runtime needs the same
  mapping (Vite `resolve.alias`, or `imports` in package.json).
- **Declaration files** (`.d.ts`): types with no implementation. They describe JavaScript
  libraries (bundled typings or `@types/*`), globals and non-code imports, and `tsc
  --declaration` emits them for libraries you publish.

## Coming from C/C++/Java/Python/JS

- C/C++: a `.d.ts` is a header: declarations the compiler trusts, for code compiled
  elsewhere. There is no preprocessor and no include-once guard to write; the module map
  guarantees one evaluation per specifier.
- Java: a module is closer to a file-level class with static members than to a package.
  Imports name files, not namespaces, and `import type` has no Java equivalent because
  Java's types exist at run time.
- Python: modules are also evaluated once and cached, but `from m import x` copies the
  binding, while an ES named import stays live. Python's circular imports fail with a
  partially initialized module; ES ones fail with a ReferenceError for `let`, `const`
  and `class`.
- Go: a Go package is a directory and every file in it shares one namespace; a TS module
  is a single file. Go forbids import cycles at compile time; JavaScript allows them and
  fails at run time. A barrel is roughly a package's exported API, which Go gets without
  a separate file.
- JS: CommonJS `require()` is an ordinary function call that returns the `module.exports`
  object, and `const { count } = require("./c")` copies values out of it. ESM imports are
  static declarations with live bindings, and ES modules allow top-level `await`.

## How it works

A feature folder and the configuration that makes `@/` work everywhere:

```ts
// src/shared/money.ts
export const CURRENCIES = ["TRY", "EUR", "USD"] as const;
export function formatMoney(n: number): string { /* ... */ }

// src/shared/index.ts  (a barrel: convenient, and every importer pays for all of it)
export * from "./money.ts";
export * from "./charts.ts";
// error TS2308: Module "./money.ts" has already exported a member named 'VERSION'. Consider explicitly re-exporting to resolve the ambiguity.

// src/features/billing/invoice.ts
import { formatMoney } from "@/shared/money.ts"; // direct: loads money.ts only
import type { Customer } from "@/features/customers/types.ts"; // erased
```

The tsconfig has no `baseUrl` (removed in TypeScript 7.0), and `paths` entries start with
`./` because they resolve relative to the tsconfig file:

```json
{ "compilerOptions": { "verbatimModuleSyntax": true, "paths": { "@/*": ["./src/*"] } } }
```

```ts
// vite.config.ts: the same mapping for the bundler
resolve: { alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) } }
```

This repo's `tsconfig.app.json` and `vite.config.ts` configure exactly this pair. The
runtime-native alternative is a subpath import in package.json, `"imports": { "#/*":
"./src/*" }`, used as `import { formatMoney } from "#/shared/money.ts"`. Node supports
the `#/` prefix since 24.14 and 25.4 (older versions need a name such as `#src/*`),
and TypeScript 6.0 resolves it under `moduleResolution` `nodenext` and `bundler`.

A declaration file in an app usually extends what the platform gives you:

```ts
// src/vite-env.d.ts in this repo: pulls in Vite's types for import.meta.env and asset imports
/// <reference types="vite/client" />

// a hand-written one for an untyped script
declare module "legacy-chart" {
  export function draw(el: HTMLElement, values: readonly number[]): void;
}
```

## Design notes

Turn on `verbatimModuleSyntax` and write `import type` whenever a whole import is types;
the `type` modifier inside braces is for mixed imports. The output then contains exactly
the imports you wrote, which keeps side effects and load order visible.

Import from the file that defines a thing. Keep barrels for the public surface of a
package that other packages consume, where one entry point is the point; inside an app,
a barrel per folder mostly buys import cycles and slower dev reloads. Choose one alias
scheme (`@/` with tsconfig plus bundler, or `#/` in package.json), configure it in every
tool that resolves imports, and make dependencies point one way: features import shared
code, never the reverse (lesson ts.architecture.folders).

Treat module-level `let` and singletons as global state with a module-shaped name. They
are shared by everything that imports them, survive between tests in the same worker,
and can't be reset from outside. Export functions that take their state as parameters,
or a factory, instead.
