import { useEffect } from 'react';
import { Link, Navigate, useParams } from 'react-router-dom';
import { ArrowRight, Bookmark, BookmarkCheck, CircleCheck, Loader2, TriangleAlert } from 'lucide-react';
import { curriculum } from '../../features/curriculum/load';
import { findLesson } from '../../features/curriculum/types';
import { loadCurriculum } from '../../features/curriculum/load';
import { lessonSummary, useLesson } from '../../features/lessons/load';
import { exampleExtras, exampleSource, type Example, type Lesson, type UnverifiedClaim } from '../../features/lessons/types';
import { LessonContent } from '../../features/lessons/LessonContent';
import { RunPanel } from '../../features/runner/RunPanel';
import type { RunnableExample } from '../../features/runner/types';
import { LessonNotes } from '../../features/learning/LessonNotes';
import { LessonQuiz } from '../../features/learning/LessonQuiz';
import { LevelBadge } from '../../shared/ui/LevelBadge';
import { markLesson, useIsBookmarked, useProgressStatus } from '../../features/learning/store';

/**
 * Lesson view: header (with any unconfirmed-claims warning) plus the tabbed
 * lesson content, whose example panel edits and runs code.
 *
 * This is where the features meet. The header comes from the lesson index,
 * so it renders at once; the content loads as the lesson's own chunk. The
 * run panel, quiz and notes are filled into LessonContent's slots here, so
 * lessons, runner and learning don't import each other.
 */
export function LessonPage() {
  const { lessonId } = useParams();
  const found = lessonId ? findLesson(curriculum, lessonId) : null;
  if (!found) return <Navigate to="/" replace />;
  const { lesson, module, track } = found;
  const content = lessonSummary(lesson.id);

  return (
    <article className="mx-auto max-w-6xl px-8 py-8">
      <header>
        <p className="text-xs font-medium uppercase tracking-wide text-slate-400">
          {track.title} · {module.title}
        </p>
        <div className="mt-1 flex flex-wrap items-center gap-3">
          <h1 className="text-2xl font-semibold tracking-tight">{lesson.title}</h1>
          <LevelBadge level={lesson.level} />
          {content && (
            <span className="text-sm text-slate-500 dark:text-slate-400">
              {content.minutes} min · since {content.since}
            </span>
          )}
        </div>
        {content && <p className="mt-2 max-w-3xl leading-relaxed text-slate-700 dark:text-slate-300">{content.summary}</p>}
        {content && <UnverifiedWarning claims={content.claims} />}
        <LessonActions lessonId={lesson.id} />
      </header>

      {content ? (
        <div className="mt-6">
          <LessonBody key={lesson.id} lessonId={lesson.id} />
        </div>
      ) : (
        <div className="mt-8 rounded-xl border border-amber-200 bg-amber-50 px-5 py-4 text-sm leading-relaxed text-amber-900 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-200">
          This lesson is in the curriculum but isn't in the generated lesson data yet. Run
          <code className="mx-1">node scripts/build-lessons.mjs</code> to rebuild it.
        </div>
      )}

      <NextUp lessonId={lesson.id} />
    </article>
  );
}

/** The lesson's content once its chunk has loaded, with the other features in its slots. */
function LessonBody({ lessonId }: { lessonId: string }) {
  const load = useLesson(lessonId);
  switch (load.kind) {
    case 'loading':
      return (
        <p className="flex items-center gap-2 text-sm text-slate-500 dark:text-slate-400">
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> Loading lesson…
        </p>
      );
    case 'failed':
      return (
        <div role="alert" className="rounded-xl border border-rose-200 bg-rose-50 px-5 py-4 text-sm leading-relaxed text-rose-900 dark:border-rose-800 dark:bg-rose-950/40 dark:text-rose-200">
          The lesson's content didn't load: {load.message}. Reload the page; if it persists, run
          <code className="mx-1">node scripts/build-lessons.mjs</code>.
        </div>
      );
    case 'ready': {
      const { lesson } = load;
      return (
        <LessonContent
          lesson={lesson}
          runPanel={(example) => (
            <RunPanel
              key={example.id}
              example={runnable(lesson, example)}
              onRan={() => void markLesson(lesson.id, 'ran')}
            />
          )}
          quiz={<LessonQuiz key={lesson.id} lessonId={lesson.id} questions={lesson.quiz} />}
          notes={<LessonNotes key={lesson.id} lessonId={lesson.id} lessonTitle={lesson.title} />}
        />
      );
    }
    default: {
      const unknown: never = load;
      throw new Error(`unknown lesson load state ${JSON.stringify(unknown)}`);
    }
  }
}

/** A lesson example in the runner's terms. */
function runnable(lesson: Lesson, example: Example): RunnableExample {
  const { expected, fixtures } = exampleExtras(lesson, example);
  return {
    lessonId: lesson.id,
    exampleId: example.id,
    runtime: example.runtime,
    fileName: example.file.split('/').pop() ?? 'main.go',
    source: exampleSource(lesson, example),
    stdin: example.stdin,
    setup: example.setup,
    fixtures,
    expected,
    check: example.check,
    timeoutS: example.timeout_s,
  };
}

/**
 * Tells the reader up front which tabs hold claims the author couldn't confirm
 * in official docs; the claims themselves are flagged where they appear.
 */
function UnverifiedWarning({ claims }: { claims: UnverifiedClaim[] }) {
  if (claims.length === 0) return null;
  const tabs = [...new Set(claims.map((c) => c.tab))];
  return (
    <p className="mt-3 flex max-w-3xl items-start gap-2 rounded-lg border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-950 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-100">
      <TriangleAlert className="mt-0.5 h-4 w-4 shrink-0 text-amber-600 dark:text-amber-400" aria-hidden />
      <span>
        {claims.length === 1 ? "1 claim in this lesson isn't" : `${claims.length} claims in this lesson aren't`} confirmed
        in official docs ({tabs.length === 1 ? 'tab' : 'tabs'}: {tabs.join(', ')}). They're marked in the
        text;{' '}
        <Link to="/unverified" className="underline underline-offset-2">
          all claims to check
        </Link>
        .
      </span>
    </p>
  );
}

/** "Next up": the next lesson in the default path, else the next in the track. */
function NextUp({ lessonId }: { lessonId: string }) {
  const next = nextLessonAfter(curriculum, lessonId);
  if (!next) return null;
  return (
    <footer className="mt-10 border-t border-slate-200 pt-5 dark:border-slate-800">
      <p className="text-xs font-medium uppercase tracking-wide text-slate-400">Next up</p>
      <Link
        to={`/lessons/${next.id}`}
        className="group mt-2 inline-flex items-center gap-2 rounded-lg px-3 py-2 -ml-3 text-sm font-medium text-teal-800 transition-colors hover:bg-teal-50 dark:text-teal-300 dark:hover:bg-teal-900/30"
      >
        {next.title}
        <ArrowRight className="h-4 w-4 text-slate-300 transition-colors group-hover:text-teal-700 dark:text-slate-600 dark:group-hover:text-teal-400" aria-hidden />
      </Link>
    </footer>
  );
}

function nextLessonAfter(curriculum: ReturnType<typeof loadCurriculum>, lessonId: string) {
  const order = curriculum.paths.find((p) => p.id === 'default')?.sequence ?? [];
  const flat = order.flatMap((m) => m.lessons);
  if (!flat.some((l) => l.id === lessonId)) {
    // Lesson not on the default path: fall back to the next lesson in its track.
    const found = findLesson(curriculum, lessonId);
    if (!found) return null;
    const { module, track } = found;
    const idx = module.lessons.findIndex((l) => l.id === lessonId);
    const inModule = module.lessons[idx + 1];
    if (inModule) return inModule;
    const modIdx = track.modules.findIndex((m) => m.id === module.id);
    return track.modules[modIdx + 1]?.lessons[0] ?? null;
  }
  const idx = flat.findIndex((l) => l.id === lessonId);
  return flat[idx + 1] ?? null;
}

/**
 * Header actions live in their own component so the hooks inside run
 * unconditionally (LessonPage can early-return on an unknown lesson).
 */
function LessonActions({ lessonId }: { lessonId: string }) {
  const status = useProgressStatus(lessonId);
  const { bookmarked, toggle } = useIsBookmarked(lessonId);

  // Opening a lesson records "viewed" (never downgrades ran/completed).
  useEffect(() => {
    void markLesson(lessonId, 'viewed');
  }, [lessonId]);

  return (
    <div className="mt-3 flex items-center gap-2">
      {status && (
        <span
          className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${
            status === 'completed'
              ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/50 dark:text-emerald-200'
              : status === 'ran'
                ? 'bg-sky-100 text-sky-800 dark:bg-sky-900/50 dark:text-sky-200'
                : 'bg-slate-200 text-slate-700 dark:bg-slate-700 dark:text-slate-200'
          }`}
        >
          {status}
        </span>
      )}
      <button
        type="button"
        onClick={toggle}
        aria-pressed={bookmarked}
        className="flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 py-1 text-xs font-medium text-slate-600 transition-colors hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-300 dark:hover:bg-slate-800"
      >
        {bookmarked ? (
          <BookmarkCheck className="h-3.5 w-3.5 text-teal-700 dark:text-teal-400" aria-hidden />
        ) : (
          <Bookmark className="h-3.5 w-3.5" aria-hidden />
        )}
        {bookmarked ? 'Bookmarked' : 'Bookmark'}
      </button>
      <button
        type="button"
        onClick={() => void markLesson(lessonId, 'completed')}
        disabled={status === 'completed'}
        className="flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-2.5 py-1 text-xs font-medium text-slate-600 transition-colors hover:bg-slate-50 dark:border-slate-700 dark:bg-slate-900 dark:text-slate-300 dark:hover:bg-slate-800 disabled:opacity-50"
      >
        <CircleCheck className="h-3.5 w-3.5 text-emerald-700 dark:text-emerald-400" aria-hidden />
        Mark completed
      </button>
    </div>
  );
}
