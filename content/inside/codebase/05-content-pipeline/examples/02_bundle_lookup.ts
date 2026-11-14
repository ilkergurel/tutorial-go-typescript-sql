// The shape of content/lessons.bundle.json, which build-lessons.mjs writes next to
// the frontend's per-lesson files, cut down to one lesson.
const bundle = {
  generatedAt: '2026-11-13T12:25:08.753Z',
  lessonCount: 1,
  lessons: [
    {
      id: 'sql.transactions.isolation-levels',
      title: 'İzolasyon seviyeleri',
      dir: 'sql/transactions/02-isolation-levels',
      examples: [{ id: 'dirty', file: 'examples/01_dirty.sql', fixtures: ['fixtures/seed.csv'] }],
    },
  ],
  files: { 'sql/transactions/02-isolation-levels/examples/01_dirty.sql': 'SELECT 1;' },
  expected: { 'sql/transactions/02-isolation-levels/dirty': 'SELECT 1\n' },
  fixtures: { 'sql/transactions/02-isolation-levels/fixtures/seed.csv': 'id\n1\n' },
};

type Lesson = (typeof bundle.lessons)[number];
const lessonById = (id: string): Lesson | null => bundle.lessons.find((l) => l.id === id) ?? null;

// The sidebar lists curriculum.yaml; content comes from the bundle. The id joins them.
for (const id of ['sql.transactions.isolation-levels', 'sql.transactions.serializable']) {
  const lesson = lessonById(id);
  if (!lesson) {
    console.log(`${id}: in the curriculum, no content -> "not authored yet" box`);
    continue;
  }
  const ex = lesson.examples[0];
  console.log(`${id}: source ${JSON.stringify(bundle.files[`${lesson.dir}/${ex.file}`])},`,
    `expected ${JSON.stringify(bundle.expected[`${lesson.dir}/${ex.id}`])},`,
    `fixture sent as ${JSON.stringify(ex.fixtures[0].replace('fixtures/', ''))}`);
}

// There is no search. The obvious client-side filter meets Turkish casing:
const title = bundle.lessons[0].title;
for (const query of ['izolasyon', 'İZOLASYON', 'IZOLASYON']) {
  console.log(query.padEnd(10),
    'toLowerCase:', title.toLowerCase().includes(query.toLowerCase()),
    '| tr:', title.toLocaleLowerCase('tr').includes(query.toLocaleLowerCase('tr')));
}
console.log([...'İ'.toLowerCase()].map((c) => 'U+' + c.codePointAt(0)!.toString(16).toUpperCase().padStart(4, '0')));
