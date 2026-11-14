---
id: go.files-io.embed
title: "embed.FS: shipping static assets in the binary"
track: go
module: files-io
order: 5
level: intermediate
minutes: 12
since: go1.22
prerequisites: [go.files-io.directories, go.cli.templates]
compare_with: [ts.tooling.vite, go.performance.cross-compile, go.http.handlers]
tags: [embed, io-fs, static-files, templates, single-binary]
summary: >
  A //go:embed directive makes the compiler copy files into the binary as a
  string, a []byte or a read-only embed.FS. Because embed.FS is an fs.FS,
  templates and HTTP file servers use it directly: one file to deploy.
examples:
  - id: embed-basics
    file: examples/01_embed_basics.go
    title: "string, embed.FS, the hidden-file rule and all:"
    runtime: go
    fixtures: [fixtures/version.txt, fixtures/static/index.html, fixtures/static/app.css, fixtures/static/_draft.html, fixtures/static/.env.local]
    check: exact
    annotations:
      - lines: "11-12"
        note: >
          The directive sits directly above a package-level variable. For a
          string or []byte it must match exactly one file, and the file's
          content becomes the variable's value at compile time.
      - lines: "14-18"
        note: >
          Naming a directory embeds its whole subtree except names that start
          with . or _, so editor backups and .env files stay out. The all:
          prefix turns that filter off, which is how secrets end up in a
          binary.
      - lines: "34-38"
        note: >
          The binary doesn't read version.txt at run time; the compiler already
          did. Changing the file changes nothing until the next go build.
      - lines: "40-43"
        note: >
          Embedded names are slash-separated, relative to the package
          directory, and include the directory you named: static/app.css, not
          app.css.
      - lines: "48-49"
        note: >
          A file excluded by the pattern simply isn't there: Open returns an
          error wrapping fs.ErrNotExist.
  - id: templates-server
    file: examples/02_templates_server.go
    title: ParseFS and FileServerFS on embedded files, a MapFS in tests
    runtime: go
    fixtures: [fixtures/templates/welcome.tmpl, fixtures/static/index.html, fixtures/static/app.css, fixtures/static/_draft.html]
    check: exact
    annotations:
      - lines: "15-19"
        note: >
          Two variables, two patterns. templates/*.tmpl matches files only; a
          * pattern would also match .hidden files at that level, unlike a
          bare directory name.
      - lines: "21-23"
        note: >
          The handler depends on fs.FS, the interface, so the embedding
          decision stays in main. This is what lets the last request run
          against an in-memory test file system.
      - lines: "25"
        note: >
          http.FileServerFS (Go 1.22) serves any fs.FS. StripPrefix removes
          /static/ from the URL so the remaining path matches names in the
          sub-tree.
      - lines: "41-45"
        note: >
          ParseFS takes path.Match patterns over the FS. fs.Sub returns the
          static directory as its own FS, so names lose the static/ prefix.
      - lines: "48-49"
        note: >
          r.PathValue returns the decoded name, Ayşe, and html/template
          escapes it for HTML: the <script> attempt arrives as text.
      - lines: "51"
        note: >
          _draft.html is on disk next to the source but was never embedded, so
          the file server answers 404.
      - lines: "53-54"
        note: >
          fstest.MapFS is a map from names to files that satisfies fs.FS, so a
          test can swap the assets without touching disk or the build.
use_cases:
  - "Single-binary deployment: HTML templates, CSS, JS and a built SPA inside one executable (or one FROM scratch image)"
  - "SQL migrations embedded next to the code that runs them"
  - "Default config files, license texts, test fixtures and seed data"
strengths:
  - "No path resolution at run time: the binary works from any working directory, unlike reading ./templates relative to where the process started"
  - "Plugs into everything that takes an fs.FS: html/template.ParseFS, http.FileServerFS, fs.WalkDir, migration tools"
  - "Built into the go command, no bundler, resource compiler or build step, unlike Java resources (which need the JAR packaging) or Node, which needs a bundler or its experimental single-executable feature"
weaknesses:
  - "Every change needs a rebuild; in development you usually want os.DirFS for live edits"
  - "Read-only, and files carry no modification time, so http.FileServer can't send Last-Modified or answer 304 on its own"
  - "The binary grows by the size of the files, and patterns can't reach outside the package directory (no .., no parent folders)"
pitfalls:
  - title: "all: embeds secrets"
    detail: "all:static includes .env, .git leftovers and _drafts. Prefer a plain directory pattern and a build step that writes only what should ship. You need all: when a framework's output uses _ names (SvelteKit's _app, Next.js's _next)."
  - title: "Assets outside the package directory"
    detail: "//go:embed ../web/dist is an invalid pattern. Put a tiny package next to the assets (web/embed.go exporting an fs.FS) and import it."
    bad: "//go:embed ../web/dist"
    good: "// in package web, next to dist/:\n//go:embed dist\nvar dist embed.FS"
  - title: "Building before the frontend exists"
    detail: "A pattern that matches nothing fails the build (\"pattern dist: no matching files found\", or \"cannot embed directory dist: contains no embeddable files\" for an empty directory). Build the frontend first in CI and in the Dockerfile."
  - title: "Forgetting the directory prefix"
    detail: "static.ReadFile(\"app.css\") fails: the name is static/app.css. Use fs.Sub to serve a directory as the root."
quiz:
  - q: "//go:embed static is above an embed.FS. Which of static/app.css, static/_draft.html, static/.env end up in the binary?"
    options: ["All three", "Only static/app.css", "static/app.css and static/.env"]
    answer: 1
    why: "Directory patterns skip names that start with . or _; the all: prefix would include them."
  - q: "You edit a template file while the compiled server is running. When do users see the change?"
    options: ["On the next request", "After a restart", "After a rebuild and a restart"]
    answer: 2
    why: "The compiler copied the file into the binary; the running program never reads the file on disk."
  - q: "Why should newHandler take an fs.FS instead of an embed.FS?"
    options: ["embed.FS can't be passed to functions", "So tests and development can pass fstest.MapFS or os.DirFS", "fs.FS is faster"]
    answer: 1
    why: "Depending on the interface keeps the decision about where files come from in main."
further_reading:
  - title: "package embed"
    url: "https://pkg.go.dev/embed"
  - title: "package io/fs: FS and Sub"
    url: "https://pkg.go.dev/io/fs"
  - title: "net/http: FileServerFS"
    url: "https://pkg.go.dev/net/http#FileServerFS"
  - title: "package testing/fstest: MapFS"
    url: "https://pkg.go.dev/testing/fstest#MapFS"
---

## Concept

`//go:embed pattern` above a package-level variable tells the compiler to read files from the
package directory (or below it) at build time and store them in the binary:

- `string` or `[]byte`: exactly one file, its content.
- `embed.FS`: any number of files and directories, as a read-only file system that
  implements `fs.FS`, `fs.ReadDirFS` and `fs.ReadFileFS`.

Patterns use `/` on every OS and may not contain `.`, `..` or a leading `/`. A directory
pattern takes the whole subtree except names beginning with `.` or `_`; `all:dir` takes
everything. The program never looks for those files at run time, so it works from any
working directory, in a `FROM scratch` container, or copied to another machine.

## Coming from C/C++/Java/Python/JS

- C/C++: C23 (and C++26) add `#embed` for one file; before that, `xxd -i` or linker tricks.
  `embed.FS` adds the directory tree and a file-system API on top.
- Java: like resources on the classpath (`getResourceAsStream`), but there's no JAR or
  classloader: the files are bytes in the executable, and the API is the same `fs.FS` you'd
  use for a directory.
- Python: `importlib.resources` for package data, minus the packaging metadata you need to
  get the files into the wheel.
- Node: the closest is a bundler inlining assets, or Node's single executable applications
  (still experimental). In Go it's part of the plain `go build`.

## How it works

The `go` command finds the `//go:embed` lines, checks the patterns, and hands the matching
files to the compiler, which stores their contents in the binary's read-only data. An
`embed.FS` value is just a sorted list of names and contents, so `Open`, `ReadDir` and
`ReadFile` are lookups in memory. Its files report a zero modification time; `go list
-json` shows each package's `EmbedPatterns` and `EmbedFiles` if you want to see what was
matched.

## Design notes

Put embedded assets behind a small package that exports an `fs.FS` (for example a `web`
package next to the frontend's `dist/`), and make every consumer take `fs.FS`. Then `main`
chooses: the embedded files in production, `os.DirFS("web/dist")` behind a `-dev` flag for
live editing, `fstest.MapFS` in tests. Since embedded files have no modification time, give
built assets content-hashed names (Vite does this by default) and serve them with a long
`Cache-Control`, or add an `ETag` middleware; otherwise browsers have nothing to revalidate
against and download them again.
