---
id: ts.services.realtime-cors
title: "WebSockets, EventSource (SSE), CORS from both sides, tokens vs cookies (concepts)"
track: ts
module: services
order: 5
level: advanced
minutes: 15
since: es2022
prerequisites: [ts.services.fetch, ts.services.zod]
compare_with: [go.http.cors-cookies-httptest, go.http.uploads-streams, go.services.listen-notify, fullstack.tracelite.step-09-realtime]
tags: [websocket, server-sent-events, eventsource, cors, origin, cookies, tokens, streams]
summary: >
  Server-sent events push text over one HTTP response; WebSockets carry messages
  both ways. CORS doesn't stop requests: the server labels its response and the
  browser decides whether the page's origin, even the opaque "null", may read it.
examples:
  - id: sse
    file: examples/01_sse.ts
    title: "Parsing a text/event-stream from a fetch body"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "10-13"
        note: >
          EventSource can't send an Authorization header or use POST, so
          streaming APIs that need either are read with fetch and parsed by
          hand. TextDecoderStream decodes UTF-8 across chunk boundaries: a
          character split between two chunks is held until its second byte
          arrives.
      - lines: "18-22"
        note: >
          A chunk can end in the middle of a line, so the last piece waits for
          the next chunk. The stream ends without a final blank line in this
          example, and the unfinished event is dropped, as EventSource drops it.
          The format also allows a lone CR as a line end; this sketch doesn't.
      - lines: "24-31"
        note: >
          A line starting with a colon is a comment, which servers send as a
          keep-alive. A blank line dispatches: data lines are joined with \n,
          the last \n is removed, the type defaults to "message", and an event
          without data lines dispatches nothing.
      - lines: "32-38"
        note: >
          Each line is field: value, with one leading space removed. The id
          carries over to later events (the second event shows #41 too), and
          EventSource sends it back as Last-Event-ID when it reconnects, so the
          server can resume. retry sets EventSource's reconnection delay.
      - lines: "52-58"
        note: >
          The wrong way to decode a stream: a fresh TextDecoder per chunk turns
          each half of İ into U+FFFD. Any Turkish text breaks this way sooner or
          later; decode with one streaming decoder.
      - lines: "66-73"
        note: >
          In a page, the body comes from fetch, with whatever method and headers
          the API needs. for await consumes the events as they arrive; break
          stops reading, and an AbortSignal on the fetch closes the connection.
  - id: websocket
    file: examples/02_websocket.ts
    title: "A WebSocket client: validated messages, resubscribe on reconnect"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "4-8"
        note: >
          WebSocket gives you frames, not a protocol. Define the messages as a
          discriminated union and validate each one: a frame is untrusted input
          like any response body.
      - lines: "10-16"
        note: >
          An exception thrown in an event listener is reported as uncaught, and
          the rest of that message's handling is lost. Turn bad JSON into a
          value the schema rejects.
      - lines: "18-19"
        note: >
          The client depends on two members of WebSocket. The real class
          satisfies the type structurally, and the fake below does too, with no
          shared base class.
      - lines: "29-33"
        note: >
          Server-side state such as subscriptions belongs to one connection.
          After a reconnect the server knows nothing, so the client subscribes
          again on every open, not once at startup.
      - lines: "34-40"
        note: >
          Two malformed frames are dropped and the stream continues. Throwing
          here, or trusting a cast, would break the live view on the first odd
          message.
      - lines: "41-46"
        note: >
          wasClean is false when the connection died without a close handshake;
          1006 is the code the browser reports then, and no endpoint ever sends
          it. Reconnect with exponential backoff, and jitter in production, or
          every client hits a restarted server at once.
      - lines: "51-59"
        note: >
          A real socket fires these events from the network. Here the example
          fires them itself, which makes the order deterministic. A real
          send() throws InvalidStateError while the socket is still connecting
          and silently drops data once it is closing or closed.
  - id: cors
    file: examples/03_cors.ts
    title: "CORS: when the browser preflights, and when the page may read the answer"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-9"
        note: >
          Real, not modelled: this sandboxed iframe (sandbox without
          allow-same-origin) has an opaque origin, which serializes as "null",
          and no access to cookies: reading document.cookie throws.
      - lines: "10-11"
        note: >
          fetch's defaults: mode "cors" for cross-origin requests, and
          credentials "same-origin", so cookies go only to the page's own
          origin unless you pass credentials: "include".
      - lines: "13-30"
        note: >
          Requests an HTML form could always send (GET, HEAD, POST with form or
          text bodies and a few plain headers) go out directly. Anything else,
          JSON bodies and Authorization headers included, is preceded by an
          OPTIONS preflight, and the real request is sent only if the answer
          allows it. Simplified: the real rules also limit header values.
      - lines: "43-51"
        note: >
          The decision after the response arrives. A simple request has already
          reached the server and run, so CORS is no protection for its side
          effects. "*" never works with credentials; then the origin must match
          exactly and Access-Control-Allow-Credentials must be "true".
      - lines: "59-61"
        note: >
          A null-origin page can read "*" responses, and anything that
          allow-lists the string "null". Every sandboxed iframe, data: URL and
          file: page in the world has that origin, so never allow-list it.
use_cases:
  - "SSE: notifications, progress of server-side jobs, streamed generated text; one direction, plain HTTP, automatic reconnect"
  - "WebSocket: chat, collaborative editing, multiplayer state; both directions, low overhead per message"
  - "CORS: a SPA on app.example.com calling an API on api.example.com; a public read-only API answering with *"
  - "Same-origin instead of CORS: a reverse proxy (or Vite's dev proxy, as in this app) serving page and API from one origin"
strengths:
  - "EventSource reconnects and resumes (Last-Event-ID) on its own, unlike a raw socket in Go or Java where you write that loop"
  - "SSE is ordinary HTTP: proxies, HTTP/2 multiplexing, cookies and server logs all just work"
  - "WebSocket is supported everywhere, bidirectional, and has little per-message overhead after the handshake"
  - "CORS lets the server opt in per origin and per credential mode, without the browser ever exposing a response by default"
weaknesses:
  - "EventSource is GET-only with no custom headers; over HTTP/1.1 each stream holds one of the browser's 6 connections per domain"
  - "WebSocket has no backpressure, no built-in reconnect and no message schema; all three are your code"
  - "CORS failures reach JavaScript as an opaque TypeError; only the devtools console says why"
  - "CORS protects reading, not acting: simple cross-origin requests still execute on the server"
pitfalls:
  - title: "Treating CORS as access control"
    detail: "The server still receives and executes simple cross-origin requests, and non-browser clients ignore CORS entirely. Authenticate every request and protect cookie-authenticated writes against CSRF."
  - title: "Reflecting any Origin with credentials"
    detail: "Echoing the request's Origin into Access-Control-Allow-Origin together with Allow-Credentials: true lets every website read your users' data. Compare against an allow-list, send Vary: Origin, and never allow-list null."
  - title: "Not checking Origin on the WebSocket handshake"
    detail: "CORS doesn't govern WebSockets. The browser sends the page's Origin and the user's cookies with the handshake; a server that doesn't check Origin can be driven from any site (cross-site WebSocket hijacking)."
  - title: "Bearer tokens in localStorage"
    detail: "Any script on the page, including an injected one, can read them. Prefer HttpOnly cookies, or keep a short-lived access token in memory only."
  - title: "Forgetting to resubscribe after a reconnect"
    detail: "Server-side subscriptions die with the connection. Send them again on every open event."
quiz:
  - q: "A page on https://app.example.com POSTs a form-encoded body to https://api.example.com, which sends no CORS headers. What happens?"
    options: ["The browser blocks the request before it is sent", "The request reaches the server and runs; the page just can't read the response", "The browser sends a preflight first"]
    answer: 1
    why: "A form-encoded POST is a simple request: no preflight. CORS only decides whether the response is exposed to the page."
  - q: "Which of these requests triggers a preflight?"
    options: ["GET with Accept: application/json", "POST with Content-Type: multipart/form-data", "POST with Content-Type: application/json", "HEAD without extra headers"]
    answer: 2
    why: "application/json is not one of the three form content types, so the browser asks permission with OPTIONS first."
  - q: "fetch(url, { credentials: \"include\" }) gets Access-Control-Allow-Origin: *. Can the page read the response?"
    options: ["Yes, * allows every origin", "No: credentialed responses need the exact origin and Access-Control-Allow-Credentials: true", "Only if the cookie is HttpOnly"]
    answer: 1
    why: "The wildcard is ignored for credentialed requests. The server must echo the specific origin and allow credentials."
  - q: "Your app needs live order updates from server to browser, and requests carry a session cookie. What is the simplest fit?"
    options: ["EventSource (SSE)", "A WebSocket", "Polling every 100 ms"]
    answer: 0
    why: "One-way push over plain HTTP: EventSource sends cookies (same-origin, or with withCredentials), reconnects itself and resumes with Last-Event-ID."
further_reading:
  - title: "MDN: Using server-sent events"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events"
  - title: "MDN: The WebSocket API"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API"
  - title: "MDN: Cross-Origin Resource Sharing (CORS)"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CORS"
  - title: "MDN: Origin header (when it is null)"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Origin"
  - title: "MDN: Set-Cookie (HttpOnly, SameSite)"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Set-Cookie"
  - title: "MDN: CloseEvent.code"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/CloseEvent/code"
---

## Concept

**Server-sent events (SSE)** keep one HTTP response open and send UTF-8 text in a tiny
format (`text/event-stream`): `field: value` lines, events separated by a blank line.
`EventSource` parses it, dispatches `message` events (or named ones from the `event`
field), and reconnects on its own, sending the last `id` it saw as `Last-Event-ID`:

```ts
const es = new EventSource("/api/orders/events"); // same origin: cookies go along
es.addEventListener("order", (e) => render(JSON.parse(e.data)));
es.onerror = () => {
  // readyState CONNECTING: it is reconnecting; CLOSED: it gave up
};
es.close(); // when the view goes away
```

The constructor takes a URL and one option, `withCredentials` (send cookies to another
origin), nothing else: always GET, no custom headers. When an API needs POST or an
`Authorization` header, read the stream from fetch and parse it yourself (example 1).

**WebSockets** upgrade an HTTP request into a two-way channel of messages.
`new WebSocket("wss://…")` fires `open`, `message`, `error` and `close`; `send()` throws
while the socket is still connecting. There is no reconnect, no message schema and no
backpressure: when messages arrive faster than you handle them, they pile up in memory.
`WebSocketStream` was an attempt to add backpressure; MDN now points to WebTransport
instead.

**CORS** relaxes the same-origin policy, which lets a page *send* requests to other
origins but not *read* the answers. The browser attaches `Origin` to cross-origin
requests. For anything beyond what an HTML form could send, it first sends an `OPTIONS`
preflight. Then it reads the response's `Access-Control-Allow-*` headers and decides
whether the page may see the response. A refusal surfaces as a `TypeError` with no
details. `Access-Control-Expose-Headers` widens the short list of response headers a
page may read. An opaque origin, from a sandboxed iframe like the lesson runner, a
`data:` URL or a `file:` page, is sent as `Origin: null`.

**Tokens vs cookies** decides how a request proves who sends it:

- A **cookie** is sent by the browser automatically. `HttpOnly` hides it from JavaScript
  (an XSS bug can't steal it, though it can still make requests while the user is on the
  page), `Secure` keeps it off plain HTTP, and `SameSite` controls cross-site sending.
  Because it is automatic, cookie-authenticated writes need CSRF protection. Cross-origin,
  fetch sends it only with `credentials: "include"`, and the server must answer with the
  exact origin plus `Access-Control-Allow-Credentials: true`.
- A **bearer token** in an `Authorization` header is sent only when your code adds it, so
  there is no CSRF. But your JavaScript must be able to read the token, so XSS can steal
  it, and the header makes every cross-origin request preflighted. EventSource can't send
  it at all, and a WebSocket can't send custom headers either: there the token goes in the
  first message, or in the URL, where proxies and access logs record it.

## Coming from C/C++/Java/Python/JS

- **Go**: a server sends SSE by writing `data: …\n\n` and flushing
  (`http.ResponseController.Flush`); go.http.uploads-streams shows the server side and
  go.http.cors-cookies-httptest the CORS middleware and cookies. The Go standard library
  has no WebSocket server; teams use a module such as `github.com/coder/websocket`.
- **Java**: Spring's `SseEmitter` and `@ServerEndpoint` (Jakarta WebSocket) are the server
  halves of this lesson. `HttpClient` has a WebSocket client too. Nothing in Java resembles
  CORS: it exists because a browser runs code from many mutually untrusting origins, all
  with the user's cookies.
- **Python**: `sse-starlette` or `websockets` on the server; a `requests` client ignores
  CORS entirely, which is the point: CORS restrains browsers, not clients.
- **C/C++**: a WebSocket is a framed TCP stream after an HTTP upgrade handshake; SSE is a
  chunked HTTP response you read line by line, as you would with `fgets`.

## How it works

A preflighted cross-origin request with a session cookie, from both sides:

```text
OPTIONS /orders/7              Origin: https://app.example.com
                               Access-Control-Request-Method: PUT
                               Access-Control-Request-Headers: content-type
204                            Access-Control-Allow-Origin: https://app.example.com
                               Access-Control-Allow-Methods: GET, PUT
                               Access-Control-Allow-Headers: content-type
                               Access-Control-Allow-Credentials: true
                               Access-Control-Max-Age: 600
                               Vary: Origin
PUT /orders/7                  Origin: https://app.example.com, Cookie: session=…
200                            Access-Control-Allow-Origin: https://app.example.com
                               Access-Control-Allow-Credentials: true
                               Vary: Origin
```

The preflight carries no cookies; its answer may be cached for `Max-Age` seconds.
`Vary: Origin` stops a shared cache from serving one origin's permission to another. On
the client the only line that matters is
`fetch(url, { method: "PUT", credentials: "include", headers, body })`.

## Design notes

Prefer one origin. A reverse proxy that serves the page and `/api` from the same origin
removes CORS, preflights and cross-site cookie questions in one move. This app does that:
the frontend calls `/api/...` on its own origin and Vite's dev server proxies it to the Go
backend, which sends no CORS headers at all (see `server.proxy` in `vite.config.ts` and
`API_BASE` in `src/shared/api.ts`).

When you do need CORS, keep the policy in one middleware on the server: an allow-list of
origins, credentials only where needed, `Vary: Origin`, and a short `Max-Age`. Never
reflect arbitrary origins and never allow-list `null`.

Pick the realtime channel by direction. Server-to-client only: SSE, with cookies for
authentication and `id` fields so a reconnect resumes without gaps. Both directions with
high message rates: WebSocket, with a validated message union, resubscription on every
`open`, backoff with jitter, a heartbeat to detect dead connections, and an Origin check
on the server's handshake. Either way, close the connection when the view unmounts.
Over HTTP/1.1 each open SSE stream occupies one of the browser's six connections per
domain, shared by all tabs; HTTP/2 multiplexes streams over one connection.
