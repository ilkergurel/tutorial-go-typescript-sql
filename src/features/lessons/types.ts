/**
 * Lesson content types: what scripts/build-lessons.mjs generates from
 * content/ (lesson.schema.json frontmatter plus prose and files).
 *
 * The data comes in two sizes (load.ts): a light index of every lesson for
 * lists and headers, and each lesson's full content, loaded when its page
 * opens. This file holds only types and pure helpers, so other features'
 * code never needs it: the app maps a lesson onto what the runner and the
 * learning feature take (app/routes/LessonPage.tsx).
 */

export type Runtime =
  | 'go'
  | 'go-test'
  | 'go-server'
  | 'sql'
  | 'sql-nontx'
  | 'sql-multisession'
  | 'ts-browser'
  | 'ts-node'
  | 'ts-test';

export interface Annotation {
  lines: string;
  note: string;
}

export interface Example {
  id: string;
  file: string;
  title: string;
  runtime: Runtime;
  fixtures?: string[];
  stdin?: string;
  setup?: string;
  check: 'exact' | 'contains' | 'regex' | 'sorted-lines' | 'none';
  nondeterminism_note?: string;
  timeout_s?: number;
  annotations?: Annotation[];
}

export interface Pitfall {
  title: string;
  detail: string;
  bad?: string;
  good?: string;
}

export interface QuizItem {
  q: string;
  options: string[];
  answer: number;
  why: string;
}

/** One `<!-- VERIFY: ... -->` claim and the lesson tab that shows it. */
export interface UnverifiedClaim {
  tab: string;
  text: string;
}

/** The index entry: everything lists and the lesson header need. */
export interface LessonSummary {
  id: string;
  title: string;
  level: 'basic' | 'intermediate' | 'advanced';
  minutes: number;
  since: string;
  summary: string;
  /** Precomputed at build time, so the dashboard and /unverified need no lesson chunk. */
  claims: UnverifiedClaim[];
}

/** One lesson's full content: generated/<id>.json. */
export interface Lesson {
  id: string;
  title: string;
  track: string;
  module: string;
  order: number;
  level: 'basic' | 'intermediate' | 'advanced';
  minutes: number;
  since: string;
  prerequisites: string[];
  compare_with: string[];
  tags: string[];
  summary: string;
  examples: Example[];
  use_cases: string[];
  strengths: string[];
  weaknesses: string[];
  pitfalls: Pitfall[];
  exercises: { id: string; check: 'output' | 'tests' }[];
  quiz: QuizItem[];
  further_reading: { title: string; url: string }[];
  /** content/<track>/<module>/<NN-slug>, "/"-separated. */
  dir: string;
  body: string;
  /** Example sources keyed by their lesson-relative path (example.file). */
  files: Record<string, string>;
  /** expected/<example id>.txt keyed by example id. */
  expected: Record<string, string>;
  /** Fixture files keyed by their lesson-relative path ("fixtures/..."). */
  fixtures: Record<string, string>;
}

/** The example's source code. */
export function exampleSource(lesson: Lesson, example: Example): string {
  return lesson.files[example.file] ?? '';
}

/**
 * Expected output and fixtures for one example, as the runner takes them:
 * fixtures lose their "fixtures/" prefix, because the program opens them from
 * its working directory. scripts/verify-lessons.mjs builds its requests the
 * same way from the shared bundle.
 */
export function exampleExtras(lesson: Lesson, example: Example): {
  expected: string | null;
  fixtures: Record<string, string> | undefined;
} {
  const expected = lesson.expected[example.id] ?? null;
  const fixtures: Record<string, string> = {};
  let has = false;
  for (const fx of example.fixtures ?? []) {
    if (fx in lesson.fixtures) {
      fixtures[fx.replace('fixtures/', '')] = lesson.fixtures[fx];
      has = true;
    }
  }
  return { expected, fixtures: has ? fixtures : undefined };
}
