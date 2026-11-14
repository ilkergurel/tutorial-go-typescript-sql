---
id: inside.codebase.ts-runner
title: "The TS CodeRunner strategy: ServerRunner vs BrowserTsRunner"
track: inside
module: codebase
order: 6
level: advanced
minutes: 14
since: es2022
prerequisites: [ts.architecture.patterns, ts.async.event-loop, ts.narrowing.discriminated-unions]
compare_with: [design.patterns.classic, go.interfaces.implicit, inside.codebase.runner-sandbox, inside.codebase.contract]
tags: [strategy, iframe, sandbox, esbuild, web-worker, csp, event-loop]
summary: >
  The Run button calls runnerFor(runtime).run(request) and never asks where the code
  runs. The interface is one line; the real work is making a browser iframe honor the
  same contract as a backend process, including when a run is over.
examples:
  - id: strategy
    file: examples/01_strategy.ts
    title: "One interface, two runners, one factory: the shape of CodeRunner.ts"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "6-8"
        note: >
          The whole strategy interface, as in src/features/runner/CodeRunner.ts.
          Everything that differs between runtimes hides behind one async method
          that takes the wire request and returns the wire result.
      - lines: "10-20"
        note: >
          ServerRunner's error contract: a non-2xx answer is a refusal (busy,
          unsupported runtime), so it throws with the problem body's detail. The
          real class calls the global fetch; this one takes it as a parameter
          because the lesson iframe has no network.
      - lines: "22-36"
        note: >
          A stand-in for BrowserTsRunner. The real one transpiles in a worker and
          runs the module in a sandboxed iframe; this one imports a data: URL in
          the same page, which keeps the shape and drops the isolation. Note that
          a crash resolves with exit 1: failing lesson code is a result, not an
          error.
      - lines: "48"
        note: >
          The factory is the only place that knows the mapping. In the app,
          ts-browser goes to the browser and all eight other runtimes, ts-node and
          ts-test included, go to the server.
      - lines: "56-63"
        note: >
          The caller's two branches are the ones RunPanel.tsx turns into
          { kind: 'done' } and { kind: 'failed' }. A thrown TypeError from the
          lesson would have been the wrong kind of failure; the runner turns it
          into a result.
  - id: completion
    file: examples/02_completion.ts
    title: "When is a browser run over? A fixed delay vs the pending-timer rule"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-24"
        note: >
          A virtual clock so the output is exact. The real shim (IFRAME_SHIM in
          browserTs.ts) wraps window.setTimeout and setInterval and keeps the ids
          of pending timers in a Set; pending here plays that role.
      - lines: "36-41"
        note: >
          An interval that is never cleared keeps the run alive, as it would
          keep a Node process alive. The shim counts intervals as pending until
          clearInterval.
      - lines: "46-47"
        note: >
          The real marker is a line appended after the lesson module's last
          statement, window.__stacklabMainDone(), so it runs after every
          top-level await. A throw skips it and the error handler marks the main
          part done instead.
      - lines: "49-51"
        note: >
          The rule the runner used before: finish a fixed delay after the frame
          loaded. Anything logged later is silently lost, and the expected file
          would record the truncated output as correct.
      - lines: "52-55"
        note: >
          Today's rule: done when the main part ran and no timer is pending. The
          shim then waits two more task turns: promise continuations can start
          new timers, and the unhandledrejection event for a promise rejected in
          the last task arrives a task later. This miniature has no promises, so
          it skips both. The 10 s ceiling is BrowserTsRunner's default timeout.
use_cases:
  - "Reading src/features/runner/ before adding a runtime: a new server runtime needs a case in runexec.execute and nothing in the frontend beyond the Runtime union"
  - "Debugging a ts-browser lesson that passes in Node but not in the app: different strategy, different console formatting and timeout"
  - "A model for any UI that must run work in two places (local and remote) behind one call"
strengths:
  - "RunPanel never branches on where code runs, so fetch, problem details, workers and iframes stay out of the view; unlike a switch in the component, a third strategy would touch only runnerFor"
  - "The same BrowserTsRunner class runs in the app and in verify.html, so scripts/verify-lessons.mjs checks the code the learner uses, not a copy"
  - "The iframe's opaque origin and CSP are enforced by the browser, which is a stronger boundary than any check the app could write itself"
  - "Like Go's implicit interfaces, structural typing means implements CodeRunner is an extra compile-time check, not a requirement"
weaknesses:
  - "The two strategies honor the same type, not the same behavior: the browser runner ignores fixtures, stdin and setup, uses only the first file, defaults to a 10 s timeout (server: 5 s) and reports a timeout as exit 124 (server: -1)"
  - "No output cap in the browser: truncated is always false, and a logging loop floods postMessage until the timeout fires; the server caps each stream at 64 KB"
  - "Browser runs never reach the backend, so they are missing from run history"
  - "Module-level singletons (serverRunner, browserTsRunner) are created at import; fine in a browser app, but a unit test can't swap fetch without patching globals"
pitfalls:
  - title: "Rejecting for failing lesson code"
    detail: "If a runner threw on a compile error or an uncaught exception, RunPanel would show a red 'failed' banner instead of the output and the exit code. Only machinery failures reject; everything the lesson did is a RunResult."
  - title: "Expecting ts-browser and ts-node to print the same thing"
    detail: "The browser shim formats values itself: strings as-is, Map, Set and Error readably, everything else as JSON. Node uses util.inspect. console.log({ a: 1 }) prints {\"a\":1} in one and { a: 1 } in the other, so an expected file recorded under one runtime fails under the other."
  - title: "Hiding the iframe with display: none"
    detail: "The runner keeps the frame rendered, 1x1 and transparent, because Edge throttles a hidden cross-origin frame: requestAnimationFrame stops (Microsoft documents this), and in Edge 157 its timers woke up only about once a second. Async lessons would get slow and their timing claims wrong."
quiz:
  - q: "A ts-browser example has a syntax error. What does BrowserTsRunner.run do?"
    options: ["Rejects, so RunPanel shows the 'failed' banner", "Resolves with exitCode 1 and the esbuild message as a diagnostic", "Resolves with exitCode 0 and empty output"]
    answer: 1
    why: "transpile() rejects inside run(), and run() catches it and returns a RunResult with exit 1 and one diagnostic. Only machinery failures reject."
  - q: "Why is the lesson iframe sandboxed with allow-scripts but without allow-same-origin?"
    options: ["So scripts run faster", "So the lesson runs in an opaque origin and can't touch the app's DOM, storage or cookies", "So the iframe can fetch the lesson libraries itself"]
    answer: 1
    why: "Without allow-same-origin the frame's origin is opaque ('null'). That is also why the libraries arrive as data: URLs: the frame couldn't fetch them from the app's origin."
  - q: "A ts-browser lesson starts a setInterval and never clears it. What does the learner see?"
    options: ["The run ends when the module's last statement finishes", "The run ends at the 10 s timeout with exit 124 and a timed-out note", "The run never ends"]
    answer: 1
    why: "A pending interval keeps the run alive, as in Node. The runner's own timer then fires, removes the iframe and reports a timeout."
  - q: "Which runner executes a ts-node lesson?"
    options: ["BrowserTsRunner, because it is TypeScript", "ServerRunner: runnerFor sends everything except ts-browser to POST /api/run"]
    answer: 1
    why: "runnerFor only checks for 'ts-browser'. ts-node runs in Node on the backend side (gorun.RunNode), with Node's own console formatting."
further_reading:
  - title: "MDN: <iframe> (sandbox, srcdoc)"
    url: "https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Elements/iframe"
  - title: "MDN: Window.postMessage"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Window/postMessage"
  - title: "MDN: setTimeout, reasons for longer delays than specified"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Window/setTimeout"
  - title: "MDN: Using Web Workers"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Web_Workers_API/Using_web_workers"
  - title: "TypeScript Handbook: implements clauses"
    url: "https://www.typescriptlang.org/docs/handbook/2/classes.html#implements-clauses"
---

## Concept

`src/features/runner/CodeRunner.ts` is the strategy pattern in about 40 lines: an
interface with one method, two implementations and a factory.

```ts
export interface CodeRunner {
  run(req: RunRequest): Promise<RunResult>;
}
export function runnerFor(runtime: RunRequest['runtime']): CodeRunner {
  return runtime === 'ts-browser' ? browserTsRunner : serverRunner;
}
```

`RunPanel.tsx` builds a `RunRequest` from the editor and calls
`runnerFor(example.runtime).run(...)`. It still looks at the runtime for the editor
language, the stdin box and SQL tables, but never to decide where code runs.

- **`ServerRunner`** posts the request to `/api/run`. Vite proxies that to the Go
  backend, which runs the code in-process in local mode (`internal/runexec`, ADR-0008)
  or forwards it to the runner container in Docker mode. Eight runtimes take this path:
  `go`, `go-test`, `go-server`, `ts-node`, `ts-test`, `sql`, `sql-nontx` and
  `sql-multisession`.
- **`BrowserTsRunner`** (`browserTs.ts`) handles `ts-browser`: it transpiles in a Web
  Worker and runs the result in a sandboxed iframe. Nothing reaches the backend.

The interface is the easy part. The contract behind it is the hard part: **resolve with a
`RunResult` for anything the lesson code did** (compile errors, crashes, timeouts) and
**reject only when the machinery refused or broke**. RunPanel relies on that split: a
resolved result becomes `{ kind: 'done' }` with the output, a rejection becomes
`{ kind: 'failed' }` with a banner. The server side gets most of the contract from the
operating system, which supplies exit codes, stdout and stderr, and the end of a process.
The browser runner has to build each of these itself.

## Coming from C/C++/Java/Python/JS

- **Java**: the same shape as a `Comparator` or a JDBC `Driver`: an interface, classes,
  a factory. The difference is structural typing: `implements CodeRunner` asks the
  compiler to check the class, but any object with a matching `run` is a `CodeRunner`.
- **Go**: the backend repeats the pattern one layer down. `httpapi.Runner` is declared by
  its consumer in `httpapi/run.go`; `runexec.Executor` (in-process) and
  `runclient.Client` (HTTP to the container) satisfy it without naming it, and `main.go`
  picks one from `STACKLAB_RUNNER_URL`. The frontend strategy chooses *where* (browser or
  server); the backend one chooses *how* (in this process or in a container).
- **C++**: a pure virtual base class, or `std::function` when there's one method. TS has
  both options too; ts.architecture.patterns argues that the stateless `ServerRunner`
  could just as well be a function.
- **Python/JS**: a `Protocol` or plain duck typing. The type adds a compile-time check,
  nothing at run time.

## How it works

What `BrowserTsRunner.run` does for one ts-browser example:

1. **Transpile off the main thread.** `esbuild.worker.ts` loads esbuild-wasm once (about
   14 MB of WebAssembly) and answers `{ id, code, loader }` messages with
   `{ id, ok, js | error }`. The id pairs answers with requests on the shared worker. The
   output is an ES module (`format: 'esm'`, `target: 'es2022'`), because lessons use
   top-level `await`. Types are stripped, not checked.
2. **Resolve libraries.** If the module imports `react`, `zod` or another library from
   ADR-0010, the runner fetches `/lesson-libs/manifest.json` and `bundle.js` once and
   builds an import map of `data:` URLs. The iframe can't fetch them itself (next step).
3. **Build the frame.** `<iframe sandbox="allow-scripts">` with a `srcdoc` holding a CSP
   `<meta>` (`default-src 'none'`, scripts only inline or `data:`), the import map, the
   shim and the module, followed by `window.__stacklabMainDone()`. Without
   `allow-same-origin` the frame's origin is opaque: no access to the app's DOM,
   `localStorage` or cookies, and with the CSP no network at all.
4. **Keep it running at full speed.** The frame is 1x1 px, transparent and fixed in the
   corner rather than `display: none`. The comment in `browserTs.ts` records why: a hidden
   cross-origin frame (the sandbox's opaque origin makes it one) is throttled. Microsoft
   documents that Edge throttles cross-origin frames that are `display: none` or outside
   the viewport, and that such frames "lose access to `requestAnimationFrame` and
   `ResizeObserver`" ([iframe throttling](https://learn.microsoft.com/en-us/deployedge/edge-learnmore-iframe-throttling-display)).
   Timers aren't on that page; a run in Edge 157.0.4341.46 (headless and headed,
   2026-11-14) showed them too. In a `display: none` sandboxed frame, a 10 ms
   `setInterval` fired a few times in the first 80 ms, then once a second, and
   `requestAnimationFrame` stopped after three callbacks; the 1x1 frame got about 300
   ticks and 180 frames in the same 3 s.
5. **Collect output.** The shim replaces `console.*` and posts every call to the parent;
   `log`/`info`/`debug` become stdout, `warn`/`error` stderr, an uncaught error or
   unhandled rejection stderr plus exit 1. One exception: React 18's development build
   replays a failed render inside a `react-invokeguardedcallback` event so that devtools
   see it, which also reaches the window's error handler even when an error boundary
   catches it. The shim ignores errors reported during that event; React rethrows the
   ones no boundary caught, and those count. The parent drops any message whose
   `e.source` isn't its own frame, and the `switch` over the message type ends in a
   `never` check.
6. **Decide when it's over** (example 2): when the main part has run and no timer is
   pending, or at the timeout (`timeoutS`, default 10 s). An unhandled rejection doesn't
   end the main part; the module keeps running. Then the frame is removed, which also
   kills an infinite loop inside it, and the result is resolved.

`ServerRunner` is ten lines by comparison: `fetch`, a problem-details error path, and
`(await res.json()) as RunResult`. That cast trusts the backend completely; the contract
lesson (inside.codebase.contract) shows where that trust is misplaced.

## Design notes

The pattern earns its keep in two places. The run logic holds no list of server
runtimes, so the backend went from three runtimes to eight without touching it. And
because the strategy is a plain object that doesn't
depend on React, `src/verify/browserVerify.ts` can expose the real
`BrowserTsRunner` to headless Edge, so `npm run verify:lessons` tests what the learner
runs.

The strategies share a type, not a behavior, and the differences are worth knowing
before you debug a lesson:

- The browser runner reads only the first file and ignores `fixtures`, `stdin` and
  `setup`. Its default timeout is 10 s (the server's run timeout is 5 s), and it reports
  a timeout as exit 124 where gorun reports -1. Both fail the "matches expected" check.
- It has no output cap: `truncated` is always `false`.
- Its `console.log` formatting is the shim's `fmt` (strings as-is, `Map`/`Set`/`Error`
  readable, everything else as JSON). `ts-node` lessons get Node's `util.inspect`, so the
  same line can print differently under the two runtimes.
- Browser runs are never recorded in run history; only `handleRun` on the backend writes
  history.

Comments drift like this one did: until 2026-11-14 the doc comment at the top of
`CodeRunner.ts` said `ServerRunner` "proxies go/sql/ts-node through the backend to the
runner container (ADR-0003)", which predated ADR-0008 and named three runtimes of eight.
It now says what happens: the backend runs the code itself in local mode and forwards it
to the runner service in Docker mode.

Classes are justified for `BrowserTsRunner`: it owns a worker, a request counter and a
lazily built import map, and the class keeps that state in one place. `ServerRunner` has
no state. If you ever need to test `RunPanel` without a backend, the module-level
singletons are the obstacle: pass the runner in (a prop or a context) rather than
patching the global `fetch`.

Two alternatives were possible here. A `switch (runtime)` inside RunPanel would have
spread transport details through the view. Sending ts-browser to the server as well
would have needed a browser on the backend, which is exactly what verify-lessons starts
for itself, and a far heavier way to run a learner's three lines of TypeScript.
