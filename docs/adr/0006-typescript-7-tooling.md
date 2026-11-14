# ADR-0006: TypeScript 7 tooling setup

Date: 2026-11-12 · Status: accepted

## Context

TypeScript 7.0 (GA 2026-07-08) is the native (Go) compiler: `tsc` is a fast binary, but
the programmatic compiler API that TS 5/6 exposed is not shipped in 7.0 (planned for
7.1). Tools that import the compiler API (typescript-eslint's type-aware rules,
ts-morph, ts-jest, some Vite plugins) may not work against 7.

## Decision

- TypeScript 7.0.x (devDependency `typescript7`, see below) is used for type-checking
  (`npm run typecheck`) and by `verify-content` for type-checking TS examples.
- The browser runner transpiles with `esbuild-wasm` in a Web Worker (TS 7 has no JS
  compiler API to call at runtime; bundlers only strip types anyway).
- Monaco editor keeps its own bundled TypeScript version for editor diagnostics; a doc
  note explains that editor diagnostics can differ slightly from `tsc` 7 output.
- Owner decision (2026-11-12): **type-aware ESLint is on.** typescript-eslint keeps
  needing the TS 5/6 JS API, so the side-by-side install is permanent until the 7.1
  API lands; `tsc` 7 stays authoritative for type-checking.
- Setup (2026-11-13). Tools load the compiler with `require('typescript')`, so the JS-API
  version must own the `typescript` name and TS 7 is the alias (aliasing the old one, as
  first planned here, would leave typescript-eslint loading TS 7's API-less stub):
  - `typescript` = `~6.0.3`: the newest line typescript-eslint supports (8.72.2:
    `>=4.8.4 <6.1.0`) and the one that already reports TS 7's deprecations.
  - `typescript7` = `npm:typescript@~7.0.2`. Both packages ship a `tsc` bin and
    `node_modules/.bin/tsc` resolves to 6.0, so `typecheck` calls
    `node node_modules/typescript7/bin/tsc` explicitly.

## Alternatives

- **Stay on TS 5/6**: maximum tool compatibility, but the app is a teaching artifact for
  TS 7; teaching the old compiler contradicts the style guide's "mark modern vs legacy"
  rule.

## Consequences

- `tsc --noEmit` becomes nearly instant on this codebase.
- Editor diagnostics and real `tsc` can disagree in edge cases; `contentcheck` always
  uses real `tsc`, which is authoritative.
- Revisit this ADR when 7.1 ships the programmatic API.

## Update (2026-11-14)

The setup holds: 7.0.2 and 6.0.3 are installed, and typescript-eslint 8.72.2 still
declares `typescript >=4.8.4 <6.1.0`. Three planned parts were never built:

- Nothing type-checks lesson examples with tsc 7. contentcheck
  (`scripts/build-lessons.mjs`) validates files without compiling them, and
  `npm run verify:lessons` runs examples the way the Run button does (esbuild for
  ts-browser, Node's type stripping for ts-node). The one type check of lesson code is
  ts-test's `*.test-d.ts` files, which Vitest's `--typecheck` checks with the `tsc` in
  `node_modules/.bin`, i.e. 6.0 (ADR-0010).
- Lint isn't type-aware: `eslint.config.js` extends typescript-eslint's `recommended`
  config without type information, so the 6.0 install serves its parser.
- No doc note on Monaco's diagnostics was written.

TS 7.0.2's main entry exports only `version` and `versionMajorMinor`; it also ships
experimental `typescript/unstable/*` API entry points.
