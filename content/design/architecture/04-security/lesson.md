---
id: design.architecture.security
title: "Security across the stack: validation, SQL injection, XSS, CSRF, secrets, least privilege"
track: design
module: architecture
order: 4
level: advanced
minutes: 15
since: go1.25
prerequisites: [go.postgres.transactions, go.http.cors-cookies-httptest, ts.services.zod]
compare_with: [go.cli.templates, sql.security-ops.roles, sql.security-ops.rls, ts.services.realtime-cors, inside.codebase.runner-sandbox]
tags: [security, sql-injection, xss, csrf, secrets, least-privilege, owasp]
summary: >
  Each layer has its own attack surface and one default defense: validate shape at the
  boundary, keep values out of SQL text, encode at the output sink, refuse cross-site
  writes, keep secrets out of code and logs, and give every process the least privilege
  that works.
examples:
  - id: order-by-injection
    file: examples/01_order_by_injection.go
    title: "SQL: the part parameters can't cover, and the error message that leaks a secret"
    runtime: go
    check: exact
    annotations:
      - lines: "14-19"
        note: >
          The value is a parameter, so the classic quote-breaking attack is gone. ORDER
          BY takes an identifier or an expression, not a value, so the sort column is
          concatenated: the injection point that survives in code written by people who
          know about $1.
      - lines: "23-32"
        note: >
          The fix for anything that can't be a parameter: an allow-list mapping public
          keys to SQL the program wrote. The user's string never reaches the query, and
          an unknown key is a 400 before any SQL exists.
      - lines: "52"
        note: >
          A stacked query fails, but only by luck of the protocol: with arguments, pgx
          uses the extended query protocol, which accepts one statement per query. pgx's
          Exec without arguments switches to the simple protocol, where both would run.
      - lines: "53"
        note: >
          No second statement needed. A subquery in ORDER BY casts the secret to int,
          and PostgreSQL's error message quotes the value it failed to convert.
      - lines: "55-56"
        note: >
          The output shows the secret inside an error. An API that returns err.Error() to
          the client has just published it; map errors to generic messages in one place
          and keep details in the server log.
  - id: xss-sinks
    file: examples/02_xss_sinks.tsx
    title: "XSS: the same stored string through five output sinks"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "4-6"
        note: >
          Both values would pass any reasonable input validation. XSS is not an input
          problem: the same string is harmless as text and dangerous as markup, so the
          defense belongs where it is output.
      - lines: "8-14"
        note: >
          innerHTML parsed the comment and created an img element with an onerror handler.
          The example leaves out src so nothing loads; with any src, the handler would run
          as soon as the image failed, no click needed. textContent created no element.
      - lines: "16-17"
        note: >
          React's default is the safe one: text children and attribute values are
          escaped, so the payload renders as visible text.
      - lines: "18-19"
        note: >
          Escaping is correct and not enough: a javascript: URL contains nothing to
          escape, and React 18 (this app's version) renders the href as given. A click
          runs it.
      - lines: "20-21"
        note: >
          dangerouslySetInnerHTML is innerHTML. If you must render user HTML, sanitize it
          with a maintained sanitizer first, and treat that as a dependency to keep
          patched.
      - lines: "23-34"
        note: >
          URLs need an allow-list of schemes, checked with the URL parser rather than a
          prefix test: the parser lower-cases the scheme, strips leading spaces and
          encodes the Turkish host ışıl.dev as punycode.
  - id: localhost-csrf
    file: examples/03_localhost_csrf.go
    title: "CSRF: a JSON endpoint on 127.0.0.1 is reachable from any page in the browser"
    runtime: go-server
    check: exact
    annotations:
      - lines: "13-22"
        note: >
          A run endpoint with no cookies and no authentication. Its only credential is
          that it listens on 127.0.0.1, which keeps other machines out but not other
          origins: the browser on this machine can reach it.
      - lines: "24-33"
        note: >
          A cross-site page can send POST with a text/plain body without a CORS preflight,
          but not with application/json. Requiring JSON turns the simple request into one
          the browser must preflight, and the preflight fails without CORS headers.
      - lines: "42"
        note: >
          http.CrossOriginProtection (Go 1.25) rejects non-safe requests that the browser
          marks as cross-site with Sec-Fetch-Site, or whose Origin host differs from Host.
          Requests without either header (curl, other servers) pass.
      - lines: "44-54"
        note: >
          What a page on evil.example can send with fetch and mode: "no-cors": the
          response is opaque to the page, but the server has already acted. Sec- headers
          are set by the browser and cannot be set by page scripts.
      - lines: "55-76"
        note: >
          The undefended endpoint ran the cross-site request (200). Each defense blocks it
          (415, 403) while the app's own page and non-browser clients keep working.
use_cases:
  - "Boundary validation (zod in TS, a strict decoder plus checks in Go): reject malformed input early, with a useful 400"
  - "Parameters for every value in SQL; an allow-list for identifiers, sort keys and directions; generic error messages to clients"
  - "Output encoding by sink: React text and html/template by default, a URL scheme allow-list for href and src, a sanitizer for user HTML"
  - "CSRF protection on every state-changing endpoint a browser can reach, cookie-authenticated or merely local"
  - "Secrets from the environment or a secret store, typed so loggers redact them; least-privilege database roles per process"
strengths:
  - "pgx and database/sql make the safe form the easy form: values go as $n arguments and travel separately from the SQL text"
  - "React and Go's html/template escape by default, so XSS requires an explicit opt-out (dangerouslySetInnerHTML, template.HTML) that review can grep for"
  - "Go 1.25's http.CrossOriginProtection gives token-free CSRF protection in one wrapper, built on headers browsers set and scripts cannot forge"
  - "PostgreSQL roles, GRANTs and row-level security enforce least privilege below the application, where a bug in the app cannot switch them off"
weaknesses:
  - "No mechanism covers identifiers: table, column and sort names need allow-lists written by hand"
  - "Escaping is context-specific; HTML-escaping does nothing for a javascript: URL, a CSS value or a string inside a script"
  - "Header-based CSRF defenses assume a browser; non-browser clients need real authentication, and DNS rebinding needs a Host check"
  - "Least privilege costs setup: separate roles for migrations and the app, explicit GRANTs, and default privileges for tables created later"
pitfalls:
  - title: "A masking function that masks the wrong half"
    detail: "StackLab's maskDSN in cmd/stacklab/main.go used to cut the DSN at '@' and keep the part before it, which is postgres://user:password. The host was hidden and the password logged. A keyword-style DSN (password=...) has no '@' and was logged whole. net/url's Redacted (Go 1.15) replaces the password and keeps the rest; maskDSN uses it now."
    bad: |
      return dsn[:i] + "@..." // "postgres://stacklab:s3cret@..."
    good: |
      u, err := url.Parse(dsn)
      if err != nil {
          return "(unparsable DSN)"
      }
      return u.Redacted() // "postgres://stacklab:xxxxx@127.0.0.1:5432/stacklab"
  - title: "Validation as an XSS defense"
    detail: "Rejecting < and > on input breaks legitimate text (a lesson about generics) and still misses javascript: URLs and attribute contexts. Validate shape on input; encode for the sink on output."
  - title: "Returning database errors to clients"
    detail: "PostgreSQL error messages quote values, constraint and table names. Log them; send the client a stable message and a status from one mapping function, as StackLab's problem helper does."
  - title: "One all-powerful database role"
    detail: "An app connecting as a superuser or the table owner turns any injection into DROP TABLE or COPY ... TO PROGRAM. Migrate with an owner role; run the app with a role that has only the GRANTs it needs (sql.security-ops.roles)."
quiz:
  - q: "Why can't the sort column be passed as $2?"
    options: ["pgx doesn't support two parameters", "Parameters stand for values; ORDER BY $2 would sort by a constant, not by the named column", "PostgreSQL forbids parameters after WHERE", "It can; the example is just old-fashioned"]
    answer: 1
    why: "A parameter is a value. Identifiers and keywords are part of the query's structure, so they must come from code, chosen through an allow-list."
  - q: "React rendered <a href={website}> with website = \"javascript:alert(1)\" unchanged. Why didn't escaping help?"
    options: ["React doesn't escape attributes", "The string contains no characters that need escaping; the danger is the URL's scheme", "renderToStaticMarkup skips escaping", "Escaping only applies to children"]
    answer: 1
    why: "Escaping makes the value parse as an attribute value, which it already does. Whether a URL is safe to follow is a separate rule: allow-list its scheme."
  - q: "A backend with no CSRF defense listens on 127.0.0.1 and uses no cookies. Can a website the owner visits make it run code?"
    options: ["No: 127.0.0.1 is unreachable from the internet", "No: without cookies there is no CSRF", "Yes: the browser on this machine can send a simple cross-site POST to it", "Only over HTTPS"]
    answer: 2
    why: "The browser is on the same machine. A text/plain POST needs no preflight, the server acts before the browser hides the response, and local reachability is the only credential."
  - q: "Which header does http.CrossOriginProtection check first, and why can't page JavaScript fake it?"
    options: ["Referer; it is signed", "Sec-Fetch-Site; Sec- headers are forbidden request headers that only the browser sets", "X-Requested-With; CORS blocks it", "Cookie; it is HttpOnly"]
    answer: 1
    why: "Browsers attach Sec-Fetch-Site to every request and scripts may not set Sec- headers. Without it, the Origin header is compared with Host."
further_reading:
  - title: "net/http: CrossOriginProtection"
    url: "https://pkg.go.dev/net/http#CrossOriginProtection"
  - title: "net/url: URL.Redacted"
    url: "https://pkg.go.dev/net/url#URL.Redacted"
  - title: "MDN: Cross-site request forgery (CSRF)"
    url: "https://developer.mozilla.org/en-US/docs/Web/Security/Attacks/CSRF"
  - title: "MDN: Cross-site scripting (XSS)"
    url: "https://developer.mozilla.org/en-US/docs/Web/Security/Attacks/XSS"
  - title: "React: dangerouslySetInnerHTML"
    url: "https://react.dev/reference/react-dom/components/common#dangerously-setting-the-inner-html"
  - title: "PostgreSQL: Message flow, extended query (one statement per Parse)"
    url: "https://www.postgresql.org/docs/current/protocol-flow.html"
---

## Concept

Security is easier to reason about as a list of surfaces, each with one default defense
that should be on unless you can say why not. The OWASP Top 10 is the industry's version
of this list; here it is by layer, for this stack.

- **Input (every boundary).** Validate shape early: types, lengths, formats, enums. zod
  at the TS trust boundary (ts.services.zod), a strict JSON decoder and explicit checks
  in Go (go.http.json-validation). Validation is about correctness and good errors; it is
  **not** the defense against injection or XSS, because safe input depends on where it
  ends up.
- **SQL.** Every value is a parameter (`$1`), never part of the SQL text
  (go.postgres.transactions shows the classic attack). What can't be a parameter
  (table and column names, sort keys and directions) comes from an allow-list in code.
  Error details go to the log, not the client.
- **Output (XSS).** Encode for the sink. React text and `html/template`
  (go.cli.templates) escape by default; `href` and `src` need a scheme allow-list;
  `innerHTML` and `dangerouslySetInnerHTML` need a sanitizer or nothing at all.
- **Requests (CSRF).** A browser will send requests to any server it can reach, with
  whatever ambient credentials apply: cookies, or simply being on the same machine.
  Refuse cross-site state changes: `http.CrossOriginProtection` (Go 1.25), `SameSite`
  cookies (go.http.cors-cookies-httptest), and JSON-only bodies.
- **Secrets.** Out of the repository (only `.env.example` is committed here), out of
  logs (typed secrets with `slog.LogValuer`, as go.cli.slog shows; `url.URL.Redacted`
  for DSNs), out of error messages, and out of the frontend bundle entirely.
- **Privilege.** Every process and database role gets the least that works: an app role
  without DDL rights, a migration role that owns the schema, row-level security for
  tenant isolation (sql.security-ops.roles, sql.security-ops.rls), services bound to
  127.0.0.1 when nothing else needs them.

## Coming from C/C++/Java/Python/JS

- **Java**: `PreparedStatement` is the same idea as `$1`; JPA's Criteria API exists
  partly to avoid string-built JPQL. Spring Security's CSRF tokens are the synchronizer
  token pattern; Go's `CrossOriginProtection` reaches the same goal with Fetch metadata
  headers and no token.
- **Python**: DB-API placeholders (`%s` in psycopg) are parameters, unlike `%`
  formatting of the SQL string, which looks almost identical. Django and Jinja2
  autoescape HTML, with the same blind spot for `javascript:` URLs.
- **JS/Node**: template literals make string-built SQL tempting; use the driver's
  placeholders. On the frontend, React's escaping is the default you get for free.
- **C/C++**: buffer overflows are mostly gone in Go and TS; injection is the memory
  corruption of web code: data crossing into a context where it is interpreted as code.

## How it works

PostgreSQL has two ways to send a query. The simple protocol takes a string that may
hold several statements; the extended protocol parses one statement, then binds
parameter values separately, as data. pgx uses the extended protocol whenever there are
arguments, which is why `title; DROP TABLE api_keys` failed with "cannot insert multiple
commands into a prepared statement". That is a side effect, not a defense: the subquery
in ORDER BY needed no second statement, and the error message did the rest.

A "simple" cross-origin request is a GET, HEAD or POST whose headers and content type
(`text/plain`, `application/x-www-form-urlencoded`, `multipart/form-data`) a plain HTML
form could have produced. Browsers send it without asking the server first; CORS only
decides whether the page may *read* the response. Since 2023, every major browser also
attaches `Sec-Fetch-Site` (`same-origin`, `same-site`, `cross-site` or `none`), which
page scripts cannot set, and `CrossOriginProtection` rejects unsafe methods when it says
`cross-site`, falling back to comparing `Origin` with `Host`.

## Design notes

StackLab gets several defaults right. Every SQL value is a parameter; lesson SQL runs as
`playground`, created `NOSUPERUSER NOCREATEDB NOCREATEROLE` precisely so `COPY ... TO
PROGRAM` is impossible (ADR-0008); errors leave through one `problem` helper and the run
handler's 502 says "details are in the log"; request bodies are capped with
`MaxBytesReader`; PostgreSQL is told to listen on `localhost` only; and only
`.env.example` is in the repository.

Reviewing StackLab for this lesson turned up three problems, in order of severity. All
three are fixed now; each is still worth knowing, because each was easy to miss:

1. **`POST /api/run` accepted cross-site requests.** It decoded JSON without checking
   `Content-Type` or `Origin`, and the server sent no CORS headers but had no CSRF
   check either. A page on any website the owner visited could send the third example's
   `text/plain` POST to `http://127.0.0.1:8080/api/run`; in local mode the code then ran
   in the backend process as the owner's Windows user. ADR-0003's threat model ("protect
   the owner from their own mistakes", void "if this app is ever exposed to a network")
   did not consider the browser as the path in. ADR-0011 closed it: `withRequestGuards`
   in `internal/httpapi/security.go` wraps the router in `http.CrossOriginProtection`,
   requires `application/json` on writes (415 otherwise) and allows only loopback `Host`
   values on `/api/` (403 otherwise). That last check stops DNS rebinding, where a
   hostile domain re-resolves to 127.0.0.1 and its requests look same-origin. Browsers
   have begun to restrict this: under Local Network Access, Chrome 142 and Edge 143 ask
   the user before a public page's `fetch()`, subresource or iframe request reaches a
   local or loopback address ([Chrome](https://developer.chrome.com/blog/local-network-access),
   [Edge](https://learn.microsoft.com/en-us/deployedge/ms-edge-local-network-access)).
   Microsoft's page says Edge has no plans to restrict main-frame navigation, which is
   what a plain form submission is, and the user can click Allow. It is the browser's
   policy, not your defense.
2. **`maskDSN` logged the password** (pitfall above). Low impact with the local
   `stacklab:stacklab` default, but it is the habit that matters once a real password is
   configured. It now parses the URL and logs `Redacted()`, masks `password=` in
   keyword-style DSNs, and logs a placeholder for anything it can't parse.
3. **Database privileges were wider than needed.** PostgreSQL grants `CONNECT` and
   `TEMPORARY` on every new database to `PUBLIC`, so `playground`, and with it every
   lesson, could connect to the app's database (not read its tables, which belong to
   `stacklab`). `setup-local-db.sql` now runs `REVOKE ALL ON DATABASE stacklab FROM
   PUBLIC`, and a connection attempt as `playground` fails with "permission denied for
   database". The app itself connects as `stacklab`, the owner of its schema, so app and
   migrations share one role; a separate migration role is the next step if the app ever
   leaves the owner's machine.
