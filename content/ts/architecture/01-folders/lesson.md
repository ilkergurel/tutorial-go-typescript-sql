---
id: ts.architecture.folders
title: "Feature-based folders and layering: UI, state, API client"
track: ts
module: architecture
order: 1
level: intermediate
minutes: 13
since: es2015
prerequisites: [ts.objects-classes.modules, ts.react.components]
compare_with: [go.design.packages, go.tooling.project-layout, go.design.layers, design.architecture.styles]
tags: [architecture, folders, layering, dependencies, modules, eslint]
summary: >
  Group code by feature, not by file kind, and let dependencies point one way:
  app, then features, then shared. Features never import each other; the app
  composes them, and a linter keeps the rule from eroding.
examples:
  - id: dependency-check
    file: examples/01_dependency_check.ts
    title: "Checking this app's real imports: upward, sideways, and in a cycle"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-20"
        note: >
          The data is real: every import in src/features and src/shared that
          leaves its own folder, collected from this repo on 2026-11-13. app/ is
          left out because importing everything below it is its job.
      - lines: "22-32"
        note: >
          The whole policy is two comparisons. "upward" means a lower layer
          knows a higher one (shared/ depending on a feature); "sibling" means a
          feature reaches into another feature instead of meeting it in app/.
          The return type is a union, so callers can't invent a third verdict.
      - lines: "36-43"
        note: >
          Eleven of fifteen crossings break a rule. Type-only imports are
          counted separately: they disappear from the JavaScript, so they can't
          cause a runtime cycle, but runner couldn't be reused without lessons
          while RunPanel needed the Lesson type.
      - lines: "46-50"
        note: >
          On that date learning, lessons and runner each depended on the other
          two: three folders, one feature. A per-import lint rule flags the imports but
          never names the cycle; a graph does.
  - id: compose-in-app
    file: examples/02_compose_in_app.tsx
    title: "UI, state and API client in one feature; two features meeting in app/"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "6-15"
        note: >
          The bottom of the feature: a transport type from shared/ and an API
          client that turns paths and JSON into the feature's own types. Only
          this layer knows URLs. ReturnType derives the client's type from the
          factory, so the two can't drift apart.
      - lines: "17-30"
        note: >
          The state layer is a hook that knows the client, not the component
          that renders it. api is an effect dependency, which is why main
          creates it once instead of on every render.
      - lines: "32-42"
        note: >
          ProgressBadge knows only the hook. LessonView belongs to a different
          feature and takes a ReactNode slot, so it has no import of progress at
          all: the dependency was turned into a prop.
      - lines: "44-48"
        note: >
          The route is the only module that names both features. This repo now
          does the same: app/routes/LessonPage.tsx fills LessonContent's slots,
          which removed the cycles the first example found.
      - lines: "50-57"
        note: >
          The real transport is chosen once at the top, like the app's pickApi()
          choosing between the backend and localStorage. A test passes a fake the
          same way, with no module mocking.
use_cases:
  - "A React app past a handful of screens: one folder per domain noun (curriculum, lessons, runner, learning) with UI, hooks and API client inside it"
  - "Deleting or rewriting a feature: if only app/ imports it, the blast radius is one routes file"
  - "Splitting work between people or teams: a feature folder is a unit of ownership with an explicit surface"
  - "Reviewing a pull request: an import from ../otherFeature/ or from shared/ into features/ is a design change, not a detail"
strengths:
  - "Unlike a layer-first tree (components/, hooks/, services/), one feature lives in one folder, so a change touches one place and deleting the folder deletes the feature"
  - "One direction of dependencies makes testing cheap: a feature's UI gets its client as a parameter and needs no module mocking"
  - "Enforceable with tools you already run: ESLint's no-restricted-imports per folder, no extra build step"
  - "Composition through slots (children, ReactNode props, callbacks) turns a cross-feature import into data flowing down from app/, which React already does well"
weaknesses:
  - "Unlike Go, which rejects import cycles at compile time, neither ES modules nor tsc forbid them: the rule exists only as convention plus lint"
  - "Lint rules match import strings, not resolved files: relative paths make patterns depth-dependent, and a path alias (@/features/...) is needed for robust rules"
  - "Deciding what is shared is a judgment call; a shared/ folder that grows domain types becomes a second, unstructured feature"
  - "Strict 'no sibling imports' sometimes costs a prop or two of plumbing that a direct import would have avoided"
pitfalls:
  - title: "shared/ that knows a domain"
    detail: "A shared component importing a feature's type makes everything that uses shared/ depend on that feature. With structural typing, shared code can declare the shape it needs and the feature's type still fits."
    bad: "// shared/ui/LevelBadge.tsx\nimport type { Level } from '../../features/curriculum/types';"
    good: "// shared/ui/LevelBadge.tsx\ntype Level = 'basic' | 'intermediate' | 'advanced';\n// curriculum's Level is the same union, so it is assignable"
  - title: "Composing features inside a feature"
    detail: "A page component that imports three features' components belongs in app/. Inside a feature it makes that feature depend on all the others, and they usually depend back."
    bad: "// features/lessons/LessonContent.tsx\nimport { RunPanel } from '../runner/RunPanel';\nimport { LessonNotes } from '../learning/LessonNotes';"
    good: "// app/routes/LessonPage.tsx\n<LessonContent lesson={lesson}\n  runPanel={(example) => <RunPanel example={runnable(lesson, example)} onRan={...} />}\n  notes={<LessonNotes lessonId={lesson.id} lessonTitle={lesson.title} />} />"
  - title: "Configuration living in a random feature"
    detail: "Transport settings such as an API base URL are needed by every client. Put them in shared/, not in the first feature that needed them (this app kept API_BASE in features/runner/types.ts, so learning imported runner, until it moved to shared/api.ts)."
  - title: "Fixing a cycle with a barrel"
    detail: "Re-exporting both features from an index.ts hides the cycle from the import list and keeps it in the module graph. Merge the features, move the shared part down, or invert the dependency with a slot or callback."
quiz:
  - q: "features/runner/RunPanel.tsx used to import markLesson from features/learning/store to record that the learner ran an example. What is the layering-friendly fix?"
    options: ["Re-export markLesson from shared/index.ts", "Give RunPanel an onRan callback prop and let app/ pass markLesson in", "Move RunPanel into features/learning"]
    answer: 1
    why: "The callback turns runner's dependency on learning into a value passed down by the composition root, which is what the app now does. Re-exporting hides the import without removing it; moving the file only relocates the cycle."
  - q: "Module a.js imports b.js, and b.js imports a.js and reads a `const` from it at its top level. What happens?"
    options: ["The bundler or tsc reports an import cycle", "A ReferenceError at run time, if a.js hasn't evaluated that const yet", "Nothing: ES modules evaluate both at once"]
    answer: 1
    why: "ES modules allow cycles. A let/const/class binding read before its module has initialized it throws ReferenceError; function declarations are initialized earlier and survive. Go would reject the cycle at compile time."
  - q: "Which import breaks the layering app → features → shared?"
    options: ["app/routes/LessonPage.tsx importing features/lessons/LessonContent", "features/lessons/ExampleView.tsx importing shared/ui/Markdown", "shared/ui/LevelBadge.tsx importing a type from features/curriculum/types"]
    answer: 2
    why: "The first two point down. The third points up, so every user of the shared badge transitively depends on the curriculum feature; being type-only makes it harmless at run time but not in the design."
further_reading:
  - title: "MDN: Understanding the module graph (cyclic imports)"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Modules/Module_graph#cyclic_imports"
  - title: "MDN: JavaScript modules"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Guide/Modules"
  - title: "React: Passing JSX as children"
    url: "https://react.dev/learn/passing-props-to-a-component#passing-jsx-as-children"
  - title: "TSConfig reference: paths"
    url: "https://www.typescriptlang.org/tsconfig/#paths"
  - title: "The Go Programming Language Specification: Import declarations"
    url: "https://go.dev/ref/spec#Import_declarations"
---

## Concept

Organize `src/` by **what the code is about**, not by what kind of file it is. A
layer-first tree (`components/`, `hooks/`, `services/`, `types/`) spreads every feature
over four folders; a feature-first tree keeps one feature in one folder and makes its
boundary visible. This app uses the common three tiers:

- `src/app/`: `main.tsx`'s routes, the layout, the sidebar, the route components. This is
  the **composition root**, the only code allowed to know every feature.
- `src/features/<name>/` (`curriculum`, `lessons`, `runner`, `learning`): one domain each,
  with its own UI, state and API client.
- `src/shared/`: code without domain knowledge (`shared/ui/Markdown.tsx`), usable from
  anywhere.

Two rules make the structure pay off:

1. **Dependencies point down**: `app → features → shared`. Inside a feature they point
   down too: UI → state (hooks, stores) → API client → transport, with the feature's types
   at the bottom.
2. **Features don't import each other.** When two features must meet, `app/` composes
   them: one feature leaves a slot (`children`, a `ReactNode` prop, a callback) and the
   route fills it. Domain-free code moves down into `shared/`; two features that can't
   stop importing each other are one feature and should be merged.

The payoff is local reasoning: you can rewrite, test or delete `features/runner` knowing
that only `app/` can break.

## Coming from C/C++/Java/Python/JS

- **Go:** a feature folder plays the role of a package, and Go's compiler rejects import
  cycles outright. ES modules accept them: MDN's module-graph guide shows a module reading
  a `const` from a module that hasn't finished evaluating and getting
  `ReferenceError: Cannot access 'a' before initialization`. tsc doesn't report cycles
  either, so the rule has to come from you and a linter.
- **Java:** package-by-feature versus package-by-layer is the same debate. There is no
  package-private: an ES module exports what it exports, so "internal to the feature" is a
  convention that lint enforces, the way ArchUnit tests enforce layering in a Java build.
- **Python:** Django apps are feature folders. A circular import surfaces as `ImportError:
  cannot import name ... (most likely due to a circular import)`. In ES modules the same
  mistake is a `ReferenceError` at run time; under CommonJS, a half-filled `exports` object.
- **C/C++:** the layering you'd enforce with include directories and library link order.
  A cycle between static libraries shows up as a link error; here it shows up at run time,
  or not at all until the load order changes.
- **JS/React:** Redux-era trees grouped by `actions/`, `reducers/`, `components/`. Hooks
  made colocation natural: the hook lives next to the component that uses it.

## How it works

ESLint's core `no-restricted-imports` rule, scoped per folder in a flat config, is enough
to enforce both rules. This configuration was run against this repo's `src/` (ESLint
9.12.0, 2026-11-13):

```js
{
  files: ["src/shared/**"],
  rules: {
    "no-restricted-imports": ["error", { patterns: [
      { regex: "/(features|app)/", message: "shared/ is the bottom layer: take the value as a prop." },
    ] }],
  },
},
{
  files: ["src/features/**"],
  rules: {
    "no-restricted-imports": ["error", { patterns: [
      { regex: "^\\.\\./(?!\\.\\.)", message: "Features don't import each other: compose them in app/." },
      { regex: "/app/", message: "features/ must not depend on app/." },
    ] }],
  },
},
```

It reports 13 problems; an excerpt, with the absolute path prefix removed:

```
src\features\learning\api.ts
  8:1  error  '../runner/types' import is restricted from being used by a pattern. Features don't import each other: compose them in app/  no-restricted-imports

src\shared\ui\LevelBadge.tsx
  1:1  error  '../../features/curriculum/types' import is restricted from being used by a pattern. shared/ is the bottom layer: take the value as a prop  no-restricted-imports

✖ 13 problems (13 errors, 0 warnings)
```

Thirteen, not eleven as in the first example, because the rule counts import statements:
`RunPanel.tsx` imports `lessons/types` twice (once with `import type`), and
`lessons/types.ts` both imports and re-exports `TrackId`. Swapping in typescript-eslint's
`@typescript-eslint/no-restricted-imports` with `allowTypeImports: true` on the feature
pattern leaves 7: the six value imports between features plus `LevelBadge`.

The rule matches the **import string as written**, not the resolved file. The regex
`^\.\./(?!\.\.)` means "one level up, then into a sibling", which works only because every
feature here is one flat folder. A path alias (`@/` is already configured in
`tsconfig.app.json` and `vite.config.ts`) makes patterns independent of depth. Plugins that
resolve real paths, such as eslint-plugin-import's `no-restricted-paths`, describe zones
directly; they are not installed in this repo.

After the fixes in the design notes (2026-11-14), the configuration reports no problems.
It is now part of the repo's `eslint.config.js`, with one more feature pattern for the
`@/` alias, so `npm run lint` keeps the layering from eroding.

## Design notes

This app is a fair case study because it is small and honest about its history. The
first example's findings, and the fix applied to each on 2026-11-14:

- `shared/ui/LevelBadge.tsx` imported `Level` from `features/curriculum`. The badge now
  declares the union it renders; curriculum's identical `Level` is assignable to it
  because TypeScript compares shapes, not names.
- `features/learning/api.ts` imported `API_BASE` from `features/runner/types.ts`.
  Transport configuration belongs below the features: it now lives in `shared/api.ts`.
- `features/lessons/LessonContent.tsx` imported `RunPanel`, `LessonNotes` and
  `LessonQuiz`, and `RunPanel` imported `markLesson` from `learning/store`. That was page
  composition done inside a feature, and it tied lessons, learning and runner into one
  cycle. The composition moved to `app/routes/LessonPage.tsx`, which passes the panels in
  as slots, and `RunPanel` got an `onRan` callback. ADR-0005 credits "no cross-feature
  store reads" with keeping feature folders independent; this import was the one place
  the code disagreed.

Type-only imports (`Lesson`, `Example`) are the mildest coupling and often acceptable:
treat `features/lessons/types.ts` as the domain kernel others may read, or move those
types into `shared/`. This app took a third way: `RunPanel` takes a `RunnableExample` and
the quiz a list of `QuizQuestion`s, each declared by its own feature, and the lesson page
maps a lesson onto them, so a mismatch is a compile error in one place. Decide
deliberately and encode the decision in the lint config, so the next import is a
conscious exception rather than drift.

Two smaller habits keep the structure cheap. Don't give each feature a barrel `index.ts`
as its "public API": lesson ts.objects-classes.modules showed what barrels cost, and a
per-folder lint rule gives the same guarantee. And start every feature flat; add
sub-folders (`ui/`, `api/`) only when a feature outgrows a dozen files.
