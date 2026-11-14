---
id: inside.codebase.content-pipeline
title: "Content loading and full-text indexing"
track: inside
module: codebase
order: 5
level: intermediate
minutes: 14
since: pg18
prerequisites: [ts.node.type-stripping, ts.tooling.vite]
compare_with: [sql.search.fts, inside.codebase.contract, go.files-io.embed, sql.tables-types.text-numbers]
tags: [content, build-step, codegen, bundle, full-text-search, tsvector, turkish]
summary: >
  Lessons go from files to the page through a build script, not a database:
  scripts/build-lessons.mjs validates content/ and writes an index plus one file
  per lesson for the frontend. There is no lessons table, no full-text index and no
  Ctrl+K search; this lesson shows what exists and what the planned index would take.
examples:
  - id: mini-contentcheck
    file: examples/01_mini_contentcheck.ts
    title: "contentcheck in small: collect every error, write nothing if there is one"
    runtime: ts-node
    check: exact
    annotations:
      - lines: "4-6"
        note: >
          The real script reads the ids from `content/curriculum.yaml`, and its
          allow-list has nine hosts (go.dev, pkg.go.dev, typescriptlang.org,
          developer.mozilla.org, react.dev, postgresql.org, nodejs.org, with
          www. variants). The H2 list is copied verbatim.
      - lines: "15-16"
        note: >
          contentcheck scans every example file for LEARN markers, this one
          included. A literal marker naming a non-curriculum id here failed this
          very lesson's check while it was being written, hence the escape.
      - lines: "25-27"
        note: >
          The frontmatter regex is the real one. The real parser is the `yaml`
          package, and validation against `lesson.schema.json` is a hand-written
          subset validator (about 80 lines) instead of ajv, as the script's
          comment explains.
      - lines: "28-30"
        note: >
          The directory decides the id: `<track>/<module>/<NN-slug>` must be
          `track.module.slug`. `--filter` matches on the directory too, so a
          lesson with a broken id can still be selected and checked.
      - lines: "32-34"
        note: >
          The real H2 check, verbatim: the headings present must appear in the
          fixed order, and the optional "How it works" may be missing.
      - lines: "39-43"
        note: >
          The real marker regex: only id-shaped arguments count, so prose such
          as `LEARN(<lesson-id>)` in CLAUDE.md isn't a marker. On an unfiltered
          run the real script also scans `server/`, `src/` and `scripts/`.
      - lines: "45-50"
        note: >
          Errors are collected, not thrown, so one run reports all of them. Any
          error means exit 1 and no file written: the frontend never imports a
          half-valid bundle.
  - id: bundle-lookup
    file: examples/02_bundle_lookup.ts
    title: "The generated module, the lookups, and the search that doesn't exist"
    runtime: ts-node
    check: exact
    annotations:
      - lines: "1-17"
        note: >
          The shape of `content/lessons.bundle.json` and of the private bundle
          verify-lessons reads: every lesson's frontmatter plus `dir` and `body`,
          and three maps keyed by path: example sources, expected outputs,
          fixtures. Until 2026-11-14 the frontend imported this whole object as
          one `.ts` module; it now gets the same data split per lesson.
      - lines: "19-20"
        note: >
          A lookup by id, `Array.find` over every lesson, as the app's old
          `lessonById` did. `load.ts` now keeps the index in a Map by id, and its
          imports are typed by assertion (`index as LessonSummary[]`,
          `import.meta.glob<Lesson>`): no runtime check, it trusts the generator
          that validated the data.
      - lines: "22-28"
        note: >
          The sidebar and lesson list come from `curriculum.yaml`; the content
          comes from the generated lesson data. `LessonPage` joins them by id,
          and a curriculum lesson with no content renders the "not authored yet"
          box.
      - lines: "30-32"
        note: >
          The key schemes: `<dir>/<file>` for sources, `<dir>/<example id>` for
          expected output (a lesson's own file drops the `<dir>/`). Fixtures lose their `fixtures/` prefix on the way to
          the runner, so the program opens `seed.csv` in its working directory.
      - lines: "35-41"
        note: >
          `toLowerCase` turns "İ" into "i" plus a combining dot, so "izolasyon"
          misses. `toLocaleLowerCase('tr')` fixes that and breaks the other
          side: "I" becomes "ı", so a query typed on a non-Turkish keyboard
          misses. A search has to fold İ, I, ı and i to one letter.
      - lines: "42"
        note: >
          The proof: U+0069 U+0307, two code points for one visible letter.
          `.length` is 2, and a naive `startsWith` or highlight offset is off by one.
  - id: planned-index
    file: examples/03_planned_index.sql
    title: "The lessons table and search index the docs planned (not built)"
    runtime: sql
    check: exact
    annotations:
      - lines: "1-10"
        note: >
          docs/ARCHITECTURE.md's ERD: a `search tsvector GENERATED ALWAYS AS
          (...) STORED` column with a GIN index, so the vector is computed on
          write and every query is an index lookup. Weights mark title (A)
          above body (B).
      - lines: "17-18"
        note: >
          Read what the stemmer did: "satırları" in the body became `satır`, but
          the title's "Satır" became `sa`. PostgreSQL's Turkish (Snowball)
          stemmer is aggressive and not always consistent; measure it on your
          vocabulary before relying on it.
      - lines: "20-24"
        note: >
          `kilit` matched "kilitleri" in one title and "kilitler" in the other
          body, and the title match ranks higher. `websearch_to_tsquery`
          accepts what people type (quotes, or, -word) without raising syntax
          errors.
      - lines: "26-31"
        note: >
          Lowercasing uses the database's own case mapping. This one (builtin
          C.UTF-8, see `scripts/setup-local-db.sql`) maps I to i, so "YALITIM"
          becomes `yalit` and misses the `yalıt` stored for "Yalıtım". The same
          trap as the JS filter, one layer down.
      - lines: "33-35"
        note: >
          PostgreSQL 18 made generated columns VIRTUAL by default (computed on
          read), and those can't be indexed, which is why the ERD insists on
          STORED. The script stops here: an error ends a sql run.
use_cases:
  - "Content that should be reviewed like code: files in the repo, validated by a script, turned into a generated module"
  - "Small, read-only data sets a frontend can ship whole (docs, help text, a few hundred records) instead of serving through an API"
  - "Deciding between a client-side filter and server-side full-text search for a Ctrl+K box"
strengths:
  - "No moving parts at runtime: lessons render with the backend down, and a broken lesson.md is caught by the build script, not by a user"
  - "Content diffs are ordinary text diffs, and every check (schema, ids, links, LEARN markers) runs before anything is written"
  - "`verify-lessons` builds a private bundle with `--out` and `--filter`, so several authors can check their own lessons without racing on the shared files"
weaknesses:
  - "Each lesson is a separate chunk: opening one costs a request and a loading state, and a page that outlives a rebuild can ask for a chunk file that no longer exists (it shows an error and asks for a reload)"
  - "Nothing regenerates it automatically: not `npm start`, not `npm run build`'s prebuild. After editing content you run `node scripts/build-lessons.mjs` yourself"
  - "No search of any kind, and lesson ids join three places (curriculum.yaml, the generated lesson data, app-db rows) with no foreign key: renaming an id silently orphans its progress, notes and bookmarks"
  - "The import is typed with `as`, so a generator bug that changes the shape surfaces as an `undefined` deep in a component, not as an error at load"
pitfalls:
  - title: "Expecting an edited lesson.md to show up"
    detail: "The page reads the generated files. Run `node scripts/build-lessons.mjs` (Vite then hot-reloads), or `npm run gen:curriculum` after editing curriculum.yaml."
  - title: "Renaming a lesson id"
    detail: "Progress, notes, bookmarks, quiz answers and run history are keyed by the id as text. Treat ids as immutable, as the schema's description says."
  - title: "Lowercasing Turkish with one function"
    detail: "Neither `toLowerCase` nor `toLocaleLowerCase('tr')` makes İ, I, ı and i match each other, and neither does PostgreSQL's `lower` under a non-Turkish locale. Fold explicitly on both sides of the comparison."
    bad: "title.toLowerCase().includes(query.toLowerCase())"
    good: "const fold = (s: string) => s.toLocaleLowerCase('tr').replace(/ı/g, 'i').normalize('NFD').replace(/\\p{M}/gu, '');\nfold(title).includes(fold(query))"
quiz:
  - q: "When you open a lesson, where does its text come from?"
    options:
      - "Generated files bundled by Vite: an index in the main chunk, the lesson itself in a chunk of its own"
      - "GET /api/lessons/{id}, from the lessons table"
      - "The backend reads content/ on each request"
    answer: 0
    why: "There is no lessons route or table. `load.ts` imports `lessonIndex.generated.ts` for the header and loads `generated/<id>.json` with import() for the content."
  - q: "You fix a typo in a lesson.md while `npm start` is running. What makes the page show it?"
    options:
      - "Nothing, the backend re-indexes content on change"
      - "Running `node scripts/build-lessons.mjs`; Vite then reloads the regenerated module"
      - "Restarting the backend"
    answer: 1
    why: "No watcher, SSE event or startup loader exists. The build script is the only writer of the generated files."
  - q: "Why does the query IZOLASYON miss the title İzolasyon with both JavaScript lowercasings?"
    options:
      - "JavaScript can't handle Turkish letters"
      - "includes is case-sensitive"
      - "toLowerCase maps İ to i plus a combining dot, and the Turkish locale maps I to ı: neither gives plain i on both sides"
    answer: 2
    why: "The example prints both results and the two code points of the lowercased İ."
further_reading:
  - title: "PostgreSQL: Tables and indexes for text search"
    url: "https://www.postgresql.org/docs/current/textsearch-tables.html"
  - title: "PostgreSQL: Controlling text search (websearch_to_tsquery, ranking)"
    url: "https://www.postgresql.org/docs/current/textsearch-controls.html"
  - title: "PostgreSQL: Generated columns (stored vs virtual)"
    url: "https://www.postgresql.org/docs/current/ddl-generated-columns.html"
  - title: "MDN: String.prototype.toLocaleLowerCase"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/String/toLocaleLowerCase"
---

## Concept

The curriculum's goal for this lesson is "files to lessons table to Ctrl+K". Two of those
three don't exist. What exists is shorter:

1. **Files.** `content/<track>/<module>/<NN-slug>/` holds `lesson.md` (YAML frontmatter
   plus prose), `examples/`, `expected/` and `fixtures/`. `content/curriculum.yaml` holds
   the full plan, including lessons nobody has written yet.
2. **Two generators, run by hand.** `npm run gen:curriculum` turns curriculum.yaml into
   `src/features/curriculum/curriculum.generated.ts` (sidebar, tracks, paths).
   `node scripts/build-lessons.mjs` validates every lesson and writes
   `content/lessons.bundle.json` (everything), `src/features/lessons/lessonIndex.generated.ts`
   (each lesson's title, header facts and unconfirmed claims) and
   `src/features/lessons/generated/<lesson id>.json` (one lesson's full content).
3. **A static import and a lazy one.** `src/features/lessons/load.ts` imports the index,
   which Vite bundles into the app. It lists the lesson files with `import.meta.glob`, so
   Vite emits each as its own chunk, and `LessonPage` loads one with `import()` when the
   lesson opens: the header renders from the index at once, the tabs when the chunk
   arrives.

The backend never sees lesson content. Its migrations
(`server/cmd/stacklab/migrations`) create `schema_info`, `runs`, `progress`, `notes`,
`bookmarks` and `quiz_answers`, each keyed by `lesson_id text` without a foreign key,
because there is no `lessons` table to reference. There is no `tsvector` column, no
search endpoint, and no Ctrl+K handler anywhere in `src/`.

## Coming from C/C++/Java/Python/JS

- **Static site generators** (Hugo, Jekyll, Sphinx, Docusaurus) are the closest model:
  Markdown in, a validated build artefact out, no database. StackLab's artefacts are a
  JavaScript module and one JSON file per lesson instead of HTML pages.
- **Java**: comparable to resources processed at build time into generated classes
  (annotation processors, protobuf). The generated file is checked in, like
  `curriculum.generated.ts`, so the app builds without running the generator first.
- **C/C++**: `xxd -i` or `#embed` (C23) turns a file into data in the binary; here the
  "binary" is the JS bundle. Go's `embed.FS` is the backend equivalent and is used for
  migrations, not content.
- **Python**: like loading a validated YAML file at import time, except the validation
  happens once at build time instead of on every start.

## How it works

`build-lessons.mjs` walks `content/`, and for each `lesson.md` it checks the frontmatter
against `lesson.schema.json`, the id against the directory, unique ids, that every
`prerequisites` and `compare_with` id exists, that example, fixture and expected files
exist, the H2 order, official-docs-only links, `nondeterminism_note` on `check: none`,
and LEARN markers. On success it writes the bundle and the frontend's files, each only
when its content changed, so an unchanged lesson doesn't reload open tabs; on any error
it writes nothing and exits 1.

Three flags exist for `npm run verify:lessons`, which never touches the shared files:
`--out <file>` writes only a private bundle, `--filter a,b` (with `--out`) restricts the
run to lessons whose directory matches, and `--allow-missing-expected` turns a missing
expected file into a warning while you generate it. verify-lessons then sends every
example to `POST /api/run` with `X-Stacklab-No-History` (or runs it in headless Edge for
ts-browser) and judges the output with the UI's own `judge.ts`, twice.

## Design notes

The docs describe a different design. ADR-0004 decided that "the backend loads and
indexes it into app-db at startup (upsert on content hash, delete removed lessons) so
search and the lessons table still work", with a `cmd/contentcheck` and a dev-mode file
watcher pushing an SSE `content-changed` event. `docs/ARCHITECTURE.md` draws the
`lessons` table with its `search` column. None of it was built: contentcheck is a Node
script, and what shipped is the alternative ADR-0004 rejected, "content in the frontend
bundle". The README's "it runs as part of the build" is also out of date: `prebuild`
runs `copy-wasm.mjs` and `build-lesson-libs.mjs` only.

For one user and a couple of hundred lessons, the shipped design is the boring, right
one: no loader to keep in sync, nothing to fail at startup, lessons readable with the
backend down. Its costs were a large main chunk, until the content was split per lesson
on 2026-11-14 (4.7 MB of first-chunk JavaScript before, 0.48 MB after), and no search.
If you add Ctrl+K, there are two honest options:

- **Client-side**, over the lesson index already in memory: titles and summaries are
  there (add tags to the index to search them too); fold Turkish casing as in example 2's pitfall, and it is instant with no
  backend change. This is where to start.
- **Server-side**, as drawn: a loader that upserts lessons by content hash at startup,
  the STORED tsvector with a GIN index, a search route. Worth it only if search must
  cover lesson bodies, which sit in per-lesson chunks the page doesn't load, or needs
  ranking and stemming.
  Example 3 shows what to check first: the `turkish` stemmer's quirks and the database's
  I/ı case mapping.

Either way, if lessons ever become rows, give `progress` and `notes` a foreign key with
`ON UPDATE CASCADE`, so an id rename stops orphaning learning state.
