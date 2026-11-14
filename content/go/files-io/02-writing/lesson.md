---
id: go.files-io.writing
title: "Writing files: atomic writes, permissions, bufio.Writer"
track: go
module: files-io
order: 2
level: basic
minutes: 12
since: go1.22
prerequisites: [go.files-io.reading-files, go.basics.defer]
compare_with: [ts.node.fs-streams, ts.browser-dom.generate, sql.transactions.acid-mvcc]
tags: [files, io, bufio, os, atomic-write, permissions]
summary: >
  Buffer many small writes with bufio.Writer and always Flush; check the error
  from Close on files you wrote. To replace a file that matters, write a temp
  file in the same directory and rename it over the target.
examples:
  - id: buffered-flags
    file: examples/01_buffered_flags.go
    title: bufio.Writer, the Close error, O_APPEND and O_EXCL
    runtime: go
    check: exact
    annotations:
      - lines: "15"
        note: >
          os.Create truncates an existing file to zero bytes before you write
          anything. From this line on, the old content is gone, which is the
          problem the second example solves.
      - lines: "19"
        note: >
          A buffered write can succeed in memory and fail when the OS finally
          writes it (disk full, network drive). Close is where that surfaces,
          so for files you write, join its error into the result instead of a
          bare defer f.Close().
      - lines: "21-28"
        note: >
          bufio.Writer collects small writes and hands them to the file 4096
          bytes at a time. The output shows the file holding exactly one full
          buffer while 2583 bytes still sit in memory: return without Flush and
          the tail of the report is silently lost.
      - lines: "32"
        note: >
          OpenFile is the general form behind Open and Create. O_APPEND makes
          every write land at the current end of file, which is what log files
          want. The 0o644 is used only if the file is created.
      - lines: "57-58"
        note: >
          O_CREATE|O_EXCL creates the file only if it doesn't exist, in one
          step, with no gap between checking and creating. Use it for lock
          files and to avoid overwriting a user's file. Test the error with
          errors.Is and fs.ErrExist: the message text differs between Windows
          and Linux.
  - id: atomic-write
    file: examples/02_atomic_write.go
    title: Temp file plus rename versus writing in place
    runtime: go
    check: exact
    annotations:
      - lines: "13-20"
        note: >
          The naive version: truncate, then write. If the process dies, the
          disk fills or the encoder fails in between, the file holds a prefix
          of the new content and the old content is gone.
      - lines: "25"
        note: >
          The temp file goes in the target's directory, not os.TempDir(): a
          rename cannot move a file to another file system or volume, and
          /tmp is often a separate one (tmpfs, or a different Docker volume).
      - lines: "29-34"
        note: >
          On any error after this point, close and delete the temp file. A hard
          crash skips this cleanup, so a stray "atomic.json.tmp-123" can still
          appear; a recognizable name lets a startup job remove them.
      - lines: "38"
        note: >
          CreateTemp creates the file with mode 0o600. Without this Chmod, every
          save would silently tighten the target from 0o644 to 0o600. On
          Windows only the owner-write bit is used, as the read-only attribute.
      - lines: "41-47"
        note: >
          Sync asks the OS to put the data on stable storage before the rename
          makes it visible under the real name. Without it, a power cut right
          after the rename can leave a correctly named but empty file on some
          file systems.
      - lines: "61-64"
        note: >
          The in-place file is now invalid JSON; the atomic one still has
          version 1, because the failed write never got the real name.
use_cases:
  - "bufio.Writer: CSV and report exports, any loop that writes many small pieces"
  - "Temp file plus rename: config and state files, caches, generated assets, anything another process reads while you write"
  - "O_APPEND: log and audit files several writers append to"
  - "O_CREATE|O_EXCL: lock files and \"don't overwrite the user's file\" saves"
strengths:
  - "os.WriteFile covers the small case in one call, like Python's Path.write_text or Node's fs.writeFileSync"
  - "bufio.Writer wraps any io.Writer (file, socket, gzip stream), not just files, unlike Java's separate BufferedWriter and BufferedOutputStream classes"
  - "The atomic pattern needs only CreateTemp, Sync, Close and Rename from the standard library"
weaknesses:
  - "The standard library has no WriteFileAtomic; you write (and test) the helper yourself"
  - "Errors surface late: a buffered write succeeds in memory and fails at Flush or Close, so both must be checked"
  - "Permission bits are Unix concepts: on Windows only the owner-write bit does anything (it maps to the read-only attribute)"
pitfalls:
  - title: "Forgetting Flush"
    detail: "bufio.Writer holds up to 4096 bytes in memory. Return without Flush and the end of the file is missing, with no error anywhere."
    bad: "w := bufio.NewWriter(f)\nfmt.Fprintln(w, \"total;42\")\nreturn nil"
    good: "w := bufio.NewWriter(f)\nfmt.Fprintln(w, \"total;42\")\nreturn w.Flush()"
  - title: "Ignoring Close on a file you wrote"
    detail: "defer f.Close() discards the error. For files you read that's fine; for files you wrote, join Close's error into the function's result."
    bad: "defer f.Close()"
    good: "defer func() { err = errors.Join(err, f.Close()) }()"
  - title: "Truncating a file that matters, then writing it"
    detail: "os.WriteFile and os.Create truncate first. The os.WriteFile docs say it outright: a failure mid-operation can leave the file partially written. Write a temp file and rename it."
  - title: "Temp file in os.TempDir()"
    detail: "Rename across file systems fails (EXDEV on Linux, and Windows refuses to move across volumes). Create the temp file in the target's directory."
  - title: "Losing the original permissions"
    detail: "CreateTemp uses 0o600 and rename keeps the temp file's mode, so call f.Chmod(perm) before renaming."
quiz:
  - q: "A function writes 300 short lines through bufio.NewWriter(f) and returns without calling Flush. What ends up in the file?"
    options: ["All 300 lines: Close flushes the buffer", "Only the bytes that filled whole 4096-byte buffers", "Nothing until the program exits"]
    answer: 1
    why: "Closing the *os.File knows nothing about the bufio.Writer wrapped around it; whatever is still buffered is lost."
  - q: "Why must the temp file be created in the target's directory?"
    options: ["CreateTemp only works there", "Rename can't move a file across file systems or volumes", "So the file inherits the directory's permissions"]
    answer: 1
    why: "os.TempDir() is often another file system or volume, where rename fails. Same directory means same file system."
  - q: "You save config.json (mode 0o644) with CreateTemp plus Rename and no Chmod. What mode does it have afterwards on Linux?"
    options: ["0o644, rename keeps the target's mode", "0o600, the temp file's mode", "0o666"]
    answer: 1
    why: "Rename replaces the directory entry with the temp file, mode included. CreateTemp creates files with 0o600 before umask."
further_reading:
  - title: "package os: WriteFile, CreateTemp, Rename, OpenFile"
    url: "https://pkg.go.dev/os"
  - title: "package bufio: Writer"
    url: "https://pkg.go.dev/bufio#Writer"
  - title: "package io/fs: ErrExist and FileMode"
    url: "https://pkg.go.dev/io/fs"
---

## Concept

Three layers, from simple to safe:

- `os.WriteFile(name, data, perm)` writes a whole `[]byte` in one call. It creates or
  **truncates** the file, so a failure halfway leaves it partially written.
- `os.Create` or `os.OpenFile` give you an `*os.File`, an `io.Writer`. Wrap it in a
  `bufio.Writer` when you make many small writes, and call `Flush` at the end.
- To replace a file that something else depends on, write a temp file in the same
  directory, `Sync` and `Close` it, then `os.Rename` it over the target. Until the rename,
  readers see the complete old file; after it, the complete new one.

The `perm` argument (`0o644`, `0o600`) is only used when the file is **created**, and the
process umask is subtracted from it on Unix (a umask of `022` turns `0o666` into `0o644`).
On Windows only the owner-write bit means anything: without it the file gets the
read-only attribute. That's why the examples never print modes: they would differ by OS.

## Coming from C/C++/Java/Python/JS

- C: `os.OpenFile` flags are `open(2)`'s (`O_APPEND`, `O_CREATE`, `O_EXCL`, `O_TRUNC`),
  and `bufio.Writer` is the buffering `stdio` does for `FILE*`, made explicit. As with
  `fclose`, the close can be where a write error shows up.
- Java: no atomic write either. The idiom is `Files.createTempFile(dir, ...)`, write, then
  `Files.move(tmp, target, ATOMIC_MOVE)`; Go's `os.Rename` is that move. Java's checked
  `IOException` on `close()` forces you to think about a failed close; in Go only you (and
  linters such as `errcheck`) notice a dropped `Close` error.
- Python: `os.replace(tmp, target)` is the same rename, and `tempfile.mkstemp(dir=...)`
  is `os.CreateTemp(dir, pattern)`, mode `0o600` included.
- Node: `fs.writeFileSync` truncates and writes like `os.WriteFile`; there's no built-in
  atomic write there either (npm packages such as `write-file-atomic` do this dance).

## How it works

`writeAtomic` in the second example, step by step:

1. `os.CreateTemp(dir, "atomic.json.tmp-*")` creates a new file with a random name and
   mode `0o600`. Several processes can do this at once without colliding.
2. Write the content. Nobody reads this name, so a crash here only leaves a stray temp file.
3. `f.Chmod(perm)`, then `f.Sync()` so the bytes are on stable storage, then `f.Close()`.
4. `os.Rename(tmp, target)` swaps the name. On Linux and macOS, `rename(2)` within one file
   system is atomic: other processes see the old file or the new one, never a mix.

Go's own docs for `os.Rename` add a caveat: "Even within the same directory, on non-Unix
platforms Rename is not an atomic operation." On Windows the pattern still guarantees the
target is never a half-written file, because the new content is complete before it gets
the name; just don't rely on readers never seeing a gap. Replacing a file that another
Windows process holds open (an editor, an antivirus scan) can also fail, so retry or report.

## Design notes

Accept an `io.Writer` (or a `func(io.Writer) error`, as `writeAtomic` does) in code that
produces content, and decide at the edge whether it goes to a file, a buffer in a test, or an
HTTP response. Reserve the atomic dance for files that are read while you write or must
survive a crash; for an export the user downloads once, `bufio` plus a checked `Close` is
enough. On Linux, a rename is durable only after the parent directory is synced too
(`fsync` on the directory), which databases do and most applications skip; PostgreSQL's
write-ahead log is the industrial version of the same idea.
