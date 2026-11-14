/**
 * Unconfirmed claims: the `<!-- VERIFY: ... -->` markers authors leave in
 * lesson prose when official docs don't confirm a claim (CLAUDE.md rule 7).
 * Collected per lesson, with the tab that shows each one, for the lesson
 * header's warning, the dashboard link and the /unverified checklist page.
 *
 * The marker in the text stays the single source: removing it (and
 * regenerating) is the whole fix. scripts/build-lessons.mjs extracts the
 * claims into the lesson index, so the dashboard and /unverified don't load
 * any lesson's content; unverifiedClaims below is the specification that
 * extraction must match, and unverified.test.ts checks it on every lesson.
 */
import { VERIFY_COMMENT } from '../../shared/ui/markdownBlocks';
import { lessonSummaries } from './load';
import type { LessonSummary, UnverifiedClaim } from './types';

// "How it works" renders under the Concept tab (LessonContent.tsx).
const TAB_FOR_SECTION: Record<string, string> = {
  Concept: 'Concept',
  'How it works': 'Concept',
  'Coming from C/C++/Java/Python/JS': 'Coming from…',
  'Design notes': 'Design notes',
};

/** The claims in one lesson body, in reading order. */
export function unverifiedClaims(body: string): UnverifiedClaim[] {
  const claims: UnverifiedClaim[] = [];
  for (const part of body.split(/^## /m)) {
    const heading = part.split('\n', 1)[0].trim();
    for (const m of part.matchAll(VERIFY_COMMENT)) {
      claims.push({ tab: TAB_FOR_SECTION[heading] ?? heading, text: m[1].replace(/\s+/g, ' ') });
    }
  }
  return claims;
}

/** Every lesson with at least one unconfirmed claim, in index order. */
export function lessonsWithUnverifiedClaims(): LessonSummary[] {
  return lessonSummaries().filter((lesson) => lesson.claims.length > 0);
}
