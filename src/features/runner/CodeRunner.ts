/**
 * CodeRunner strategy: each runtime knows how to run itself. The lesson UI
 * depends only on this interface, so adding runtimes never touches the view.
 *
 * - ServerRunner sends every other runtime to the backend's POST /api/run.
 *   In local mode (the default, ADR-0008) the backend runs the code itself
 *   (internal/runexec); in Docker mode it forwards the run to the runner
 *   service (ADR-0003).
 * - BrowserTsRunner transpiles with esbuild-wasm in a Web Worker and executes
 *   in a sandboxed iframe — TS browser lessons never leave the machine.
 */
export interface CodeRunner {
  run(req: RunRequest): Promise<RunResult>;
}

import type { RunRequest, RunResult } from './types';
import { API_BASE } from '../../shared/api';
import { BrowserTsRunner } from './browserTs';

export class ServerRunner implements CodeRunner {
  async run(req: RunRequest): Promise<RunResult> {
    const res = await fetch(`${API_BASE}/api/run`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req),
    });
    if (!res.ok) {
      // Problem-details body: surface the detail, fall back to status text.
      let detail = res.statusText;
      try {
        const body = (await res.json()) as { detail?: string };
        if (body.detail) detail = body.detail;
      } catch {
        // non-JSON error body: keep statusText
      }
      throw new Error(detail);
    }
    return (await res.json()) as RunResult;
  }
}

/** Shared singletons: the browser runner owns one worker for the app lifetime. */
export const serverRunner = new ServerRunner();
export const browserTsRunner = new BrowserTsRunner();

/** Pick the runner for a runtime. ts-browser stays in the browser. */
export function runnerFor(runtime: RunRequest['runtime']): CodeRunner {
  return runtime === 'ts-browser' ? browserTsRunner : serverRunner;
}
