---
id: ts.node.fs-streams
title: "node:fs/promises and streams, side by side with Go equivalents"
track: ts
module: node
order: 2
level: intermediate
minutes: 14
since: es2018
prerequisites: [ts.node.type-stripping, ts.async.promises]
compare_with: [go.files-io.reading-files, go.files-io.composing, go.files-io.writing, ts.browser-dom.reading-files]
tags: [node, fs, streams, readline, pipeline, utf-8, files]
summary: >
  The Go file-reading lesson again, in Node: readFile for the whole file,
  FileHandle.readLines() for constant memory, and stream.pipeline as the io.Copy
  that propagates errors and closes every stage. Chunks are bytes, so decode them
  with the stream, not one by one.
examples:
  - id: read-whole
    file: examples/01_read_whole.ts
    title: "readFile: the whole file as a Buffer, failures as rejections"
    runtime: ts-node
    fixtures: [fixtures/app.log]
    check: exact
    annotations:
      - lines: "3-6"
        note: >
          The same output as the Go example, from the same app.log. Without an
          encoding readFile returns a Buffer (a Uint8Array of bytes, Go's
          []byte); toString("utf8") decodes it into a new string, the copy that
          string(data) makes in Go. The file ends with "\n", hence trimEnd()
          before split, where Go used TrimSpace.
      - lines: "8-14"
        note: >
          There is no error return value: the promise rejects and await turns
          that into an exception. Under strict, `err` in catch is unknown, so
          it is cast to NodeJS.ErrnoException from @types/node, which Node
          itself never reads. Compare err.code with "ENOENT", the analogue of
          errors.Is(err, fs.ErrNotExist). The message holds the absolute path
          and errno differs per OS, so print neither.
      - lines: "16-21"
        note: >
          31 bytes, 26 characters: ğ, ı, Ş and Ç take two bytes each in
          UTF-8. A Buffer's length counts bytes like len() in Go; a string's
          length counts UTF-16 code units, which matches the character count
          for Turkish text but not for emoji (two units each). Node prints a
          Buffer as hex: 49 is "I", c4 9f is "ğ".
  - id: stream-lines
    file: examples/02_stream_lines.ts
    title: "Line by line with FileHandle.readLines()"
    runtime: ts-node
    fixtures: [fixtures/app.log]
    check: exact
    annotations:
      - lines: "3"
        note: >
          open() is os.Open: it returns a FileHandle or rejects (ENOENT,
          EACCES) right here, before any reading starts.
      - lines: "6-12"
        note: >
          readLines() wraps a read stream in a readline interface. The file is
          read 64 KiB at a time and memory stays flat however big it is, as
          with bufio.Scanner. Unlike Scanner, readline has no maximum line
          length: a file without newlines becomes one huge string.
      - lines: "13-14"
        note: >
          No defer here: the read stream closes the file at end of input or on
          an error, and leaving the loop early (break or throw) closes it too.
          A FileHandle you open and never read from must be closed yourself,
          in finally, or with `await using` on Node 24.
      - lines: "16-20"
        note: >
          Go randomizes map iteration on purpose; a JS Map always iterates in
          insertion order, so the output would already be deterministic.
          Sorting matches the Go version's output. Plain sort() compares
          UTF-16 code units, the right order for identifiers; for Turkish
          words use localeCompare with "tr" or Intl.Collator.
  - id: pipeline
    file: examples/03_pipeline.ts
    title: "Chunks, decoding and stream.pipeline"
    runtime: ts-node
    fixtures: [fixtures/app.log]
    check: exact
    annotations:
      - lines: "5-10"
        note: >
          highWaterMark 4 forces 4-byte chunks (the default is 64 KiB) so the
          bug shows on a short line. Decoding each chunk alone turns a
          two-byte ı or ğ split across chunks into two U+FFFD characters. With
          real files it happens wherever a 64 KiB boundary falls inside a
          letter, which ASCII-only test data never catches.
      - lines: "11-16"
        note: >
          With an encoding the stream decodes for you and keeps an incomplete
          trailing sequence until the next chunk arrives. readline and
          readLines() do the same. Go's Scanner can't hit this bug: it splits
          raw bytes at "\n", a byte that never occurs inside a multi-byte
          UTF-8 letter.
      - lines: "20-25"
        note: >
          An async generator is a complete transform stage: it pulls a line,
          yields zero or more chunks, and pipeline() asks for the next line
          only when the file stream has accepted the output. That is
          backpressure. In Go the same pull happens because a Reader is read
          only when someone calls Read.
      - lines: "26-28"
        note: >
          pipeline is Node's io.Copy with the error handling included. It
          connects the stages, passes along the first error, and resolves
          only after the write stream has finished writing errors.log, which
          is why the next line can read the file back.
      - lines: "30-38"
        note: >
          The rejected pipeline destroyed the write stream, so no file
          descriptor leaks. partial.log still exists with 0 bytes, because
          createWriteStream had already opened and truncated it. When the
          target matters, write a temp file and rename it, exactly like the
          atomic write in the Go lesson.
use_cases:
  - "readFile: config files, templates, small JSON or CSV, fixtures in tests"
  - "open() + readLines(): logs, large exports, anything of unknown size read line by line"
  - "pipeline: copy, compress (zlib.createGzip), hash or filter data between files, sockets and HTTP bodies with backpressure and one error path"
  - "A CLI that reads stdin line by line: readline.createInterface({ input: process.stdin }) takes the same loop as readLines()"
strengths:
  - "Async iteration turns push-based streams into a pull loop that reads like Go's Scanner loop, and async generators make transforms plain functions"
  - "pipeline gives streams what io.Copy has: backpressure, one error, and every stage destroyed on failure"
  - "Waiting for the disk never blocks the event loop: fs promises run on the libuv threadpool, so one thread keeps serving other work in the meantime"
  - "Setting an encoding on the stream handles multi-byte UTF-8 split across chunk boundaries"
weaknesses:
  - "Several API generations side by side (callbacks, *Sync, promises, streams, web streams), where Go has one io.Reader/io.Writer pair"
  - "Stream errors are 'error' events: with no listener the process crashes, and .pipe() doesn't forward them. You need pipeline() to get Go's single returned error"
  - "readline has no maximum line length; Go's Scanner stops at 64 KB with ErrTooLong instead of growing a string up to V8's limit (about 512 MiB of UTF-16, then ERR_STRING_TOO_LONG)"
  - "The libuv threadpool has 4 threads by default (UV_THREADPOOL_SIZE), so many slow file operations queue behind each other; goroutines have no such shared pool"
pitfalls:
  - title: "Connecting streams with .pipe()"
    detail: ".pipe() moves data but not errors: if the source fails, the destination stays open, and an 'error' event nobody listens to terminates the process. pipeline() forwards the error and destroys every stage."
    bad: "createReadStream(src).pipe(createGzip()).pipe(createWriteStream(dst));"
    good: "await pipeline(createReadStream(src), createGzip(), createWriteStream(dst));"
  - title: "Decoding Buffer chunks one at a time"
    detail: "chunk.toString() on each chunk corrupts any letter split across a boundary into U+FFFD. Pass encoding: \"utf8\" to the stream, use readline, or a TextDecoder with { stream: true }."
  - title: "readFileSync in a server"
    detail: "A *Sync call blocks the only JavaScript thread, so every other request waits for the disk. It is fine at startup or in a CLI; in request handlers use the promise API."
  - title: "Relative paths resolve against process.cwd()"
    detail: "They don't resolve against the script's directory, the same trap as in Go. Build paths from import.meta.dirname or new URL(\"./data.txt\", import.meta.url) when the file ships next to the code."
  - title: "Opening a FileHandle and relying on garbage collection"
    detail: "Node may close an unclosed FileHandle on garbage collection and warn, and its docs say not to rely on it. Close it in finally or use `await using`."
    bad: "const f = await open(path);\nconst header = await f.read(buf, 0, 16, 0);"
    good: "const f = await open(path);\ntry {\n  await f.read(buf, 0, 16, 0);\n} finally {\n  await f.close();\n}"
quiz:
  - q: "You decode each chunk of createReadStream(path) with chunk.toString(). What can go wrong with Turkish text?"
    options:
      - "Nothing; UTF-8 is self-synchronizing"
      - "A two-byte letter split across two chunks turns into replacement characters"
      - "toString() throws on invalid UTF-8"
    answer: 1
    why: "Each chunk is decoded alone, so a sequence cut in half is invalid in both halves and becomes U+FFFD. An encoding on the stream keeps the incomplete bytes for the next chunk."
  - q: "Which is Node's counterpart of Go's `_, err := io.Copy(dst, src)`, error handling included?"
    options:
      - "src.pipe(dst)"
      - "await pipeline(src, dst)"
      - "dst.write(await readFile(path))"
    answer: 1
    why: "pipeline propagates the first error from any stage and destroys the others; .pipe() forwards neither errors nor cleanup, and readFile loads everything into memory."
  - q: "A 2 GB log has to be counted on a small VM. Which do you use?"
    options:
      - "readFile(path, \"utf8\") and split(\"\\n\")"
      - "open(path) and for await over readLines()"
    answer: 1
    why: "readLines holds one 64 KiB chunk and one line at a time. readFile would need the whole file in memory, and a 2 GB string exceeds V8's maximum string length anyway."
  - q: "Why does the Go version sort map keys, while the Node version would be deterministic without sorting?"
    options:
      - "Go maps randomize iteration order; a JS Map iterates in insertion order"
      - "Node sorts Map keys automatically"
      - "Both randomize; Node just happens to be lucky"
    answer: 0
    why: "Go deliberately randomizes map iteration. A JS Map iterates in insertion order, a guarantee of the language."
further_reading:
  - title: "Node.js: File system (fs/promises, FileHandle.readLines)"
    url: "https://nodejs.org/api/fs.html"
  - title: "Node.js: Stream (pipeline, backpressure, async iteration)"
    url: "https://nodejs.org/api/stream.html"
  - title: "Node.js: Readline"
    url: "https://nodejs.org/api/readline.html"
  - title: "Node.js: Buffer"
    url: "https://nodejs.org/api/buffer.html"
  - title: "package io (Go)"
    url: "https://pkg.go.dev/io"
---

## Concept

The Go lesson had two levels: `os.ReadFile` for the whole file, `os.Open` plus
`bufio.Scanner` for a constant-memory pass. Node has the same two, plus a third piece
that Go builds into `io.Reader` itself:

- `readFile(path)` returns a `Buffer` (bytes), or a string when you pass an encoding.
- `open(path)` returns a `FileHandle`; `for await (const line of handle.readLines())`
  is the Scanner loop.
- **streams** carry data in chunks between a source and a destination, and
  `stream.pipeline()` connects them with backpressure and a single error path, which is
  what `io.Copy` does in Go.

Everything here is asynchronous. `await` doesn't block a thread the way a Go read
blocks a goroutine: it hands control back to the event loop until the threadpool has
the data.

## Coming from C/C++/Java/Python/JS

- Go: `os.ReadFile` ↔ `readFile`; `os.Open` + `bufio.Scanner` ↔ `open` + `readLines()`;
  `io.Copy`/`io.Pipe` ↔ `pipeline`; `errors.Is(err, fs.ErrNotExist)` ↔
  `err.code === "ENOENT"`; `defer f.Close()` ↔ `try`/`finally` (or `await using` on
  Node 24); `[]byte` ↔ `Buffer`.
- Python: `Path.read_bytes()` ↔ `readFile`; `for line in f` ↔ `readLines()`. Python's
  text mode decodes across buffer boundaries for you; Node does so only when you set
  an encoding.
- Java: `Files.readAllBytes` / `Files.lines`. Instead of an InputStream decorator
  hierarchy, Node composes stages with `pipeline` and async generators.
- C: the chunk-boundary bug in example 3 is the multibyte problem `mbrtowc` keeps
  conversion state for.
- Browser JS: `File.text()` and `File.stream()` (lesson ts.browser-dom.reading-files)
  are web streams. Node has those too (`Readable.toWeb`, `Readable.fromWeb`), but its
  fs and http APIs hand you Node streams.

## How it works

```mermaid
flowchart LR
  disk["app.log"] -->|"64 KiB reads on the libuv threadpool"| rs["ReadStream buffer"]
  rs -->|"for await pulls"| gen["async generator"]
  gen -->|"write; waits when full"| ws["WriteStream buffer"]
  ws --> out["errors.log"]
```

A readable stream reads ahead until its buffer holds `highWaterMark` bytes (64 KiB for
files), then stops until someone consumes. A writable stream reports "full" the same
way. `for await` and `pipeline` consume only as fast as the next stage accepts, so a
fast disk can't flood a slow network socket: that is backpressure. Without it, a
naive `on("data")` handler that writes to a slow destination buffers the whole file in
memory.

## Design notes

Same advice as in Go: keep file opening at the edge and give the logic an iterable. In
Go that is `func countLevels(r io.Reader)`. In TypeScript it is
`async function countLevels(lines: AsyncIterable<string> | Iterable<string>)`, since
`for await` accepts both kinds. Production passes `handle.readLines()` or a readline
over `process.stdin`; a test passes a plain array of strings, no file needed. Reach
for `pipeline` whenever two streams meet: it is the only form where errors and cleanup
are correct by construction. Choose `readFile` only when the size is bounded and
known.
