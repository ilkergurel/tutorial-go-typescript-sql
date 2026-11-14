---
id: go.http.server-lifecycle
title: "Server timeouts, graceful shutdown, static files with embed"
track: go
module: http
order: 4
level: intermediate
minutes: 14
since: go1.26
prerequisites: [go.http.middleware, go.concurrency.context, go.files-io.embed]
compare_with: [ts.node.cli-server, inside.codebase.wiring, go.services.client]
tags: [http, server, timeouts, shutdown, embed, static-files]
summary: >
  The zero http.Server has no timeouts, so one slow client can hold a connection
  forever: set them. Shut down with Server.Shutdown, which drains requests but
  cancels none, and ship static assets inside the binary with embed and FileServerFS.
examples:
  - id: timeouts
    file: examples/01_timeouts.go
    title: "What ReadHeaderTimeout and WriteTimeout do, and what the client sees"
    runtime: go-server
    check: exact
    timeout_s: 10
    annotations:
      - lines: "13-29"
        note: >
          The slowloris attack in miniature: valid but unfinished headers,
          sent by a client that then just waits. Each such connection pins a
          goroutine and a file descriptor; a few thousand of them exhaust a
          server without sending much traffic.
      - lines: "25"
        note: >
          errors.AsType (Go 1.26) finds the net.Error and returns it typed.
          A read deadline that passes is a timeout; anything else here means
          the server hung up.
      - lines: "39"
        note: >
          httptest.NewServer, like http.ListenAndServe, uses a zero
          http.Server: every timeout is zero, which means none. That is the
          dangerous default this lesson is about.
      - lines: "41-44"
        note: >
          NewUnstartedServer exposes the *http.Server as Config, so a test can
          set fields before Start. ReadHeaderTimeout closes the connection
          without any response, not even a 408.
      - lines: "50-57"
        note: >
          WriteTimeout is not a request timeout with a polite answer: when it
          passes, the client gets a closed connection and no status at all,
          while the handler's write succeeded into a buffer and it never
          learned. Use it as a backstop, and per-request deadlines (previous
          lesson) for answers.
  - id: graceful-shutdown
    file: examples/02_graceful_shutdown.go
    title: "Shutdown drains in-flight requests; long-lived ones need a signal"
    runtime: go-server
    check: exact
    timeout_s: 10
    annotations:
      - lines: "13-17"
        note: >
          Listening on 127.0.0.1:0 lets the OS choose a free port; the
          listener knows which. Binding to 127.0.0.1 rather than ":8080"
          keeps a development server off the network.
      - lines: "19-21"
        note: >
          Shutdown does not cancel request contexts and does not wait for
          hijacked connections such as WebSockets. A context canceled from
          RegisterOnShutdown gives streams and long polls their stop signal.
      - lines: "34-39"
        note: >
          BaseContext is the root of every request's context on this server,
          so canceling it reaches every handler at once. RegisterOnShutdown
          functions run in their own goroutines when Shutdown starts; they
          should start the goodbye, not wait for it.
      - lines: "61-64"
        note: >
          Always give Shutdown a deadline. Without one, a single stuck request
          blocks the process forever; with one, Shutdown returns the context's
          error and you can call srv.Close() to cut the remaining connections.
      - lines: "66-68"
        note: >
          Serve returns http.ErrServerClosed immediately, while two requests
          are still running. A main that returns at that moment kills them
          mid-response: main must wait for Shutdown, not for Serve.
      - lines: "69-72"
        note: >
          The stream ended because RegisterOnShutdown canceled its context. The
          report finished its work and got its 200: drained, not dropped. Only
          then did Shutdown return nil.
  - id: embed-static
    file: examples/03_embed_static.go
    title: "Serving embedded files with fs.Sub and http.FileServerFS"
    runtime: go-server
    check: exact
    timeout_s: 10
    fixtures: [fixtures/static/index.html, fixtures/static/app.css, fixtures/static/img/logo.svg, fixtures/static/_draft.html]
    annotations:
      - lines: "13-17"
        note: >
          //go:embed is read at compile time: the files must sit next to the
          source, and the build fails if they are missing. Embedding a
          directory skips names starting with "." or "_" (_draft.html below);
          the "all:static" pattern would include them.
      - lines: "20-25"
        note: >
          Forgetting fs.Sub is the classic bug: every URL then needs a
          /static/ segment, or nothing matches. StackLab's backend hit the same
          trap with its embedded migrations; main.go's migrationsFS exists for
          it.
      - lines: "27"
        note: >
          FileServerFS (Go 1.22) takes any fs.FS: embed.FS, os.DirFS, or a
          testing/fstest.MapFS in tests. StripPrefix removes /assets/ so the
          remaining path matches names inside the FS.
      - lines: "41-49"
        note: >
          Content-Type comes from the extension. Last-Modified is empty because
          embedded files have a zero ModTime, so browsers can't revalidate
          with If-Modified-Since. index.html redirects to its directory, and a
          directory without an index gets a listing.
use_cases:
  - "Any production http.Server: ReadHeaderTimeout always, the others according to the slowest legitimate request"
  - "Rolling deploys and container restarts: SIGTERM, stop accepting, drain, exit before the orchestrator's kill deadline"
  - "Single-binary tools and admin UIs that carry their HTML, CSS and JavaScript (or a built React app) inside the executable"
  - "Embedding SQL migrations or templates so a distroless image needs nothing but the binary"
strengths:
  - "Shutdown is built in and correct by default (stop accepting, close idle, wait for active), where many stacks need a library or careful signal handling"
  - "Per-route exceptions without a second server: http.ResponseController.SetWriteDeadline extends one handler's deadline"
  - "embed.FS is an fs.FS, so the same code serves embedded files in production and os.DirFS or fstest.MapFS in development and tests"
  - "One static binary with its assets: no working-directory surprises, unlike Java resources resolved at run time or Node's static folders"
weaknesses:
  - "Safe timeouts are opt-in: http.ListenAndServe and the zero http.Server have none, unlike Node's 60 s headersTimeout and 300 s requestTimeout (since Node 18)"
  - "Server-wide ReadTimeout and WriteTimeout can't tell a slow upload or a stream from an attack; long routes need per-request handling"
  - "Shutdown neither cancels handlers nor tracks WebSockets; long-lived work needs BaseContext or RegisterOnShutdown wiring"
  - "Embedded files have no modification time, so FileServerFS sends no Last-Modified; caching needs ETags or fingerprinted file names"
pitfalls:
  - title: "http.ListenAndServe in production"
    detail: "It uses a zero http.Server, so there is no way to set timeouts. Build an http.Server and set at least ReadHeaderTimeout."
    bad: "log.Fatal(http.ListenAndServe(\":8080\", mux))"
    good: "srv := &http.Server{Addr: \"127.0.0.1:8080\", Handler: mux,\n\tReadHeaderTimeout: 5 * time.Second, IdleTimeout: time.Minute}"
  - title: "Exiting when ListenAndServe returns"
    detail: "After Shutdown starts, ListenAndServe returns http.ErrServerClosed at once. If main returns there, in-flight requests die. Wait for Shutdown to return."
  - title: "Treating ErrServerClosed as a failure"
    detail: "It is the expected result after Shutdown or Close. Check errors.Is(err, http.ErrServerClosed) before logging an error or exiting non-zero."
  - title: "A global WriteTimeout shorter than your slowest route"
    detail: "The client of a long export or SSE stream gets a dropped connection, and the handler's writes may look successful. Extend the deadline for that route with http.NewResponseController(w).SetWriteDeadline, which needs Unwrap on any wrapping middleware."
  - title: "Expecting dotfiles and _files in an embedded directory"
    detail: "//go:embed static skips names starting with . or _. Frameworks that emit _app.js or similar need //go:embed all:static."
quiz:
  - q: "What are the timeouts of a server started with http.ListenAndServe(addr, mux)?"
    options: ["30 s for everything", "ReadHeaderTimeout 5 s, the rest unlimited", "None: every timeout is zero, meaning no deadline", "They depend on the OS"]
    answer: 2
    why: "ListenAndServe builds a zero http.Server, and a zero timeout means no timeout. Set them on your own http.Server."
  - q: "A handler takes longer than the server's WriteTimeout. What does the client see?"
    options: ["503 Service Unavailable", "504 Gateway Timeout", "The connection closes without a response", "The partial body with status 200"]
    answer: 2
    why: "The write deadline expires on the connection; the buffered response can't be sent and the connection is closed. No status line reaches the client."
  - q: "Shutdown(ctx) is called while an SSE handler waits on r.Context().Done(). Without further wiring, what happens?"
    options: ["The handler's context is canceled at once", "Shutdown waits for the handler until ctx expires", "The handler is killed", "Shutdown returns nil immediately"]
    answer: 1
    why: "Shutdown waits for active connections but cancels nothing. Cancel a BaseContext from RegisterOnShutdown to end such handlers."
  - q: "Why does FileServerFS over an embed.FS send no Last-Modified header?"
    options: ["embed strips HTTP headers", "Embedded files have a zero modification time, and ServeContent omits a zero time", "FileServerFS never sends it", "Only os.DirFS supports headers"]
    answer: 1
    why: "embed.FS reports a zero ModTime; ServeContent only sets Last-Modified for a non-zero time. Use an ETag or fingerprinted names for caching."
further_reading:
  - title: "package net/http: Server (timeouts, Shutdown, RegisterOnShutdown)"
    url: "https://pkg.go.dev/net/http#Server"
  - title: "package net/http: ResponseController"
    url: "https://pkg.go.dev/net/http#ResponseController"
  - title: "package embed"
    url: "https://pkg.go.dev/embed"
  - title: "package net/http: FileServerFS and ServeContent"
    url: "https://pkg.go.dev/net/http#FileServerFS"
  - title: "package os/signal: NotifyContext"
    url: "https://pkg.go.dev/os/signal#NotifyContext"
---

## Concept

An `http.Server` is configuration plus a loop that accepts connections. Its zero value,
which `http.ListenAndServe` uses, has **no timeouts**: a client that sends headers slowly,
never finishes a body, or never reads the response holds a goroutine and a socket for as
long as it likes. Four fields bound the phases of a connection:

| Field | Bounds | Notes |
|---|---|---|
| `ReadHeaderTimeout` | reading the request line and headers | set it always; the slowloris defense |
| `ReadTimeout` | reading the whole request, body included | blunt for uploads; often left to per-request limits |
| `WriteTimeout` | from the end of the headers to the end of the response | effectively caps handler time; the client gets no response when it fires |
| `IdleTimeout` | waiting for the next request on a keep-alive connection | falls back to `ReadTimeout` when zero |

Header size has a limit by default: `MaxHeaderBytes` defaults to 1 MB, and Go 1.27 added
`MaxHeaderValueCount` (default 500 values). One route can relax the server-wide deadlines
with `http.NewResponseController(w).SetWriteDeadline(...)` (Go 1.20).

**Graceful shutdown** is `srv.Shutdown(ctx)`: close the listeners, close idle connections,
wait for active requests to finish. `Serve` returns `http.ErrServerClosed` as soon as it
starts, so `main` must wait for `Shutdown` to return. Shutdown never cancels a request's
context and ignores hijacked connections, so streams and WebSockets need their own signal.

**Static files** can live in the binary: `//go:embed static` fills an `embed.FS` at compile
time, `fs.Sub` drops the directory prefix, and `http.FileServerFS` (Go 1.22) serves it with
content types, range requests and conditional GETs.

## Coming from C/C++/Java/Python/JS

- **Node**: `http.createServer` has defaults: `headersTimeout` 60 s and, since Node 18,
  `requestTimeout` 300 s. Go starts at zero, and you choose. Draining on SIGTERM is your
  code in both: `process.on('SIGTERM', ...)` there, `signal.NotifyContext` here.
- **Java**: a servlet container ships preconfigured connection timeouts and its own
  graceful-shutdown setting. In Go the server is a struct in your `main`, and so is its
  lifecycle.
- **Python**: gunicorn or uvicorn own the timeouts and the SIGTERM handling outside your
  code. In Go there is no separate application server: your binary is it.
- **C/C++**: `go:embed` is what `xxd -i` or C23's `#embed` do for a byte array, but it
  produces a read-only file system with directories, not one blob.

## Design notes

StackLab's `cmd/stacklab/main.go` is a complete lifecycle to read: `ReadHeaderTimeout` 5 s,
`ReadTimeout` and `WriteTimeout` from configuration (15 s), `IdleTimeout` 60 s;
`signal.NotifyContext` for Ctrl+C and SIGTERM; `ListenAndServe` in a goroutine, ignoring
`ErrServerClosed`; and `Shutdown` with a 10 s budget. The run route is the one exception
to the 15 s write deadline: it extends its own deadline with `ResponseController`, because
a lesson run may take longer.

Pick the drain budget from the slowest request you are willing to wait for, and keep it
below the time your process manager waits before killing the process. If `Shutdown` returns
`context.DeadlineExceeded`, call `srv.Close()`, log, and exit. For a service behind a load
balancer, fail the readiness check first (StackLab has `/readyz`), so traffic stops arriving
before the listener closes.

For embedded assets, decide on caching explicitly. With content-hashed file names, as a
Vite build emits, serve them with `Cache-Control: public, max-age=31536000, immutable`. For
fixed names, compute a hash per file at startup and set it as the `ETag` header before
calling the file server; `ServeContent` then answers `If-None-Match` with 304. A single-page
app also needs a fallback that serves `index.html` for unknown paths outside `/api/` and
`/assets/`.
