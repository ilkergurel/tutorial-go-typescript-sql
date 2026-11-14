// contentcheck: validate every lesson under content/ against lesson.schema.json
// and the style guide's mechanical rules, then emit the lesson data: one JSON
// bundle with everything (content/lessons.bundle.json, or the --out file
// verify-lessons reads) and, for the frontend, the same data split per lesson.
//
// Checks performed here:
//   - frontmatter matches lesson.schema.json
//   - lesson id matches track/module of its directory location
//   - unique ids; every curriculum lesson exists; no unknown lessons
//   - prerequisites and compare_with reference existing lessons
//   - example files, fixtures, and expected outputs exist on disk
//   - prose has the fixed H2 sections in order (Concept, Coming from..., [How it
//     works], Design notes)
//   - further_reading URLs are official docs only
//   - examples whose check is "none" carry a nondeterminism_note
// Running the examples is scripts/verify-lessons.mjs, which calls this script
// with --out first.
//
// Usage: node scripts/build-lessons.mjs [--quiet]
// Output: content/lessons.bundle.json,
//         src/features/lessons/lessonIndex.generated.ts (every lesson, light),
//         src/features/lessons/generated/<lesson id>.json (one lesson, full).

import { readFileSync, writeFileSync, readdirSync, statSync, existsSync, mkdirSync, rmSync } from 'node:fs';
import { join, dirname, relative, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');
const contentDir = join(root, 'content');
const quiet = process.argv.includes('--quiet');
// --out <file>: write the bundle JSON there and nothing else. verify-lessons
// uses it so parallel checks never race on the shared bundle files.
const outIdx = process.argv.indexOf('--out');
const outFile = outIdx > 0 ? process.argv[outIdx + 1] : null;
// --filter a,b (with --out only): check and bundle just the lessons whose
// directory names an id equal to or under a prefix ("go.basics" or a full id).
// Lessons someone else is halfway through writing then can't block your check.
// Matching uses the directory, not the frontmatter, so a broken id still matches.
const filterIdx = process.argv.indexOf('--filter');
const filters = filterIdx > 0 && outFile ? process.argv[filterIdx + 1].split(',').filter(Boolean) : [];
const inFilter = (rel) => {
  if (filters.length === 0) return true;
  const [track, module, dir = ''] = rel.split('/');
  const dirId = `${track}.${module}.${dir.replace(/^\d+-/, '')}`;
  return filters.some((f) => dirId === f || dirId.startsWith(f + '.'));
};
// --allow-missing-expected (with --out only): a missing expected file is a
// warning, so verify-lessons --print/--write-expected can run on a new lesson.
const allowMissingExpected = outFile !== null && process.argv.includes('--allow-missing-expected');

const errors = [];
const warnings = [];
const fail = (msg) => errors.push(msg);
const warn = (msg) => warnings.push(msg);

// --- JSON Schema validation without a schema-library dependency: implement the
// subset lesson.schema.json uses (type, required, enum, pattern, min/max, items,
// properties, allOf/if/then). Rejected alternative: pulling in ajv — the subset is
// ~80 lines and contentcheck runs before dependencies exist in some environments.

function validate(value, schema, path) {
  if (schema.enum && !schema.enum.includes(value)) {
    return fail(`\`${path}\` must be one of ${schema.enum.join(', ')} (got ${JSON.stringify(value)})`);
  }
  switch (schema.type) {
    case 'object': {
      if (typeof value !== 'object' || value === null || Array.isArray(value)) {
        return fail(`\`${path}\` must be an object`);
      }
      for (const req of schema.required ?? []) {
        if (!(req in value)) fail(`\`${path}\` is missing required property \`${req}\``);
      }
      for (const [key, sub] of Object.entries(schema.properties ?? {})) {
        if (key in value) validate(value[key], sub, `${path}.${key}`);
      }
      break;
    }
    case 'array': {
      if (!Array.isArray(value)) return fail(`\`${path}\` must be an array`);
      if (schema.minItems !== undefined && value.length < schema.minItems) {
        fail(`\`${path}\` needs at least ${schema.minItems} item(s)`);
      }
      if (schema.maxItems !== undefined && value.length > schema.maxItems) {
        fail(`\`${path}\` allows at most ${schema.maxItems} item(s)`);
      }
      value.forEach((item, i) => validate(item, schema.items ?? {}, `${path}[${i}]`));
      break;
    }
    case 'string': {
      if (typeof value !== 'string') return fail(`\`${path}\` must be a string`);
      if (schema.pattern && !new RegExp(schema.pattern).test(value)) {
        fail(`\`${path}\` must match ${schema.pattern} (got ${JSON.stringify(value)})`);
      }
      if (schema.minLength !== undefined && value.length < schema.minLength) {
        fail(`\`${path}\` is shorter than ${schema.minLength} chars`);
      }
      if (schema.maxLength !== undefined && value.length > schema.maxLength) {
        fail(`\`${path}\` is longer than ${schema.maxLength} chars`);
      }
      break;
    }
    case 'integer': {
      if (!Number.isInteger(value)) return fail(`\`${path}\` must be an integer`);
      if (schema.minimum !== undefined && value < schema.minimum) fail(`\`${path}\` must be >= ${schema.minimum}`);
      if (schema.maximum !== undefined && value > schema.maximum) fail(`\`${path}\` must be <= ${schema.maximum}`);
      break;
    }
  }
  if (schema.allOf) {
    for (const sub of schema.allOf) {
      if (sub.if) {
        const matches = Object.entries(sub.if.properties ?? {}).every(
          ([k, v]) => value[k] === v.const
        );
        if (matches && sub.then) {
          for (const req of sub.then.required ?? []) {
            if (!(req in value)) fail(`\`${path}\` with check "none" must declare \`${req}\``);
          }
        }
      }
    }
  }
}

// --- curriculum

const { parse } = await import('yaml');
const curriculum = parse(readFileSync(join(contentDir, 'curriculum.yaml'), 'utf8'));
const curriculumLessonIds = new Set();
for (const track of Object.values(curriculum.tracks)) {
  for (const mod of track.modules) {
    for (const lesson of mod.lessons) curriculumLessonIds.add(lesson.id);
  }
}

// --- official docs allow-list (style guide rule: further_reading links only here)

const OFFICIAL_HOSTS = new Set([
  'go.dev', 'pkg.go.dev',
  'www.typescriptlang.org', 'typescriptlang.org',
  'developer.mozilla.org',
  'react.dev',
  'www.postgresql.org', 'postgresql.org',
  'nodejs.org',
]);

// --- fixed H2 order per style guide

const H2_ORDER = ['Concept', 'Coming from C/C++/Java/Python/JS', 'How it works', 'Design notes'];

function checkProse(id, body) {
  const h2s = [...body.matchAll(/^## (.+)$/gm)].map((m) => m[1]);
  const expected = H2_ORDER.filter((h) => h2s.includes(h));
  if (JSON.stringify(h2s) !== JSON.stringify(expected)) {
    fail(`${id}: prose H2 sections must be ${JSON.stringify(expected)} in order, found ${JSON.stringify(h2s)}`);
  }
  if (!h2s.includes('Concept')) fail(`${id}: missing "## Concept" section`);
  if (!h2s.includes('Coming from C/C++/Java/Python/JS')) {
    fail(`${id}: missing "## Coming from C/C++/Java/Python/JS" section`);
  }
  if (!h2s.includes('Design notes')) fail(`${id}: missing "## Design notes" section`);
}

// --- walk content/<track>/<module>/<NN-slug>/lesson.md

function* walkLessons(dir) {
  for (const entry of readdirSync(dir)) {
    const p = join(dir, entry);
    if (!statSync(p).isDirectory()) continue;
    if (entry === 'shared') continue;
    yield* walkLessons(p);
    if (existsSync(join(p, 'lesson.md'))) yield p;
  }
}

/** Minimal frontmatter splitter: frontmatter ends at the first line that is exactly `---`. */
function splitFrontmatter(text) {
  const m = text.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n?([\s\S]*)$/);
  if (!m) throw new Error('no frontmatter block found (must start with --- and close with ---)');
  return { fm: m[1], body: m[2] };
}

// LEARN(<lesson-id>) markers (CLAUDE.md conventions) tie illustrative code to
// the lesson that explains it; a marker naming no curriculum lesson is a
// broken link. Lesson examples are checked with their lesson; the app's own
// code (server/, src/, scripts/) on full runs, since a filtered run is about
// specific lessons.
function checkLearnMarkers(source, where) {
  // Only id-shaped arguments: prose such as LEARN(<lesson-id>) is not a marker.
  for (const m of source.matchAll(/LEARN\(([a-z][a-z0-9.-]*)\)/g)) {
    if (!curriculumLessonIds.has(m[1])) fail(`${where}: LEARN(${m[1]}) names no curriculum lesson`);
  }
}

function* sourceFiles(dir) {
  for (const entry of readdirSync(dir)) {
    if (entry === 'node_modules' || entry.endsWith('.generated.ts')) continue;
    const p = join(dir, entry);
    if (statSync(p).isDirectory()) yield* sourceFiles(p);
    else if (/\.(go|ts|tsx|mjs|sql)$/.test(entry)) yield p;
  }
}

if (filters.length === 0) {
  for (const dir of ['server', 'src', 'scripts']) {
    for (const file of sourceFiles(join(root, dir))) {
      checkLearnMarkers(readFileSync(file, 'utf8'), relative(root, file).split(sep).join('/'));
    }
  }
}

const lessons = [];
const seenIds = new Set();

for (const lessonDir of walkLessons(contentDir)) {
  // Always "/"-separated: it becomes the bundle's lesson.dir and key prefix,
  // which the frontend looks up with "/". On Windows relative() returns "\".
  const rel = relative(contentDir, lessonDir).split(sep).join('/');
  if (!inFilter(rel)) continue;
  const file = join(lessonDir, 'lesson.md');
  let front, body;
  try {
    ({ fm: front, body } = splitFrontmatter(readFileSync(file, 'utf8')));
  } catch (e) {
    fail(`${rel}: ${e.message}`);
    continue;
  }
  // A YAML mistake (say an unquoted title with a colon) is a content error to
  // report next to the others, not a crash with a stack trace.
  let fm;
  try {
    fm = parse(front);
  } catch (e) {
    fail(`${rel}: frontmatter is not valid YAML: ${e.message.split('\n')[0]}`);
    continue;
  }
  validate(fm, JSON.parse(readFileSync(join(contentDir, 'lesson.schema.json'), 'utf8')), rel);

  // id must match directory: content/<track>/<module>/<slug>
  const parts = rel.split('/');
  if (parts.length !== 3) {
    fail(`${rel}: lessons live at content/<track>/<module>/<NN-slug>/`);
  } else {
    const [dirTrack, dirModule] = parts;
    const slug = parts[2].replace(/^\d+-/, '');
    if (fm.track !== dirTrack) fail(`${rel}: frontmatter track "${fm.track}" != directory "${dirTrack}"`);
    if (fm.module !== dirModule) fail(`${rel}: frontmatter module "${fm.module}" != directory "${dirModule}"`);
    if (fm.id !== `${dirTrack}.${dirModule}.${slug}`) {
      fail(`${rel}: id "${fm.id}" must be ${dirTrack}.${dirModule}.${slug}`);
    }
  }

  if (seenIds.has(fm.id)) fail(`${rel}: duplicate lesson id ${fm.id}`);
  seenIds.add(fm.id);

  checkProse(fm.id, body);

  // example/fixture/expected files exist
  for (const ex of fm.examples ?? []) {
    if (!existsSync(join(lessonDir, ex.file))) fail(`${fm.id}: example file missing: ${ex.file}`);
    else checkLearnMarkers(readFileSync(join(lessonDir, ex.file), 'utf8'), `${rel}/${ex.file}`);
    for (const fix of ex.fixtures ?? []) {
      if (!existsSync(join(lessonDir, fix))) fail(`${fm.id}: fixture missing: ${fix}`);
    }
    const expectedPath = join(lessonDir, 'expected', ex.id + '.txt');
    if (ex.check !== 'none' && !existsSync(expectedPath)) {
      (allowMissingExpected ? warn : fail)(`${fm.id}: expected/${ex.id}.txt missing for check "${ex.check}"`);
    }
  }

  // exercise directories follow the layout the runner expects
  for (const exercise of fm.exercises ?? []) {
    const exDir = join(lessonDir, 'exercises', exercise.id);
    if (!existsSync(exDir)) fail(`${fm.id}: exercise dir missing: exercises/${exercise.id}`);
  }

  for (const ref of fm.further_reading ?? []) {
    let host;
    try {
      host = new URL(ref.url).host;
    } catch {
      fail(`${fm.id}: further_reading url invalid: ${ref.url}`);
      continue;
    }
    if (!OFFICIAL_HOSTS.has(host)) {
      fail(`${fm.id}: further_reading must be official docs only, got ${host}`);
    }
  }

  if (!curriculumLessonIds.has(fm.id)) {
    warn(`${fm.id}: exists on disk but not in curriculum.yaml`);
  }

  lessons.push({ ...fm, _rel: rel, _body: body });
}

for (const id of curriculumLessonIds) {
  if (!seenIds.has(id)) {
    // Not an error in Phase 2 (batches land in Phase 5); surface as a warning count.
  }
}

// prerequisites / compare_with resolve (against disk + curriculum ids)
const allKnown = new Set([...curriculumLessonIds, ...seenIds]);
for (const lesson of lessons) {
  for (const pre of lesson.prerequisites ?? []) {
    if (!allKnown.has(pre)) fail(`${lesson.id}: prerequisite "${pre}" does not exist`);
  }
  for (const cmp of lesson.compare_with ?? []) {
    if (!allKnown.has(cmp)) fail(`${lesson.id}: compare_with "${cmp}" does not exist`);
  }
}

// --- emit bundle (with example sources embedded for the reading view)

const files = {};
const expectedFiles = {};
const fixtureFiles = {};
for (const lesson of lessons) {
  for (const ex of lesson.examples ?? []) {
    const srcPath = join(contentDir, lesson._rel, ex.file);
    if (existsSync(srcPath)) {
      files[`${lesson._rel}/${ex.file}`] = readFileSync(srcPath, 'utf8');
    }
    // Expected outputs ride along so the UI can diff run results without a
    // backend round trip.
    const expectedPath = join(contentDir, lesson._rel, 'expected', ex.id + '.txt');
    if (existsSync(expectedPath)) {
      expectedFiles[`${lesson._rel}/${ex.id}`] = readFileSync(expectedPath, 'utf8');
    }
    // Fixtures are needed client-side too: the runner API sends them per run.
    // fx is lesson-relative and already starts with "fixtures/", exactly as
    // in the existence check above.
    for (const fx of ex.fixtures ?? []) {
      const fxPath = join(contentDir, lesson._rel, fx);
      if (existsSync(fxPath)) {
        fixtureFiles[`${lesson._rel}/${fx}`] = readFileSync(fxPath, 'utf8');
      }
    }
  }
}

const bundle = {
  generatedAt: new Date().toISOString(),
  lessonCount: lessons.length,
  lessons: lessons.map(({ _rel, _body, ...fm }) => ({ ...fm, dir: _rel, body: _body })),
  files,
  expected: expectedFiles,
  fixtures: fixtureFiles,
};

if (!quiet) {
  for (const w of warnings) console.warn('WARN:', w);
  for (const e of errors) console.error('ERROR:', e);
}
if (errors.length > 0) {
  console.error(`contentcheck: ${errors.length} error(s)`);
  process.exit(1);
}
if (outFile) {
  writeFileSync(outFile, JSON.stringify(bundle));
  if (!quiet) console.log(`contentcheck OK: ${lessons.length} lesson(s) validated, bundle written to ${outFile}`);
  process.exit(0);
}
writeFileSync(join(contentDir, 'lessons.bundle.json'), JSON.stringify(bundle, null, 2));

// --- frontend data: a light index plus one chunk per lesson
//
// The app imports the index statically (lists, lesson headers, /unverified)
// and each lesson's chunk with import() when its page opens, so Vite gives
// every lesson its own file and the first chunk carries only the index.
// Rejected alternative: one module with every lesson, imported statically.
// That put all prose, sources and expected outputs (about 4.5 MB) into the
// main chunk, parsed before the first page rendered.
//
// Files are rewritten only when their content changes, and nothing in them
// depends on the time of the run: an unchanged lesson must not trigger a Vite
// reload in every open tab.

/**
 * The `<!-- VERIFY: ... -->` claims of one lesson body, with the tab that shows
 * each. Mirrors unverifiedClaims in src/features/lessons/unverified.ts, which
 * unverified.test.ts holds this output to for every lesson. A copy rather than
 * an import: this script runs on plain Node, which can't import TypeScript on
 * every version the project supports.
 */
const VERIFY_COMMENT = /<!--\s*VERIFY:?\s*([\s\S]*?)\s*-->/g;
const TAB_FOR_SECTION = {
  Concept: 'Concept',
  'How it works': 'Concept',
  'Coming from C/C++/Java/Python/JS': 'Coming from…',
  'Design notes': 'Design notes',
};
function unverifiedClaims(body) {
  const claims = [];
  for (const part of body.split(/^## /m)) {
    const heading = part.split('\n', 1)[0].trim();
    for (const m of part.matchAll(VERIFY_COMMENT)) {
      claims.push({ tab: TAB_FOR_SECTION[heading] ?? heading, text: m[1].replace(/\s+/g, ' ') });
    }
  }
  return claims;
}

function writeIfChanged(file, text) {
  if (existsSync(file) && readFileSync(file, 'utf8') === text) return false;
  writeFileSync(file, text);
  return true;
}

const lessonsSrc = join(root, 'src/features/lessons');
const chunkDir = join(lessonsSrc, 'generated');
mkdirSync(chunkDir, { recursive: true });
let changedChunks = 0;
const chunkNames = new Set();
for (const { _rel, _body, ...fm } of lessons) {
  // Keys are lesson-relative: the chunk is one lesson's data, so the
  // directory prefix the shared bundle needs would only repeat lesson.dir.
  const own = (map) =>
    Object.fromEntries(
      Object.entries(map)
        .filter(([key]) => key.startsWith(_rel + '/'))
        .map(([key, value]) => [key.slice(_rel.length + 1), value])
    );
  const chunk = { ...fm, dir: _rel, body: _body, files: own(files), expected: own(expectedFiles), fixtures: own(fixtureFiles) };
  const name = `${fm.id}.json`;
  chunkNames.add(name);
  if (writeIfChanged(join(chunkDir, name), JSON.stringify(chunk) + '\n')) changedChunks++;
}
// A renamed or deleted lesson must not leave a chunk the app could still load.
for (const name of readdirSync(chunkDir)) {
  if (name.endsWith('.json') && !chunkNames.has(name)) {
    rmSync(join(chunkDir, name));
    changedChunks++;
  }
}

const index = lessons.map(({ _body, ...fm }) => ({
  id: fm.id,
  title: fm.title,
  level: fm.level,
  minutes: fm.minutes,
  since: fm.since,
  summary: fm.summary,
  claims: unverifiedClaims(_body),
}));
writeIfChanged(
  join(lessonsSrc, 'lessonIndex.generated.ts'),
  '// Generated from content/ by scripts/build-lessons.mjs. Do not edit.\n' +
    '// Each lesson\'s full content is in generated/<lesson id>.json, loaded on demand.\n' +
    'const index = ' + JSON.stringify(index) + ';\n' +
    'export default index;\n'
);
console.log(
  `contentcheck OK: ${lessons.length} lesson(s) validated, index and lesson chunks written (${changedChunks} chunk(s) changed)`
);
