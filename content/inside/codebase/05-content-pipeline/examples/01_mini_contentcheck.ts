// LEARN(inside.codebase.content-pipeline): scripts/build-lessons.mjs in small.
type Files = Record<string, string>;

const curriculumIds = new Set(['go.basics.defer', 'go.basics.loops']);
const OFFICIAL_HOSTS = new Set(['go.dev', 'pkg.go.dev', 'www.postgresql.org']);
const H2_ORDER = ['Concept', 'Coming from C/C++/Java/Python/JS', 'How it works', 'Design notes'];

const content: Files = {
  'go/basics/03-defer/lesson.md':
    '---\nid: go.basics.defer\nreading: https://go.dev/ref/spec\n---\n' +
    '## Concept\n## Coming from C/C++/Java/Python/JS\n## Design notes\n',
  'go/basics/04-loops/lesson.md':
    '---\nid: go.basics.loop\nreading: https://stackoverflow.com/q/1\n---\n' +
    '## Concept\n## Design notes\n## Coming from C/C++/Java/Python/JS\n',
  // \u0028 is "(": the real contentcheck scans this file too and would reject a literal one.
  'go/basics/04-loops/examples/01_range.go': '// LEARN\u0028go.basics.loop): range over int\n',
};

const errors: string[] = [];
const lessons: { id: string; dir: string; body: string }[] = [];

for (const [path, text] of Object.entries(content)) {
  if (!path.endsWith('/lesson.md')) continue;
  const dir = path.slice(0, -'/lesson.md'.length);
  const m = text.match(/^---\r?\n([\s\S]*?)\r?\n---\r?\n?([\s\S]*)$/); // the real regex
  if (!m) { errors.push(`${dir}: no frontmatter block found`); continue; }
  const fm = Object.fromEntries(m[1].split('\n').map((l) => l.split(/: (.*)/s, 2))); // real: the yaml package
  const [track, module, nnSlug] = dir.split('/');
  const want = `${track}.${module}.${nnSlug.replace(/^\d+-/, '')}`;
  if (fm.id !== want) errors.push(`${dir}: id "${fm.id}" must be ${want}`);

  const h2s = [...m[2].matchAll(/^## (.+)$/gm)].map((h) => h[1]);
  const expected = H2_ORDER.filter((h) => h2s.includes(h));
  if (JSON.stringify(h2s) !== JSON.stringify(expected)) errors.push(`${fm.id}: H2 sections out of order`);
  const host = new URL(fm.reading).host;
  if (!OFFICIAL_HOSTS.has(host)) errors.push(`${fm.id}: further_reading must be official docs only, got ${host}`);
  lessons.push({ id: fm.id, dir, body: m[2] });
}
for (const [path, src] of Object.entries(content)) {
  for (const mk of src.matchAll(/LEARN\(([a-z][a-z0-9.-]*)\)/g)) {
    if (!curriculumIds.has(mk[1])) errors.push(`${path}: LEARN(${mk[1]}) names no curriculum lesson`);
  }
}

for (const e of errors) console.log('ERROR:', e);
if (errors.length > 0) {
  console.log(`contentcheck: ${errors.length} error(s); no bundle written`); // real: process.exit(1)
} else {
  console.log(JSON.stringify({ lessonCount: lessons.length, lessons }));
}
