// scripts/dev.mjs — `npm start`: the whole app in local mode (ADR-0008).
//
// Builds the Go backend, then runs it next to the Vite dev server; PostgreSQL
// is the only other moving part and runs as a Windows service. Ctrl+C stops
// both; if either process exits on its own, the other is stopped too.
//
// Why a script instead of `concurrently`/`npm-run-all`: node:child_process
// covers it, so there is no new dependency. The backend is built to a binary
// instead of run with `go run`, because killing `go run` on Windows leaves its
// child (the real server) running and holding port 8080.

import { spawn, spawnSync } from 'node:child_process';
import { existsSync, mkdirSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const viteBin = path.join(root, 'node_modules', 'vite', 'bin', 'vite.js');
const backendBin = path.join(root, '.bin', process.platform === 'win32' ? 'stacklab.exe' : 'stacklab');

if (!existsSync(viteBin)) {
  console.error('[start] node_modules is missing: run `npm install` once, then `npm start`.');
  process.exit(1);
}

// Libraries ts-browser lessons import (react etc.) are generated, not checked
// in; build them once, and again after `npm install` changes their versions.
const libsManifest = path.join(root, 'public', 'lesson-libs', 'manifest.json');
if (!existsSync(libsManifest)) {
  console.log('[start] bundling lesson libraries (first run only)…');
  const libs = spawnSync(process.execPath, [path.join(root, 'scripts', 'build-lesson-libs.mjs')], {
    cwd: root,
    stdio: 'inherit',
  });
  if (libs.status !== 0) process.exit(libs.status ?? 1);
}

mkdirSync(path.dirname(backendBin), { recursive: true });
console.log('[start] building the Go backend…');
const build = spawnSync('go', ['build', '-o', backendBin, './cmd/stacklab'], {
  cwd: path.join(root, 'server'),
  stdio: 'inherit',
});
if (build.error) {
  console.error(`[start] could not run "go" (${build.error.message}). Is Go installed and on PATH?`);
  process.exit(1);
}
if (build.status !== 0) process.exit(build.status ?? 1);

const children = [];
let stopping = false;

function stopAll(exitCode) {
  if (stopping) return;
  stopping = true;
  process.exitCode = exitCode;
  for (const child of children) {
    if (child.exitCode === null && child.signalCode === null) child.kill();
  }
}

function run(name, command, args, hint) {
  const child = spawn(command, args, { cwd: root, stdio: 'inherit' });
  children.push(child);
  child.on('exit', (code, signal) => {
    if (!stopping) {
      console.error(`[start] ${name} exited (${signal ?? code}); stopping the rest.`);
      if (code !== 0 && hint) console.error(`[start] ${hint}`);
    }
    stopAll(code ?? 1);
  });
}

run(
  'backend',
  backendBin,
  [],
  'Is PostgreSQL running, and was scripts/setup-local-db.sql applied? See README "Run locally".',
);
// 127.0.0.1, not "localhost": on Windows localhost can resolve to ::1 only,
// and the README (and CLAUDE.md's local-only rule) promise 127.0.0.1.
run('vite', process.execPath, [viteBin, '--host', '127.0.0.1', '--port', '5173', '--strictPort']);

// Ctrl+C reaches both children directly (same console) and each shuts down
// gracefully by itself; this handler only keeps Node from exiting first, with
// a hard stop if a child ignores the signal.
process.on('SIGINT', () => {
  stopping = true;
  setTimeout(() => {
    for (const child of children) if (child.exitCode === null) child.kill();
  }, 5000).unref();
});
