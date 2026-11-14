---
id: ts.tooling.vite
title: "Vite: dev server, proxy, env variables, build, source maps, DevTools debugging"
track: ts
module: tooling
order: 2
level: intermediate
minutes: 15
since: es2020
prerequisites: [ts.mental-model.erasable, ts.services.zod]
compare_with: [go.files-io.embed, go.performance.cross-compile, ts.mental-model.erasable, ts.tooling.bundle]
tags: [vite, dev-server, proxy, env, build, source-maps, devtools]
summary: >
  Vite serves source files to the browser as native ES modules in development and bundles
  them for production. Neither mode type-checks. Its proxy keeps the API same-origin, its
  env variables are build-time strings, and source maps take DevTools back to your TypeScript.
examples:
  - id: env
    file: examples/01_env.ts
    title: "import.meta.env: strings at build time, parsed once at startup"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "4-6"
        note: >
          import.meta is part of JavaScript (import.meta.url works in any
          module); env is Vite's addition. Code that runs outside Vite (this
          runner, a Node script, a Vitest test without Vite's config) sees
          undefined.
      - lines: "8-21"
        note: >
          The values are what `vite build` substituted for this .env in the
          lesson's scratch build. vite/client declares ImportMetaEnv as
          Record<string, any> plus the built-in keys, so the compiler accepts
          any VITE_ name and any use of it.
      - lines: "23-24"
        note: >
          "false" is a non-empty string, so it is truthy, and "20" + 1 is "201".
          In the real build both expressions were constant-folded to exactly
          these wrong answers.
      - lines: "26-32"
        note: >
          The schema states what each variable means and converts it once.
          z.infer gives the rest of the app a real type (boolean, number)
          instead of any.
      - lines: "37-43"
        note: >
          A bad value fails at startup with a precise message, instead of NaN
          travelling into a request ten screens later.
  - id: source-map
    file: examples/02_source_map.ts
    title: "Reading a real source map: minified column to TypeScript line"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-14"
        note: >
          Real esbuild output and map. Minified, the whole program is one line
          and the parameter names are e, n and r; sourcesContent embeds the
          original TypeScript, so DevTools can show it without fetching the
          .ts file.
      - lines: "16-32"
        note: >
          Each mapping number is base64 VLQ: variable-length, 5 bits per
          character with a continuation bit, sign in the lowest bit. Small
          deltas take one or two characters, which keeps maps compact.
      - lines: "34-47"
        note: >
          Every field is relative to the previous segment, so the decoder keeps
          running totals. The optional fifth field indexes names: how the map
          remembers that t was totalMinutes.
      - lines: "49-57"
        note: >
          A stack frame gives a generated line and column; the last segment at
          or before that column names the original position. DevTools does this
          for every frame, breakpoint and console location.
use_cases:
  - "Running the frontend against a local backend without CORS: proxy /api in server.proxy, as this app does"
  - "Per-environment configuration that is safe to publish: API base URLs, feature flags, build labels"
  - "Debugging production errors: build hidden source maps, keep them out of the deploy, feed them to the error tracker"
  - "Setting breakpoints in TypeScript in the browser's DevTools while the dev server runs"
strengths:
  - "Unlike webpack-style dev servers that bundle before serving, Vite transforms only the modules the page requests, so dev startup stays roughly flat as the app grows"
  - "Env replacement at build time lets the minifier delete whole branches: the DEV-only code in the scratch build never reached the bundle"
  - "The proxy makes development same-origin, so the backend needs no CORS configuration it would never use in production"
  - "Source maps are a standard format, so the same map works in Chrome, Edge and Firefox DevTools and in error trackers"
weaknesses:
  - "No type-checking anywhere in Vite: `vite build` ships code with type errors unless tsc runs separately"
  - "Env values are strings typed as any by default; nothing stops `VITE_DEBUG ? ... : ...` from always being true"
  - "Dev and build use different pipelines (per-module transforms vs a Rollup bundle in Vite 7), so a bug can appear in only one of them"
  - "server.proxy exists only in the dev server; production needs the same routing in a real reverse proxy or the backend"
pitfalls:
  - title: "Secrets in VITE_ variables"
    detail: "Everything Vite exposes is compiled into JavaScript that every visitor downloads. Keep keys on the server and call the server."
  - title: "A proxy key that matches more than you meant"
    detail: "Keys are prefix matches on the raw path, so '/api' also proxies '/apiary'. In this app a request to /apiary returns the Go backend's '404 page not found' instead of Vite's index.html fallback. End the key with a slash or use a ^ regular expression."
    bad: "proxy: { '/api': apiTarget }"
    good: "proxy: { '/api/': apiTarget }"
  - title: "Testing a flag with truthiness"
    detail: "VITE_FEATURE=false is the string \"false\". Compare with === 'true' or parse the whole env once."
    bad: "if (import.meta.env.VITE_DEBUG) enableDebug();"
    good: "if (config.VITE_DEBUG) enableDebug(); // parsed to boolean at startup"
  - title: "Trusting `vite build` to catch type errors"
    detail: "It strips types without checking them (lesson ts.mental-model.erasable). This repo's build script is plain `vite build`, so run `npm run typecheck` before it in CI."
quiz:
  - q: "`.env` contains `VITE_DEBUG=false`. What does `import.meta.env.VITE_DEBUG ? \"on\" : \"off\"` produce in a production build?"
    options: ["\"off\"", "\"on\"", "A build error"]
    answer: 1
    why: "The value is the string \"false\", which is truthy. Vite substituted it at build time and the minifier folded the expression to \"on\"."
  - q: "vite.config.ts has `server.proxy: { '/api': 'http://127.0.0.1:8080' }`. Which request does the dev server forward?"
    options: ["Only /api", "/api/progress but not /apiary", "/api/progress and /apiary"]
    answer: 2
    why: "Non-regex keys are matched with startsWith on the request path. '/api/' or '^/api(/|$)' limits it to the API."
  - q: "You set `build.sourcemap: 'hidden'`. What changes compared to `true`?"
    options: ["No map files are written", "Map files are written, but the bundles don't reference them, so browsers won't load them", "The map is inlined into each bundle"]
    answer: 1
    why: "'hidden' writes the maps without the sourceMappingURL comment: upload them to an error tracker and keep them off the public server."
  - q: "`.env` sets `VITE_API_URL=/api` and `.env.production` sets `VITE_API_URL=https://api.example.com`. Which value does `vite build` (mode production) compile in?"
    options: ["/api, because .env is always loaded", "https://api.example.com, because mode-specific files win", "Neither: builds read only .env.local"]
    answer: 1
    why: "Vite loads .env, .env.local, .env.production and .env.production.local; variables in mode-specific files take precedence over the generic ones."
further_reading:
  - title: "MDN: import.meta"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/import.meta"
  - title: "MDN: import() (dynamic import)"
    url: "https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Operators/import"
  - title: "MDN Glossary: Source map"
    url: "https://developer.mozilla.org/en-US/docs/Glossary/Source_map"
  - title: "TSConfig reference: isolatedModules"
    url: "https://www.typescriptlang.org/tsconfig/#isolatedModules"
---

## Concept

Vite (this repo: 7.3.7) has two modes that share a config file and little else:

- **Dev server** (`npm run dev`; `npm start` runs it for you). Nothing is bundled. The
  browser loads `index.html`, follows `<script type="module">`, and requests each module
  by URL. Vite transforms each file on request: esbuild strips the types (one file at a time,
  no type information, hence `isolatedModules` in `tsconfig.app.json`), bare imports such as
  `react` are rewritten to dependencies pre-bundled once into a cache, and hot module
  replacement swaps edited modules in place.
- **Build** (`vite build`). Rollup bundles the module graph, esbuild minifies, every
  `import.meta.env.*` is replaced by its value, dynamic `import()` becomes a separate chunk,
  file names get content hashes. Vite 8 (released 2026-03-12) replaced esbuild and Rollup
  with Rolldown and Oxc; the concepts below carry over.

Three features matter daily:

- **`server.proxy`** forwards matching paths from the dev server to another server. This
  app's `vite.config.ts` sends `/api` and `/healthz` to `http://127.0.0.1:8080` (or to
  `STACKLAB_API_PROXY_TARGET` under Compose), so the page and its API share an origin and the
  backend needs no CORS headers. The proxy is dev-only.
- **Env variables.** Vite loads `.env`, `.env.local`, `.env.[mode]` and `.env.[mode].local`
  (mode-specific files win) and exposes only names starting with `VITE_` (configurable with
  `envPrefix`). Built-ins: `MODE`, `BASE_URL`, `DEV`, `PROD`, `SSR`. User values are
  **strings**, and they end up **in the bundle**.
- **Source maps.** In dev, each transformed TypeScript module carries an inline map. In a build,
  `build.sourcemap` is `false` by default; `true` writes `.map` files, `'inline'` embeds them,
  `'hidden'` writes them without the comment that tells browsers where to find them.

## Coming from C/C++/Java/Python/JS

- **C/C++:** a source map is debug info: DWARF or a PDB maps machine addresses to source
  lines; a source map maps generated columns to TypeScript lines. `'hidden'` maps are the
  equivalent of shipping stripped binaries and keeping the symbols for your crash reporter.
- **Go:** `go build` type-checks and links everything. Vite's dev server never type-checks
  and never links; it serves files. The built `dist/` can be embedded in a Go binary with
  `embed` (lesson go.files-io.embed), which is how one Go server ships the SPA and its API
  from the same origin in production.
- **Java:** the closest analogues are a servlet container serving static files with hot
  reload in dev, and a fat-jar build for production. Env values are build-time constants
  like Maven resource filtering, not runtime `System.getenv`.
- **nginx users:** `location /api` is also a prefix match and also matches `/apiary`; the
  same `/api/` habit applies to Vite's proxy keys.
- **JS:** webpack-dev-server bundles in memory before the first page load; Vite relies on
  the browser's native module loader and transforms files on request, so startup stays fast
  as the app grows.

## How it works

Fetching a module from this app's running dev server shows the dev pipeline directly.
`GET http://127.0.0.1:5173/src/shared/api.ts` returns (map shortened):

```js
import.meta.env = {"BASE_URL": "/", "DEV": true, "MODE": "development", "PROD": false, "SSR": false};export const API_BASE = import.meta.env.VITE_API_URL ?? "";

//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJzb3VyY2VzIjpbImFwaS50cyJdLC...
```

The doc comment and the `as string | undefined` cast are gone, Vite has prepended the env
object, and the base64 map decodes to `{"version":3,"sources":["api.ts"],...}` with the
original file inside.

For the build, a scratch project had this `.env`:

```
VITE_API_URL=/api
VITE_DEBUG=false
VITE_PAGE_SIZE=20
SECRET_TOKEN=do-not-ship
```

and this `src/main.ts` (`format.ts` exports `formatTitle` and an unused `unusedHelper`;
`editor.ts` builds a 2000-line array):

```ts
import { formatTitle } from "./format";

const api: string = import.meta.env.VITE_API_URL ?? "";
const debug = import.meta.env.VITE_DEBUG;
console.log(formatTitle("istanbul"), api, typeof debug, debug ? "debug on" : "debug off");
console.log(import.meta.env.VITE_PAGE_SIZE + 1, import.meta.env.MODE);
if (import.meta.env.DEV) console.log("dev only");

document.querySelector("#open")?.addEventListener("click", async () => {
  const { openEditor } = await import("./editor");
  console.log(openEditor());
});
```

`vite build --sourcemap` printed:

```
vite v7.3.7 building client environment for production...
transforming...
✓ 6 modules transformed.
rendering chunks...
computing gzip size...
dist/index.html                 0.17 kB │ gzip: 0.14 kB
dist/assets/editor-C4QPI_bQ.js  0.15 kB │ gzip: 0.16 kB │ map: 0.40 kB
dist/assets/index-Bm7OXp9I.js   2.21 kB │ gzip: 1.13 kB │ map: 1.24 kB
✓ built in 136ms
```

and the entry chunk contains `console.log(P("istanbul"),L,"string","debug on");console.log("201","production");`.
The minifier folded both naive reads to wrong constants, the `DEV` branch is gone,
`unusedHelper` and `SECRET_TOKEN` appear nowhere, and `editor` became its own chunk because
it was loaded with `import()`. Adding `console.log(JSON.stringify(import.meta.env))` showed
that reading the whole object compiles to
`{BASE_URL:"/",DEV:!1,MODE:"production",PROD:!0,SSR:!1,VITE_API_URL:"/api",VITE_DEBUG:"false",VITE_PAGE_SIZE:"20"}`.

In DevTools, open Sources, find the `.ts` file under the dev server's origin, and set
breakpoints as usual; a `debugger;` statement also stops there when DevTools is open.
Stack traces in the console already point at TypeScript lines, because DevTools applies the
same lookup the second example performs.

## Design notes

Parse the env once, at the edge, the way the first example does, and export a typed
`config` object; nothing else in the app should read `import.meta.env`. For stricter types
without a schema, Vite's documentation shows declaring the keys on `ImportMetaEnv` in
`src/vite-env.d.ts`, and opting into `strictImportMetaEnv` through `ViteTypeOptions` so that
unknown keys stop being `any`. This app reads one variable, in
`src/shared/api.ts`, with a cast to `string | undefined`; that is the minimal honest
version.

The proxy key in this app is `'/api'`, which also forwards `/apiary`; harmless today, but
`'/api/'` says what is meant. And because `"build": "vite build"` never type-checks, the
order that keeps a broken build out of `dist/` is `npm run typecheck && npm run build`.
Production source maps are a policy decision: `'hidden'` plus an error tracker gives
readable stack traces without publishing the original source.
