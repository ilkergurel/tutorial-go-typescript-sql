---
id: go.http.uploads-streams
title: "File upload (multipart, size limits), streaming download, SSE, WebSockets"
track: go
module: http
order: 5
level: advanced
minutes: 15
since: go1.26
prerequisites: [go.http.server-lifecycle, go.files-io.composing]
compare_with: [ts.services.realtime-cors, go.services.pagination-streaming, ts.browser-dom.reading-files]
tags: [http, multipart, upload, streaming, sse, websocket]
summary: >
  Request and response bodies are streams: read uploads part by part under a
  size cap, write downloads and Server-Sent Events as they are produced, and
  flush. WebSockets leave HTTP after a 101 handshake; the standard library stops there.
examples:
  - id: upload
    file: examples/01_upload.go
    title: "Streaming a multipart upload under a size limit"
    runtime: go-server
    check: exact
    timeout_s: 10
    annotations:
      - lines: "19"
        note: >
          The cap goes on the whole body before anything reads it. Every part,
          header and boundary counts, so one limit covers files and fields.
      - lines: "20"
        note: >
          MultipartReader hands out parts in order as they arrive. Its
          alternative, ParseMultipartForm (also called by r.FormValue and
          r.FormFile, with 32 MB of memory), reads the whole body first:
          memory up to the limit, temp files for the rest.
      - lines: "25"
        note: >
          On HTTP/1.x a handler should finish reading the body before it
          writes the response. Collecting the report and writing it at the
          end also keeps a 413 possible until the last byte.
      - lines: "39-44"
        note: >
          io.Copy moves the part in 32 KiB chunks straight into the hash. Swap
          the hash for an os.File from os.CreateTemp or an object-storage
          writer and memory use stays the same for a 5 GB file.
      - lines: "45-46"
        note: >
          FileName() is the client's claim, passed through filepath.Base, which
          depends on the OS. "../../etc/passwd" arrives as "passwd". Never
          use it as a path: store under a name you generate, and keep the
          original (here "rapor-İş.csv", UTF-8) only as metadata.
      - lines: "53-61"
        note: >
          The client streams too: multipart.Writer writes into a pipe from a
          goroutine while http.Post reads the other end. Closing the pipe with
          mw.Close()'s error ends the body, or fails it.
      - lines: "77"
        note: >
          200,000 bytes against a 64 KiB cap: the server stops reading,
          answers 413 and closes the connection. Go's client returns that
          response even though it never finished sending the body.
  - id: sse
    file: examples/02_sse.go
    title: "Server-Sent Events: flush per event, stop when the client leaves"
    runtime: go-server
    check: exact
    timeout_s: 10
    annotations:
      - lines: "17-18"
        note: >
          text/event-stream is what makes a browser's EventSource accept the
          response. no-cache keeps proxies and the browser from storing a
          live stream.
      - lines: "21"
        note: >
          The wire format: "field: value" lines, a blank line ends the event.
          id lets EventSource send Last-Event-ID when it reconnects, so the
          server can resume; event picks the listener; data is the payload.
      - lines: "22-25"
        note: >
          Without Flush the event sits in a 4 KiB buffer until it fills.
          ResponseController.Flush reports when flushing is impossible, for
          example behind a middleware wrapper without Unwrap.
      - lines: "26-31"
        note: >
          A stream runs until someone ends it. The request context is canceled
          when the client disconnects, so the select is the only way out.
          Without it this goroutine would loop forever.
      - lines: "50-51"
        note: >
          No Content-Length (-1), so HTTP/1.1 uses chunked encoding: each
          flush sends a chunk. Any download written with io.Copy(w, src)
          without a length is streamed the same way.
      - lines: "64-67"
        note: >
          Canceling the request context closes the connection. The server
          notices only because it was waiting on r.Context().Done(); a handler
          that never looks would find out at its next failed write.
  - id: websocket-handshake
    file: examples/03_websocket_handshake.go
    title: "The WebSocket upgrade handshake with net/http, and one raw frame"
    runtime: go-server
    check: exact
    timeout_s: 10
    annotations:
      - lines: "15-18"
        note: >
          The only cryptography in the handshake, and it isn't security: it
          proves the server understood the WebSocket request, so a
          misconfigured plain HTTP server can't accidentally accept one.
      - lines: "21-25"
        note: >
          426 Upgrade Required with an Upgrade header tells a plain HTTP
          client which protocol this endpoint wants.
      - lines: "26-27"
        note: >
          Browsers send Origin on WebSocket requests but enforce nothing, so
          checking it is the server's job. Skipping the check enables
          cross-site WebSocket hijacking: another site's page opens a socket
          with your user's cookies.
      - lines: "28-33"
        note: >
          Hijack takes the TCP connection away from net/http: no more
          headers, timeouts or Shutdown tracking. It works only on HTTP/1.x;
          HTTP/2 connections don't support it.
      - lines: "37-42"
        note: >
          A minimal frame: FIN plus opcode 1 (text), then a length byte. The
          length counts bytes: "Merhaba dünya" is 14 bytes but 13 characters,
          and text frames must be valid UTF-8.
      - lines: "53"
        note: >
          The sample key from RFC 6455. Real clients send 16 random bytes in
          base64; the expected accept value s3pPLMBiTxaQ9kYGzzhZRbK+xOo= is
          printed in the RFC and on MDN, so you can check the computation.
      - lines: "64"
        note: >
          Since Go 1.12 the client hands back a 101 response whose Body is
          also an io.Writer: the raw connection, readable and writable.
use_cases:
  - "Uploads of CSV imports, images and documents, written to disk or object storage without holding them in memory"
  - "Exports and backups streamed with io.Copy or http.ServeContent (which adds Range and conditional requests)"
  - "SSE for one-way live updates: job progress, notifications, log tails, LLM token streams"
  - "WebSockets for two-way, low-latency traffic: chat, collaborative editing, games"
strengths:
  - "io.Reader and io.Writer make streaming the default shape: MultipartReader, io.Copy and Flush compose without callbacks, unlike Node's stream events or Java's servlet async API"
  - "Size limits are one wrapper (MaxBytesReader) with a typed error"
  - "SSE needs nothing beyond net/http: a header, a format and Flush; the browser's EventSource adds reconnection and Last-Event-ID for free"
weaknesses:
  - "No WebSocket implementation in the standard library: Hijack gives you a raw connection, and framing, masking, ping/pong and the close handshake are a library's job"
  - "Hijacked connections are invisible to Server.Shutdown and to server timeouts; you manage their lifecycle yourself"
  - "Streams fight with server-wide WriteTimeout, TimeoutHandler and buffering middleware such as compression, which need per-route exceptions"
  - "SSE is one-way, text-only, and over HTTP/1.1 limited to six connections per browser and domain"
pitfalls:
  - title: "r.FormValue on an upload endpoint"
    detail: "It calls ParseMultipartForm(32 MB) for you: up to 32 MB in memory and the rest in temp files, before your code sees a byte. Use MultipartReader for large or untrusted uploads."
  - title: "Trusting the file name and Content-Type of a part"
    detail: "Both come from the client. Generate the storage name, keep the original for display, and sniff the type with http.DetectContentType on the first 512 bytes if it matters."
    bad: "os.Create(filepath.Join(dir, part.FileName()))"
    good: "f, err := os.CreateTemp(dir, \"upload-*\") // original name goes to metadata"
  - title: "Writing an SSE stream without Flush"
    detail: "The events wait in the server's buffer and arrive in bursts or at the end. Flush after every event, and check that no middleware hides Flush."
  - title: "A non-ASCII download name in Content-Disposition"
    detail: "filename=\"rapor-İş.csv\" in a raw header is not portable. mime.FormatMediaType(\"attachment\", map[string]string{\"filename\": name}) produces filename*=utf-8''rapor-%C4%B0%C5%9F.csv, the RFC 2231 encoding browsers understand."
  - title: "Accepting WebSocket upgrades from any origin"
    detail: "CORS doesn't apply to WebSockets. Compare Origin against an allow-list before upgrading, or any site can open a socket with your users' cookies."
quiz:
  - q: "An upload handler calls r.FormFile(\"file\") on a 2 GB upload. What happens before the call returns?"
    options: ["Nothing; it streams lazily", "The whole body is parsed: up to 32 MB in memory, the rest in temp files", "It fails: the body is too large", "Only the headers are read"]
    answer: 1
    why: "FormFile calls ParseMultipartForm with 32 MB of memory if the form isn't parsed yet, which consumes the entire body."
  - q: "An SSE handler writes events but never calls Flush. What does the browser see?"
    options: ["Every event immediately", "Events in bursts when the buffer fills, or at the end", "An error", "Nothing ever"]
    answer: 1
    why: "Small writes sit in the server's response buffer until it fills or the handler returns; Flush pushes them out."
  - q: "Which statement about WebSockets in Go's standard library is true?"
    options: ["net/http implements WebSockets", "net/http can perform the upgrade through Hijack, but framing, masking and control frames need code of your own or a library", "WebSockets work only over HTTP/2", "Server.Shutdown closes WebSocket connections gracefully"]
    answer: 1
    why: "The standard library has no WebSocket package; after the 101 the connection is yours. Shutdown doesn't track hijacked connections."
  - q: "Why must a WebSocket server check the Origin header itself?"
    options: ["Browsers don't send Origin on WebSocket requests", "Browsers don't apply CORS to WebSocket connections", "Origin is required by TLS", "To pick the subprotocol"]
    answer: 1
    why: "Browsers send Origin but don't block the connection; without a server-side check, any site can connect with the user's cookies (cross-site WebSocket hijacking)."
further_reading:
  - title: "package net/http: Request.MultipartReader, MaxBytesReader, ResponseController"
    url: "https://pkg.go.dev/net/http#Request.MultipartReader"
  - title: "package mime/multipart (including its limits)"
    url: "https://pkg.go.dev/mime/multipart"
  - title: "MDN: Using server-sent events"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events"
  - title: "MDN: Writing WebSocket servers"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API/Writing_WebSocket_servers"
  - title: "package golang.org/x/net/websocket (and the alternatives its docs recommend)"
    url: "https://pkg.go.dev/golang.org/x/net/websocket"
---

## Concept

A request body is an `io.Reader` and a response is an `io.Writer`. As long as you keep
them that way, memory use doesn't depend on the size of what passes through. Buffering
creeps in through convenient calls: `io.ReadAll(r.Body)`, `r.FormFile`, a JSON encoder
fed one giant slice.

**Uploads.** Cap the body with `http.MaxBytesReader`, then walk the parts with
`r.MultipartReader()` and `NextPart()`, copying each file part into its destination with
`io.Copy`. Exceeding the cap surfaces as `*http.MaxBytesError`: answer 413.

**Downloads.** `io.Copy(w, src)` streams. Without a known length, HTTP/1.1 switches to
chunked encoding. For files and seekable data, `http.ServeContent` adds `Content-Length`,
range requests (resumable downloads) and conditional GETs.

**Server-Sent Events.** A response that never ends: `Content-Type: text/event-stream`,
events made of `id:`, `event:` and `data:` lines, each event ended by a blank line and
followed by a `Flush`. The browser's `EventSource` parses them, and reconnects on its own
with a `Last-Event-ID` header. A line starting with `:` is a comment, useful as a
keep-alive through proxies.

**WebSockets.** A GET with `Upgrade: websocket` and a random `Sec-WebSocket-Key`; the server
answers `101 Switching Protocols` with `Sec-WebSocket-Accept`, and from then on both sides
exchange binary frames on the same TCP connection. `net/http` gets you to the 101 through
`Hijack`. Everything after it is yours or a library's.

## Coming from C/C++/Java/Python/JS

- **Node**: `req` and `res` are streams there too, and `busboy` or `multer` stream
  multipart. `res.write` + `res.flushHeaders` is the SSE counterpart. `ws` and Socket.IO
  are separate packages, as in Go.
- **Java**: Servlet 3.0's `request.getParts()` buffers to disk like `ParseMultipartForm`;
  Spring's `SseEmitter` wraps the SSE format. Java EE/Jakarta has a standard WebSocket API
  (JSR 356), which Go's standard library has no equivalent of.
- **Python**: Starlette's `StreamingResponse` and `request.stream()` play the same roles;
  WebSockets need an ASGI server, never WSGI.
- **C/C++**: think `sendfile` and a write loop; `io.Copy` uses such fast paths when both
  sides allow it.

## How it works

What a WebSocket library adds after the handshake, all of it from RFC 6455:

- **Masking**: client-to-server frames are XOR-masked with a fresh 4-byte key, and a server
  must close the connection on an unmasked client frame. Server frames are unmasked.
- **Lengths**: 0-125 fit in the second byte; 126 means a 16-bit length follows, 127 a
  64-bit one.
- **Fragmentation**: a message may span frames (opcode 0, continuation), the last with FIN.
- **Control frames**: ping (0x9), pong (0xA) and close (0x8) with a status code and the
  close handshake; pings keep idle connections alive through NATs and proxies.
- **Checks and limits**: UTF-8 validation of text messages, maximum message sizes, one
  writer at a time per connection, optional `permessage-deflate` compression, Origin
  checks and subprotocol negotiation.

`golang.org/x/net/websocket` exists outside the standard library, and its own documentation
points to more actively maintained packages, github.com/gorilla/websocket and
github.com/coder/websocket.

## Design notes

Choose the simplest transport that fits. Polling suits data that changes every few
minutes. SSE suits server-to-client updates: it is plain HTTP, so cookies, CORS,
compression and HTTP/2 multiplexing work unchanged, and the browser reconnects for you.
Choose WebSockets when the client sends a steady stream too, and accept the cost: a
library, Origin checks, your own heartbeat, and connections that `Shutdown` doesn't see.

Long-lived responses need the server configured for them: a write deadline extended for
that route (or none), no `TimeoutHandler`, no buffering middleware, and a context that
`RegisterOnShutdown` cancels, as in the previous lesson. For uploads, decide the limit per
endpoint, stream into the destination, and record the client's file name only as metadata.
