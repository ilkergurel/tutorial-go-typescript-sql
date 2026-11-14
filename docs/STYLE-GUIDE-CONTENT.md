# StackLab content style guide

The rules every lesson must satisfy. `scripts/build-lessons.mjs` (contentcheck) and
`npm run verify:lessons` enforce the mechanical ones; you enforce the rest in review.

## Quality bar

1. **Correct beats complete.** Never invent APIs, flags or functions. Check every
   version-specific claim against official docs and set `since`. If you can't verify,
   write `<!-- VERIFY: the claim, and what to check where -->` on its own line(s) inside
   a displayed section, and tell the owner. Don't guess. The app shows each marker as a
   warning box, flags the lesson header, and lists all of them at `/unverified`.
2. Every example **runs**, is **deterministic** (sort map keys, seed randomness, fix
   times), is self-contained, idiomatic, gofmt/Prettier-clean, ideally ≤ 50 lines and
   never over 80: one idea per example, readable top to bottom. Shrink before you split,
   and keep a lesson to at most four examples.
3. Show the naive or wrong version next to the idiomatic one when a common mistake exists.
4. Strengths and weaknesses are **concrete and comparative**: "unlike Java's checked
   exceptions, the compiler won't force you to handle the error". Never generic praise.
5. Annotations explain **why**, not what the line literally says.
6. One lesson ≤ 15 minutes. Split it if longer.
7. Mark modern vs legacy explicitly ("before Go 1.22 the loop variable was shared…").
8. Tone: a senior colleague explaining to an experienced engineer. No filler, no hype,
   no emoji.
9. Audience calibration: no beginner padding. Explain what is **different**, what is
   **idiomatic**, and **why**. The learner knows C/C++, Java, Python, JS/React/Node;
   is new to Go, TS's type system, and serious Postgres work.
10. Turkish-text gotchas (İ/ı case mapping, `turkish` FTS config, UTF-8 bytes vs
    characters) appear wherever they are relevant.

## File layout

```
content/<track>/<module>/<NN-slug>/
  lesson.md
  examples/*.go|.ts|.sql
  expected/*.txt
  fixtures/
  exercises/<slug>/{prompt.md, starter.*, solution.*, solution_test.go}
```

## Frontmatter

See `content/lesson.schema.json` (added in Phase 2). Required fields: `id`
(`track.module.slug`, immutable), `title`, `track`, `module`, `order`, `level`,
`minutes`, `since`, `prerequisites`, `compare_with`, `tags`, `summary` (≤ 2 sentences),
`examples` (each with `id`, `file`, `title`, `runtime`, `check`), `use_cases`,
`strengths`, `weaknesses`, `pitfalls`, `exercises`, `quiz` (2–4), `further_reading`
(**official docs only**).

Runtimes: `go`, `go-test`, `go-server`, `sql`, `sql-nontx`, `sql-multisession`,
`ts-browser`, `ts-node`, `ts-test`.

Checks: `exact`, `contains`, `regex`, `sorted-lines`, `none` (`none` only for genuinely
nondeterministic demos, with a note explaining why). `contains` passes when every
expected line (trimmed) appears in stdout. Only stdout is compared; stderr is shown
but never judged, and a non-zero exit code always fails the check.

Directories are `NN-slug` with `NN` = the lesson's position in its curriculum module
(01-based), and `order` in the frontmatter is the same number.

## What each runtime does

Every example is one file; the Run button sends it as the only file.

- **go**: package `main`, standard library only (no network, `GOPROXY=off`). Built and run
  in a fresh temp dir with `go.mod` saying `go 1.27`; fixtures are copied next to it.
  Default timeout 5 s, `timeout_s` up to 15. It runs on Windows in local mode and on
  Linux in Docker: never print OS-specific text (paths, OS error messages) under an
  `exact` check.
- **go with pgx** (any Go runtime): an example that imports `github.com/jackc/pgx/v5`
  builds against the backend's own pgx version, offline, and gets `DATABASE_URL`: the
  playground database as the non-superuser `playground`, with `search_path` set to an
  empty schema of its own that is dropped after the run. Connect with
  `pgx.Connect(ctx, os.Getenv("DATABASE_URL"))` or `pgxpool.New`; committed data stays
  in that schema. `golang.org/x/sync` (errgroup, semaphore) and `golang.org/x/text`
  (`cases`, `language`, `collate`) are available the same way. Local mode only for now:
  the Docker runner image has none of these modules.
- **go-server**: the same, for HTTP lessons. The program must be self-contained and
  exit: start servers with `httptest.NewServer` (or on `127.0.0.1:0`), make the requests,
  print what matters, shut down.
- **go-test**: `go test -v` over one `_test.go` file in package `main` holding both the
  code and its tests. Compare with `contains` (timings vary).
- **ts-node**: Node with type stripping: erasable syntax only (no `enum`, `namespace`,
  parameter properties), ES module, top-level `await` allowed, `node:` modules available.
  `console.log` uses Node's formatting.
- **ts-browser**: esbuild transpiles (all TS syntax), then the code runs as an ES module
  in a sandboxed iframe with no network (CSP): mock `fetch` in the example. DOM, `File`,
  `Blob` and streams work. The run ends when the module finished and no timer is pending;
  clear every `setInterval`. `console.log` prints strings as-is, `Map`/`Set`/`Error`
  readably, and everything else as `JSON.stringify` output.
- **ts-browser libraries**: lessons may import `react`, `react-dom`, `react-dom/client`,
  `react-dom/server`, `react/jsx-runtime`, `react-router-dom`, `@tanstack/react-query`
  and `zod` (the app's installed versions); name the file `.tsx` for JSX. Render with
  `flushSync(() => root.render(...))` and print `textContent`, so output doesn't depend
  on scheduling; give effects a turn with `await new Promise(r => setTimeout(r, 0))`.
- **ts-test** (local mode only): the real tools, run by the backend with the app's own
  packages. A `*.test.ts(x)` file runs in Vitest with jsdom: `@testing-library/react`,
  `@testing-library/user-event` and MSW (`msw/node`) are available. A `*.test-d.ts`
  file adds Vitest's typecheck (`expectTypeOf`). A file that imports `@playwright/test`
  (`*.spec.ts`) runs in Playwright driving the installed Microsoft Edge, headless; build
  pages with `page.setContent` or `page.route`, there is no app server. Output includes
  timings: check with `contains`, on test names and the pass/fail summary.
- **sql**: one transaction in a throwaway schema, rolled back afterwards. Statements run
  one by one and the first error stops the script. `COMMIT`, `ROLLBACK`, `END` and
  `ABORT` are refused because they would end the sandbox; `SAVEPOINT` and
  `ROLLBACK TO SAVEPOINT` work. Create tables visibly in the example unless the lesson
  is not about them; use `setup` only for data the learner shouldn't have to read.
  `COPY ... FROM STDIN` is refused in every SQL runtime (the runner can't stream rows);
  use `INSERT`, or pgx's `CopyFrom` in a Go example.
- **sql-nontx**: autocommit, for `VACUUM`, `CREATE INDEX CONCURRENTLY` and scripts with
  their own `BEGIN`/`COMMIT`. The schema is dropped afterwards.
- **sql-multisession**: two concurrent sessions. Each step starts with `A:` or `B:` and
  holds one statement. `setup` runs first and commits (both sessions see it). A step still
  running after 400 ms is reported as `waiting` and the script moves on; its result
  appears when it arrives (after the other session commits, or when the deadlock
  detector picks it). `lock_timeout` is 4 s, `statement_timeout` 5 s.

## SQL output as text

SQL results also come back as text, which is what expected files hold: one block per
statement, the command tag (or `ERROR: <message> (SQLSTATE <code>)`), then notices and
the result table indented by four spaces. `NULL` is spelled out. Multisession steps are
prefixed `[n] A:` and a step that had to wait prints `[n] A: waiting` first. Never write
these by hand: generate them (below) and review them.

## Authoring workflow

The app must be running (`npm start`). For a lesson `go.basics.defer`:

1. Write the examples and `lesson.md`, then run
   `npm run verify:lessons -- --filter go.basics.defer --print` and read the output.
2. When it is what the lesson claims, record it:
   `npm run verify:lessons -- --filter go.basics.defer --write-expected`, and review
   the written `expected/*.txt` (`contains`/`regex` expectations are written by hand).
3. Check every annotation against its code:
   `npm run verify:lessons -- --filter go.basics.defer --annotations`.
4. `npm run verify:lessons -- --filter go.basics.defer` must pass. It runs every example
   twice; different output is a failure.
5. Verify version claims (`since`, "Go 1.22 changed…") against official docs or the
   toolchain itself: a `go.mod` with an older `go` line makes the compiler name the
   version a feature needs.

## Prose body

Fixed H2 sections, in this order:

1. `## Concept`
2. `## Coming from C/C++/Java/Python/JS`
3. `## How it works` (optional; mermaid diagrams allowed)
4. `## Design notes`

The app renders a Markdown subset (`src/shared/ui/Markdown.tsx`): paragraphs, `###` and
`####` headings, `-` and `1.` lists (nested items flatten), fenced code, pipe tables with
a `| --- |` separator row, inline code, `**bold**`, `*italic*` and `[links](https://…)`.
No underscore emphasis and no raw HTML; HTML comments are hidden except `VERIFY` markers.

## The reference lesson

`content/go/files-io/01-reading-files` (§7.4 of the build prompt) is the quality bar:
compiled, run, expected outputs verified. When in doubt about depth, tone, or annotation
style, match that lesson.
