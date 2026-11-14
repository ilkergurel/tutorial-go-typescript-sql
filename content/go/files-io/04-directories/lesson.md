---
id: go.files-io.directories
title: "Directories: ReadDir, WalkDir, path vs path/filepath, os.Root"
track: go
module: files-io
order: 4
level: intermediate
minutes: 14
since: go1.25
prerequisites: [go.files-io.writing]
compare_with: [ts.node.fs-streams, go.http.uploads-streams]
tags: [files, directories, filepath, io-fs, os-root, security]
summary: >
  os.ReadDir lists one level and filepath.WalkDir a whole tree, both in byte
  order. Use path for "/" names, path/filepath for OS paths, and os.Root
  (Go 1.24) to open user-supplied names without escaping a directory.
examples:
  - id: readdir-walkdir
    file: examples/01_readdir_walkdir.go
    title: ReadDir for one level, WalkDir for a tree, io/fs for "/" names
    runtime: go
    check: exact
    annotations:
      - lines: "18-23"
        note: >
          MkdirAll is mkdir -p: it creates missing parents and succeeds if the
          directory already exists. Forward slashes work in os calls on
          Windows too; paths you get back from filepath use backslashes there.
      - lines: "26-32"
        note: >
          ReadDir returns fs.DirEntry values sorted by name as bytes:
          uppercase Z before lowercase n, and İ (0xC4 0xB0) after every ASCII
          letter. Fine for deterministic output, wrong for a Turkish user's
          list, which needs collation (golang.org/x/text/collate).
      - lines: "34-37"
        note: >
          A non-nil err means WalkDir couldn't read path (permissions, a
          vanished directory). Returning it stops the whole walk; returning nil
          skips that directory and carries on.
      - lines: "38-40"
        note: >
          fs.SkipDir on a directory skips its subtree without reading it, so
          node_modules costs nothing. fs.SkipAll (Go 1.20) ends the walk early
          without an error.
      - lines: "41-43"
        note: >
          filepath.WalkDir hands you OS paths: data\raporlar\Ocak.csv on
          Windows. ToSlash makes the printed form the same everywhere.
      - lines: "50-52"
        note: >
          os.DirFS("data") is the same directory as an fs.FS. Names are
          relative to it and always use "/", which is what embed.FS, zip
          archives and fstest.MapFS use too, so code written against fs.FS
          works with all of them.
  - id: path-filepath
    file: examples/02_path_filepath.go
    title: path vs path/filepath, Join, IsLocal and Localize
    runtime: go
    check: exact
    annotations:
      - lines: "10-12"
        note: >
          The path package is pure string manipulation with "/" on every OS.
          Use it for URL paths and io/fs names; using it on Windows file
          paths would mishandle backslashes and drive letters.
      - lines: "15-17"
        note: >
          filepath uses the OS separator. Rel computes the path of p relative
          to "data". Printing through ToSlash keeps the output identical on
          Windows and Linux.
      - lines: "19-20"
        note: >
          Join calls Clean, which resolves ".." lexically. With a name taken
          from a request, uploads/../../etc/passwd becomes ../etc/passwd:
          Join is not a security boundary.
      - lines: "22-25"
        note: >
          IsLocal (Go 1.20) is true only for a non-empty, relative name that
          stays inside the directory it is joined to; a/../b stays inside, so
          it passes. It checks the string only: a symlink inside the tree can
          still point outside.
      - lines: "27-31"
        note: >
          Localize (Go 1.23) takes an io/fs-style name, as in a URL, and gives
          an OS path, or an error for names that aren't valid fs paths
          (.., leading /) or that the OS can't represent, such as a\b on
          Windows.
  - id: os-root
    file: examples/03_os_root.go
    title: os.Root keeps every lookup inside one directory
    runtime: go
    check: exact
    annotations:
      - lines: "19-23"
        note: >
          OpenRoot (Go 1.24) opens the directory once and keeps a handle to it.
          Every method on root resolves names relative to that handle and
          refuses any step that leaves it.
      - lines: "25-31"
        note: >
          Writes stay inside as well. Names such as kasım are UTF-8 strings in
          Go; on Windows the os package converts them to the UTF-16 the system
          calls expect, so the same code works on both.
      - lines: "35-39"
        note: >
          The naive Join plus os.ReadFile leaks secret.env twice. root.ReadFile
          refuses both: ".." may not climb above the root, and neither may
          absolute names or symlinks that point outside it.
use_cases:
  - "Listing a folder for a UI or an API: os.ReadDir"
  - "Indexing, cleanup and import jobs over a tree: filepath.WalkDir with SkipDir for vendor and node_modules"
  - "Serving or reading user-named files (uploads, report downloads, archive extraction): os.Root or os.OpenInRoot"
  - "Code that should work on disk, in embed.FS, in a zip and in tests: accept an fs.FS"
strengths:
  - "WalkDir avoids an lstat per entry (unlike the older filepath.Walk), and DirEntry.Info only stats when you ask"
  - "Deterministic order out of the box: ReadDir and WalkDir sort by name, unlike Node's fs.readdir or Python's os.scandir, which return directory order"
  - "os.Root resolves names with openat-style lookups, so it is safe against symlink tricks that a lexical check like IsLocal (or Java's normalize().startsWith()) misses"
weaknesses:
  - "Two path packages that look alike; mixing them up compiles and fails only on Windows"
  - "Byte-order sorting is wrong for display: İ, Ç, Ş sort after z"
  - "os.Root confines names, not resources: it doesn't stop bind mounts, /proc files or device files on Unix, as its docs say"
pitfalls:
  - title: "filepath.Join as a sandbox"
    detail: "Join cleans .. away from the result; it never refuses. Use os.Root (or at least filepath.IsLocal) for names that come from users, archives or URLs."
    bad: "f, err := os.Open(filepath.Join(uploadDir, r.PathValue(\"name\")))"
    good: "f, err := root.Open(r.PathValue(\"name\")) // root, _ := os.OpenRoot(uploadDir)"
  - title: "path where filepath was meant"
    detail: "path.Join(\"C:\\\\data\", \"x\") treats the whole \"C:\\\\data\" as one name. Use filepath for anything the OS opens, path for URLs and fs.FS names."
  - title: "Case-insensitive file systems"
    detail: "NTFS (Windows) and macOS's default APFS treat Rapor.csv and rapor.csv as one file; Linux ext4 sees two. Code that dedupes or looks up names behaves differently in Docker than on your machine."
  - title: "Returning err from the WalkDir callback by reflex"
    detail: "One unreadable directory then aborts the whole walk. Decide: stop, or log and return nil to skip it."
quiz:
  - q: "filepath.Join(\"uploads\", \"../../etc/passwd\") returns what?"
    options: ["An error", "uploads/etc/passwd", "../etc/passwd"]
    answer: 2
    why: "Join cleans the result lexically. It doesn't confine anything, which is why user-supplied names need os.Root or IsLocal."
  - q: "Which package do you use to join the parts of a URL path or an embed.FS name?"
    options: ["path", "path/filepath", "either; they're the same on Linux"]
    answer: 0
    why: "URL and io/fs names always use \"/\". filepath would produce backslashes on Windows."
  - q: "filepath.IsLocal(name) is true. Is opening filepath.Join(dir, name) now safe against escaping dir?"
    options: ["Yes, always", "Only lexically: a symlink inside dir can still lead outside", "No, IsLocal allows .."]
    answer: 1
    why: "IsLocal checks the string. os.Root checks each step against the real file system, symlinks included."
further_reading:
  - title: "package os: ReadDir, Root, DirFS"
    url: "https://pkg.go.dev/os"
  - title: "package path/filepath"
    url: "https://pkg.go.dev/path/filepath"
  - title: "package io/fs"
    url: "https://pkg.go.dev/io/fs"
  - title: "Traversal-resistant file APIs (Go blog)"
    url: "https://go.dev/blog/osroot"
---

## Concept

- `os.ReadDir(dir)` reads one directory and returns `[]os.DirEntry` (an alias of
  `fs.DirEntry`), sorted by name. A `DirEntry` knows its name and type from the directory
  read; `Info()` fetches size and times only if you call it.
- `filepath.WalkDir(root, fn)` visits a whole tree in lexical order, calls `fn` for every
  directory and file, and lets `fn` prune with `fs.SkipDir` or stop with `fs.SkipAll`.
  It doesn't follow symbolic links.
- `path` handles `"/"`-separated names (URLs, `io/fs`, `embed.FS`, zip entries) on every
  OS; `path/filepath` handles the OS's own paths, with `\` and drive letters on Windows.
- `os.Root` (Go 1.24, with `MkdirAll`, `ReadFile`, `WriteFile`, `Rename` and more in 1.25)
  opens a directory and then refuses any name that would leave it, through `..`, an
  absolute path or a symlink.

## Coming from C/C++/Java/Python/JS

- C: `opendir`/`readdir` plus `nftw`; `os.Root` is the `openat`-relative-to-a-dirfd
  discipline, wrapped so every method uses it.
- Java: `Files.list` and `Files.walk` (lazy streams, unsorted) vs Go's sorted slices.
  `path.normalize().startsWith(base)` is the lexical check that `filepath.IsLocal` replaces,
  with the same symlink blind spot.
- Python: `os.scandir` ≈ `os.ReadDir` (both avoid a stat per entry), `os.walk` ≈ `WalkDir`,
  `posixpath` vs `os.path` ≈ `path` vs `filepath`.
- Node: `fs.readdir(dir, { withFileTypes: true })` ≈ `ReadDir`; `path.posix` vs `path` ≈
  `path` vs `filepath`. Node has no equivalent of `os.Root`.

## How it works

`os.Root` resolves a name one component at a time from the directory handle it holds (on
Unix with `openat(2)` and `O_NOFOLLOW`, following symlinks itself). `..` may not climb
above the root, a leading `/` (or `C:` on Windows) is rejected, and symlinks are followed
only if they are relative and their target stays inside. Because it never builds a full
path string and then opens it, there is no gap between "checked" and "opened" for an
attacker to swap a directory for a symlink (the docs list the few exceptions, such as
`Root.Chmod` on Unix). On Windows it also refuses reserved device names such as `NUL` and
`COM1`. `root.FS()` gives the same protection behind the `fs.FS` interface; `os.DirFS`
does not, as its own documentation warns.

## Design notes

Keep the OS at the edge. Functions that only read files can take an `fs.FS`, and `main`
decides whether that is `os.DirFS(dir)`, `root.FS()`, an `embed.FS` or a zip archive;
tests pass a `fstest.MapFS`. For anything that turns user input into a file name (download
endpoints, upload storage, archive extraction), open an `os.Root` once at startup for the
directory and route every access through it; `filepath.IsLocal` is a good input check but
not a substitute. Sort for display with locale-aware collation, and remember that the same
two names can be one file on your Windows machine and two in the Linux container.
