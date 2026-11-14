---
id: go.http.cors-cookies-httptest
title: "CORS, cookies, and httptest"
track: go
module: http
order: 6
level: advanced
minutes: 14
since: go1.27
prerequisites: [go.http.middleware, go.testing.table-driven]
compare_with: [ts.services.realtime-cors, go.testing.fakes-httptest, inside.codebase.ts-runner]
tags: [http, cors, cookies, csrf, httptest, origin]
summary: >
  CORS is the server telling browsers which other origins may read its responses;
  it stops no request. Cookies carry sessions, so they need HttpOnly, Secure,
  SameSite and CSRF protection. httptest runs all of it in tests without a network.
examples:
  - id: cors
    file: examples/01_cors.go
    title: "A CORS middleware: preflights, an allow-list, and what it doesn't stop"
    runtime: go-server
    check: exact
    timeout_s: 10
    annotations:
      - lines: "11-12"
        note: >
          The key fact about CORS: the server only labels responses. The
          browser decides whether the page's JavaScript may read them. curl,
          Go programs and attackers' scripts ignore the labels entirely.
      - lines: "16-19"
        note: >
          Reflect the request's Origin only when it is on the list, and add
          Vary: Origin so a cache never serves one origin's permission to
          another. "*" also works, but never together with cookies or other
          credentials.
      - lines: "20-28"
        note: >
          A preflight is an OPTIONS request with Access-Control-Request-Method.
          The browser sends one before a PUT, a DELETE, or a POST with
          Content-Type: application/json, and sends the real request only if
          the answer allows it. Max-Age lets the browser cache that answer.
      - lines: "41"
        note: >
          CORS wraps the mux, not the route. The mux only knows "PUT
          /notes/{id}", so a preflight reaching it would get 405 and the
          browser would never send the PUT.
      - lines: "47-48"
        note: >
          For the other site the preflight succeeds as an HTTP exchange but
          carries no permission, so a browser stops there. A non-browser
          client simply sends the PUT, and the handler runs.
      - lines: "49"
        note: >
          A sandboxed iframe without allow-same-origin has an opaque origin,
          sent as the string "null". StackLab's TS runner iframe is exactly
          that. Never allow-list "null": every sandboxed iframe, data: URL and
          file: page in the world has that same origin.
      - lines: "67"
        note: >
          Three executions, not one: the handler ran for every PUT. Protect
          state-changing endpoints with authentication and CSRF defenses;
          CORS only protects the response from being read.
  - id: cookies
    file: examples/02_cookies.go
    title: "Session cookies, non-ASCII values, and CSRF protection (Go 1.25)"
    runtime: go-server
    check: exact
    timeout_s: 10
    annotations:
      - lines: "13-23"
        note: >
          The usual session cookie. The value is a random ID pointing at
          server-side state, so the cookie carries nothing worth reading.
          SameSite=Lax withholds it from cross-site POSTs and subrequests but
          keeps it on ordinary links into your site.
      - lines: "15"
        note: >
          The __Host- prefix is a promise browsers enforce: such a cookie is
          rejected unless it is Secure, has Path=/ and no Domain, so a
          sibling subdomain can't set or overwrite it.
      - lines: "29-30"
        note: >
          Cookie values allow only a subset of ASCII. net/http drops the
          bytes of ş and logs a warning on stderr; the log even shows the
          first byte as Å, UTF-8 read as Latin-1. Encode such values
          (base64url, or percent-encoding) and decode them on the way back.
      - lines: "43"
        note: >
          Deleting a cookie is setting it again with the same name, path and
          domain and an expiry in the past. In Go, MaxAge < 0 sends Max-Age=0.
      - lines: "46"
        note: >
          CrossOriginProtection rejects every method except GET, HEAD and
          OPTIONS when a browser sends it from another origin, judged by
          Sec-Fetch-Site or, failing that, Origin versus Host. Requests without either header (curl, server
          to server) pass, so it complements authentication.
      - lines: "49-50"
        note: >
          srv.Client() trusts the TLS test certificate; a cookiejar makes the
          client keep and resend cookies like a browser, Secure ones included,
          because the server speaks HTTPS.
      - lines: "70"
        note: >
          A browser sets Sec-Fetch-Site: cross-site when another site's form
          posts here. The 403 means the logout never happened: the next /me
          still finds the session.
  - id: handlers-test
    file: examples/03_handlers_test.go
    title: "Testing handlers with ResponseRecorder and NewTestServer (Go 1.27)"
    runtime: go-test
    check: contains
    annotations:
      - lines: "25-45"
        note: >
          The fastest handler test: build a request with httptest.NewRequest,
          call ServeHTTP with a ResponseRecorder, inspect the result. The whole
          router runs, 405 included, with no socket and no goroutine. The CORS
          middleware from example 1 is tested the same way: set Origin on the
          request and check Access-Control-Allow-Origin in the result.
      - lines: "37"
        note: >
          httptest.NewRequest builds a server-side request (Host example.com
          unless the target is absolute) and panics on bad input, which is
          what you want in a test. http.NewRequest builds a client request
          instead.
      - lines: "38-39"
        note: >
          rec.Header() is the live map a handler mutates; rec.Result().Header
          is a snapshot of what was sent at the first write. Assert on the
          snapshot, or a header set too late will look correct. http.Error
          sets text/plain itself, which is why the error rows expect it.
      - lines: "49-55"
        note: >
          NewTestServer (Go 1.27) runs a real server and client over an
          in-memory network: any host name reaches it, no port is opened, and
          t.Cleanup shuts it down. It also fails the test if a handler panics.
          httptest.NewServer is the loopback alternative when a test needs a
          real address.
use_cases:
  - "An API used by a browser app on another origin, such as a Vite dev server on :5173 calling a backend on :8080 without a proxy"
  - "Cookie-based sessions for server-rendered pages and same-site SPAs"
  - "CSRF protection for any cookie-authenticated endpoint that changes state"
  - "Handler, middleware and router tests that run in milliseconds without ports"
strengths:
  - "CORS is a few headers in one middleware; no framework configuration DSL like Spring's CorsConfiguration"
  - "http.CrossOriginProtection (Go 1.25) gives token-free CSRF protection in one wrapper, unlike Django's or Spring Security's synchronizer tokens"
  - "http.Cookie models every attribute (SameSite, Partitioned since Go 1.23) and SetCookie validates what it writes"
  - "httptest covers three levels, recorder, in-memory server and loopback server, all in the standard library, where Java needs MockMvc or a test container"
weaknesses:
  - "CORS mistakes fail silently on the server: the request succeeds, only the browser console complains"
  - "Invalid cookie bytes are dropped with a log line, not an error; nothing stops a deploy"
  - "CrossOriginProtection relies on headers that only browsers send, so it is a browser-facing defense, not authentication"
  - "A ResponseRecorder skips the transport: connection handling, Flush and Hijack behavior need a test server"
pitfalls:
  - title: "Allowing the null origin"
    detail: "Access-Control-Allow-Origin: null admits every sandboxed iframe, data: URL and file: document, including hostile ones. Serve such clients through a proxy or a data: URL design instead."
  - title: "Reflecting any Origin with credentials"
    detail: "Echoing whatever Origin arrives plus Access-Control-Allow-Credentials: true lets every website read authenticated responses. Compare against an explicit allow-list."
    bad: "w.Header().Set(\"Access-Control-Allow-Origin\", r.Header.Get(\"Origin\"))"
    good: "if allowed[origin] {\n\tw.Header().Set(\"Access-Control-Allow-Origin\", origin)\n}\nw.Header().Add(\"Vary\", \"Origin\")"
  - title: "CORS middleware inside the router"
    detail: "With method patterns, OPTIONS reaches no route and the mux answers 405, so every preflighted request fails. Wrap the mux."
  - title: "Treating CORS as access control"
    detail: "Simple requests, such as a form POST, are sent cross-origin without any preflight, and non-browser clients ignore CORS entirely. Authenticate, and protect cookie-based state changes against CSRF."
  - title: "Non-ASCII in cookie values"
    detail: "\"Ayşe\" becomes \"Aye\" with only a log line to show for it. Base64url- or percent-encode the value."
quiz:
  - q: "A page on https://evil.example calls fetch(\"https://api.example/notes/1\", {method: \"PUT\"}). Your server answers the preflight without Access-Control-Allow-Origin. What happens?"
    options: ["The PUT is sent and its response hidden", "The browser never sends the PUT", "The server must answer 403", "The PUT is sent without cookies"]
    answer: 1
    why: "PUT is not a simple method, so the browser preflights first and stops when the preflight grants nothing."
  - q: "Why does a sandboxed iframe (sandbox=\"allow-scripts\", no allow-same-origin) send Origin: null?"
    options: ["Iframes never send Origin", "Its document gets an opaque origin, which serializes as null", "srcdoc documents have no URL", "The CSP strips the header"]
    answer: 1
    why: "Without allow-same-origin the sandbox gives the document a unique opaque origin, and opaque origins serialize as \"null\"."
  - q: "http.SetCookie(w, &http.Cookie{Name: \"n\", Value: \"Ayşe\"}). What reaches the browser?"
    options: ["n=Ayşe", "n=\"Ayşe\"", "n=Aye, with a warning logged by net/http", "Nothing: SetCookie returns an error"]
    answer: 2
    why: "Cookie values are limited to a subset of ASCII; net/http drops the invalid bytes and logs, so encode non-ASCII values."
  - q: "What does an httptest.ResponseRecorder test not exercise?"
    options: ["The router's 404 and 405 handling", "Middleware", "The real HTTP transport: connections, serialization, Flush and Hijack", "Response headers"]
    answer: 2
    why: "ServeHTTP is called directly with a recorder, so routing and middleware run, but no connection or wire format is involved; use a test server for those."
further_reading:
  - title: "MDN: Cross-Origin Resource Sharing (CORS)"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Guides/CORS"
  - title: "MDN: Access-Control-Allow-Origin (why not null)"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Access-Control-Allow-Origin"
  - title: "MDN: Set-Cookie"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Set-Cookie"
  - title: "package net/http: CrossOriginProtection"
    url: "https://pkg.go.dev/net/http#CrossOriginProtection"
  - title: "package net/http/httptest"
    url: "https://pkg.go.dev/net/http/httptest"
---

## Concept

An **origin** is scheme + host + port: `http://127.0.0.1:5173` (Vite) and
`http://127.0.0.1:8080` (the Go backend) are different origins. Browsers let a page *send*
many cross-origin requests (links, forms, images, simple fetches) but not *read* the
responses, unless the server opts in with CORS headers:

- **Simple requests** (GET, HEAD, POST with a form or plain-text body) are sent directly;
  the browser shows the response to the page only if `Access-Control-Allow-Origin` matches.
- **Preflighted requests** (PUT, DELETE, JSON bodies, custom headers) start with an
  `OPTIONS` request; the real one follows only if the preflight answer allows it.
- With cookies (`credentials: "include"`), the server must name the exact origin and send
  `Access-Control-Allow-Credentials: true`; `*` is rejected.

The `Origin` header is the string `null` for **opaque origins**: documents in an
`<iframe sandbox>` without `allow-same-origin`, `data:` and `file:` documents, and requests
after cross-origin redirects.

**Cookies** carry sessions automatically, which is both their convenience and their
danger. `HttpOnly` hides them from scripts, `Secure` keeps them off plain HTTP, `SameSite`
limits cross-site sending, and the `__Host-` prefix pins them to one host. Because the
browser attaches them to requests other sites trigger, cookie-authenticated state changes
need **CSRF** protection; since Go 1.25 that is `http.CrossOriginProtection`.

**httptest** tests all of this without a network: `NewRecorder` and `NewRequest` call a
handler directly, `NewTestServer` (Go 1.27) runs a real client and server over an in-memory
network, and `NewServer`/`NewTLSServer` listen on loopback.

## Coming from C/C++/Java/Python/JS

- **JS/React**: you have met CORS as a red console error. This is the other side: the
  server's headers decide, and the browser enforces. The Express `cors` package is the same
  middleware as example 1.
- **Java/Spring**: `@CrossOrigin` and `CorsConfiguration` generate these headers; Spring
  Security's CSRF tokens solve what `CrossOriginProtection` solves with `Sec-Fetch-Site`.
  `MockMvc` corresponds to the `ResponseRecorder` style, a random-port `@SpringBootTest` to
  `httptest.NewServer`.
- **Python**: Django's CSRF middleware uses tokens; `flask-cors` and Starlette's
  `CORSMiddleware` add the headers. Flask's `test_client()` calls the app without a socket,
  like a recorder.
- **C/C++**: a non-browser client never sees CORS at all, which is the point to remember.

## How it works

StackLab's TS runner, `src/features/runner/browserTs.ts`, is a worked example of the
null origin:

    <iframe sandbox="allow-scripts" srcdoc="...lesson...">
      - no allow-same-origin   -> opaque origin, Origin: null, no access to the app's
                                  DOM, storage or cookies
      - CSP default-src 'none' -> no fetch at all, so lessons stay deterministic
      - libraries              -> data: URLs in an import map (ADR-0010), which need
                                  neither CORS nor a CSP exception

The app itself avoids CORS too: the Vite dev server proxies `/api` to the backend, so the
browser only ever talks to one origin, and the backend sends no CORS headers at all. If a
lesson iframe were allowed to call the API, the backend would have to allow `null`, which
would also allow every other sandboxed document on the web.

## Design notes

Avoid CORS when you can: serve the frontend and the API from one origin, in development
through a proxy as StackLab does, in production behind one reverse proxy. When you need
it, keep an explicit allow-list, reflect only listed origins, add `Vary: Origin`, answer
preflights before routing, and keep `Allow-Credentials` off unless the client sends
cookies.

For sessions, store a random ID in a `__Host-` cookie with `HttpOnly`, `Secure` and
`SameSite=Lax`, keep the data on the server, and wrap state-changing routes in
`CrossOriginProtection`. Generate IDs with `crypto/rand.Text`. Keep tokens for third-party
API clients in the `Authorization` header instead: no cookies, no CSRF, but the client has
to store the token somewhere scripts can reach.

Test at the cheapest level that covers the behavior: recorder tests for routing,
validation, status codes and headers; `NewTestServer` when the client side matters
(cookies in a jar, redirects, TLS); loopback servers only for code that needs a real
address. Lesson go.testing.fakes-httptest goes further with fakes and golden files.
