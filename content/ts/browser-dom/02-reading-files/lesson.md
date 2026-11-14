---
id: ts.browser-dom.reading-files
title: "Reading user-selected files: File and Blob"
track: ts
module: browser-dom
order: 2
level: intermediate
minutes: 10
since: ts4.4
prerequisites: [ts.async.promises, ts.narrowing.discriminated-unions]
compare_with: [go.files-io.reading-files, sql.files-movement.csv-staging]
tags: [files, blob, file-api, browser]
summary: >
  The browser wraps a user-selected file in a File (a Blob with a name), and
  hands you .text(), .arrayBuffer() and .stream() — Promises instead of the
  legacy FileReader callbacks.
examples:
  - id: read-file
    file: examples/01_read_file.ts
    title: Size guards, text(), and streaming a big file
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "6-8"
        note: >
          File extends Blob: the bytes plus name/type/lastModified. You never
          get a path — the browser gives you the content, not the filesystem.
      - lines: "10"
        note: >
          Check size before reading. .text() loads everything; a 2 GB "CSV"
          without this guard freezes the tab — the trap Go's os.ReadFile has too.
      - lines: "23"
        note: >
          .text() returns a Promise: await it. The legacy FileReader and its
          onload callbacks are unnecessary for whole-file reads.
      - lines: "12-18"
        note: >
          .stream() is a ReadableStream: constant memory for big files, the
          browser twin of bufio.Scanner. getReader() + read() yields chunks.
use_cases:
  - "CSV/JSON import in a UI before upload (validate client-side first)"
  - "Drag-and-drop handlers (e.dataTransfer.files gives File objects)"
  - "Generating exports: new Blob([...]) + object URL + click an <a download>"
strengths:
  - ".text()/.arrayBuffer()/.stream() are one-liners with no callback plumbing"
  - ".stream() handles files far larger than memory"
  - "File/Blob work in drag-and-drop, input elements and clipboard identically"
weaknesses:
  - "No path access: you cannot re-read the same file later without asking the user again"
  - ".text() decodes as UTF-8 by default; other encodings need TextDecoder with an explicit label"
  - "Legacy Safari-era code and many tutorials still use FileReader callbacks"
pitfalls:
  - title: "Reading whole files without a size guard"
    detail: "One .text() on a huge file locks the tab. Guard with file.size, stream with .stream() beyond a threshold."
  - title: "Assuming File.name is a safe path"
    detail: "It's a name, possibly '../evil'. If you send it anywhere, treat it as untrusted input."
  - title: "Mixing up Blob and File types in APIs"
    detail: "Accept Blob in helpers; File is a subtype, so both work."
quiz:
  - q: "file.name is '../etc/passwd'. What is safe?"
    options: ["Open it relative to cwd", "Display it, but treat it as untrusted data — never a path", "Nothing; reject all files with slashes"]
    answer: 1
    why: "The name is display metadata only; the browser gives you content, not filesystem access."
  - q: "You need to read a possibly-500 MB log in the browser. Which API?"
    options: ["file.text()", "file.stream() and process chunks", "FileReader.readAsBinaryString"]
    answer: 1
    why: ".text() materializes the whole content; .stream() yields chunks with constant memory."
further_reading:
  - title: "MDN: File"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/File"
  - title: "MDN: Blob.text()"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Blob/text"
  - title: "MDN: ReadableStream"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/ReadableStream"
---

## Concept

`<input type="file">` (or drag-and-drop) gives you `FileList` of `File` objects. A `File` is
a `Blob` — immutable bytes plus metadata — with `.text()`, `.arrayBuffer()`, `.stream()` and
`.slice()`. Whole-file reads are one awaited call; big files stream chunk by chunk.

## Coming from C/C++/Java/Python/JS

- Go: `os.ReadFile` vs `bufio.Scanner` maps exactly onto `.text()` vs `.stream()` — but the
  browser starts from the user's choice, not a path.
- Node: `fs.readFile` takes paths; the browser deliberately never does (security).
- C: no FILE* here. The sandbox is the point: the page sees only what the user picked.

## Design notes

Treat client-side reading as *pre-validation before upload*: parse, check, then send only
what's needed. The browser sandbox (no paths, user-initiated reads) is worth internalizing —
it's the same least-privilege thinking as the SQL playground role, which owns only sandbox
objects. The TraceLite import step (Phase 7) wires this lesson's pattern to a Go server.
