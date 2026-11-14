/**
 * Entry for verify.html: exposes the real BrowserTsRunner to
 * scripts/verify-lessons.mjs, which drives a headless browser over the
 * DevTools protocol and calls window.__stacklabRun with ready-made requests.
 *
 * Why the real runner in a real browser: ts-browser lessons depend on browser
 * APIs (File, streams, DOM) and on the runner's own rules (module scripts,
 * pending-timer tracking, console formatting). Re-implementing those in Node
 * would verify a copy, not what the learner runs.
 */
import { BrowserTsRunner } from '../features/runner/browserTs';
import type { RunRequest, RunResult } from '../features/runner/types';

declare global {
  interface Window {
    __stacklabRun?: (reqs: RunRequest[]) => Promise<RunResult[]>;
  }
}

const runner = new BrowserTsRunner();

window.__stacklabRun = async (reqs) => {
  const results: RunResult[] = [];
  // One at a time, like a learner pressing Run: runs share the esbuild worker.
  for (const req of reqs) results.push(await runner.run(req));
  return results;
};
document.body.dataset.ready = '1';
