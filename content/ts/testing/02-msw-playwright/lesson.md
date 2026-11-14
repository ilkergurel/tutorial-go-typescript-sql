---
id: ts.testing.msw-playwright
title: "MSW for the API; Playwright basics; type tests with expectTypeOf"
track: ts
module: testing
order: 2
level: advanced
minutes: 15
since: ts4.1
prerequisites: [ts.testing.vitest, ts.services.fetch]
compare_with: [go.testing.fakes-httptest, ts.services.zod, ts.type-level.utility]
tags: [testing, msw, playwright, e2e, expectTypeOf, type-tests, mocking]
summary: >
  Mock the network, not fetch: MSW answers real requests from production code.
  Playwright drives a real browser and can answer its requests too, so end-to-end
  tests need no backend. expectTypeOf tests the types themselves, which no
  runtime test can see.
examples:
  - id: msw
    file: examples/01_msw.test.tsx
    title: "MSW: production fetch code, mocked at the network boundary"
    runtime: ts-test
    check: contains
    annotations:
      - lines: "10-15"
        note: >
          This is production code that calls the real global fetch. There is
          no injected client and no vi.mock of a module, so the test exercises
          the URL building, encodeURIComponent and the status check exactly as
          shipped. res.json() returns any; real code validates it (lesson
          ts.services.zod).
      - lines: "34-44"
        note: >
          setupServer describes the network: a method and a URL pattern map
          to a resolver that receives a standard Fetch API Request and returns
          an HttpResponse. searchParams.get decodes the query, so "Güneydoğu
          Anadolu" arrives with its ğ and ü intact, as a real server would
          see it.
      - lines: "45"
        note: >
          onUnhandledFrame is MSW 3's name (MSW 2 called it
          onUnhandledRequest). The default "warn" lets an unmocked request
          through to the real network. "error" stops it, so a missing mock
          fails the test instead of calling a live service.
      - lines: "46-50"
        note: >
          resetHandlers removes what server.use added, so one test's error
          scenario can't leak into the next. close() stops the interception
          after the last test. cleanup is the Testing Library unmount from
          the previous lesson.
      - lines: "53-60"
        note: >
          findAllByRole waits for the effect, the intercepted fetch and the
          re-render. The test never mentions MSW: it renders and reads, like a
          user.
      - lines: "62-70"
        note: >
          server.use prepends a handler for this test only, and the newest
          matching handler wins. The 503 path is tested without touching the
          component or the client.
      - lines: "72-74"
        note: >
          With "error", an unmocked request rejects with "fetch failed", and
          MSW prints the method and URL on stderr. Without that option, this
          request would have gone out to the internet.
  - id: browser
    file: examples/02_browser.spec.ts
    title: "Playwright: a real browser, with page.route as the backend"
    runtime: ts-test
    check: contains
    timeout_s: 15
    annotations:
      - lines: "3-19"
        note: >
          The app under test is a string of HTML with an inline script,
          because this runner has no app server. In a project,
          playwright.config's webServer starts the dev server and tests call
          page.goto("/").
      - lines: "21-26"
        note: >
          page.route intercepts requests from the browser itself, including
          the navigation to https://app.test/, where nothing is listening.
          route.fulfill answers with any status, headers and body.
      - lines: "28-33"
        note: >
          The browser-side counterpart of an MSW handler: read the request,
          answer with JSON (`json:` also sets the Content-Type). When several
          routes match, the most recently registered one wins.
      - lines: "34-36"
        note: >
          Locators are lazy queries by role and label, the same vocabulary as
          Testing Library. press("Enter") submits the form through the
          browser's own implicit submission, in real Edge, not a simulation.
      - lines: "37-38"
        note: >
          Web-first assertions retry until they pass or time out (5 s by
          default), so no findBy or sleep is needed. toHaveText with an array
          checks every list item in order. toBeHidden asks the browser's real
          rendering, something jsdom can't compute.
      - lines: "41-49"
        note: >
          The failure path: one route answers 503, and the test checks what
          the user sees, an alert and an empty list.
  - id: types
    file: examples/03_types.test-d.ts
    title: "Type tests with expectTypeOf in a *.test-d.ts file"
    runtime: ts-test
    check: contains
    annotations:
      - lines: "6-10"
        note: >
          The kind of type worth testing: the path picks the response type
          through an indexed access. A careless refactor turns it into
          Promise<any> and every caller still compiles. `declare function` is
          enough, because a .test-d.ts file is only type-checked and never
          executed.
      - lines: "19-27"
        note: >
          expectTypeOf(x) captures x's static type. .resolves unwraps a
          Promise, .parameter(0) reads a parameter, and toEqualTypeOf demands
          identical types. The @ts-expect-error line is a negative test: that
          call must not compile, and if it ever does, tsc reports an unused
          directive (TS2578) and the test fails.
      - lines: "29-34"
        note: >
          Narrowing pinned down: on the union, data doesn't exist; after
          `if (r.ok)` it is a City. If someone loosens Result so that data is
          always present, this test fails.
      - lines: "43-47"
        note: >
          toExtend checks assignability, the structural rule (extra properties
          allowed); toEqualTypeOf checks identity. expect-type 1.2 deprecated
          the older toMatchTypeOf in favor of toExtend and toMatchObjectType.
use_cases:
  - "Component and client tests against a mocked API: success, empty, 4xx/5xx, slow responses, malformed JSON"
  - "End-to-end tests of critical flows (login, checkout, a form wizard) in a real browser, with page.route for unreliable or paid third-party APIs"
  - "Type tests for library-style code: generic helpers, typed API clients, discriminated unions and mapped types that callers depend on"
  - "Sharing one set of MSW handlers between tests and local development (setupWorker in the browser) while the backend is unfinished"
strengths:
  - "MSW leaves the client untouched: fetch, URL building, headers and JSON parsing run as in production, unlike vi.mock or a spy on fetch that replaces exactly the code you wanted to test"
  - "No port and no injected base URL: compared with Go's httptest.NewServer, which needs the client to accept srv.URL, MSW intercepts in process"
  - "Playwright runs real browser engines with auto-waiting locators and assertions, so the flakiness of manual waits mostly disappears"
  - "Type tests catch regressions no runtime test can, such as a generic quietly widening to any, and they cost nothing at run time"
weaknesses:
  - "Mocks are a second description of the API and can drift from the real server; derive them from the OpenAPI contract or keep a few tests against the real backend"
  - "Playwright is slow next to jsdom: this two-test file takes about 8 s here, mostly starting Edge, against about 1 s for the jsdom files, so keep end-to-end tests few and focused"
  - "Type-test failures are tsc errors about generic constraints and can be hard to read; Vitest's typecheck mode still prints an experimental-feature notice"
  - "MSW patches globals in the test process: every test in a file shares one set of handlers, so overrides need resetHandlers discipline, and concurrent tests need server.boundary() to keep them apart"
pitfalls:
  - title: "Mocking fetch instead of the network"
    detail: "A spy on fetch must hand-build Response objects and breaks when the client changes (another header, a switch to a different client), while the request the server would have received is never checked."
    bad: "vi.spyOn(globalThis, \"fetch\").mockResolvedValue(new Response(\"[]\"));"
    good: "server.use(http.get(`${API}/cities`, () => HttpResponse.json([])));"
  - title: "Copying an MSW 2 setup into MSW 3"
    detail: "MSW 3 renamed onUnhandledRequest to onUnhandledFrame. tsc reports the old key as an unknown property (TS2353), but at run time it is ignored: the default \"warn\" applies, and unmocked requests go out to the real network."
  - title: "Sleeping in Playwright tests"
    detail: "A fixed wait is slow when the app is fast and flaky when it is slow. Web-first assertions retry until the condition holds."
    bad: "await page.waitForTimeout(1000);\nexpect(await page.locator(\"li\").count()).toBe(1);"
    good: "await expect(page.getByRole(\"listitem\")).toHaveCount(1);"
  - title: "MSW's browser worker and page.route together"
    detail: "Requests answered by a Service Worker (MSW's setupWorker in a dev build) never reach page.route. Playwright's docs recommend serviceWorkers: \"block\" when you route requests yourself."
  - title: "Type tests that never run"
    detail: "*.test-d.ts files are checked only in typecheck mode (vitest --typecheck); a plain `vitest run` skips them, and nothing fails. This lesson's runner adds the flag. In a project, put --typecheck into the CI test command or enable typecheck in the Vitest config."
quiz:
  - q: "What does MSW replace in the code under test?"
    options:
      - "The fetch function"
      - "Nothing: it intercepts the request after the code made it, at the network layer"
      - "The React component's effect"
    answer: 1
    why: "The client code runs unchanged; MSW answers the outgoing request (in Node by intercepting the request modules, in the browser with a Service Worker)."
  - q: "A test adds `server.use(...)` for a 503 and the next test unexpectedly sees 503 too. What is missing?"
    options:
      - "server.close() after each test"
      - "server.resetHandlers() in afterEach"
      - "onUnhandledFrame: \"error\""
    answer: 1
    why: "server.use prepends handlers until they are reset; resetHandlers returns to the handlers given to setupServer."
  - q: "Why does `await expect(locator).toHaveText(\"...\")` need no explicit wait in Playwright?"
    options:
      - "Playwright waits for network idle before every assertion"
      - "Web-first assertions retry until the condition holds or the timeout passes"
      - "The page is static, so nothing changes"
    answer: 1
    why: "Assertions on locators re-query and retry (5 s by default), which is what makes them robust against timing."
  - q: "`expectTypeOf({ name: \"a\", plate: 1, extra: true })` passes which check against `City`?"
    options:
      - "toEqualTypeOf<City>() only"
      - "toExtend<City>() only"
      - "Both"
    answer: 1
    why: "The object is assignable to City (structural typing allows the extra property), but its type is not identical to City."
further_reading:
  - title: "MDN: Fetch API (Request, Response)"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API"
  - title: "MDN: Service Worker API"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Service_Worker_API"
  - title: "TypeScript handbook: Indexed access types"
    url: "https://www.typescriptlang.org/docs/handbook/2/indexed-access-types.html"
  - title: "package net/http/httptest (Go)"
    url: "https://pkg.go.dev/net/http/httptest"
---

## Concept

Three tools, three boundaries:

- **MSW** (Mock Service Worker) mocks **the network**. You describe endpoints with
  handlers (`http.get(url, resolver)`); in Node, `setupServer` intercepts outgoing
  requests and answers them from the handlers. Your `fetch` code runs unchanged. See
  the [MSW docs](https://mswjs.io/docs).
- **Playwright** drives **a real browser** (here the installed Microsoft Edge,
  headless) through locators and auto-retrying assertions. `page.route` answers the
  browser's requests, the network mock one level up. See the
  [Playwright docs](https://playwright.dev/docs/intro).
- **Type tests** check **the types**. `expectTypeOf` from Vitest turns "this returns
  `Result<City[]>`" into an assertion that tsc verifies; files named `*.test-d.ts`
  are type-checked by `vitest --typecheck` and never executed. See Vitest's
  [testing types guide](https://vitest.dev/guide/testing-types).

A sensible split: many fast jsdom tests with MSW for components and clients, a
handful of Playwright tests for the flows that must work in a real browser, and type
tests for the types other code leans on.

## Coming from C/C++/Java/Python/JS

- Go: `httptest.NewServer` runs a real server on a loopback port, and your client
  must accept `srv.URL`. MSW intercepts in process: no port, no injected URL, but a
  patched global. Go has no type tests because the compiler is one; the closest idiom
  is a compile-time assertion such as `var _ io.Reader = (*MyReader)(nil)`, which is
  what `toExtend` expresses.
- Java: MSW sits between Mockito (replace the client) and WireMock (a real HTTP
  server); Playwright fills Selenium's role, with auto-waiting built in.
- Python: `responses` or `respx` patch the HTTP library much as MSW patches Node's;
  Playwright has Python bindings with the same API.
- JS: `jest.mock("axios")` and hand-made `fetch` stubs are the pattern MSW replaced.

## How it works

```mermaid
flowchart LR
  subgraph vitest["Vitest (Node + jsdom)"]
    comp["component"] --> client["fetchCities()"] --> fetch1["fetch"]
    fetch1 -. intercepted .-> msw["MSW handlers"]
  end
  subgraph pw["Playwright test (Node)"]
    spec["spec"] -- "protocol" --> edge["Edge: page + script"]
    edge -- "fetch /api/cities" --> route["page.route handler (in the test)"]
  end
```

In Node, MSW hooks the modules that issue requests (the MSW README describes it as a
low-level interception algorithm, since Node has no Service Workers); in a browser it
registers a Service Worker. Playwright runs your test in Node and controls the browser
over a protocol; `page.route` pauses each matching request inside the browser and
lets the test continue, fulfill or abort it. For type tests, Vitest runs tsc with
`--noEmit` over the test files and maps each error to the `test()` block it falls in.
In this repo that is the `typescript` package (6.0); `npm run typecheck` uses 7.0.

## Design notes

Mock at the outermost boundary you can. The further out the mock sits, the more of
your real code each test covers: MSW keeps the client honest, and `page.route` keeps
the whole page honest. Keep handlers realistic (status codes, error bodies, Turkish
text in the payloads) and make unhandled requests fail. A mock that quietly lets a
request through, or answers everything with 200, gives you green tests that prove
nothing. Type tests earn their place on code with clever types: generic helpers, typed
clients, unions that callers narrow. If a type is simple enough that a reviewer sees
it at a glance, a type test adds nothing.
