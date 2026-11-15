import { Link } from 'react-router-dom';
import { TriangleAlert } from 'lucide-react';
import { lessonsWithUnverifiedClaims } from '../../features/lessons/unverified';
import { Inline } from '../../shared/ui/Markdown';

/**
 * Checklist of every claim in the lessons that official docs didn't confirm.
 * Each one is also flagged inside its lesson; this page is the to-do list for
 * checking them. A claim leaves the list when its marker leaves the lesson.
 */
export function UnverifiedPage() {
  const entries = lessonsWithUnverifiedClaims();
  const total = entries.reduce((n, lesson) => n + lesson.claims.length, 0);

  return (
    <div className="mx-auto max-w-3xl px-8 py-10">
      <h1 className="flex items-center gap-2 text-2xl font-semibold tracking-tight">
        <TriangleAlert className="h-6 w-6 text-amber-600 dark:text-amber-400" aria-hidden />
        Claims to check
      </h1>
      <p className="mt-2 max-w-2xl leading-relaxed text-slate-600 dark:text-slate-400">
        {total === 0
          ? 'Every claim in the lessons is confirmed by official documentation.'
          : `${total} statement${total === 1 ? '' : 's'} in ${entries.length} lesson${entries.length === 1 ? '' : 's'} couldn't be confirmed in official documentation. Each is also marked with a warning inside its lesson.`}
      </p>

      <ol className="mt-8 space-y-6">
        {entries.map((lesson) => (
          <li key={lesson.id}>
            <Link to={`/lessons/${lesson.id}`} className="font-medium text-teal-800 underline-offset-2 hover:underline dark:text-teal-300">
              {lesson.title}
            </Link>
            <p className="text-xs text-slate-400">{lesson.id}</p>
            <ul className="mt-2 space-y-2">
              {lesson.claims.map((claim, i) => (
                <li key={i} className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-2.5 text-sm leading-relaxed text-amber-950 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-100">
                  <span className="mr-2 rounded bg-amber-100 px-1.5 py-0.5 text-xs font-medium text-amber-800 dark:bg-amber-900/60 dark:text-amber-200">{claim.tab}</span>
                  <Inline text={claim.text} />
                </li>
              ))}
            </ul>
          </li>
        ))}
      </ol>
    </div>
  );
}
