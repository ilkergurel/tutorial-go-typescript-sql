/**
 * BrowserTsRunner: runs TS browser lessons entirely in the browser.
 *
 * Pipeline: esbuild-wasm worker transpiles TS -> an ES module, which executes
 * as <script type="module"> in an <iframe sandbox="allow-scripts"> (opaque
 * origin, no same-origin access to the app, no network: see IFRAME_CSP). The
 * iframe shim routes console.* and errors to postMessage; the runner collects
 * them into the same RunResult shape the server produces, with a hard timeout
 * that destroys the iframe (an infinite loop dies with it).
 *
 * When is a run over? A browser has no "event loop idle" signal, so the shim
 * tracks it: the module ran to its end (after any top-level await), or threw,
 * and no timer is pending. Promise chains driven by timers, the usual
 * async-lesson pattern, therefore finish completely. An uncaught error
 * anywhere else (a timer callback, a listener) is reported with exit 1 but
 * doesn't end the run, as it doesn't stop a page. Rejected alternative:
 * finishing a fixed delay after the iframe loads, which cut off any output
 * logged later.
 *
 * Why not eval in-page: the iframe's opaque origin is the actual security
 * boundary for DOM access; the worker only transpiles.
 */
import type { CodeRunner } from './CodeRunner';
import type { RunRequest, RunResult } from './types';

interface WorkerOut {
  id: number;
  ok: boolean;
  js?: string;
  error?: string;
}

/** Messages the iframe shim posts to the runner. */
type ShimMessage = {
  __stacklab_run: true;
  /** log/info/debug -> stdout; warn/error -> stderr; uncaught -> stderr and exit 1. */
  type: 'log' | 'info' | 'debug' | 'warn' | 'error' | 'uncaught' | 'done';
  args: string[];
};

/**
 * The script that runs inside the sandboxed iframe, before the lesson module:
 * console capture, uncaught-error capture, and pending-timer tracking.
 * Exported for the tests, which run it in jsdom.
 */
export const IFRAME_SHIM = `
(function () {
  var post = function (type, args) {
    parent.postMessage({ __stacklab_run: true, type: type, args: args }, '*');
  };
  // Values print the way a reader expects; JSON.stringify alone would turn a
  // Map into {} and undefined into nothing.
  function fmt(v) {
    if (typeof v === 'string') return v;
    if (v === undefined || typeof v === 'function' || typeof v === 'symbol' || typeof v === 'bigint') return String(v);
    if (v instanceof Error) return v.name + ': ' + v.message;
    if (v instanceof Map) return 'Map(' + v.size + ') {' + Array.from(v, function (e) { return fmt(e[0]) + ' => ' + fmt(e[1]); }).join(', ') + '}';
    if (v instanceof Set) return 'Set(' + v.size + ') {' + Array.from(v, fmt).join(', ') + '}';
    try { var s = JSON.stringify(v); return s === undefined ? String(v) : s; } catch (e) { return String(v); }
  }
  ['log', 'info', 'warn', 'error', 'debug'].forEach(function (k) {
    var orig = console[k] ? console[k].bind(console) : function () {};
    console[k] = function () { post(k, Array.prototype.map.call(arguments, fmt)); orig.apply(console, arguments); };
  });

  var mainDone = false, finished = false, pending = new Set();
  var realSetTimeout = setTimeout, realClearTimeout = clearTimeout;
  var realSetInterval = setInterval, realClearInterval = clearInterval;
  function check() {
    if (finished || !mainDone || pending.size > 0) return;
    // Two more task turns before the run is over. The first lets promise
    // continuations run; they may start timers. The second lets the browser
    // fire unhandledrejection for a promise rejected in the run's last task:
    // that event is a task queued after the microtask checkpoint, behind the
    // first turn's timer. With one turn such a rejection was lost every time
    // (20 of 20 runs in Edge, 2026-11-14) and the run passed with exit 0.
    realSetTimeout(function () {
      if (finished || pending.size > 0) return;
      realSetTimeout(function () {
        if (finished || pending.size > 0) return;
        finished = true;
        post('done', []);
      }, 0);
    }, 0);
  }
  window.setTimeout = function (fn, ms) {
    var args = Array.prototype.slice.call(arguments, 2);
    var id = realSetTimeout(function () {
      pending.delete(id);
      try { if (typeof fn === 'function') fn.apply(null, args); } finally { check(); }
    }, ms);
    pending.add(id);
    return id;
  };
  window.clearTimeout = function (id) { pending.delete(id); realClearTimeout(id); check(); };
  // An interval keeps the run alive until cleared, as it would keep Node alive.
  window.setInterval = function (fn, ms) {
    var args = Array.prototype.slice.call(arguments, 2);
    var id = realSetInterval(function () {
      try { if (typeof fn === 'function') fn.apply(null, args); } finally { check(); }
    }, ms);
    pending.add(id);
    return id;
  };
  window.clearInterval = function (id) { pending.delete(id); realClearInterval(id); check(); };

  // React 18 development builds (the lesson libraries are dev builds) replay
  // a failed render or effect inside a "react-invokeguardedcallback" event
  // dispatched on a <react> element, so the error reaches window.onerror the
  // way an uncaught one would, even when an error boundary then catches it.
  // An error reported during that dispatch is not uncaught yet: React either
  // hands it to a boundary or rethrows it after the dispatch, and the rethrow
  // is reported as usual. Counting the dispatches tells the two apart.
  var reactReplays = 0, realDispatch = EventTarget.prototype.dispatchEvent;
  EventTarget.prototype.dispatchEvent = function (event) {
    var replay = this instanceof Element && this.localName === 'react' &&
      typeof event.type === 'string' && event.type.indexOf('react-') === 0;
    if (!replay) return realDispatch.call(this, event);
    reactReplays++;
    try { return realDispatch.call(this, event); } finally { reactReplays--; }
  };

  // An uncaught error or unhandled rejection is reported, and the run goes
  // on: an error in a timer callback or a listener doesn't stop a page, nor
  // a module that is awaiting something else meanwhile. Only the module's own
  // end, finished or thrown, ends the main part; the page reports it through
  // __stacklabMainDone (see iframeDoc). Ending the main part on any error, as
  // the shim once did, cut off the output of a module still awaiting a fetch
  // mock or a MessageChannel.
  window.addEventListener('error', function (e) {
    if (reactReplays > 0) return;
    post('uncaught', ['Uncaught ' + (e.error ? fmt(e.error) : e.message)]);
  });
  window.addEventListener('unhandledrejection', function (e) {
    post('uncaught', ['Uncaught (in promise) ' + fmt(e.reason)]);
  });
  window.__stacklabMainDone = function () { mainDone = true; check(); };
})();
`;

/**
 * No network and no other origins: lessons stay deterministic and can't call
 * home. Inline scripts are the shim and the end-of-module watcher; data:
 * scripts are the lesson and the libraries from the import map (below);
 * blob: workers and data:/blob: images cover the DOM lessons that create them.
 */
const IFRAME_CSP =
  "default-src 'none'; script-src 'unsafe-inline' data:; worker-src blob:; img-src data: blob:; style-src 'unsafe-inline'";

const dataUrl = (source: string) => 'data:text/javascript;charset=utf-8,' + encodeURIComponent(source);

/**
 * The iframe document: CSP, the import map when the lesson imports libraries,
 * the shim, the lesson module, and a module that reports the lesson's end.
 *
 * All of it sits in <head>, so <body> holds only what the lesson puts there:
 * with the scripts in <body>, document.body.textContent began with the
 * shim's source. The order still holds: the classic shim runs while the head
 * is parsed, the modules after parsing (deferred, as on any page), so the
 * lesson finds document.body and runs before DOMContentLoaded.
 *
 * The lesson is a data: URL module rather than an inline one so that the
 * last script can import() the same URL. The module map hands it the same
 * module, already running, and the promise settles once the lesson has
 * finished (after every top-level await) or failed, which marks the end of
 * the main part either way; the browser still reports the lesson's own error
 * to window.onerror. Rejected: a done marker appended to an inline module. A
 * throw skips the marker, and the error event that remains can't be told
 * apart from one thrown in a timer callback or a listener, which must not end
 * the run. The sourceURL comment names the module after the lesson's file in
 * stack traces and error events, which would otherwise quote the whole data:
 * URL. Exported for the tests.
 */
export function iframeDoc(js: string, importMap: string | null, fileName: string): string {
  const close = `</scr${''}ipt>`;
  // encodeURIComponent leaves no double quote, < or &: safe in the attribute
  // and, JSON-quoted, in the script. The name is kept to one harmless line.
  const lesson = dataUrl(`${js}\n//# sourceURL=${fileName.replace(/[^\w.-]/g, '_')}`);
  return (
    `<!doctype html><html><head><meta charset="utf-8">` +
    `<meta http-equiv="Content-Security-Policy" content="${IFRAME_CSP}">` +
    (importMap ? `<script type="importmap">${importMap}${close}` : '') +
    `<script>${IFRAME_SHIM}${close}` +
    `<script type="module" src="${lesson}">${close}` +
    // Both outcomes end the main part; handling the rejection also keeps it
    // from being reported a second time as an unhandled one.
    `<script type="module">import(${JSON.stringify(lesson)})` +
    `.then(function () { window.__stacklabMainDone(); }, function () { window.__stacklabMainDone(); });${close}` +
    `</head><body></body></html>`
  );
}

/** public/lesson-libs/manifest.json, written by scripts/build-lesson-libs.mjs. */
interface LibsManifest {
  versions: Record<string, string>;
  /** specifier -> shim module source re-exporting from "stacklab:lesson-libs" */
  modules: Record<string, string>;
}

/** Bare module specifiers the transpiled lesson imports (static and dynamic). */
function bareImports(js: string): string[] {
  const found = new Set<string>();
  for (const m of js.matchAll(/(?:\bfrom\s*|\bimport\s*\(?\s*)["']([^"'./][^"']*)["']/g)) found.add(m[1]);
  return [...found];
}

export class BrowserTsRunner implements CodeRunner {
  private worker: Worker | null = null;
  private nextId = 1;
  /** The import map for lessons that use libraries, built once on first use. */
  private libsImportMap: Promise<{ map: string; specifiers: Set<string> }> | null = null;

  /**
   * Loads the prebuilt libraries (react, react-dom, react-router-dom,
   * @tanstack/react-query, zod) from the app's own origin and turns them into
   * an import map of data: URLs; the opaque-origin iframe can't fetch them
   * itself. All shims import one bundle, so the lesson gets one React.
   */
  private loadLibs(): Promise<{ map: string; specifiers: Set<string> }> {
    this.libsImportMap ??= (async () => {
      const [manifestRes, bundleRes] = await Promise.all([
        fetch('/lesson-libs/manifest.json'),
        fetch('/lesson-libs/bundle.js'),
      ]);
      if (!manifestRes.ok || !bundleRes.ok) {
        throw new Error('lesson libraries missing: run npm run build:lesson-libs (npm start does it once)');
      }
      const manifest = (await manifestRes.json()) as LibsManifest;
      const imports: Record<string, string> = { 'stacklab:lesson-libs': dataUrl(await bundleRes.text()) };
      for (const [spec, shim] of Object.entries(manifest.modules)) imports[spec] = dataUrl(shim);
      return { map: JSON.stringify({ imports }), specifiers: new Set(Object.keys(manifest.modules)) };
    })();
    // A failed load (e.g. libraries not built yet) is retried on the next run.
    this.libsImportMap.catch(() => {
      this.libsImportMap = null;
    });
    return this.libsImportMap;
  }

  /** The import map this lesson needs, or null when it imports no library. */
  private async importMapFor(js: string): Promise<string | null> {
    const specs = bareImports(js);
    if (specs.length === 0) return null;
    const libs = await this.loadLibs();
    // An unknown specifier (say "lodash") gets no entry and fails in the
    // iframe with the browser's own "failed to resolve module" error.
    return specs.some((s) => libs.specifiers.has(s)) ? libs.map : null;
  }

  private spawn(): Worker {
    if (this.worker) return this.worker;
    this.worker = new Worker(new URL('./esbuild.worker.ts', import.meta.url), { type: 'module' });
    return this.worker;
  }

  /** Transpile via the shared worker; .tsx files get JSX. */
  private transpile(code: string, fileName: string): Promise<string> {
    return new Promise((resolve, reject) => {
      const id = this.nextId++;
      const w = this.spawn();
      const onMsg = (e: MessageEvent<WorkerOut>) => {
        if (e.data.id !== id) return;
        w.removeEventListener('message', onMsg);
        if (e.data.ok && typeof e.data.js === 'string') resolve(e.data.js);
        else reject(new Error(e.data.error ?? 'transpile failed'));
      };
      w.addEventListener('message', onMsg);
      w.postMessage({ id, code, loader: fileName.endsWith('.tsx') ? 'tsx' : 'ts' });
    });
  }

  async run(req: RunRequest): Promise<RunResult> {
    const started = Date.now();
    const [fileName, source] = Object.entries(req.files)[0] ?? ['main.ts', ''];
    let js: string;
    let importMap: string | null;
    try {
      js = await this.transpile(source, fileName);
      importMap = await this.importMapFor(js);
    } catch (err) {
      return {
        stdout: '',
        stderr: '',
        exitCode: 1,
        durationMs: Date.now() - started,
        timedOut: false,
        truncated: false,
        diagnostics: [{ file: '', line: 0, col: 0, message: err instanceof Error ? err.message : String(err) }],
        files: [],
      };
    }

    return new Promise<RunResult>((resolve) => {
      const timeoutMs = (req.timeoutS ?? 10) * 1000;
      const stdout: string[] = [];
      const stderr: string[] = [];
      let timedOut = false;
      let crashed = false;
      let settled = false;

      const iframe = document.createElement('iframe');
      iframe.setAttribute('sandbox', 'allow-scripts');
      // Invisible but rendered: a display:none cross-origin frame is treated
      // like a background tab, so its timers get batched into ~1 s wake-ups
      // and requestAnimationFrame never fires. A 1x1 transparent frame inside
      // the viewport runs at full speed and never takes a click.
      Object.assign(iframe.style, {
        position: 'fixed',
        right: '0',
        bottom: '0',
        width: '1px',
        height: '1px',
        border: '0',
        opacity: '0',
        pointerEvents: 'none',
      });
      iframe.setAttribute('aria-hidden', 'true');
      iframe.tabIndex = -1;

      const finish = () => {
        if (settled) return;
        settled = true;
        window.clearTimeout(timer);
        window.removeEventListener('message', onMsg);
        iframe.remove(); // destroying the iframe kills any loop inside it
        resolve({
          // Trailing newline, like a program's stdout: expected files end in one.
          stdout: stdout.map((l) => l + '\n').join(''),
          stderr: stderr.map((l) => l + '\n').join(''),
          exitCode: timedOut ? 124 : crashed ? 1 : 0,
          durationMs: Date.now() - started,
          timedOut,
          truncated: false,
          diagnostics: [],
          files: [],
        });
      };

      const onMsg = (e: MessageEvent) => {
        // Only our iframe: other frames could post look-alike messages.
        if (e.source !== iframe.contentWindow) return;
        const d = e.data as ShimMessage | null;
        if (!d || d.__stacklab_run !== true) return;
        const line = (d.args ?? []).join(' ');
        switch (d.type) {
          case 'done':
            finish();
            return;
          case 'uncaught':
            crashed = true;
            stderr.push(line);
            return;
          case 'warn':
          case 'error':
            stderr.push(line);
            return;
          case 'log':
          case 'info':
          case 'debug':
            stdout.push(line);
            return;
          default: {
            const unknown: never = d.type;
            stderr.push(`[runner: unknown message ${String(unknown)}]`);
          }
        }
      };
      window.addEventListener('message', onMsg);

      const timer = window.setTimeout(() => {
        timedOut = true;
        stderr.push(`[timed out after ${timeoutMs}ms]`);
        finish();
      }, timeoutMs);

      iframe.srcdoc = iframeDoc(js, importMap, fileName);
      document.body.appendChild(iframe);
    });
  }
}
