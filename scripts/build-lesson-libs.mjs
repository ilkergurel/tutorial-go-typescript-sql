// build-lesson-libs: pre-bundles the libraries ts-browser lessons may import
// (react, react-dom, react-router-dom, @tanstack/react-query, zod; versions
// from package.json) into public/lesson-libs/, for BrowserTsRunner's import map.
//
// Why a bundle: the lesson iframe has no network and no bundler, and React 18
// ships CommonJS only. One ES module holds every library, so all specifiers
// share one React instance (two Reacts break hooks). Each specifier gets a
// tiny shim module that re-exports its part with static named exports, which
// `import { useState } from "react"` needs; the names are read from the real
// packages here, so they always match the installed versions.
//
// Rejected alternative: serving the libraries from the Vite dev server. The
// iframe has an opaque origin, so module fetches would need CORS from Vite and
// a CSP hole for the app origin; data: URLs need neither.
//
// Usage: node scripts/build-lesson-libs.mjs (runs from prebuild and npm start).

import { mkdirSync, writeFileSync } from 'node:fs';
import { createRequire } from 'node:module';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import * as esbuild from 'esbuild-wasm';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const require = createRequire(join(root, 'package.json'));
const outDir = join(root, 'public', 'lesson-libs');

// specifier -> name of its namespace export in the bundle
const LIBS = {
  react: 'React',
  'react/jsx-runtime': 'JSXRuntime',
  'react-dom': 'ReactDOM',
  'react-dom/client': 'ReactDOMClient',
  'react-dom/server': 'ReactDOMServer',
  'react-router-dom': 'RouterDOM',
  '@tanstack/react-query': 'ReactQuery',
  zod: 'Zod',
};
// react-dom/server's default entry targets Node streams; the browser build is
// the one that works inside the iframe.
const ENTRY_PATH = { 'react-dom/server': 'react-dom/server.browser' };

const entry =
  Object.entries(LIBS)
    .map(([spec, name]) => `import * as ${name} from ${JSON.stringify(ENTRY_PATH[spec] ?? spec)};`)
    .join('\n') + `\nexport { ${Object.values(LIBS).join(', ')} };\n`;

const result = await esbuild.build({
  stdin: { contents: entry, resolveDir: root, loader: 'js' },
  bundle: true,
  format: 'esm',
  platform: 'browser',
  target: 'es2022',
  // Development builds: React's warnings (missing keys, bad hook calls) are
  // exactly what a learner should see; they go to stderr, never compared.
  define: { 'process.env.NODE_ENV': '"development"' },
  legalComments: 'none',
  write: false,
});

const identifier = /^[A-Za-z_$][A-Za-z0-9_$]*$/;
const modules = {};
for (const [spec, name] of Object.entries(LIBS)) {
  // All of these ship CommonJS for Node, so require() shows the real export
  // names (what the bundle's namespace objects will hold).
  const mod = require(ENTRY_PATH[spec] ?? spec);
  const names = Object.keys(mod).filter((k) => k !== 'default' && identifier.test(k)).sort();
  // Export names can be reserved words (zod exports `catch`), which a const
  // can't be called; so bind to neutral locals and export them under the
  // real names, which `export { x as catch }` allows.
  const locals = names.map((n, i) => `${n}: _${i}`).join(', ');
  const exported = names.map((n, i) => `_${i} as ${n}`).join(', ');
  modules[spec] =
    `import { ${name} as m } from "stacklab:lesson-libs";\n` +
    `export default m.default ?? m;\n` +
    (names.length ? `const { ${locals} } = m;\nexport { ${exported} };\n` : '');
}

const versions = Object.fromEntries(
  ['react', 'react-dom', 'react-router-dom', '@tanstack/react-query', 'zod'].map((p) => [
    p,
    require(`${p}/package.json`).version,
  ])
);
mkdirSync(outDir, { recursive: true });
writeFileSync(join(outDir, 'bundle.js'), result.outputFiles[0].text);
writeFileSync(join(outDir, 'manifest.json'), JSON.stringify({ versions, modules }, null, 2) + '\n');
console.log(
  `lesson-libs: ${Object.keys(LIBS).join(', ')} (react ${versions.react}) -> public/lesson-libs ` +
    `(${Math.round(result.outputFiles[0].contents.length / 1024)} KB)`
);
