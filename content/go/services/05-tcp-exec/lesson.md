---
id: go.services.tcp-exec
title: "TCP with net, os/exec without a shell, ReverseProxy"
track: go
module: services
order: 5
level: advanced
minutes: 15
since: go1.26
prerequisites: [go.services.client, go.files-io.composing]
compare_with: [ts.node.cli-server, go.files-io.binary, design.architecture.security]
tags: [net, tcp, os-exec, reverse-proxy, framing, security]
summary: >
  Below HTTP, a TCP connection is a byte stream: you choose the framing and set
  deadlines. os/exec runs programs directly with an argument list, never a shell.
  httputil.ReverseProxy with Rewrite forwards HTTP with headers you control.
examples:
  - id: tcp
    file: examples/01_tcp.go
    title: "A line protocol over TCP: framing, half-close, deadlines"
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "19-34"
        note: >
          The classic Go server: an accept loop and a goroutine per
          connection. Blocking reads are fine, because goroutines are cheap and
          the runtime's network poller parks them; there is no callback or
          async API to learn.
      - lines: "27"
        note: >
          A deadline is an absolute time and covers all I/O on the connection
          from now on. Without one, a client that connects and goes quiet holds
          this goroutine and its file descriptor forever.
      - lines: "37"
        note: >
          127.0.0.1 keeps the listener off the network; port 0 asks the OS for
          any free port, which ln.Addr() then reports. Tests use the same
          trick, so they never collide on a fixed port.
      - lines: "44-45"
        note: >
          The context bounds only the dial. Once connected, deadlines are the
          tool; context.AfterFunc(ctx, func() { conn.SetDeadline(time.Now()) })
          (Go 1.21) ties a connection's I/O to a context when you need both.
      - lines: "51-53"
        note: >
          One Write carries three messages, and the server's Scanner still sees
          three lines: TCP delivers bytes, not messages. The newline is the
          framing; a length prefix (encoding/binary) is the other common
          choice. CloseWrite sends FIN, so the server's Scan ends cleanly
          while we can still read the replies.
      - lines: "55-57"
        note: >
          Upper-casing changes byte lengths: Turkish i becomes İ (two bytes),
          ı becomes I (one byte). A protocol that sends lengths must compute
          them after encoding. strings.ToUpper would have produced ISTANBUL.
      - lines: "67-69"
        note: >
          A deadline turns "wait forever" into an error that wraps
          os.ErrDeadlineExceeded. The connection stays usable: set a new
          deadline and read again.
  - id: exec
    file: examples/02_exec.go
    title: "os/exec: an argument list, exit codes, stderr, and a kill on timeout"
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "14-26"
        note: >
          The program re-runs its own binary as the child, so the example
          needs nothing but itself. Go's own os/exec tests use the same
          trick.
      - lines: "29-32"
        note: >
          The parent starts without arguments and the child with a role, so
          one main serves both.
      - lines: "36-37"
        note: >
          "go" has no path separator, so Command looks it up on PATH. Since Go
          1.19 that lookup never resolves to a program in the current
          directory (exec.ErrDot), which closes a classic Windows hijack.
          Tools such as gopls drive go list exactly like this.
      - lines: "44-45"
        note: >
          Each string arrives as one argument, exactly as written: no word
          splitting, no globbing, no $(...) substitution, because no shell is
          involved. Passing user input this way is safe; building a string for
          sh -c is how command injection happens.
      - lines: "47-50"
        note: >
          A non-zero exit is an *exec.ExitError. Output captures stderr into
          its Stderr field when you didn't set cmd.Stderr yourself, so the
          error message of a failing tool is not lost.
      - lines: "52-57"
        note: >
          CommandContext kills the child (os.Process.Kill) when the context
          ends. WaitDelay (Go 1.20) bounds how long Wait then waits for the
          child's pipes, which a grandchild could otherwise hold open forever.
          Only booleans are printed: the error text differs between Windows
          and Linux.
  - id: reverse-proxy
    file: examples/03_reverse_proxy.go
    title: "httputil.ReverseProxy with Rewrite, ModifyResponse and ErrorHandler"
    runtime: go
    check: exact
    timeout_s: 10
    annotations:
      - lines: "26-30"
        note: >
          Rewrite (Go 1.20) receives the inbound request (pr.In) and edits the
          outbound one (pr.Out). SetURL joins paths and sets Host to the
          backend's; SetXForwarded adds the X-Forwarded-* headers from the
          real connection.
      - lines: "29"
        note: >
          Headers set in Rewrite survive. Hop-by-hop headers, including any the
          client names in Connection, are removed before Rewrite runs. With
          the deprecated Director they were removed after it, so line 46 could
          strip a header the proxy added.
      - lines: "31-34"
        note: >
          ModifyResponse sees every backend response, whatever its status, and
          can change headers or the body, or return an error to turn it into a
          proxy error.
      - lines: "35-37"
        note: >
          Without an ErrorHandler the proxy logs the error and answers 502. A
          custom one decides what the client sees; the dial error names
          internal hosts and ports, so log it instead of sending it.
      - lines: "45-46"
        note: >
          The client tries two tricks. The spoofed X-Forwarded-For is gone:
          Rewrite starts from a request without X-Forwarded-* headers, and
          SetXForwarded writes the connection's real address. X-Gateway
          arrives intact.
use_cases:
  - "TCP: device and legacy protocols, a health probe for a database port, a tiny line protocol between your own processes"
  - "os/exec: driving git, ffmpeg, pg_dump or the go command from a service or a build tool"
  - "ReverseProxy: an API gateway or BFF in front of internal services, a dev server that proxies /api to a backend, gradual migrations where some paths go to a new service"
strengths:
  - "net.Conn is an io.Reader and io.Writer, so bufio, encoding/binary and every reader wrapper from the files-io module work on sockets unchanged"
  - "Blocking I/O on goroutines scales to many thousands of connections without callbacks, unlike thread-per-connection Java or a Node event loop"
  - "os/exec never involves a shell unless you start one, so arguments can't be reinterpreted"
  - "ReverseProxy is production-grade in the standard library: streaming bodies, trailers, WebSocket upgrades, hop-by-hop handling"
weaknesses:
  - "Raw TCP leaves framing, timeouts, retries and versioning to you; that is most of what HTTP and gRPC provide"
  - "Deadlines are per connection and absolute, not per call, and they don't come from a context automatically"
  - "Child processes are OS-specific in the details: signals, exit codes, Windows command-line quoting and batch files"
  - "ReverseProxy needs explicit configuration for timeouts (its Transport) and for which headers are trusted"
pitfalls:
  - title: "Running user input through a shell"
    detail: "exec.Command(\"sh\", \"-c\", ...) or cmd.exe /C with concatenated input turns data into code. Pass arguments separately. If the tool itself interprets arguments starting with a dash, end options with \"--\" before user input."
    bad: |
      exec.Command("sh", "-c", "convert "+name+" out.png")
    good: |
      exec.Command("convert", "--", name, "out.png")
  - title: "Batch files on Windows"
    detail: "Command quotes arguments for programs that parse their command line like CommandLineToArgvW. cmd.exe, and therefore every .bat and .cmd file, unquotes differently, so user input passed to a batch file can still break out. Don't pass untrusted arguments to batch files."
  - title: "Assuming one Read is one message"
    detail: "TCP may split one Write across several Reads or merge several Writes into one. Frame messages (newline, length prefix) and read with bufio or io.ReadFull."
  - title: "No deadlines on a server connection"
    detail: "A client that connects and never sends holds a goroutine and a file descriptor. Set a deadline per connection, and refresh it per message for long-lived connections."
  - title: "Trusting X-Forwarded-For from anywhere"
    detail: "Clients can send any header. Only the proxy you operate should set it; backends should accept it only from that proxy. Director-based proxies, including NewSingleHostReverseProxy, keep the client's X-Forwarded-* headers; prefer Rewrite."
quiz:
  - q: "The client writes \"a\\n\" and then \"b\\n\" in two Write calls. How many Read calls does the server need?"
    options: ["Exactly two", "Exactly one", "One or more: TCP has no message boundaries, so frame with the newlines", "It depends on GOMAXPROCS"]
    answer: 2
    why: "TCP is a byte stream. Both writes may arrive in one read, or split anywhere; the newline delimiter is what defines the messages."
  - q: "exec.Command(\"git\", \"log\", userInput) runs with userInput = \"; rm -rf ~\". What happens?"
    options: ["The rm runs", "git receives one argument containing \"; rm -rf ~\" and probably rejects it", "Go escapes the semicolon", "Command returns an error before running"]
    answer: 1
    why: "No shell is involved, so ; and ~ have no special meaning; the string arrives as one argument."
  - q: "Why is ReverseProxy.Director deprecated (Go 1.26) in favour of Rewrite?"
    options: ["It was slower", "Hop-by-hop headers were removed after Director ran, so a client could strip headers the proxy added, and inbound X-Forwarded-* headers were kept", "It couldn't change the URL", "It didn't support HTTP/2"]
    answer: 1
    why: "Both problems are listed in the Director documentation; Rewrite sees the request after hop-by-hop removal and without inbound X-Forwarded-* headers."
further_reading:
  - title: "package net: Listen, Dialer, Conn deadlines"
    url: "https://pkg.go.dev/net"
  - title: "package os/exec (no shell; executables in the current directory)"
    url: "https://pkg.go.dev/os/exec"
  - title: "package net/http/httputil: ReverseProxy, ProxyRequest"
    url: "https://pkg.go.dev/net/http/httputil#ReverseProxy"
  - title: "Go 1.26 release notes: ReverseProxy.Director deprecated"
    url: "https://go.dev/doc/go1.26"
---

## Concept

HTTP covers most service-to-service traffic, but three tools one level down come up in
real backends.

**TCP with `net`.** `net.Listen` and `net.Dial` give you a `net.Conn`, which is an
`io.Reader` and `io.Writer` over a byte stream. Everything HTTP does for you is now
yours: **framing** (where one message ends; a newline or a length prefix), **deadlines**
(`SetDeadline`, absolute times, per connection), and shutdown (`Close`, or `CloseWrite`
to say "no more data" while still reading).

**`os/exec`.** `exec.Command(name, args...)` starts a program directly with an argument
list. There is no shell: no word splitting, no globbing, no `$()` or `;`. That makes it
safe to pass user input as an argument, and it means pipes and redirections are done in
Go (`cmd.Stdin`, `cmd.Stdout`, `StdoutPipe`). `CommandContext` kills the process when a
context ends; a non-zero exit is an `*exec.ExitError`.

**`httputil.ReverseProxy`.** An `http.Handler` that forwards each request to a backend
and streams the response back. Configure it with `Rewrite` (the deprecated `Director` is
unsafe), add `ModifyResponse` to adjust responses and `ErrorHandler` to answer when the
backend is down.

## Coming from C/C++/Java/Python/JS

- **C**: `socket`/`bind`/`listen`/`accept` and `read`/`write` are the same concepts;
  Go hides `select`/`epoll` behind blocking calls on goroutines. `exec.Command` is the
  `fork`/`execvp` family, not `system()`, which goes through `/bin/sh`.
- **Java**: `ServerSocket` with a thread per connection is the same shape, but a
  goroutine costs kilobytes, not a megabyte of stack. `ProcessBuilder(List<String>)` is
  `exec.Command`: an argument list, no shell.
- **Python**: `socketserver`/`asyncio.start_server` for TCP. `subprocess.run([...])` is
  `exec.Command`; `shell=True` is the dangerous variant.
- **Node**: `net.createServer` with `data` events that, like Go's reads, carry arbitrary
  chunks, not messages. `child_process.execFile` (no shell) versus `exec` (shell) is the
  same distinction as here.

## Design notes

Reach for raw TCP when a protocol is given (a device, a legacy system, Redis or
PostgreSQL wire protocols) or when you need something HTTP can't do cheaply. For your own
services, HTTP or gRPC (lesson 7) give you framing, status codes, deadlines and tooling
for free.

Treat external programs like remote services: bound them with a context and
`WaitDelay`, capture stderr for error messages, limit their output, and check
`exec.LookPath` at startup so a missing tool fails the deploy rather than the first
request. Prefer a Go library when one exists; a subprocess per request costs process
start-up time and ties you to what's installed on the host.

`ReverseProxy` is the core of a gateway, but its defaults are generic: give it a
`Transport` with timeouts (lesson 1), decide which inbound headers to trust, and set
`X-Forwarded-*` only at your outermost proxy. Backends then read the client address from
the header only when the request came from that proxy.
