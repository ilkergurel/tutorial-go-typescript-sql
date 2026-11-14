---
id: go.files-io.composing
title: "Composing readers and writers: io.Copy, TeeReader, MultiWriter, LimitReader"
track: go
module: files-io
order: 3
level: intermediate
minutes: 13
since: go1.16
prerequisites: [go.files-io.writing, go.interfaces.implicit]
compare_with: [ts.node.fs-streams, go.http.uploads-streams, ts.browser-dom.reading-files]
tags: [io, reader, writer, streams, pipe, composition]
summary: >
  io.Reader and io.Writer have one method each, so small adapters wrap them
  like Unix pipes: MultiWriter fans out, TeeReader taps, LimitReader caps,
  MultiReader concatenates and io.Pipe connects a writer to a reader.
examples:
  - id: copy-tee-multi
    file: examples/01_copy_tee_multi.go
    title: Copy once to two places; read once for two consumers
    runtime: go
    check: exact
    annotations:
      - lines: "26-27"
        note: >
          sha256.New returns a hash.Hash, which is an io.Writer. MultiWriter
          sends every chunk to the file and to the hash, so the checksum costs
          no second pass over the data. Swap the strings.Reader for an HTTP body
          and nothing else changes.
      - lines: "34"
        note: >
          io.Copy counts bytes, not characters: İ, Ş, ı and ğ take two bytes
          each in UTF-8, hence 69 bytes for 64 characters. Size limits and
          Content-Length are in bytes.
      - lines: "42-47"
        note: >
          TeeReader copies whatever its consumer reads into the hash. The hash
          sees only what was actually read: if the loop stopped early, the
          checksum would cover a prefix, so check sc.Err before trusting it.
      - lines: "52"
        note: >
          Sum(nil) appends the digest to nil and returns it; it doesn't reset
          the hash. bytes.Equal compares the two 32-byte results.
  - id: limit-multi
    file: examples/02_limit_multi.go
    title: LimitReader with the +1 trick, byte limits vs characters, MultiReader
    runtime: go
    check: exact
    annotations:
      - lines: "15-24"
        note: >
          LimitReader returns a plain EOF at the limit, so a body of exactly max
          bytes and a much longer one look the same. Reading max+1 bytes and
          checking the length tells them apart without reading the rest.
      - lines: "27-30"
        note: >
          Kastamonu is exactly 9 bytes and passes. Eskişehir is 9 characters
          but 10 bytes, so a 9-byte limit rejects it: limits are in bytes.
      - lines: "33-34"
        note: >
          Cutting at a byte count can split a character: the third byte is the
          first half of ı. Trim previews with a rune-aware loop, or back off to
          a boundary with utf8.RuneStart, before showing them to a user.
      - lines: "37-42"
        note: >
          MultiReader reads each reader to EOF, then moves to the next. Typical
          uses: prepend a header, stitch file parts together, or put back bytes
          you already consumed to sniff a format.
  - id: own-writer-pipe
    file: examples/03_own_writer_pipe.go
    title: Your own Writer, and io.Pipe between a producer and a consumer
    runtime: go
    check: exact
    annotations:
      - lines: "13-21"
        note: >
          A type satisfies io.Writer by having this method; there is nothing to
          declare. Return len(p) and nil on success: a Writer that reports
          fewer bytes than len(p) must also return an error.
      - lines: "29"
        note: >
          io.Pipe connects code that wants an io.Writer (the encoder) with code
          that wants an io.Reader (io.Copy) without a temp file or a buffer of
          the whole output. Each side must run in its own goroutine, or the
          first Write blocks forever.
      - lines: "39"
        note: >
          Always close the writer side. CloseWithError(nil) gives the reader
          EOF; a real error reaches the reader as its Read error, so a failed
          producer can't look like a short, successful stream.
      - lines: "43-46"
        note: >
          Each Encode is one Write, the pipe hands it to one Read, and io.Copy
          passes it on: three records, three writes.
use_cases:
  - "Hashing or counting while copying: checksums of uploads, progress bars, metrics"
  - "Logging to the console and a file at once with io.MultiWriter"
  - "Capping untrusted input: io.LimitReader for files and streams, http.MaxBytesReader for request bodies"
  - "io.Pipe: streaming an encoder's output (JSON, gzip, CSV) into an HTTP request or upload without a temp file"
strengths:
  - "Two one-method interfaces cover files, sockets, HTTP bodies, hashes, compressors and buffers, without Java's InputStream/Reader class hierarchies and their adapters"
  - "Adapters are values you nest, like a Unix pipeline, and they cost a function call per chunk, not a thread or an event loop"
  - "io.Copy uses WriterTo/ReaderFrom when the types have them; on Linux, file-to-file and file-to-socket copies can then go through copy_file_range, splice or sendfile"
weaknesses:
  - "Blocking calls keep backpressure simple, but io.Pipe needs a second goroutine and leaks it if a side never closes"
  - "Wrapping hides capabilities: io.MultiWriter has no ReadFrom, so a copy through it loses the zero-copy path a bare *os.File would get"
  - "LimitReader stops silently with EOF, so it can't tell you the input was too large; you need the +1 check or http.MaxBytesReader"
pitfalls:
  - title: "Trusting LimitReader to report oversize input"
    detail: "It truncates and returns EOF. Read max+1 and compare, or in HTTP handlers use http.MaxBytesReader, which returns a *http.MaxBytesError."
    bad: "data, err := io.ReadAll(io.LimitReader(r, max)) // a 2 GB body becomes a valid-looking prefix"
    good: "data, err := io.ReadAll(io.LimitReader(r, max+1))\nif err != nil { return err }\nif int64(len(data)) > max { return errTooLarge }"
  - title: "Never closing the PipeWriter"
    detail: "The reader waits for EOF forever and both goroutines leak. Close it on every path, with CloseWithError(err) so failures propagate."
  - title: "Checksum from a TeeReader whose consumer stopped early"
    detail: "The hash covers only what was read. Drain the rest (io.Copy(io.Discard, r)) or treat an early stop as an error."
  - title: "Cutting text by bytes"
    detail: "A byte limit can split a multi-byte character such as ş, which then shows up as \\xc5 or U+FFFD. Limit storage by bytes, but cut display text at rune boundaries."
quiz:
  - q: "io.ReadAll(io.LimitReader(r, 1024)) returns exactly 1024 bytes and no error. What do you know about r?"
    options: ["It had exactly 1024 bytes", "It had at least 1024 bytes", "It had more than 1024 bytes"]
    answer: 1
    why: "LimitReader reports EOF at the limit whether or not r had more. Ask for 1025 to find out."
  - q: "A goroutine writes to an io.PipeWriter and returns without closing it. What happens to the io.Copy reading the other end?"
    options: ["It gets EOF when the goroutine exits", "It blocks forever", "It panics"]
    answer: 1
    why: "Pipes have no idea the writer goroutine ended; only Close or CloseWithError ends the stream."
  - q: "You need a SHA-256 of an upload while saving it to disk, in one pass. Which composition?"
    options: ["io.Copy(io.MultiWriter(file, hash), body)", "io.Copy(file, io.LimitReader(body, hash))", "io.MultiReader(file, hash)"]
    answer: 0
    why: "MultiWriter duplicates each chunk to both writers; io.TeeReader(body, hash) as the source would work too."
further_reading:
  - title: "package io"
    url: "https://pkg.go.dev/io"
  - title: "net/http: MaxBytesReader"
    url: "https://pkg.go.dev/net/http#MaxBytesReader"
  - title: "Effective Go: Interfaces and other types"
    url: "https://go.dev/doc/effective_go#interfaces_and_types"
---

## Concept

`io.Reader` (`Read(p []byte) (n int, err error)`) and `io.Writer` (`Write(p []byte) (n int,
err error)`) are the whole contract. Because they are this small, the `io` package can offer
adapters that take readers or writers and return new ones:

- `io.Copy(dst, src)` moves bytes until EOF, in 32 KB chunks or through a fast path.
- `io.MultiWriter(w1, w2, ...)` writes every chunk to all of them (Unix `tee`).
- `io.TeeReader(r, w)` returns a reader that also writes what it reads into `w`.
- `io.LimitReader(r, n)` stops with EOF after `n` bytes.
- `io.MultiReader(r1, r2, ...)` reads one after another, as one stream.
- `io.Pipe()` gives a connected reader and writer: what one goroutine writes, another reads.

Code that accepts an `io.Reader` or `io.Writer` works with all of these, and with files,
network connections, HTTP bodies, `bytes.Buffer`, hashes and compressors.

## Coming from C/C++/Java/Python/JS

- Unix shell: this is `cat a b | tee file | head -c N`, as values in one process.
- Java: `InputStream`/`OutputStream` decorators (`BufferedInputStream`, `DigestInputStream`,
  `SequenceInputStream`) do the same job, but through a class hierarchy, and with a
  separate `Reader`/`Writer` family for characters. Go has one family, for bytes; text is
  bytes that happen to be UTF-8.
- Python: file-like objects are duck-typed the same way, but `shutil.copyfileobj` has no
  standard tee or limit wrapper; you write a small class.
- Node: closest to `stream.pipeline` with `PassThrough` and transform streams, minus events
  and callbacks. A Go `Read` blocks its goroutine; it doesn't block the process, so plain
  sequential code gets backpressure for free.

## How it works

`io.Copy` first looks for shortcuts: if `src` has a `WriteTo` method, or `dst` has
`ReadFrom`, it lets them do the copy. `*os.File` has both, and on Linux they use
`copy_file_range`, `splice` or `sendfile`, so file-to-file and file-to-socket copies may never
pass through your process's memory. Otherwise `io.Copy` loops over a 32 KB buffer.

`io.Pipe` has no buffer at all: a `Write` blocks until `Read` calls have taken every byte.
That makes it a synchronization point between two goroutines, which is why the producer in
the third example runs in its own goroutine and closes the writer when it is done.

## Design notes

Make functions take the narrowest interface they need (`io.Reader`, not `*os.File`), and
compose at the call site: the hash, the size limit and the progress counter are wrappers the
caller adds, not parameters of your parser. Put limits at the trust boundary, where untrusted
bytes enter (an HTTP handler, an upload, an archive entry), and keep them in bytes, because
memory and disks are measured in bytes; convert to characters only for display. Reach for
`io.Pipe` when one API insists on a `Writer` and the next on a `Reader`; if both sides are
yours, a direct function call with an `io.Writer` is simpler.
