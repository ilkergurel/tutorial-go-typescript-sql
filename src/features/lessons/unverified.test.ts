import { describe, expect, it } from 'vitest';
import { parseBlocks } from '../../shared/ui/markdownBlocks';
import { lessonSummaries } from './load';
import type { Lesson } from './types';
import { unverifiedClaims } from './unverified';

const COMPLETE_COMMENT = /<!--[\s\S]*?-->/g;

// Every lesson's generated chunk, read eagerly: the app loads them one at a
// time, but these checks guard all of them. As raw text, parsed here: Vitest's
// file-system module cache returns JSON modules it cached as invalid JSON on
// the next run (Vitest 5.1.0), and raw strings are plain JS modules.
const raw = import.meta.glob<string>('./generated/*.json', { eager: true, query: '?raw', import: 'default' });
const chunks = Object.fromEntries(Object.entries(raw).map(([path, text]) => [path, JSON.parse(text) as Lesson]));
const lessons = Object.values(chunks);

describe('unconfirmed claims', () => {
  it('maps each claim to the tab that shows it', () => {
    const body = [
      '## Concept',
      'A. <!-- VERIFY: one -->',
      '## How it works',
      '<!-- VERIFY: two\n  lines -->',
      '## Coming from C/C++/Java/Python/JS',
      '- x <!-- VERIFY: three -->',
      '## Design notes',
      'No claims.',
    ].join('\n');
    expect(unverifiedClaims(body)).toEqual([
      { tab: 'Concept', text: 'one' },
      { tab: 'Concept', text: 'two lines' },
      { tab: 'Coming from…', text: 'three' },
    ]);
  });

  // The index and the chunks are written by the same generator run; a lesson
  // in one but not the other would be a page that can't load, or content no
  // list shows.
  it('the index and the lesson chunks hold the same lessons', () => {
    expect(lessons.length).toBeGreaterThan(0);
    expect(lessonSummaries().map((s) => s.id).sort()).toEqual(lessons.map((l) => l.id).sort());
    for (const [path, lesson] of Object.entries(chunks)) {
      expect(path, lesson.id).toBe(`./generated/${lesson.id}.json`);
    }
  });

  // The dashboard, the lesson header and /unverified read the claims the
  // generator extracted into the index; they must be exactly what the
  // lesson's text holds.
  it("the index's claims are the ones in each lesson's text", () => {
    const byId = new Map(lessons.map((l) => [l.id, l]));
    for (const summary of lessonSummaries()) {
      const lesson = byId.get(summary.id);
      expect(lesson, summary.id).toBeDefined();
      expect(summary.claims, summary.id).toEqual(unverifiedClaims(lesson?.body ?? ''));
    }
  });

  // Guards the real lessons: a marker the parser splits apart (say, a comment
  // whose continuation lines aren't indented inside a list item) would show
  // up as raw "<!--" text instead of a warning, and drop off the checklist.
  it('every marker in every lesson renders as a warning, never as raw text', () => {
    for (const lesson of lessons) {
      const markers = lesson.body.match(/<!--\s*VERIFY/g)?.length ?? 0;
      expect(unverifiedClaims(lesson.body), lesson.id).toHaveLength(markers);

      for (const block of parseBlocks(lesson.body)) {
        const texts = block.kind === 'paragraph' ? [block.text] : block.kind === 'list' ? block.items : [];
        for (const text of texts) {
          expect(text.replace(COMPLETE_COMMENT, ''), lesson.id).not.toMatch(/<!--|-->/);
        }
        // A table the parser missed shows up as a paragraph of pipes.
        if (block.kind === 'paragraph') expect(block.text, lesson.id).not.toMatch(/\|\s*:?-{3,}:?\s*\|/);
      }
    }
  });
});
