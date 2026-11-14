---
id: go.services.pagination-streaming
title: "Cursor pagination and streaming large responses"
track: go
module: services
order: 3
level: advanced
minutes: 14
since: go1.23
prerequisites: [go.services.client, go.iterators.writing]
compare_with: [sql.indexes.stats-pagination, ts.services.advanced, go.files-io.composing]
tags: [http, pagination, cursor, keyset, iterators, streaming, downloads]
summary: >
  Wrap a paged API in an iterator that follows the cursor and fetches lazily; prefer
  cursors over offsets, which skip and repeat rows while others write. Stream big
  downloads to disk through a hash, with a stall watchdog instead of a total timeout.
examples:
  - id: cursor-iterator
    file: examples/01_cursor_iterator.go
    title: "A paged API as an iter.Seq2 that fetches pages on demand"
    runtime: go
    check: exact
    annotations:
      - lines: "13-18"
        note: >
          The usual page shape: the items plus an opaque cursor for the next
          page, absent on the last one. The client never parses the cursor; it
          only sends it back.
      - lines: "22-42"
        note: >
          iter.Seq2 (Go 1.23) lets callers write a plain for-range over every
          item and never see pages. Paging and cursors stay in one function.
      - lines: "27-30"
        note: >
          An error is yielded as a value, once, and then the sequence ends.
          That is the common convention for Seq2[T, error]: the caller decides
          whether to stop or log and continue.
      - lines: "31-35"
        note: >
          When yield returns false the caller has left its loop (break, return,
          panic). The iterator must stop right there; calling yield again
          panics at run time.
      - lines: "45-46"
        note: >
          The cursor here is just the last ID, base64-encoded so clients treat
          it as opaque. Real APIs often also sign or encrypt it, so they can
          change what's inside without breaking clients.
      - lines: "47-51"
        note: >
          The IDs are Turkish licence-plate codes, and pages are ordered by ID:
          behind this API runs a keyset query, WHERE id > $1 ORDER BY id
          LIMIT 3. Ordering by name would be harder: compared byte by byte,
          İstanbul and İzmir sort after Konya, because İ is a two-byte
          character beyond ASCII. Names need a Turkish collation
          (golang.org/x/text/collate).
      - lines: "53-56"
        note: >
          fetch is a parameter, so a fake API can stand in for the real one.
          In a client type it's the method that sends lesson 1's GET with a
          context and decodes the JSON page.
      - lines: "64-66"
        note: >
          Laziness pays off: the loop stops at Bursa, the iterator returns, and
          the third page is never requested.
  - id: offset-drift
    file: examples/02_offset_drift.go
    title: "Offset pages repeat and skip rows when the data changes; cursors don't"
    runtime: go
    check: exact
    annotations:
      - lines: "11-14"
        note: >
          OFFSET counts positions. The returned page is a copy, as it would be
          after a JSON round trip; returning a subslice would let the
          in-place DeleteFunc below change page 1 after the fact.
      - lines: "16-26"
        note: >
          A keyset cursor names the last row seen. Rows inserted before it or
          deleted after it can't shift the next page.
      - lines: "41-44"
        note: >
          A new post at the top pushes everything down one position, so the
          offset client sees post 8 twice. A deleted post pulls everything up,
          so post 7 is never seen at all. Neither failure produces an error.
  - id: stream-download
    file: examples/03_stream_download.go
    title: "Stream 10 MB to disk through a hash; rename only when verified"
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "30-34"
        note: >
          Write to a temp file next to the target, and rename only when the
          file is complete and verified. A failed download never leaves a
          half-written cities.csv for someone else to read.
      - lines: "36"
        note: >
          io.Copy moves the body through one 32 KiB buffer, and MultiWriter
          feeds each chunk to the file and the hash, so memory stays flat
          whether the body is 10 MB or 10 GB. io.ReadAll would hold all of it.
      - lines: "40-43"
        note: >
          Content-Digest (RFC 9530) carries a hash of the bytes as sent. A
          body that ends early looks like a normal end of stream to io.Copy
          (no Content-Length here); only the digest catches it, and the file
          is not renamed into place.
      - lines: "48"
        note: >
          Each line is 30 characters but 34 bytes: İ, Ş and ı take two bytes
          each in UTF-8. Sizes, Content-Length and offsets are always in
          bytes.
  - id: stall-watchdog
    file: examples/04_stall_watchdog.go
    title: "Bound stalls, not total time: a watchdog that cancels with a cause"
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "18-30"
        note: >
          A reader that wraps a reader: every chunk that arrives pushes the
          stall deadline back. This bounds "no progress", which is what you
          actually care about for a download of unknown duration.
      - lines: "34-38"
        note: >
          The transfer takes 2 s, longer than the 1 s idle limit, and is never
          cut off: only a gap without data counts.
      - lines: "43-45"
        note: >
          Clone keeps DefaultTransport's settings and adds a bound on the wait
          for headers. There is deliberately no Client.Timeout, which would
          also cut off a slow but healthy 10 GB download.
      - lines: "47-48"
        note: >
          WithCancelCause (Go 1.20) lets the watchdog say why it cancelled.
          The Transport reports context.Cause from the body read, so the error
          the caller sees is errStalled, not a bare "context canceled".
      - lines: "58-59"
        note: >
          The watchdog starts once the headers are in; until then the
          Transport's ResponseHeaderTimeout is in charge.
use_cases:
  - "Syncing every record from a SaaS API (customers, invoices, tickets) into your database"
  - "Exports and backups that are larger than the memory of the process downloading them"
  - "Infinite-scroll feeds and admin tables whose data changes while users page through it"
strengths:
  - "Range-over-func iterators let a paged API look like a slice to its callers, and stop fetching when they stop reading"
  - "io.Reader composition (wrappers, MultiWriter, TeeReader) builds a streaming pipeline in a few lines, with constant memory"
  - "Keyset pagination is stable under concurrent writes and as fast on the last page as on the first"
  - "A cancel cause makes timeouts self-describing: the error says stalled, not just canceled"
weaknesses:
  - "Cursors can't jump to page 37 or show a total page count; offsets can, at the cost of correctness and speed"
  - "Iterator errors need a convention (Seq2 with error), since a for-range has no error return"
  - "Streaming means the caller sees partial data before it is validated; the temp-file-and-rename discipline is on you"
  - "There's no built-in progress or stall timeout on Response.Body; you wrap the reader yourself"
pitfalls:
  - title: "Reading a large body with io.ReadAll"
    detail: "It holds the whole response in memory, growing the buffer by repeated reallocation along the way. Stream to the destination with io.Copy, and cap what you accept with io.LimitReader if the size is untrusted."
    bad: |
      data, err := io.ReadAll(resp.Body)
      os.WriteFile("export.csv", data, 0o644)
    good: |
      n, err := io.Copy(f, io.LimitReader(resp.Body, maxBytes))
  - title: "Writing the final file in place"
    detail: "A download that fails halfway leaves a truncated file under the real name, and the next run or another process trusts it. Write to a temp file in the same directory and rename after verification, so readers see the old file or the new one, never half of one (go.files-io.writing)."
  - title: "Verifying a digest after transparent decompression"
    detail: "If the Transport asked for gzip itself, it decompresses the body for you and deletes Content-Encoding (resp.Uncompressed is true). Your hash then covers decoded bytes, while Content-Digest covers the gzip bytes that were sent. Set Transport.DisableCompression, or request the encoding explicitly, when you verify digests."
  - title: "Forgetting to stop the iterator"
    detail: "An iterator that ignores yield's false return keeps fetching pages nobody reads, and calling yield after it returned false panics. Return as soon as yield says stop."
  - title: "Parsing or constructing cursors on the client"
    detail: "Cursors are opaque by contract; the server may change their format at any time. Store and send them back verbatim, URL-encoded."
quiz:
  - q: "A client pages through /posts?offset=N newest first. A new post is published between page 1 and page 2. What does the client see?"
    options: ["An error", "Nothing different", "The last post of page 1 again at the top of page 2", "The new post at the top of page 2"]
    answer: 2
    why: "Everything moved down one position, so offset 3 now points at what used to be the third row. Keyset cursors don't move with positions."
  - q: "In a for c, err := range cities(...) loop the caller hits break. What must the iterator function do?"
    options: ["Fetch the rest of the pages so the connection can be reused", "Return immediately: yield returned false", "Call yield once more with io.EOF", "Nothing: the runtime stops it"]
    answer: 1
    why: "yield returning false means the loop is over; calling yield again panics, and the iterator should stop fetching."
  - q: "Why is Client.Timeout a poor fit for a 5 GB download?"
    options: ["It doesn't apply to downloads", "It caps the whole exchange including the body, so a slow but healthy download is cut off", "It only applies to headers", "It disables connection reuse"]
    answer: 1
    why: "Client.Timeout includes reading the body. A stall watchdog bounds the absence of progress instead of the total time."
  - q: "What does io.Copy(io.MultiWriter(file, hash), resp.Body) hold in memory?"
    options: ["The whole body", "One buffer of a fixed size", "Two copies of the body", "Nothing: it uses sendfile"]
    answer: 1
    why: "Neither side implements WriterTo or ReaderFrom here, so io.Copy loops over a single 32 KiB buffer and writes each chunk to both destinations."
further_reading:
  - title: "package iter"
    url: "https://pkg.go.dev/iter"
  - title: "Range over function types (Go blog)"
    url: "https://go.dev/blog/range-functions"
  - title: "package io: Copy, MultiWriter"
    url: "https://pkg.go.dev/io#Copy"
  - title: "package context: WithCancelCause, Cause"
    url: "https://pkg.go.dev/context#WithCancelCause"
  - title: "MDN: Content-Digest"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Content-Digest"
  - title: "PostgreSQL: LIMIT and OFFSET"
    url: "https://www.postgresql.org/docs/current/queries-limit.html"
---

## Concept

APIs that return collections hand them out in pages, and there are two ways to say
"the next page":

- **Offset**: `?offset=40&limit=20`, "skip 40 rows". Simple, and lets a UI jump to any
  page, but the server still reads and discards the skipped rows, and positions shift
  whenever someone inserts or deletes: rows are repeated or silently skipped.
- **Cursor (keyset)**: `?cursor=...`, where the cursor encodes the last row you saw, and
  the server continues *after* it (`WHERE id > $1 ORDER BY id LIMIT 20`). Stable under
  concurrent writes, and as fast on page 10,000 as on page 1, but only forward (and
  backward), with no "page 37".

On the client, wrap the paging in an **iterator** (`iter.Seq2[T, error]`, Go 1.23):
callers range over items, pages are fetched only as the loop asks for more, and a
`break` stops the fetching.

Large responses are **streams**. `resp.Body` is an `io.Reader`; copy it to its
destination in chunks instead of reading it into memory. Downloads get two extra
guards: write to a temp file and rename only after verifying (size, digest), and bound
*stalls* rather than total time, because a healthy 10 GB transfer may take an hour.

## Coming from C/C++/Java/Python/JS

- **Python**: a generator that `yield`s items and fetches the next page inside the loop
  is exactly `iter.Seq2`; `requests.get(url, stream=True)` with `iter_content()` is the
  streaming half.
- **Java**: you would implement `Iterator<T>` or `Spliterator` (or a reactive
  `Flux`) for the pages; `InputStream.transferTo(OutputStream)` is `io.Copy`.
- **JS/TS**: an `async function*` generator consumed with `for await`, and
  `response.body` as a `ReadableStream` piped through a `TransformStream` (lesson
  ts.services.advanced). Go's version is synchronous code; the goroutine blocks, nothing
  else does.
- **C**: a `read()` loop into a fixed buffer, plus `rename(2)` for the atomic replace:
  the same design, with the error handling done by hand.

## How it works

A range-over-func loop calls the iterator once, passing `yield`; each `yield(item, nil)`
runs the loop body. A `break` in the body makes `yield` return false, and the iterator
must return then. That's how the stop propagates without channels or goroutines: the
whole thing is ordinary function calls on one stack.

In example 4, the stall watchdog is a `time.AfterFunc` that cancels the request's
context with a cause. Cancelling the context makes the Transport close the connection
and fail the pending `Read` on the body; the error the Transport reports is
`context.Cause(ctx)`, so `io.Copy` returns `errStalled` itself.

## Design notes

When you design the API (the server side), prefer cursors for anything that is written
to while being read, and make them opaque: base64 at least, signed if clients could
gain anything by forging them. Order by a unique key, or by a tuple ending in one
(`ORDER BY created_at, id` with `WHERE (created_at, id) > ($1, $2)`), or ties at page
boundaries produce the same skip-and-repeat bugs as offsets. The SQL side, including the
index that makes this fast, is in sql.indexes.stats-pagination.

On the consuming side, an iterator is the right boundary: the sync job that needs every
invoice writes `for inv, err := range client.Invoices(ctx)`, and retries (lesson 2) live
inside `fetch`. Keep the cursor of the last completed page, and a crashed sync can resume
where it stopped instead of starting over.

For downloads, decide what "done" means before starting: a byte count, a digest, or both.
Then nothing else may see the file until it is done. The same pattern (temp file, fsync
if durability matters, rename) is in go.files-io.writing.
