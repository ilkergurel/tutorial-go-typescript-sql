/**
 * Lesson data, in two sizes. The index (lessonIndex.generated.ts) is imported
 * statically: every lesson's title, header facts and unconfirmed claims, about
 * 100 KB. Each lesson's full content (generated/<id>.json) is loaded with
 * import() when its page opens, so Vite emits one chunk per lesson and the
 * first chunk doesn't grow with the curriculum.
 *
 * Rejected alternatives: one module with all content, imported statically
 * (4.5 MB in the main chunk, parsed before the first page rendered), and a
 * backend content API (ADR-0004's plan), which would make reading a lesson
 * depend on the backend being up.
 */
import { useEffect, useState } from 'react';
import index from './lessonIndex.generated';
import type { Lesson, LessonSummary } from './types';

const summaries = index as LessonSummary[];
const byId = new Map(summaries.map((s) => [s.id, s]));

// One lazy loader per chunk file, keyed "./generated/<id>.json". The glob is
// resolved at build time, which is what lets Vite split the files.
const chunks = import.meta.glob<Lesson>('./generated/*.json', { import: 'default' });

/** Every lesson, in the order build-lessons walks content/ (track and module folders A-Z). */
export function lessonSummaries(): LessonSummary[] {
  return summaries;
}

/** The index entry, or null for a curriculum lesson nobody has written yet. */
export function lessonSummary(id: string): LessonSummary | null {
  return byId.get(id) ?? null;
}

// Loaded lessons stay in memory: revisiting one renders at once, without a
// loading state. The module cache would dedupe the fetch, but not the flash.
const loaded = new Map<string, Lesson>();

/** Loads one lesson's full content (its chunk). */
export async function loadLesson(id: string): Promise<Lesson> {
  const cached = loaded.get(id);
  if (cached) return cached;
  const load = chunks[`./generated/${id}.json`];
  if (!load) throw new Error(`no content chunk for lesson ${id}: run node scripts/build-lessons.mjs`);
  const lesson = await load();
  loaded.set(id, lesson);
  return lesson;
}

export type LessonLoad =
  | { kind: 'loading' }
  | { kind: 'ready'; lesson: Lesson }
  | { kind: 'failed'; message: string };

/**
 * The lesson's content for a component. The result is tagged with the id it
 * belongs to, so a slow load for the previous lesson can never show up under
 * the next one.
 */
export function useLesson(id: string): LessonLoad {
  const [result, setResult] = useState<{ id: string; load: LessonLoad } | null>(null);

  useEffect(() => {
    if (loaded.has(id)) return;
    let live = true;
    loadLesson(id).then(
      (lesson) => {
        if (live) setResult({ id, load: { kind: 'ready', lesson } });
      },
      // A missing chunk, or (after a rebuild) a chunk URL the server no
      // longer has: the page says so instead of staying on "loading".
      (err: unknown) => {
        if (live) setResult({ id, load: { kind: 'failed', message: err instanceof Error ? err.message : String(err) } });
      }
    );
    return () => {
      live = false;
    };
  }, [id]);

  const cached = loaded.get(id);
  if (cached) return { kind: 'ready', lesson: cached };
  if (result?.id === id) return result.load;
  return { kind: 'loading' };
}
