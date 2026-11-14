---
id: ts.mental-model.tsconfig
title: "Essential tsconfig options and the strict flags"
track: ts
module: mental-model
order: 2
level: basic
minutes: 13
since: ts5.0
prerequisites: [ts.mental-model.erasable]
compare_with: [ts.tooling.lint, go.tooling.go-command, ts.everyday-types.null]
tags: [tsconfig, strict, noUncheckedIndexedAccess, exactOptionalPropertyTypes, verbatimModuleSyntax]
summary: >
  strict (the default since TypeScript 6.0) is a family of checks; three flags
  outside it close the holes that bite most often: unchecked index reads, optional
  properties that may hold undefined, and imports whose fate depends on elision.
examples:
  - id: strict-family
    file: examples/01_strict_family.ts
    title: What strict turns on, and what it catches
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-7"
        note: >
          verbatimModuleSyntax (TS 5.0): imports without `type` stay in the output,
          imports with `type` disappear, and nothing in between is guessed. Node's
          type stripping and esbuild see one file and cannot know that Prefs is only
          a type, so the rule makes the source say it. It implies isolatedModules.
      - lines: "9-15"
        note: >
          noImplicitAny is the flag that makes TypeScript a typed language rather
          than optional hints: a parameter tsc cannot infer must be annotated.
          Variables and return types are still inferred, so annotate parameters
          and exported signatures, not every local.
      - lines: "17-22"
        note: >
          strictNullChecks puts undefined into find()'s return type. Without it,
          null and undefined are members of every type, which is Java's
          NullPointerException model. Lesson ts.everyday-types.null goes deeper.
      - lines: "24-35"
        note: >
          useUnknownInCatchVariables (TS 4.4, part of strict): JavaScript can throw
          any value, here a string. The cast on line 33 compiles and prints
          undefined; narrowing with instanceof is the honest version.
  - id: unchecked-index
    file: examples/02_unchecked_index.ts
    title: noUncheckedIndexedAccess and a Turkish lookup key
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-12"
        note: >
          Record<string, string> promises a string for every key, which no object
          can keep. Without the flag tsc believes it; the `!` makes that belief
          visible. "IŞIK".toLowerCase() is "işik" (dotted i), so the lookup misses
          and the call crashes on undefined.
      - lines: "14-21"
        note: >
          With the flag (TS 4.1, not part of strict) the read is string | undefined
          and the compiler forces the miss to be handled. Lowercasing with the "tr"
          locale maps I to ı and İ to i, which is what Turkish logins need.
      - lines: "32-34"
        note: >
          Array index reads and destructuring get undefined too. for...of and
          map() don't, because they only visit elements that exist: prefer them to
          index loops and the flag costs little.
  - id: exact-optional
    file: examples/03_exact_optional.ts
    title: exactOptionalPropertyTypes and the spread that erases a default
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "5-13"
        note: >
          Without the flag, `theme?: T` also accepts an explicit undefined. The key
          then exists ("theme" in loose is true) and object spread copies it,
          overwriting the default with undefined.
      - lines: "15-20"
        note: >
          With the flag (TS 4.4, not part of strict), `theme?: T` means absent or a
          real T. If you really mean "may be present and undefined", write
          `theme?: T | undefined` and the type says so.
      - lines: "22-28"
        note: >
          Leaving the key out is the only way to say "not set", so the spread keeps
          the default. The same distinction matters for PATCH bodies, where a
          missing field means "don't change" and null means "clear".
      - lines: "30-31"
        note: >
          JSON.stringify drops undefined properties, so both objects print as {}.
          Logging the object won't show the bug; the flag catches it at compile time.
use_cases:
  - "Starting a project: strict plus noUncheckedIndexedAccess, exactOptionalPropertyTypes, verbatimModuleSyntax and erasableSyntaxOnly from day one, when turning them on costs nothing"
  - "Tightening an existing codebase one flag at a time: run tsc with the flag on the command line, count the errors, fix, then commit the tsconfig change"
  - "Reviewing a pull request that adds `!` or `as` around index reads: each one is a place where the checker was overruled"
strengths:
  - "Unlike javac's or Go's fixed rules, strictness is adjustable per project, so a JS codebase can be migrated gradually and a new one can start strict"
  - "TypeScript 6.0 and 7.0 default strict to true, so a fresh tsconfig already checks null, implicit any and catch variables"
  - "noUncheckedIndexedAccess catches the missing-key class of bugs that Java only reports at run time (a null from Map.get) and Go hides behind zero values"
weaknesses:
  - "The most useful extra flags are not part of strict, so `strict: true` alone still lets arr[i] and record[key] lie"
  - "noUncheckedIndexedAccess adds noise to index loops (`for (let i...)` reads are T | undefined even inside bounds); there is no bounds analysis"
  - "exactOptionalPropertyTypes trips over library types that write `x?: T` but would accept undefined: passing `{ x: maybe }` then needs a conditional spread or a cast"
pitfalls:
  - title: "Assuming strict covers index reads"
    detail: "strict does not include noUncheckedIndexedAccess or exactOptionalPropertyTypes. Turn them on explicitly."
  - title: "Silencing the flag with `!` everywhere"
    detail: "A `!` on every index read restores the unchecked behaviour with extra typing. Use for...of, a Map with .get(), or an explicit undefined check that does something sensible."
    bad: "const role = roles[key]!;"
    good: "const role = roles[key] ?? \"guest\";"
  - title: "Relying on a missing tsconfig field meaning \"off\""
    detail: "Defaults changed in 6.0 (strict, module, target, types). Write the important options explicitly so the file reads the same under every compiler version."
quiz:
  - q: "Which flag is NOT turned on by `strict: true`?"
    options: ["strictNullChecks", "noImplicitAny", "noUncheckedIndexedAccess", "useUnknownInCatchVariables"]
    answer: 2
    why: "noUncheckedIndexedAccess (and exactOptionalPropertyTypes) are separate opt-ins; the other three are members of the strict family."
  - q: "With noUncheckedIndexedAccess, what is the type of `names[0]` when `names: string[]`?"
    options: ["string", "string | undefined", "string | null"]
    answer: 1
    why: "The flag adds undefined to every index-signature and array element read, because the index may be out of range."
  - q: "Under verbatimModuleSyntax, what happens to `import type { User } from \"./user\"` in the JavaScript output?"
    options: ["It is kept as written", "It is dropped entirely", "It is rewritten to a side-effect import"]
    answer: 1
    why: "Imports with the type modifier are erased; imports without it are kept verbatim. That is the whole rule."
further_reading:
  - title: "TSConfig reference: strict"
    url: "https://www.typescriptlang.org/tsconfig/#strict"
  - title: "TSConfig reference: noUncheckedIndexedAccess"
    url: "https://www.typescriptlang.org/tsconfig/#noUncheckedIndexedAccess"
  - title: "TSConfig reference: exactOptionalPropertyTypes"
    url: "https://www.typescriptlang.org/tsconfig/#exactOptionalPropertyTypes"
  - title: "TSConfig reference: verbatimModuleSyntax"
    url: "https://www.typescriptlang.org/tsconfig/#verbatimModuleSyntax"
---

## Concept

`tsconfig.json` decides which program you are writing. The same file can be loosely typed
JavaScript or a tightly checked one depending on a handful of flags:

- **`strict`** is an umbrella for the "strict family": `noImplicitAny`, `strictNullChecks`,
  `strictFunctionTypes`, `strictBindCallApply`, `strictPropertyInitialization`,
  `strictBuiltinIteratorReturn`, `noImplicitThis` and `useUnknownInCatchVariables`. New
  members join the family in later releases, so `strict: true` gets stricter when you
  upgrade. **Since TypeScript 6.0 it defaults to true** (also in 7.0). `alwaysStrict`
  went the other way: 6.0 took it out of the family and turned it on unconditionally, so
  `alwaysStrict: false` is deprecated in 6.0 and rejected by 7.0.
- **`noUncheckedIndexedAccess`** (4.1): `arr[i]` and `record[key]` include `undefined`.
- **`exactOptionalPropertyTypes`** (4.4): `x?: T` means "absent or a T", not "or undefined".
- **`verbatimModuleSyntax`** (5.0): imports and exports are emitted exactly as written,
  minus anything marked `type`.

None of the last three is part of `strict`.

This repo's `tsconfig.app.json` is a typical Vite setup: `strict`, `isolatedModules`,
`noEmit` with `allowImportingTsExtensions`, `moduleResolution: "bundler"`,
`noFallthroughCasesInSwitch`. It does not turn on the three extra flags. Try them from the
command line without editing anything:
`node node_modules/typescript7/bin/tsc --noEmit -p tsconfig.app.json --noUncheckedIndexedAccess`
lists every index read the app takes on trust, such as `lines[i]` in a parsing loop or
`m[1]` from a regex match. Run it for the current count: it changes with the code.

## Coming from C/C++/Java/Python/JS

- Java: `javac`'s rules are fixed. TypeScript's are configurable because it had to absorb
  untyped JavaScript gradually. `strictNullChecks` is the switch between Java's model
  (null is in every reference type) and Kotlin's (it isn't, unless you say `T?`).
- C/C++: comparable to `-Wall -Wextra -Werror`, except that a TS flag changes what the
  type system believes (`T` vs `T | undefined`), not just which warnings print.
- Python: mypy's `--strict` is the same idea: opt-in checks bundled under one switch.
- Go: the compiler has no strictness knobs; its rules are the same for every module. A
  missing map key gives the zero value; in TS without `noUncheckedIndexedAccess` it
  gives `undefined` typed as `T`.
- JS: the flags change nothing at run time. They only change which programs tsc accepts.

## Design notes

For new code, start with everything on: `strict`, `noUncheckedIndexedAccess`,
`exactOptionalPropertyTypes`, `verbatimModuleSyntax`, `erasableSyntaxOnly`. Each one
turns a whole category of runtime surprise (the missing key, the undefined that overwrote
a default, the import that only existed for types, the enum Node can't strip) into a
compile error, and the cost is lowest on an empty codebase. Tighten an existing project
one flag at a time from the command line, as above, and commit the flag together with
the fixes. Write important options explicitly even when they match today's default: the
6.0 default changes showed that "unset" means different things to different compiler
versions.
