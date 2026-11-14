/**
 * The "matches expected" verdict for one run, per the example's check mode.
 *
 * Shared by the run panel and scripts/verify-lessons.mjs, which imports this
 * file directly through Node's type stripping. That is why it uses erasable
 * syntax only and imports nothing at runtime: the badge a learner sees and the
 * check that guards every lesson can't drift apart.
 */
import type { Check, RunResult } from './types';

/** null when there is nothing to judge (check "none" or no expected file). */
export function judge(r: RunResult, check: Check, expected: string | null): { pass: boolean } | null {
  if (check === 'none' || expected === null) return null;
  if (r.exitCode !== 0) return { pass: false };
  const actual = normalize(r.stdout, check);
  const want = normalize(expected, check);
  switch (check) {
    case 'exact':
    case 'sorted-lines':
      return { pass: actual === want };
    case 'contains':
      return { pass: want.split('\n').every((line) => actual.includes(line.trim())) };
    case 'regex':
      return { pass: new RegExp(want, 'm').test(actual) };
    default: {
      const unknown: never = check;
      throw new Error(`unknown check mode ${String(unknown)}`);
    }
  }
}

/** Line endings and trailing whitespace never decide a verdict. */
function normalize(s: string, check: Check): string {
  const text = s.replace(/\r\n/g, '\n').trimEnd();
  return check === 'sorted-lines' ? text.split('\n').sort().join('\n') : text;
}
