/// <reference lib="webworker" />
/**
 * esbuild-wasm worker: transpiles TS lesson code off the main thread.
 *
 * Why a worker: esbuild's WASM binary is ~10MB and its first initialize blocks;
 * keeping it here means the editor never janks while the wasm warms up. The
 * worker is created once per app lifetime by BrowserTsRunner.
 *
 * Protocol: { id, code, loader } in -> { id, ok, js | error } out.
 */

import * as esbuild from 'esbuild-wasm';

let ready: Promise<void> | null = null;

/** Initialize once; esbuild-wasm needs the URL of its own wasm binary. */
function ensureReady(): Promise<void> {
  ready ??= esbuild.initialize({
    // esbuild.wasm ships at the site root (copied to public/ by the prebuild
    // script), so the URL is stable regardless of worker chunk hashing.
    wasmURL: '/esbuild.wasm',
    worker: false, // we ARE the worker; nested workers are blocked on some setups
  });
  return ready;
}

self.onmessage = async (e: MessageEvent<{ id: number; code: string; loader: 'ts' | 'tsx' }>) => {
  const { id, code, loader } = e.data;
  try {
    await ensureReady();
    const out = await esbuild.transform(code, {
      loader,
      // .tsx lessons: JSX compiles to imports from react/jsx-runtime, which the
      // runner's import map provides (public/lesson-libs).
      jsx: 'automatic',
      target: 'es2022',
      // An ES module, run as <script type="module">: lessons use top-level
      // await, which esbuild refuses to emit as an iife.
      format: 'esm',
      // bundle defaults to false: lesson TS is a single module; imports would
      // need the FS API, which we don't expose.
    });
    self.postMessage({ id, ok: true, js: out.code });
  } catch (err) {
    self.postMessage({ id, ok: false, error: err instanceof Error ? err.message : String(err) });
  }
};
