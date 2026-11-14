// Regenerates src/features/curriculum/curriculum.generated.ts from content/curriculum.yaml.
// Run via `npm run gen:curriculum` after editing the curriculum.
// Rejected alternative: a Vite YAML import plugin — that adds a build dependency for
// one file, and a checked-in generated module also works without Vite (tsc, IDE).

import { parse } from 'yaml';
import { readFileSync, writeFileSync } from 'node:fs';

const src = 'content/curriculum.yaml';
const out = 'src/features/curriculum/curriculum.generated.ts';

const data = parse(readFileSync(src, 'utf8'));
const json = JSON.stringify(data);
writeFileSync(
  out,
  '// Generated from content/curriculum.yaml by scripts/gen-curriculum.mjs. Do not edit.\n' +
    'const data = ' + json + ';\n' +
    'export default data;\n'
);
console.log(`wrote ${out} (${json.length} bytes)`);
