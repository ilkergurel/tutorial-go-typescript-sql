---
id: go.http.handlers
title: "The handler model and ServeMux patterns"
track: go
module: http
order: 1
level: intermediate
minutes: 13
since: go1.22
prerequisites: [go.interfaces.implicit, go.basics.functions]
compare_with: [ts.node.cli-server, inside.codebase.http-layer, go.http.middleware]
tags: [http, net/http, servemux, routing, handlers]
summary: >
  An http.Handler is one method, ServeHTTP, and everything in net/http, the router
  included, is built from it. Since Go 1.22 ServeMux matches methods and wildcards
  and picks the most specific pattern, so most services need no router library.
examples:
  - id: handler-model
    file: examples/01_handler_model.go
    title: "Handlers, HandlerFunc, and the ResponseWriter's order of operations"
    runtime: go-server
    check: exact
    timeout_s: 10
    annotations:
      - lines: "11-17"
        note: >
          No base class and no registration magic: a type with a ServeHTTP
          method is a Handler. Its fields are shared by every request, and
          requests run concurrently (one goroutine per connection), so fields
          must be read-only or protected by a mutex.
      - lines: "16"
        note: >
          PathValue returns the segment already percent-decoded: %C4%B0zmir
          arrives as İzmir, and the client encoded the ş in Ayşe by itself.
          The value is always a string; turning it into an int and answering
          400 when that fails is your job.
      - lines: "20-24"
        note: >
          The ResponseWriter is a stream, not a response object you return:
          headers, then status, then body. A header set after WriteHeader or
          the first Write is silently ignored, because it has already been sent.
      - lines: "28-29"
        note: >
          Handle takes a Handler. HandleFunc takes a function and converts it
          to http.HandlerFunc, a function type whose ServeHTTP method calls the
          function itself. That one-line adapter is why plain functions work.
      - lines: "31"
        note: >
          httptest.NewServer accepts any Handler and serves it on a loopback
          port. The mux is just another Handler, which is why middleware can
          wrap the whole router.
      - lines: "34-46"
        note: >
          show(http.Get(...)) passes both results straight through: Go allows
          f(g()) when g's results match f's parameters. greeter set no
          Content-Type, so net/http sniffed one from the first 512 bytes.
          /users/7 matches no pattern and gets the mux's plain-text 404.
  - id: patterns
    file: examples/02_patterns.go
    title: "Wildcards, subtrees, {$}, and the most specific pattern wins"
    runtime: go-server
    check: exact
    timeout_s: 10
    annotations:
      - lines: "10-20"
        note: >
          Request.Pattern (Go 1.23) is the pattern that matched. Log and count
          by it: "GET /posts/{id}" is one value per route, while raw paths
          produce one value per ID and blow up your metrics.
      - lines: "24-33"
        note: >
          Registration order is irrelevant. ServeMux picks the most specific
          matching pattern, the one that matches a strict subset of the
          other's requests, so /posts/latest beats /posts/{id} even though it
          was registered second.
      - lines: "29-30"
        note: >
          "/" matches every path, the classic accidental catch-all; "/{$}"
          matches only the root. /posts/42/comments lands in "/" because
          {id} matches exactly one segment.
      - lines: "48-51"
        note: >
          /docs has no pattern, but "GET /docs/" owns that subtree, so the mux
          redirects to /docs/ and http.Get follows the redirect. Register
          "/docs" separately if you want to answer it yourself.
  - id: methods-conflicts
    file: examples/03_methods_conflicts.go
    title: "405 with Allow, HEAD for free, and conflicts caught at registration"
    runtime: go-server
    check: exact
    timeout_s: 10
    annotations:
      - lines: "11-26"
        note: >
          Recovering here is only so the output can show the messages. In a
          real program, let the panic stop startup: a broken routing table
          should never serve a request.
      - lines: "39-53"
        note: >
          HEAD is served by the GET handler and the server drops the body. PUT
          matches the path but no method, so the mux answers 405 with an Allow
          header instead of a misleading 404. Methods are case-sensitive:
          "get" is not GET.
      - lines: "55-59"
        note: >
          "GET /" is narrower in method, "/index.html" narrower in path, so
          neither is more specific: a conflict, whichever comes first. Renaming
          a wildcard makes a duplicate, and wildcards must be whole segments.
use_cases:
  - "JSON APIs and internal services: method + path patterns cover what chi or gorilla/mux were used for before Go 1.22"
  - "Health and readiness endpoints, admin pages and webhooks next to the main API on one mux"
  - "Mounting a sub-application under a prefix with \"/admin/\" and http.StripPrefix"
  - "Tests: the router is a Handler, so httptest exercises the whole routing table without a network"
strengths:
  - "One interface for everything: routers, middleware, file servers and test servers all take and return http.Handler, unlike Java's split between Servlet, Filter and the container"
  - "Precedence by specificity, not registration order, and ambiguous route sets panic at startup instead of misrouting in production, unlike Express's first-match rule"
  - "405 with an Allow header and HEAD support come for free"
  - "Blocking code per request: no callbacks or async/await, yet requests run concurrently on goroutines"
weaknesses:
  - "No typed or regex constraints: {id} matches any segment, and parsing \"abc\" into an int is your code, unlike Flask's <int:id> or Spring's @PathVariable conversion"
  - "No route groups or reverse routing (building a URL from a route name); you nest muxes, wrap handlers, and build URLs by hand"
  - "The ResponseWriter's ordering rules fail silently: late headers are dropped, a second WriteHeader only logs a warning"
  - "The default 404 and 405 bodies are plain text; an API that answers JSON errors must register its own \"/\" fallback"
pitfalls:
  - title: "Forgetting return after an error response"
    detail: "http.Error writes a response; it does not stop the handler. The code below it still runs, appends to the body, and the server logs \"superfluous response.WriteHeader call\"."
    bad: "if err != nil {\n\thttp.Error(w, \"bad id\", http.StatusBadRequest)\n}\nw.WriteHeader(http.StatusOK)"
    good: "if err != nil {\n\thttp.Error(w, \"bad id\", http.StatusBadRequest)\n\treturn\n}"
  - title: "Setting headers after the body"
    detail: "Header().Set after WriteHeader or the first Write has no effect. Set Content-Type, Location and cookies before writing anything."
  - title: "A go.mod older than 1.22"
    detail: "The go line sets the GODEBUG defaults, so a module saying go 1.21 gets the old mux: \"GET /posts/{id}\" is not a method pattern and every request 404s, silently. go vet does report PathValue as requiring go1.22."
  - title: "\"/\" when you meant the home page"
    detail: "\"/\" matches every path that nothing else claims, including typos, so they get your home page with a 200. Use \"/{$}\" (or \"GET /{$}\") for the root alone."
  - title: "Lowercase methods in patterns"
    detail: "\"get /users\" registers a pattern for a method literally named get. A GET request then receives 405 with \"Allow: get\". Use http.MethodGet or uppercase strings."
quiz:
  - q: "You register \"GET /posts/latest\" first and \"GET /posts/{id}\" second. Which handler serves GET /posts/latest?"
    options: ["The first registered", "The last registered", "GET /posts/latest, because it is more specific", "Registration panics: they conflict"]
    answer: 2
    why: "ServeMux ignores order and picks the pattern that matches a strict subset of the other's requests; a literal segment is narrower than a wildcard."
  - q: "Only \"GET /items/{id}\" is registered. What does PUT /items/1 get?"
    options: ["404 Not Found", "405 Method Not Allowed with \"Allow: GET, HEAD\"", "200 from the GET handler", "501 Not Implemented"]
    answer: 1
    why: "The path matches a pattern with another method, so the mux answers 405 and lists the methods that would work; GET implies HEAD."
  - q: "What happens when you register both \"GET /\" and \"/index.html\"?"
    options: ["\"/index.html\" wins for every method", "\"GET /\" wins because it was first", "The second registration panics", "Both handlers run"]
    answer: 2
    why: "One is narrower in method, the other in path; neither matches a subset of the other's requests, so they conflict and Handle panics."
  - q: "A handler calls w.WriteHeader(201) and then w.Header().Set(\"Location\", \"/users/7\"). What does the client see?"
    options: ["201 with the Location header", "201 without the Location header", "A 500 error", "A compile error"]
    answer: 1
    why: "The status line and headers are sent on WriteHeader; later changes to the header map are ignored."
further_reading:
  - title: "package net/http: ServeMux (patterns, precedence, compatibility)"
    url: "https://pkg.go.dev/net/http#ServeMux"
  - title: "The Go Blog: Routing Enhancements for Go 1.22"
    url: "https://go.dev/blog/routing-enhancements"
  - title: "package net/http: Handler and ResponseWriter"
    url: "https://pkg.go.dev/net/http#ResponseWriter"
  - title: "Go, Backwards Compatibility, and GODEBUG"
    url: "https://go.dev/doc/godebug"
---

## Concept

The whole server side of `net/http` rests on one interface:

```go
type Handler interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}
```

A router is a `Handler` that picks another `Handler`. Middleware is a function that takes a
`Handler` and returns one. `http.FileServerFS`, `http.TimeoutHandler` and
`httptest.NewServer` all speak the same type. `http.HandlerFunc` is a function type with a
`ServeHTTP` method that calls the function, so any `func(w, r)` becomes a `Handler` by a
plain conversion.

The `ResponseWriter` is a stream with a fixed order: set headers, then write the status
(`WriteHeader`, or implicitly 200 on the first `Write`), then write the body. The server
gives each connection its own goroutine, so handlers run concurrently and write blocking,
sequential code.

`http.ServeMux` has understood patterns of the form `[METHOD ][HOST]/[PATH]` since Go 1.22:

- `{name}` matches one path segment, `{name...}` the rest of the path, and `r.PathValue`
  returns the decoded value.
- A trailing slash (`/docs/`) matches a subtree; `{$}` matches the end of the path only.
- A method in the pattern restricts it; `GET` also matches `HEAD`. A path that matches only
  under another method gets **405** with an `Allow` header.
- When several patterns match, the **most specific** wins: the one whose matches are a
  strict subset of the other's. If neither is more specific, registering the second one
  **panics**, whatever the order.

## Coming from C/C++/Java/Python/JS

- **Java**: `ServeHTTP` is `HttpServlet.service(req, resp)` without the container. The
  server is a library you start from `main`. Spring's `@GetMapping("/posts/{id}")` maps to
  `mux.HandleFunc("GET /posts/{id}", ...)`, but there is no reflection: `PathValue` gives
  you a string and `strconv.Atoi` is yours to call.
- **Node/Express**: `app.get('/posts/:id', ...)` dispatches to the first registered match;
  ServeMux ignores order and rejects ambiguous route sets at startup. Express handlers share
  one thread, while Go handlers run in parallel: a map shared between requests needs a
  `sync.Mutex`.
- **Python**: WSGI's `app(environ, start_response)` has the same shape as `ServeHTTP`.
  Flask's `<int:id>` converters have no equivalent; validate inside the handler.
- **C/C++**: no event loop and no callbacks on readiness. Each connection reads blocking in
  its own goroutine, and the runtime multiplexes those goroutines over a few OS threads.

## How it works

    net.Listener ── Accept ──> one goroutine per connection
                                   │ reads a request
                                   ▼
                     server.Handler.ServeHTTP(w, r)      your root Handler (usually middleware)
                                   ▼
                     ServeMux: clean the path, find the most specific pattern
                       ├─ match           -> handler(w, r) with r.Pattern and PathValue set
                       ├─ subtree root    -> redirect to the path with a trailing slash
                       ├─ other method    -> 405 + Allow
                       └─ nothing         -> 404 (or your "/" pattern)

## Design notes

Start with `http.ServeMux`; reach for a router library only for what it lacks: typed or
regex segment constraints, route groups with their own middleware, and URL building from
route names. StackLab itself uses the standard mux (ADR-0001): every route is registered in
one function, `NewRouter` in `server/internal/httpapi/router.go`, which reads like a table
of contents of the API. A test that merely builds the router catches conflicting patterns,
because they panic at registration.

Two shapes for handlers with dependencies coexist there. A struct whose methods are
handlers (`learningHandlers`) suits a group sharing the same store and logger. A function
that returns an `http.HandlerFunc` closure (`handleReady(logger, pool)`) suits a one-off.
Both are constructor injection without globals.

Before Go 1.22, handlers checked `r.Method` by hand and cut IDs out of `r.URL.Path` with
`strings.TrimPrefix`, which is why chi and gorilla/mux were near-universal. Code like that
still works, but new code should put the method and wildcards in the pattern. Check the
`go` line in `go.mod`: below 1.22 the old matching applies (GODEBUG `httpmuxgo121=1`), and
the new patterns quietly fail to match.
