// verify-lessons: run every lesson example the way the app runs it and judge
// it the way the app judges it. contentcheck (build-lessons.mjs) proves the
// files exist; this proves they work.
//
//   - Server runtimes (go, go-test, go-server, ts-node, sql, sql-nontx,
//     sql-multisession) go through the running backend's POST /api/run, with a
//     header that keeps them out of the learner's run history.
//   - ts-browser examples run in headless Edge or Chrome, on verify.html, in
//     the app's own BrowserTsRunner (driven over the DevTools protocol).
//   - Verdicts come from src/features/runner/judge.ts, the file the run panel
//     uses, imported through Node's type stripping.
//   - Each deterministic example runs twice: different output is a failure,
//     because a learner would see a "matches expected" badge flicker.
//   - Annotation line ranges must exist in their example file.
//
// Needs the app running (npm start). Usage:
//   npm run verify:lessons [-- --filter go.basics,sql.querying] [--print]
//                          [--annotations] [--once] [--no-browser]
//                          [--write-expected]
//                          [--api http://127.0.0.1:8080] [--ui http://127.0.0.1:5173]
//
// Rejected alternative: re-implementing the runners in Node. That would check
// a copy of the runtime rules, not the code the learner's Run button uses.

import { spawn, spawnSync } from 'node:child_process';
import { existsSync, mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, relative } from 'node:path';
import { setTimeout as sleep } from 'node:timers/promises';
import { fileURLToPath } from 'node:url';
import { judge } from '../src/features/runner/judge.ts';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const args = process.argv.slice(2);
const flag = (name) => args.includes(name);
const option = (name, fallback) => {
  const i = args.indexOf(name);
  return i >= 0 && args[i + 1] ? args[i + 1] : fallback;
};
const filters = option('--filter', '').split(',').map((s) => s.trim()).filter(Boolean);
const api = option('--api', 'http://127.0.0.1:8080');
const ui = option('--ui', 'http://127.0.0.1:5173');
const twice = !flag('--once');

// --- 1. contentcheck into a private bundle file (parallel runs never race)

const work = mkdtempSync(join(tmpdir(), 'stacklab-verify-'));
const bundlePath = join(work, 'bundle.json');
// Only the filtered lessons are checked, so someone else's half-written lesson
// can't block this run; a new lesson may lack expected files while you are
// generating them.
const buildArgs = [join(root, 'scripts/build-lessons.mjs'), '--out', bundlePath];
if (filters.length) buildArgs.push('--filter', filters.join(','));
if (flag('--print') || flag('--write-expected') || flag('--annotations')) buildArgs.push('--allow-missing-expected');
const build = spawnSync(process.execPath, buildArgs, { stdio: ['ignore', 'ignore', 'inherit'] });
if (build.status !== 0) {
  console.error('contentcheck failed (errors above); nothing was run.');
  process.exit(1);
}
const bundle = JSON.parse(readFileSync(bundlePath, 'utf8'));
rmSync(work, { recursive: true, force: true });

const lessons = bundle.lessons.filter((l) => filters.length === 0 || filters.some((f) => l.id.startsWith(f)));
if (lessons.length === 0) {
  console.error(`no lesson matches ${filters.join(', ')}`);
  process.exit(1);
}

// --- 2. requests, built exactly like RunPanel.tsx builds them

/** Mirrors exampleExtras in src/features/lessons/types.ts. */
function extras(lesson, ex) {
  const expected = bundle.expected[`${lesson.dir}/${ex.id}`] ?? null;
  const fixtures = {};
  let has = false;
  for (const fx of ex.fixtures ?? []) {
    const key = `${lesson.dir}/${fx}`;
    if (key in bundle.fixtures) {
      fixtures[fx.replace('fixtures/', '')] = bundle.fixtures[key];
      has = true;
    }
  }
  return { expected, fixtures: has ? fixtures : undefined };
}

const cases = [];
const problems = [];
for (const lesson of lessons) {
  for (const ex of lesson.examples ?? []) {
    const code = bundle.files[`${lesson.dir}/${ex.file}`] ?? '';
    const { expected, fixtures } = extras(lesson, ex);
    const name = `${lesson.id} / ${ex.id} [${ex.runtime}]`;
    const lineCount = code.replace(/\n$/, '').split('\n').length;
    for (const a of ex.annotations ?? []) {
      const [from, to = from] = a.lines.split('-').map(Number);
      if (!(from >= 1 && from <= to && to <= lineCount)) {
        problems.push(`${name}: annotation lines "${a.lines}" outside ${ex.file} (1-${lineCount})`);
      }
    }
    cases.push({
      name,
      lesson,
      ex,
      code,
      expected,
      req: {
        runtime: ex.runtime,
        example: ex.id,
        files: { [ex.file.split('/').pop() ?? 'main.go']: code },
        fixtures,
        setup: ex.setup,
        stdin: ex.stdin ?? '',
        timeoutS: ex.timeout_s,
        lessonId: lesson.id,
      },
    });
  }
}

if (flag('--annotations')) {
  for (const c of cases) {
    const lines = c.code.split('\n');
    for (const a of c.ex.annotations ?? []) {
      const [from, to = from] = a.lines.split('-').map(Number);
      console.log(`\n${c.name} lines ${a.lines}: ${a.note.trim()}`);
      for (let n = from; n <= to && n <= lines.length; n++) console.log(`  ${String(n).padStart(3)} | ${lines[n - 1]}`);
    }
  }
  process.exit(problems.length ? 1 : 0);
}

// --- 3. run

async function runOnServer(req) {
  for (let attempt = 1; ; attempt++) {
    let res;
    try {
      res = await fetch(`${api}/api/run`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Stacklab-No-History': '1' },
        body: JSON.stringify(req),
      });
    } catch (err) {
      throw new Error(`backend unreachable at ${api} (is npm start running?): ${err.message}`);
    }
    const text = await res.text();
    // 503 = both run slots busy (another verify run, or the learner): retry.
    if (res.status === 503 && attempt < 30) {
      await sleep(Math.min(500 * attempt, 5000));
      continue;
    }
    if (!res.ok) {
      let detail = text;
      try {
        detail = JSON.parse(text).detail ?? text;
      } catch {
        // keep the raw body
      }
      return { httpError: `HTTP ${res.status}: ${detail}` };
    }
    return JSON.parse(text);
  }
}

const results = new Map(); // case -> [result, result?]

// npm start may still be building the backend: wait for it rather than fail.
async function waitForBackend() {
  const deadline = Date.now() + 90_000;
  for (;;) {
    try {
      if ((await fetch(`${api}/healthz`)).ok) return;
    } catch {
      // not listening yet
    }
    if (Date.now() > deadline) throw new Error(`backend not reachable at ${api}; start the app with npm start`);
    await sleep(500);
  }
}
if (cases.some((c) => c.ex.runtime !== 'ts-browser')) await waitForBackend();

for (const c of cases.filter((c) => c.ex.runtime !== 'ts-browser')) {
  const first = await runOnServer(c.req);
  const second = twice && !first.httpError && c.ex.check !== 'none' ? await runOnServer(c.req) : null;
  results.set(c, [first, second]);
}

const browserCases = cases.filter((c) => c.ex.runtime === 'ts-browser');
if (browserCases.length > 0 && !flag('--no-browser')) {
  const reqs = browserCases.map((c) => c.req);
  const out = await withBrowser(async (cdp) => {
    await cdp.send('Page.navigate', { url: `${ui}/verify.html` });
    // Vite may reload the page once while it optimizes dependencies on the
    // first visit; an evaluate that races that reload fails and is retried.
    const waitReady = async () => {
      const deadline = Date.now() + 90_000;
      for (;;) {
        try {
          const r = await cdp.send('Runtime.evaluate', {
            expression: "document.body?.dataset.ready === '1'",
            returnByValue: true,
          });
          if (r.result.value === true) return;
        } catch {
          // navigating: try again
        }
        if (Date.now() > deadline) throw new Error(`${ui}/verify.html never became ready (is npm start running?)`);
        await sleep(300);
      }
    };
    const evaluate = async () => {
      for (let attempt = 1; ; attempt++) {
        await waitReady();
        try {
          const r = await cdp.send('Runtime.evaluate', {
            expression: `window.__stacklabRun(${JSON.stringify(reqs)})`,
            awaitPromise: true,
            returnByValue: true,
          });
          if (r.exceptionDetails) {
            throw new Error(`verify.html: ${r.exceptionDetails.exception?.description ?? r.exceptionDetails.text}`);
          }
          return r.result.value;
        } catch (err) {
          if (attempt >= 3 || !/navigated|closed|context/i.test(err.message)) throw err;
          await sleep(1000);
        }
      }
    };
    return [await evaluate(), twice ? await evaluate() : null];
  });
  browserCases.forEach((c, i) => results.set(c, [out[0][i], c.ex.check !== 'none' && out[1] ? out[1][i] : null]));
} else if (browserCases.length > 0) {
  for (const c of browserCases) results.set(c, [{ skipped: true }, null]);
}

// --- 4. judge and report

let failed = 0;
let skipped = 0;
const show = (label, text) => {
  const body = (text ?? '').replace(/\r\n/g, '\n').trimEnd();
  console.log(`      ${label}:${body ? '\n' + body.split('\n').map((l) => '        ' + l).join('\n') : ' (empty)'}`);
};

// --write-expected: the golden-file workflow (like go test -update). Writes
// this run's stdout as the expected output of each exact/sorted-lines example
// that ran cleanly and the same way twice. Review the diff before keeping it:
// the tool can't know whether the output is what the lesson claims.
if (flag('--write-expected')) {
  for (const c of cases) {
    const [r, again] = results.get(c);
    const canWrite = ['exact', 'sorted-lines'].includes(c.ex.check);
    if (r.skipped || r.httpError || !canWrite) continue;
    if (r.exitCode !== 0 || r.timedOut || (again && again.stdout !== r.stdout)) {
      console.log(`not written (failed or nondeterministic): ${c.name}`);
      continue;
    }
    const file = join(root, 'content', c.lesson.dir, 'expected', `${c.ex.id}.txt`);
    mkdirSync(dirname(file), { recursive: true });
    writeFileSync(file, r.stdout.replace(/\r\n/g, '\n'));
    c.expected = r.stdout;
    console.log(`wrote ${relative(root, file)}`);
  }
}

for (const c of cases) {
  const [r, again] = results.get(c);
  if (r.skipped) {
    skipped++;
    console.log(`SKIP ${c.name} (--no-browser)`);
    continue;
  }
  const reasons = [];
  if (r.httpError) {
    reasons.push(r.httpError);
  } else {
    const verdict = judge(r, c.ex.check, c.expected);
    if (verdict === null) {
      if (r.exitCode !== 0 || r.timedOut) reasons.push(`check "none" but exit ${r.exitCode}${r.timedOut ? ', timed out' : ''}`);
    } else if (!verdict.pass) {
      reasons.push(`output does not match expected (check: ${c.ex.check}, exit ${r.exitCode}${r.timedOut ? ', timed out' : ''})`);
    }
    // Determinism means the learner's badge can't flicker: the second run must
    // pass the same check. For exact checks that is identical output; for
    // contains/regex, timings that the check ignores (go test's "ok lesson
    // 1.58s") may differ. Without an expected file yet, compare the output.
    if (again && !again.httpError) {
      const second = judge(again, c.ex.check, c.expected);
      const flickers = verdict === null ? again.stdout !== r.stdout : verdict.pass && second !== null && !second.pass;
      if (flickers) reasons.push('nondeterministic: a second run gave a different verdict');
    }
  }
  if (reasons.length === 0) {
    console.log(`ok   ${c.name}${r.durationMs !== undefined ? ` (${r.durationMs} ms)` : ''}`);
    if (flag('--print')) show('stdout', r.stdout);
    continue;
  }
  failed++;
  console.log(`FAIL ${c.name}`);
  for (const why of reasons) console.log(`      ${why}`);
  if (!r.httpError) {
    show('expected', c.expected);
    show('stdout', r.stdout);
    if (r.stderr) show('stderr', r.stderr);
    if (again && again.stdout !== r.stdout) show('stdout, second run', again.stdout);
  }
}

for (const p of problems) console.log(`FAIL ${p}`);
const total = cases.length;
console.log(
  `\n${total - failed - skipped}/${total} examples ok` +
    (skipped ? `, ${skipped} skipped` : '') +
    (problems.length ? `, ${problems.length} annotation problem(s)` : '') +
    ` (${lessons.length} lesson(s))`
);
process.exit(failed || problems.length ? 1 : 0);

// --- headless browser over the DevTools protocol (no dependency: Node has
// fetch and WebSocket built in)

function findBrowser() {
  const candidates = [
    process.env.STACKLAB_BROWSER,
    'C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe',
    'C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe',
    'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe',
    '/usr/bin/google-chrome',
    '/usr/bin/chromium',
    '/usr/bin/chromium-browser',
  ];
  const found = candidates.find((p) => p && existsSync(p));
  if (!found) throw new Error('no Edge/Chrome found; set STACKLAB_BROWSER or pass --no-browser');
  return found;
}

async function withBrowser(fn) {
  const profile = mkdtempSync(join(tmpdir(), 'stacklab-browser-'));
  const proc = spawn(
    findBrowser(),
    ['--headless=new', '--remote-debugging-port=0', `--user-data-dir=${profile}`, '--no-first-run',
      '--no-default-browser-check', '--disable-extensions', 'about:blank'],
    { stdio: 'ignore', detached: process.platform !== 'win32' }
  );
  let cdp;
  try {
    // With port 0 the browser picks a free port and writes it here.
    const portFile = join(profile, 'DevToolsActivePort');
    const deadline = Date.now() + 20_000;
    while (!existsSync(portFile) || readFileSync(portFile, 'utf8').trim() === '') {
      if (Date.now() > deadline) throw new Error('headless browser did not start');
      await sleep(100);
    }
    const port = readFileSync(portFile, 'utf8').split('\n')[0].trim();
    const targets = await (await fetch(`http://127.0.0.1:${port}/json/list`)).json();
    const page = targets.find((t) => t.type === 'page');
    if (!page) throw new Error('headless browser has no page target');
    cdp = await connectCDP(page.webSocketDebuggerUrl);
    return await fn(cdp);
  } finally {
    cdp?.close();
    if (process.platform === 'win32') spawnSync('taskkill', ['/T', '/F', '/PID', String(proc.pid)], { stdio: 'ignore' });
    else {
      try {
        process.kill(-proc.pid, 'SIGKILL');
      } catch {
        // already gone
      }
    }
    rmSync(profile, { recursive: true, force: true, maxRetries: 10, retryDelay: 200 });
  }
}

function connectCDP(url) {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(url);
    const pending = new Map();
    let nextId = 0;
    ws.onerror = () => reject(new Error('DevTools websocket failed'));
    ws.onmessage = (m) => {
      const msg = JSON.parse(m.data);
      const p = msg.id !== undefined && pending.get(msg.id);
      if (!p) return; // an event, not a reply
      pending.delete(msg.id);
      if (msg.error) p.reject(new Error(msg.error.message));
      else p.resolve(msg.result);
    };
    ws.onopen = () =>
      resolve({
        send(method, params = {}) {
          return new Promise((res, rej) => {
            const id = ++nextId;
            pending.set(id, { resolve: res, reject: rej });
            ws.send(JSON.stringify({ id, method, params }));
          });
        },
        close: () => ws.close(),
      });
  });
}
