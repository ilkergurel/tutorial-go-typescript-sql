---
id: ts.tooling.lint
title: "A ladder of tsconfig strictness; ESLint + typescript-eslint; Prettier"
track: ts
module: tooling
order: 1
level: intermediate
minutes: 14
since: ts6.0
prerequisites: [ts.mental-model.tsconfig, ts.async.promises]
compare_with: [go.tooling.go-command, go.tooling.modernizers, ts.mental-model.tsconfig]
tags: [tooling, tsconfig, eslint, typescript-eslint, prettier, ci]
summary: >
  Three tools, three jobs: tsc proves types, type-aware ESLint catches bugs that type-check
  fine, Prettier ends formatting debates. Climb the strictness ladder one measured rung at a
  time, and ratchet each new check in so existing errors never block a merge.
examples:
  - id: promise-bugs
    file: examples/01_promise_bugs.ts
    title: "Promise bugs that tsc accepts and type-aware lint rejects"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "9-11"
        note: >
          filter's predicate may return anything, so tsc accepts a Promise; every
          Promise is truthy, so nobody is filtered out. tsc does catch the plain
          `if (isAdmin(x))` (TS2801), but not this, a negation, or a forEach.
      - lines: "13-21"
        note: >
          forEach calls the async callback and drops the Promise it returns.
          Each callback stops at its first await, so countAdmins returns 0 before
          any of them resumes.
      - lines: "29-32"
        note: >
          A floating promise: submit returns while saveDraft is suspended, and a
          rejection would surface as an unhandled rejection far from this line.
          If fire-and-forget is really meant, `void saveDraft()` says so and
          satisfies the rule.
      - lines: "35-38"
        note: >
          The fixed versions. Promise.all runs the checks concurrently and keeps
          their order; by the last line saveDraft has finished, which shows the
          floating call did run, just not when submit assumed.
  - id: silent-values
    file: examples/02_silent_values.ts
    title: "Values that type-check and still print nonsense"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "11-12"
        note: >
          A template literal accepts any type, so an object becomes
          "[object Object]" in a log line, an error message or a URL.
      - lines: "14-15"
        note: >
          || falls back on every falsy value, so an explicit 0 becomes 3. ??
          falls back only on null and undefined. The rule fires only when the
          left side can actually be nullish, as an optional property can.
      - lines: "17-30"
        note: >
          A switch statement that assigns instead of returning gives tsc nothing
          to check: the missing "archived" case compiles and renders an empty
          badge. switch-exhaustiveness-check is in no preset; enable it by name.
  - id: ratchet
    file: examples/03_ratchet.ts
    title: "Ratcheting a new check in: compare per-file counts against a baseline"
    runtime: ts-node
    fixtures: [fixtures/unchecked-index.txt, fixtures/baseline.json]
    check: exact
    annotations:
      - lines: "4-7"
        note: >
          unchecked-index.txt is tsc 7.0.2's real output for this repo with
          noUncheckedIndexedAccess on (2026-11-13). The baseline is invented to
          show all three verdicts; a real one is generated once and committed.
      - lines: "9-13"
        note: >
          tsc's non-pretty format is file(line,col): error TSnnnn: message, one
          error per line plus indented continuation lines, which the regex
          skips. Counting per file, not per line, survives code moving around.
      - lines: "15-23"
        note: >
          New errors fail; fewer errors pass and ask for a lower baseline, so the
          count can only go down. browserTs.ts fails here because it had no
          entry: a new index read without a check.
      - lines: "26"
        note: >
          The run here only prints, because a non-zero exit fails the example. In
          CI the exit code is the whole point.
use_cases:
  - "Starting a project: strict tsconfig, type-aware recommended lint and a formatter on day one, when every rung costs zero errors"
  - "Tightening an existing codebase: measure each flag or rule from the command line, take the free ones immediately, ratchet the rest"
  - "Code review: lint catches floating promises and misused async callbacks so reviewers can look at design instead"
  - "Ending formatting discussions: Prettier on save and `prettier --check` in CI, ESLint left to bugs"
strengths:
  - "Unlike Go, where gofmt and vet ship with the compiler, every rung is adjustable, so a JavaScript codebase can be migrated gradually and a new one can start at the top"
  - "Type-aware rules see what tsc knows: a Promise passed where void is expected, an object stringified in a template, a non-exhaustive switch statement"
  - "A ratchet makes a new check adoptable without a big-bang cleanup: existing errors are tolerated, new ones fail"
  - "Separating formatter and linter keeps each fast and uncontroversial; formatting never shows up as a lint error"
weaknesses:
  - "Type-aware linting needs a TypeScript program per run: 3.8 s instead of 1.6 s on this repo's src, and much more on large codebases"
  - "typescript-eslint drives the TypeScript 5/6 JavaScript API, which TypeScript 7.0 doesn't ship, so a TS 7 project keeps TS 6 installed for its linter (ADR-0006)"
  - "strict and strict-type-checked are documented as not semver-stable: a minor typescript-eslint update can add errors"
  - "Three tools mean three configs and three places where an ignore list can be forgotten"
pitfalls:
  - title: "Turning on the strictest preset in one commit"
    detail: "On this repo, strict-type-checked plus stylistic-type-checked reports 53 problems in 15 files. Done in one go, that becomes a wall of eslint-disable comments. Take the free rungs, then ratchet."
  - title: "Fire-and-forget without saying so"
    detail: "A deliberately unawaited promise should be marked, and its errors handled, so readers and no-floating-promises can tell it from a forgotten await."
    bad: "saveDraft();"
    good: "void saveDraft().catch(reportError);"
  - title: "Linting generated files"
    detail: "ESLint 9 lints .js files by default. This repo's ignores list only dist, so `npm run lint` also lints the generated public/lesson-libs/bundle.js and reports errors from eslint-disable comments inside bundled libraries. Ignore build output explicitly."
    bad: "{ ignores: ['dist'] }"
    good: "{ ignores: ['dist', 'public/lesson-libs'] }"
  - title: "Formatting rules in the linter"
    detail: "ESLint deprecated its core formatting rules (indent, semi, quotes and others) in v8.53.0. Let a formatter own layout and turn off any lint rule that disagrees with it."
quiz:
  - q: "`logins.filter(async (l) => await isAdmin(l))` keeps every login. Which tool reports it?"
    options: ["tsc with strict: true", "typescript-eslint's recommended config (no type information)", "typescript-eslint's recommended-type-checked config"]
    answer: 2
    why: "no-misused-promises needs type information to know the callback returns a Promise where filter tests truthiness. tsc accepts it, and syntax-only rules can't see return types."
  - q: "A new flag produces 27 errors in 5 files. With a per-file ratchet, which pull request fails?"
    options: ["One that leaves the 27 errors untouched", "One that fixes two errors in one file", "One that adds a new error in a file whose count was 0"]
    answer: 2
    why: "The ratchet compares per-file counts with the baseline: unchanged or lower passes, higher fails. Old errors never block a merge; new ones always do."
  - q: "Prettier and an ESLint rule disagree about where a line breaks. What is the fix?"
    options: ["Configure Prettier to match the ESLint rule", "Turn the formatting rule off in ESLint and let Prettier decide", "Run ESLint --fix after Prettier in CI"]
    answer: 1
    why: "Formatting belongs to the formatter. Prettier's docs recommend eslint-config-prettier for exactly this, and ESLint itself deprecated its formatting rules."
  - q: "Why does this repo keep TypeScript 6.0 installed next to TypeScript 7.0?"
    options: ["tsc 7 can't type-check React projects", "typescript-eslint's type-aware rules need the TypeScript JavaScript API, which 7.0 doesn't ship", "Vite requires TypeScript 6 to transpile"]
    answer: 1
    why: "ADR-0006: tsc 7 is authoritative for type-checking; typescript-eslint loads the 6.0 compiler API until TypeScript 7.1 ships its new API. Vite transpiles with esbuild and needs neither."
further_reading:
  - title: "TSConfig reference"
    url: "https://www.typescriptlang.org/tsconfig/"
  - title: "TSConfig reference: noUncheckedIndexedAccess"
    url: "https://www.typescriptlang.org/tsconfig/#noUncheckedIndexedAccess"
  - title: "TSConfig reference: erasableSyntaxOnly"
    url: "https://www.typescriptlang.org/tsconfig/#erasableSyntaxOnly"
  - title: "TypeScript 6.0 release notes (new defaults and deprecations)"
    url: "https://www.typescriptlang.org/docs/handbook/release-notes/typescript-6-0.html"
---

## Concept

Three tools, three questions:

- **tsc**: is the program type-correct? Its strictness is set in `tsconfig.json`.
- **ESLint with typescript-eslint**: does the program contain patterns that are type-correct
  but almost always bugs? The rules that matter most are **type-aware**: they ask the
  TypeScript checker what a value is.
- **Prettier**: what should the code look like? A formatter with deliberately few options,
  run on save and checked in CI.

Strictness is a **ladder**: you climb it one rung at a time, measuring each rung first.
Measured on this repo with tsc 7.0.2 and ESLint 9.12 on 2026-11-13:

1. `strict: true`, the default since TypeScript 6.0. This repo is here, plus
   `noFallthroughCasesInSwitch`.
2. Free rungs, zero errors then: `noUnusedLocals`, `noUnusedParameters` (both explicitly
   `false` in `tsconfig.app.json`), `noImplicitReturns`, `noImplicitOverride`,
   `verbatimModuleSyntax`, `erasableSyntaxOnly`. Turning them on now keeps them at zero.
3. Small rungs: `exactOptionalPropertyTypes` (3 errors in 2 files),
   `noPropertyAccessFromIndexSignature` (2 in 2).
4. `noUncheckedIndexedAccess`: 27 errors in 5 files, 20 of them in
   `src/shared/ui/markdownBlocks.ts`. A ratchet rung.
5. Lint presets, in typescript-eslint's own order: `recommended` (this repo, no type
   information, 0 problems in `src`), `recommended-type-checked` (11 problems in 2 files),
   `strict-type-checked` with `stylistic-type-checked` (53 problems in 15 files), plus rules
   no preset enables, such as `switch-exhaustiveness-check`.

Measure a flag without editing anything:
`node node_modules/typescript7/bin/tsc --noEmit -p tsconfig.app.json --noUncheckedIndexedAccess`.
A day later the same command reported 40 errors in 6 files, 31 of them in
`markdownBlocks.ts`, which had grown table support in the meantime. Un-ratcheted counts
drift with every change, which is why the ratchet compares each file with its own
baseline instead of one total.

## Coming from C/C++/Java/Python/JS

- **Go:** `gofmt` is Prettier with zero options, and the whole community accepted it. `go vet`
  is a type-aware linter shipped with the toolchain; staticcheck is the stricter rung.
  Go's compiler already rejects unused variables and imports, which TypeScript leaves to
  `noUnusedLocals`.
- **Java:** javac's `-Xlint` and Error Prone (bug patterns using type information) are the
  type-aware lint; Checkstyle the stylistic one; google-java-format the formatter.
- **C/C++:** `-Wall -Wextra -Werror` is the same ladder, `clang-tidy` the type-aware linter,
  `clang-format` the formatter.
- **Python:** mypy's flags and `--strict` mirror tsconfig; ruff or pylint lint; black is
  the "uncompromising" formatter, the closest relative of Prettier.
- **JS:** plain ESLint parses JavaScript only. typescript-eslint adds a TypeScript parser
  and, for type-aware rules, a running TypeScript program.

## How it works

The first two examples type-check cleanly with tsc 7 and pass this repo's own lint
configuration. Linted with `strict-type-checked`, `stylistic-type-checked` and
`switch-exhaustiveness-check`, using `parserOptions: { projectService: true }`, they report
(path prefix shortened):

```
01_promise_bugs.ts
  10:30  error  Expected a non-Promise value to be returned  @typescript-eslint/no-misused-promises
  16:18  error  Promise returned in function argument where a void return was expected  @typescript-eslint/no-misused-promises
  30:3   error  Promises must be awaited, end with a call to .catch, end with a call to .then with a rejection handler or be explicitly marked as ignored with the `void` operator  @typescript-eslint/no-floating-promises

02_silent_values.ts
  12:23  error  'lesson' will use Object's default stringification format ('[object Object]') when stringified  @typescript-eslint/no-base-to-string
  12:23  error  Invalid type "Lesson" of template literal expression  @typescript-eslint/restrict-template-expressions
  15:40  error  Prefer using nullish coalescing operator (`??`) instead of a logical or (`||`), as it is a safer operator  @typescript-eslint/prefer-nullish-coalescing
  20:11  error  Switch is not exhaustive. Cases not matched: "archived"  @typescript-eslint/switch-exhaustiveness-check

✖ 7 problems (7 errors, 0 warnings)
```

`projectService` asks TypeScript's language service for each file's program, which is what
makes these rules possible and what makes them slower: 3.8 s for this repo's `src` against
1.6 s for the syntax-only configuration (tsc 7 type-checks the same files in about 0.26 s).
The service is TypeScript's JavaScript API. TypeScript 7.0 ships no such API (a new one is
planned for 7.1), so ADR-0006 keeps `typescript` 6.0 installed for typescript-eslint and
calls `node_modules/typescript7/bin/tsc` for type-checking.

## Design notes

Climb in this order: take every free rung now, because a flag at zero errors only costs
something once someone breaks it. Then ratchet the expensive rungs: commit per-file counts,
fail CI on increases, lower the baseline as files are touched. The third example is the
whole mechanism in twenty lines.

Two findings from measuring this repo. ADR-0006 records the owner's decision that
type-aware ESLint is on, but `eslint.config.js` still extends the syntax-only
`tseslint.configs.recommended`; switching to `recommendedTypeChecked` with `projectService`
is the next rung, and its 11 findings include the unnecessary double cast in
`src/features/learning/api.ts` discussed in ts.architecture.patterns. Second, the
`ignores` list holds only `dist`, so `npm run lint` also lints the generated
`public/lesson-libs/bundle.js` and reports errors from it.

Prettier is not installed in this repo. If it is added, keep its job separate: format on
save, `prettier --check .` in CI, and eslint-config-prettier (the integration Prettier's
documentation recommends) to switch off any lint rule that would fight it. Prettier's
documentation advises against running it as an ESLint rule (eslint-plugin-prettier): slower,
and formatting turns into red squiggles. ESLint 9.22 added `defineConfig()` in
`eslint/config`, and typescript-eslint now marks its own `tseslint.config()` helper as
deprecated in its favour. This repo's ESLint 9.12 predates it (the same helpers ship
separately as `@eslint/config-helpers`), so `tseslint.config()` stays until one of the two
dependency changes is made.
