---
id: go.files-io.reading-files
title: "Reading files: whole file vs streaming"
track: go
module: files-io
order: 1
level: basic
minutes: 12
since: go1.16
prerequisites: [go.composite.slices-and-maps]
compare_with: [ts.browser-dom.reading-files, ts.node.fs-streams, sql.files-movement.copy]
tags: [files, io, bufio, os]
summary: >
  Use os.ReadFile when the file comfortably fits in memory; stream it with
  bufio.Scanner when it might not, or when one pass is all you need.
examples:
  - id: read-whole
    file: examples/01_read_whole.go
    title: Read the whole file with os.ReadFile
    runtime: go
    fixtures: [fixtures/app.log]
    check: exact
    annotations:
      - lines: "11-14"
        note: >
          One call opens, reads and closes the file. The error is an ordinary
          return value: nothing forces you to check it, but ignoring it is a bug.
      - lines: "16-17"
        note: >
          string(data) copies the bytes. That's fine for a config file. For a
          500 MB export it doubles memory, which is the cue to stream instead.
  - id: stream-lines
    file: examples/02_stream_lines.go
    title: Stream line by line with bufio.Scanner
    runtime: go
    fixtures: [fixtures/app.log]
    check: exact
    annotations:
      - lines: "18"
        note: >
          defer schedules f.Close() for when main returns, on every return path.
          But log.Fatalf calls os.Exit, which skips deferred calls: harmless in a
          tiny program, a real bug in a server.
      - lines: "21-28"
        note: >
          Memory use stays constant no matter how big the file is: one line at a time.
      - lines: "29-31"
        note: >
          Scan returns false both at end of file and on error. Only sc.Err()
          tells the two apart, so this check isn't optional.
      - lines: "34"
        note: >
          Go randomizes map iteration order on purpose. slices.Sorted(maps.Keys(m))
          (iterators, Go 1.23) gives stable output.
use_cases:
  - "os.ReadFile: config files, small JSON/CSV, templates, test fixtures"
  - "bufio.Scanner: logs, large CSV exports, uploaded files, anything of unknown size"
strengths:
  - "os.ReadFile is one call and closes the file for you; there's no resource to leak"
  - "Scanner uses constant memory regardless of file size"
  - "Both build on io.Reader, so the same scanning code works on files, network connections, gzip streams and HTTP bodies, without Java's InputStream class hierarchy"
weaknesses:
  - "os.ReadFile holds the whole file in memory, and converting to string copies it again"
  - "Scanner's default maximum token size is 64 KB; longer lines stop the scan with bufio.ErrTooLong"
  - "Errors have to be checked in two places: at Open and after the loop"
pitfalls:
  - title: "Skipping sc.Err()"
    detail: "A too-long line ends the loop early and looks like EOF. Always check sc.Err(); raise the limit with sc.Buffer(...) if you need to."
  - title: "Relative paths resolve against the working directory"
    detail: "They don't resolve against the source file. In servers, resolve paths from configuration."
  - title: "Untrusted file names can escape a directory"
    detail: "\"../../etc/passwd\". Use os.Root (Go 1.24) or filepath.IsLocal before opening user-supplied paths."
exercises:
  - id: count-levels-reader
    check: tests
quiz:
  - q: "sc.Scan() returned false. What do you know?"
    options: ["The file ended", "An error occurred", "Either; check sc.Err()"]
    answer: 2
    why: "Scan reports false for both EOF and errors; Err() returns nil only at a clean EOF."
  - q: "You must read a 2 GB CSV on a 4 GB machine. Which do you reach for?"
    options: ["os.ReadFile", "bufio.Scanner over os.Open"]
    answer: 1
    why: "os.ReadFile would need the file (plus its string copy) in memory; Scanner holds one line."
further_reading:
  - title: "package bufio"
    url: "https://pkg.go.dev/bufio"
  - title: "package os"
    url: "https://pkg.go.dev/os"
---

## Concept

Go gives you two levels of file reading. `os.ReadFile` returns the whole content as
`[]byte`. `os.Open` returns an `*os.File`, which is an `io.Reader`, and you consume it
piece by piece, usually through `bufio.Scanner` (line by line) or `bufio.Reader`.

## Coming from C/C++/Java/Python/JS

- Python: `Path.read_text()` ≈ `os.ReadFile`; `for line in f:` ≈ the `Scanner` loop.
- Java: `Files.readAllBytes` vs `BufferedReader.lines()`. There are no checked exceptions:
  errors are return values, and linters (`errcheck`) catch the ones you ignore.
- C: a `fgets` loop without buffer overflows. An over-long line becomes an error, not memory corruption.
- Node: `fs.readFileSync` vs `readline` over a stream. Go's version is synchronous code
  that doesn't block other goroutines.

## Design notes

Accept an `io.Reader`, not a file name: `func countLevels(r io.Reader) (map[string]int, error)`.
Opening the file stays at the edge of the program (`main`, an HTTP handler), and the logic
works on files, request bodies, gzip streams and `strings.NewReader` in tests. This is Go's
everyday form of dependency inversion, and it's why `io.Reader` is the most important
interface in the standard library. The exercise has you do exactly this refactor.
