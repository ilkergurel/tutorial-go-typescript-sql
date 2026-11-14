---
id: ts.browser-dom.storage
title: "localStorage, sessionStorage, IndexedDB, Web Workers, URL/URLSearchParams"
track: ts
module: browser-dom
order: 4
level: intermediate
minutes: 15
since: es2022
prerequisites: [ts.browser-dom.dom, ts.async.event-loop]
compare_with: [ts.react.routing, ts.architecture.state, go.concurrency.goroutines-channels]
tags: [storage, localstorage, indexeddb, web-workers, url, urlsearchparams]
summary: >
  Shareable view state goes in the URL, small preferences in localStorage,
  per-tab drafts in sessionStorage, and large or structured data in IndexedDB.
  Storage can be unavailable, so wrap it. Heavy computation goes to a Worker.
examples:
  - id: safe-storage
    file: examples/01_safe_storage.ts
    title: "Web Storage that can fail: a fallback and versioned, validated JSON"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-4"
        note: >
          The consumer declares the three methods it needs. Real Storage and the
          in-memory fallback both fit, by structure, the way a Go interface is
          satisfied without an implements clause.
      - lines: "6-17"
        note: >
          The fallback keeps the contract: values are strings. lib.dom types
          setItem's value as string, so tsc rejects setItem("n", 42); plain
          JavaScript would store "42", and an object becomes "[object Object]".
      - lines: "19-30"
        note: >
          Two separate failures. Reading window.localStorage throws SecurityError
          when storage is blocked: this sandboxed iframe, or a user who blocks
          site data. setItem throws QuotaExceededError when the origin's quota
          (about 5 MiB) is full. The first line of output is the first case.
      - lines: "36-37"
        note: >
          A version in the key: when Prefs changes shape, write prefs:v2 and
          leave old data behind instead of misreading it.
      - lines: "39-58"
        note: >
          Stored data is input: an older version of your app, another tab or the
          user in devtools wrote it. Parse to unknown, check the shape, fall back
          to defaults; never cast.
      - lines: "60-65"
        note: >
          The rest of the code can't tell which store it got. In a normal page
          the same lines use localStorage and survive a reload.
  - id: url-state
    file: examples/02_url_state.ts
    title: "The URL as state: URLSearchParams, defaults, encodings"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "10-19"
        note: >
          Parse defensively: page=abc becomes 1, an unknown sort becomes "name".
          getAll returns every value of a repeated key, here the selected cities.
      - lines: "21-29"
        note: >
          Write only values that differ from the defaults: shorter links, and
          the same view always produces the same URL.
      - lines: "31-36"
        note: >
          URLSearchParams decodes + as a space and percent escapes as UTF-8
          bytes, so %C4%B0 is İ and the query reads "Ayşe Yılmaz".
      - lines: "38-46"
        note: >
          toString() percent-encodes every value (ğ in Muğla becomes %C4%9F), and
          assigning url.search replaces the whole query. On a page, history.replaceState puts the result in the address bar
          without a reload, and pushState adds an entry for the Back button.
      - lines: "48-50"
        note: >
          URLSearchParams uses form encoding (a space is +); encodeURIComponent
          uses %20. Decode with the matching function, or a + stays a +.
      - lines: "52-55"
        note: >
          new URL(relative, base) resolves a link the way the browser resolves
          an href, including ../ segments.
  - id: worker
    file: examples/03_worker.ts
    title: "A Web Worker, structured clone, and transfer"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3-12"
        note: >
          A Worker runs a script on its own thread with its own event loop. Real
          projects give it its own file; a one-file lesson ships the code as a
          string, which tsc can't check.
      - lines: "11"
        note: >
          There is no document in a worker (the output says undefined), and no
          localStorage either. In a normal page, workers do get IndexedDB, fetch
          and timers.
      - lines: "29-37"
        note: >
          postMessage is fire and forget; a Promise wrapper turns it into a
          request and a reply. This one assumes one request at a time.
      - lines: "39-43"
        note: >
          The worker parses Turkish decimal commas; the page formats the result
          back with tr-TR. terminate() stops the thread: a worker otherwise lives
          as long as the page.
      - lines: "45-58"
        note: >
          Messages are copied with the structured clone algorithm, the same one
          structuredClone uses. Map survives; a class instance arrives as a plain
          object without its prototype; a function throws DataCloneError.
      - lines: "60-65"
        note: >
          A transfer moves the ArrayBuffer instead of copying it, and the sender's
          copy is left empty. Ownership moves with the data, so there is nothing
          to race on.
use_cases:
  - "The URL: anything a link should reproduce, such as filters, sort order, page and the selected item"
  - "sessionStorage: per-tab state that should survive a reload but not appear in other tabs, such as a half-filled wizard"
  - "localStorage: small preferences that outlive the session: theme, column widths, the last opened tab"
  - "IndexedDB: large or structured data: offline caches, drafts with attachments, queued uploads"
  - "Web Workers: CPU-heavy parsing, diffing or compression without freezing the page"
strengths:
  - "localStorage is a single synchronous call with no setup or schema, unlike any database driver"
  - "The URL as state store gives shareable, bookmarkable views and a working Back button for free"
  - "Workers give parallelism without data races: messages are copied or transferred, Go's 'share memory by communicating' enforced by the platform"
weaknesses:
  - "Web Storage holds strings only, about 5 MiB per origin, and every call is synchronous on the main thread"
  - "IndexedDB's request-and-event API is verbose; most code wraps it in promises"
  - "Every script on the page can read Web Storage and IndexedDB, so an XSS bug exposes all of it"
  - "Workers can't touch the DOM, and each message pays for a structured clone unless you transfer"
pitfalls:
  - title: "Session tokens in localStorage"
    detail: "Any injected script can read and exfiltrate them. Let the server set an HttpOnly cookie, which JavaScript cannot read at all."
  - title: "Assuming storage exists"
    detail: "Access throws in sandboxed iframes, with blocked site data, and on some file: pages. Wrap it once and fall back."
    bad: "const theme = localStorage.getItem(\"theme\") ?? \"light\";"
    good: "const store = openStore(\"localStorage\"); // try/catch + memory fallback\nconst theme = store.getItem(\"theme\") ?? \"light\";"
  - title: "Casting stored JSON"
    detail: "JSON.parse(raw) as Prefs trusts whatever an older version wrote. Parse to unknown and check the shape, or validate with a schema."
  - title: "Secrets or personal data in the URL"
    detail: "URLs end up in browser history, bookmarks and server logs. Keep ids and filters there, not tokens or national ID numbers."
  - title: "Expecting class instances to survive postMessage"
    detail: "Only data crosses: the prototype is not copied, so methods are gone. Send plain data and rebuild objects on the other side."
quiz:
  - q: "A colleague should open your filtered order list from a link. Where does the filter live?"
    options: ["localStorage", "sessionStorage", "The URL query string", "IndexedDB"]
    answer: 2
    why: "Only the URL travels with a link. Storage is per browser and per origin, so a colleague never sees yours."
  - q: "Plain JavaScript runs localStorage.setItem(\"prefs\", { theme: \"dark\" }). What is stored?"
    options: ["{\"theme\":\"dark\"}", "[object Object]", "Nothing; it throws"]
    answer: 1
    why: "Values are converted to strings, and an object's default string is [object Object]. Store JSON.stringify(value)."
  - q: "Which store survives a reload but is not visible in a second tab of the same site?"
    options: ["localStorage", "sessionStorage", "IndexedDB"]
    answer: 1
    why: "sessionStorage is partitioned by origin and by tab, and lasts as long as the tab."
  - q: "After worker.postMessage(buf, [buf]), what is buf.byteLength in the sending thread?"
    options: ["Unchanged", "0, because the buffer was transferred", "postMessage throws"]
    answer: 1
    why: "Transferring moves ownership: the sender's ArrayBuffer is detached and reports 0 bytes."
further_reading:
  - title: "MDN: Web Storage API"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Web_Storage_API"
  - title: "MDN: Storage quotas and eviction criteria"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Storage_API/Storage_quotas_and_eviction_criteria"
  - title: "MDN: Using IndexedDB"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/IndexedDB_API/Using_IndexedDB"
  - title: "MDN: Using Web Workers"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Web_Workers_API/Using_web_workers"
  - title: "MDN: URLSearchParams"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/URLSearchParams"
  - title: "MDN: The structured clone algorithm"
    url: "https://developer.mozilla.org/en-US/docs/Web/API/Web_Workers_API/Structured_clone_algorithm"
---

## Concept

A browser offers several places to keep data, and the question is always who should see
it and for how long:

- **The URL** (path and query): state a link should reproduce. Visible, shareable,
  bookmarkable, and part of the Back button's history.
- **sessionStorage**: strings, per origin *and* per tab. Survives reloads, disappears with
  the tab.
- **localStorage**: strings, per origin, shared by all tabs, kept until the user or your
  code clears it. A `storage` event tells *other* tabs about changes.
- **IndexedDB**: per origin, asynchronous and transactional, with indexes. Values are
  structured clones, so `Date`, `Map` and `Blob` are stored as they are. Quotas are a share
  of the disk rather than a few megabytes.
- **Cookies**: sent to the server with every request. Session identifiers belong here, as
  `HttpOnly` cookies that scripts can't read.
- **Memory**: everything else, gone on reload.

Browsers allow about 5 MiB of localStorage and 5 MiB of sessionStorage per origin, and
throw `QuotaExceededError` beyond that. Any of these stores can be unavailable: the lesson
runner's sandboxed iframe has no storage at all, which is why example 1 falls back to
memory.

A **Web Worker** is the other half of this lesson: a second thread with its own event loop
and no DOM, talking to the page by messages.

## Coming from C/C++/Java/Python/JS

- **Go**: a Worker is the closest thing to a goroutine the browser has, but with nothing
  shared: no common heap, only messages (unless you opt into `SharedArrayBuffer`), so it
  behaves like a goroutine that only ever talks over channels.
- **Java**: `Preferences` is localStorage; IndexedDB is an embedded object store, closer to
  a key-value store than to an embedded SQL database. Workers are threads without shared
  objects.
- **Python**: `shelve` or `sqlite3` on disk for persistent data, `multiprocessing` with
  pickled messages for workers. Structured clone plays pickle's role, with fewer types.
- **JS/Node**: `worker_threads` are Node's Workers, with the same structured-clone
  messages and transfer lists.

## How it works

IndexedDB predates Promises: every operation returns a request object that later fires
`success` or `error`. Two small wrappers make it usable with `await` (the lesson runner
can't open IndexedDB, so this snippet was checked in a normal page):

```ts
function openDb(
  name: string,
  version: number,
  upgrade: (db: IDBDatabase) => void,
): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(name, version);
    req.onupgradeneeded = () => upgrade(req.result); // first open or new version
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => reject(req.error);
  });
}

function done(tx: IDBTransaction): Promise<void> {
  return new Promise((resolve, reject) => {
    tx.oncomplete = () => resolve();
    tx.onerror = tx.onabort = () => reject(tx.error);
  });
}

const db = await openDb("stacklab", 1, (db) =>
  db.createObjectStore("drafts", { keyPath: "id" }),
);
const tx = db.transaction("drafts", "readwrite");
tx.objectStore("drafts").put({ id: "order-7", note: "Merhaba", savedAt: new Date() });
await done(tx);
```

The version number is the schema migration mechanism: raising it fires `upgradeneeded`,
the only place where object stores and indexes can be created. A transaction commits on
its own once no requests are pending, so don't `await` unrelated work in the middle of one.

## Design notes

Choose by audience and lifetime. If a link should reproduce it, it is URL state (React
Router's search params build on exactly this API). If only this tab cares, sessionStorage.
If this browser should remember it, localStorage for small values, IndexedDB for anything
large, binary or queryable. If it is a credential, it is an `HttpOnly` cookie and none of
the above.

Treat every persisted value as input from an older version of your app: version the keys
or the database, parse to `unknown`, validate, fall back to defaults. And wrap access once,
behind an interface the rest of the code declares, so a blocked store degrades the app
instead of breaking it.

Move CPU-heavy work to a Worker when it takes longer than a frame or two. Keep the
message types in one module that both sides import, model them as discriminated unions,
and transfer large `ArrayBuffer`s instead of copying them.
