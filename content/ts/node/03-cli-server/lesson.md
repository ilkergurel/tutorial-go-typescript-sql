---
id: ts.node.cli-server
title: "A small CLI and a tiny HTTP server in Node, next to the Go version"
track: ts
module: node
order: 3
level: advanced
minutes: 15
since: es2024
prerequisites: [ts.node.fs-streams, ts.async.event-loop]
compare_with: [go.cli.flag-env, go.http.handlers, go.http.server-lifecycle, go.concurrency.goroutines-channels]
tags: [node, cli, parseArgs, http, event-loop, worker-threads]
summary: >
  The same two programs as in the Go track: a CLI built on util.parseArgs and
  stdin, and an HTTP server built on node:http and fetch. The code looks alike;
  the runtime differs. Every Node request shares one thread, so CPU work in a
  handler stalls them all, and a throw in an async handler can kill the process.
examples:
  - id: cli
    file: examples/01_cli.ts
    title: "parseArgs, stdin, and an exit status from a testable main"
    runtime: ts-node
    stdin: "2026-11-01T09:00:00Z INFO server started port=8080\n2026-11-01T09:00:05Z WARN slow query took=1200ms\n2026-11-01T09:01:10Z ERROR db connection lost\n2026-11-01T09:01:12Z INFO reconnected attempts=2\n2026-11-01T09:02:31Z ERROR payment failed order=111042\n"
    check: exact
    annotations:
      - lines: "9-10"
        note: >
          The Go idiom carried over: `func run(args []string, stdin io.Reader,
          stdout, stderr io.Writer) int`. Arguments and streams come in, an
          exit status goes out, and nothing touches process globals, so a test
          calls main directly instead of spawning a process.
      - lines: "13-19"
        note: >
          parseArgs (node:util, stable since Node 20) parses an explicit array,
          like a Go FlagSet. strict is the default, so unknown options and
          missing values throw. Unlike Go's flag package, options may follow
          positionals, -min means the three short flags -m -i -n, and
          --json=false is an error (allowNegative enables --no-json instead).
      - lines: "20-24"
        note: >
          Exit status 2 for a usage error, the convention Go's flag package
          follows too. The message goes to stderr; this demo prefixes it with
          (stderr) because only stdout is shown.
      - lines: "25"
        note: >
          toUpperCase() is locale-independent, which is what identifiers and
          protocol words need. toLocaleUpperCase() with no argument uses the
          host's default locale, and under a Turkish locale "info" becomes
          "İNFO" (what toLocaleUpperCase("tr") returns), which matches no
          level. Go's strings.ToUpper is locale-independent as well.
      - lines: "31-37"
        note: >
          The readLines() loop from the previous lesson, over stdin. It reads
          one line at a time, so `levels < huge.log` runs in constant memory,
          like a bufio.Scanner over os.Stdin.
      - lines: "43-46"
        note: >
          Set process.exitCode and let the process end; don't call
          process.exit(), which terminates at once, even while writes to a
          piped stdout are still pending. Go has the mirror-image problem:
          os.Exit skips deferred calls. In both languages, return a status
          and exit in one place.
  - id: http-server
    file: examples/02_http_server.ts
    title: "node:http: one listener, raw URLs, and errors you must catch"
    runtime: ts-node
    check: exact
    timeout_s: 10
    annotations:
      - lines: "8-9"
        note: >
          createServer takes one listener for every request: there is no
          ServeMux, no patterns and no 405. Go 1.22's mux matches method and
          wildcard patterns and picks the most specific; in Node that is your
          code or a framework. Each request is a callback on the one
          JavaScript thread, not a goroutine of its own.
      - lines: "11-14"
        note: >
          req.url is the raw request target, path plus query and still
          percent-encoded: /hello/Ay%C5%9Fe. Parse it with new URL against a
          fixed base (not the client's Host header) and decode segments
          yourself; Go's PathValue hands you "Ayşe" already decoded. No
          Content-Type was set, so Node sends none, where Go would sniff
          "text/plain; charset=utf-8".
      - lines: "15-18"
        note: >
          json() from node:stream/consumers reads the whole body stream and
          parses it, with no size limit. A real server caps the body (Go:
          http.MaxBytesReader) and validates its shape (lesson
          ts.services.zod) instead of trusting the cast.
      - lines: "22-26"
        note: >
          This catch is load-bearing. http.Server ignores the promise an
          async listener returns, so a throw becomes an unhandled rejection,
          and Node's default is to terminate the process, together with every
          other request in flight. Go's server recovers a panicking handler,
          logs it, and closes only that connection. Errors map to a status in
          one place: SyntaxError (bad JSON) and URIError (bad escape) give
          400, anything else 500.
      - lines: "30-32"
        note: >
          Port 0 on 127.0.0.1 lets the OS pick a free port, the job
          httptest.NewServer does in Go. listen() is asynchronous, so wait
          for the "listening" event before asking for the address.
      - lines: "39-45"
        note: >
          fetch is built into Node 18+. /hello/%ZZ makes decodeURIComponent
          throw: without the catch above, that single request would have
          taken the whole server down.
      - lines: "46"
        note: >
          close() stops accepting connections and, since Node 19, also closes
          idle keep-alive connections, so the process can exit. It does not
          cut requests that are still running; Go's Server.Shutdown(ctx) is
          the counterpart.
  - id: event-loop
    file: examples/03_event_loop.ts
    title: "One thread: what a busy handler does to every other request"
    runtime: ts-node
    check: exact
    timeout_s: 10
    annotations:
      - lines: "7-11"
        note: >
          spin is 300 ms of pure computation, a stand-in for parsing a large
          report, hashing or a heavy regex. The worker gets the same loop as
          source text because eval keeps this example in one file; a real
          project gives the worker its own .ts file,
          new Worker(new URL("./report.ts", import.meta.url)).
      - lines: "14-19"
        note: >
          The output, line by line. Spinning on the event loop: /health
          cannot even start until /report ends, because timers, sockets and
          every other request share the one JavaScript thread. Awaiting I/O:
          the handler yields at await, and /health completes in the
          meantime. A worker thread: the same CPU work runs in parallel on
          another thread with its own event loop, and the main thread stays
          free. In Go all three would interleave: every request has its own
          goroutine, goroutines run on several OS threads, and since Go 1.14
          even a tight loop is preempted.
      - lines: "21-31"
        note: >
          A fresh server per strategy logs when each request starts and ends.
          /report signals as soon as its handler is running, then does its
          300 ms of work.
      - lines: "37-40"
        note: >
          The client runs in the same process, so it can only send /health
          once the server has handed control back to the event loop. A real
          client in another process would send /health right away; the
          request would then sit unread in the socket buffer, with the same
          result.
use_cases:
  - "Small CLIs and dev tools in the same language as the frontend: code generators, data fixes, release scripts"
  - "I/O-bound services (API gateways, BFFs, webhook receivers, proxies) where handlers mostly await databases and other services"
  - "Moving CPU-heavy work (image resizing, report generation, large JSON parsing) off the request thread with worker_threads"
  - "Comparing designs: the Go version of the same server is the default choice for CPU-heavy or highly concurrent services"
strengths:
  - "Built-in parts are enough for small tools: parseArgs, readline, node:http and fetch need no dependencies, much like Go's flag and net/http"
  - "One thread means no data races: handler code needs no mutex around shared state (though it can change across an await), unlike Go handlers running concurrently on goroutines"
  - "Awaiting I/O is cheap: thousands of slow connections cost callbacks, not threads"
  - "Sane server timeouts by default (requestTimeout 300 s and headersTimeout 60 s since Node 18), where Go's zero http.Server has none"
weaknesses:
  - "CPU work in any handler stalls every request; Go spreads handlers across all cores and preempts long-running goroutines"
  - "A throw that escapes an async handler terminates the process; Go's net/http recovers a panicking handler per connection"
  - "node:http has no routing, path parameters or 405 handling; real services add a framework, where Go 1.22's ServeMux covers most needs"
  - "No Content-Type sniffing, no decoded path values and no body size limit: more of HTTP is left to your code"
  - "A Go CLI ships as one static binary; a Node CLI needs Node installed (or a bundled single-executable application)"
pitfalls:
  - title: "An async request handler without try/catch"
    detail: "http.Server ignores the returned promise, so a rejected one is an unhandled rejection, which by default terminates the process. Wrap the handler body once, or use a framework that catches async errors and answers 500."
    bad: "createServer(async (req, res) => {\n  const body = await json(req);\n  res.end(JSON.stringify(body));\n});"
    good: "createServer(async (req, res) => {\n  try {\n    const body = await json(req);\n    res.end(JSON.stringify(body));\n  } catch {\n    res.writeHead(400).end();\n  }\n});"
  - title: "CPU-heavy work inside a request handler"
    detail: "A synchronous loop, JSON.parse of a huge body, crypto.pbkdf2Sync or a catastrophic regex blocks every request and timer. Move the work to worker_threads (or a job queue), or split it into chunks that await between batches."
  - title: "process.exit() at the end of a CLI"
    detail: "It exits immediately, even while writes to stdout are still pending (Node's docs: pipes are asynchronous on Linux and macOS, terminals on Windows), so `mycli | grep x` can lose its last lines. Set process.exitCode and let the event loop drain."
    bad: "console.log(report);\nprocess.exit(failed ? 1 : 0);"
    good: "console.log(report);\nprocess.exitCode = failed ? 1 : 0;"
  - title: "Using req.url as a path"
    detail: "It is the percent-encoded request target including the query string. Compare and route on new URL(req.url, base).pathname, and decode each segment with decodeURIComponent inside a try, because a bad escape throws URIError."
  - title: "toLocaleUpperCase() with no locale on flags and identifiers"
    detail: "It follows the host's locale, so on a Turkish system \"info\" becomes \"İNFO\". Use toUpperCase() for identifiers and toLocaleUpperCase(\"tr\") for Turkish text shown to people."
quiz:
  - q: "A Node handler runs a 2-second synchronous computation. What happens to a /health request that arrives during it?"
    options:
      - "It is served immediately on another thread"
      - "It waits until the computation finishes, like every other event"
      - "Node preempts the computation after 10 ms"
    answer: 1
    why: "All JavaScript in the process runs on one thread. Nothing else runs until the handler returns or awaits. Go would serve /health on another goroutine."
  - q: "An async request listener throws and nothing catches it. What happens by default?"
    options:
      - "Node answers 500 for that request"
      - "The connection is closed and the server keeps running"
      - "The rejection is unhandled and the process terminates"
    answer: 2
    why: "http.Server doesn't look at the listener's returned promise. Since Node 15 an unhandled rejection is raised as an uncaught exception, which ends the process."
  - q: "GET /hello/Ay%C5%9Fe reaches a node:http server. What is req.url?"
    options:
      - "/hello/Ayşe"
      - "/hello/Ay%C5%9Fe"
      - "http://127.0.0.1/hello/Ayşe"
    answer: 1
    why: "req.url is the raw request target from the request line: path and query, still percent-encoded. Decoding is your job."
  - q: "Why should a CLI set process.exitCode rather than call process.exit()?"
    options:
      - "process.exit() is deprecated"
      - "process.exit() ends the process at once, possibly before pending stdout writes reach a pipe"
      - "exitCode is the only way to return a non-zero status"
    answer: 1
    why: "process.exit() terminates even while asynchronous writes are pending. With exitCode set, the process exits by itself once the event loop is empty, after the output is written."
further_reading:
  - title: "Node.js: util.parseArgs()"
    url: "https://nodejs.org/api/util.html#utilparseargsconfig"
  - title: "Node.js: HTTP (createServer, request.url, server.close, timeouts)"
    url: "https://nodejs.org/api/http.html"
  - title: "Node.js: Worker threads"
    url: "https://nodejs.org/api/worker_threads.html"
  - title: "Node.js: process.exitCode and process.exit()"
    url: "https://nodejs.org/api/process.html#processexitcode"
  - title: "Node.js: Don't block the event loop (or the worker pool)"
    url: "https://nodejs.org/en/learn/asynchronous-work/dont-block-the-event-loop"
  - title: "package net/http (Go)"
    url: "https://pkg.go.dev/net/http"
---

## Concept

Both programs from the Go track, rewritten for Node. The code reads much the same;
what sits underneath differs.

- **CLI**: `util.parseArgs` parses an explicit argument array (Go: a `FlagSet`),
  `readline` over `process.stdin` reads lines (Go: `bufio.Scanner` over `os.Stdin`),
  and the exit status is set on `process.exitCode`. Put the logic in a function that
  takes argv and streams, exactly as you would write `run(args, stdin, stdout)` in Go.
- **Server**: `http.createServer` takes one listener for every request. You route on
  `new URL(req.url, base).pathname`, decode path segments, set Content-Type and catch
  errors yourself; Go's `ServeMux` and `net/http` do more of that for you.

The **runtime model** is the real difference. A Go server gives every request its own
goroutine; goroutines run in parallel on all cores and are preempted. A Node server has
one JavaScript thread and an event loop: a handler runs until it returns or reaches an
`await`, and while it computes, nothing else in the process runs. Waiting is cheap and
computing is shared. CPU work goes to `worker_threads`.

## Coming from C/C++/Java/Python/JS

- Go: `flag` ↔ `parseArgs`; `os.Exit(2)` ↔ `process.exitCode = 2`;
  `http.HandlerFunc` ↔ the request listener; `r.PathValue` ↔ your own regex plus
  `decodeURIComponent`; `httptest.NewServer` ↔ `listen(0, "127.0.0.1")`;
  `Server.Shutdown` ↔ `server.close()`. Go recovers handler panics, while an async Node
  handler that throws takes the process down.
- Java: a servlet container runs requests on a thread pool, closer to Go. Node is like
  a single-threaded Netty event loop where your handler code runs on the loop itself.
- Python: asyncio with aiohttp is the same model as Node: one thread, cooperative
  `await`, and CPU work in a process or thread pool.
- C: a `select`/`epoll` loop with callbacks, minus the manual buffer management.
- JS: the browser's event loop, with sockets in place of DOM events. The rule you know
  from the UI (don't block the main thread) applies to every client of the server.

## How it works

```mermaid
sequenceDiagram
  participant C as client
  participant L as event loop (one JS thread)
  participant W as worker thread
  C->>L: GET /report
  L->>W: start the CPU work
  C->>L: GET /health
  L-->>C: 200 /health (the loop is free)
  W-->>L: work done
  L-->>C: 200 /report
```

The event loop runs one callback at a time: a request arriving, a timer firing, a
promise continuation. `await` ends the current callback and schedules the rest for
later, which is why the timer strategy in example 3 lets /health through. A worker
thread has its own V8 isolate and event loop. It shares no ordinary objects with
the main thread (only SharedArrayBuffer memory) and talks to it by messages (`postMessage`, or the
`exit` event used here), much like goroutines that share nothing and communicate
over a channel.

## Design notes

For a service that mostly waits on databases and other services, Node's model is
productive: one thread means no locks in handler code. Keep two rules. First, every
async handler needs a try/catch, or a framework that provides one, because an escaped
rejection ends the process. Second, CPU work above a few milliseconds per request
belongs in a worker or a job queue. If a service is CPU-bound, or needs many cores
without that bookkeeping, it is a good fit for Go, like this repo's own backend. For CLIs the choice comes down to distribution: Go ships one static
binary, while Node needs a runtime on the target machine but shares code and types with
the frontend. In both languages, write `main` as a function of argv and streams that
returns a status; that keeps the CLI testable.
