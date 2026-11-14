import { Link } from 'react-router-dom';
import { ArrowRight, Map, PlayCircle, TriangleAlert } from 'lucide-react';
import { curriculum } from '../../features/curriculum/load';
import { countPathLessons, trackOrder } from '../../features/curriculum/types';
import { useAllProgress } from '../../features/learning/store';
import { lessonsWithUnverifiedClaims } from '../../features/lessons/unverified';

/** Points at the checklist of claims official docs didn't confirm, if any. */
function UnverifiedLink() {
  const total = lessonsWithUnverifiedClaims().reduce((n, lesson) => n + lesson.claims.length, 0);
  if (total === 0) return null;
  return (
    <Link
      to="/unverified"
      className="mt-6 flex items-center gap-2 rounded-xl border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-950 transition-colors hover:border-amber-500"
    >
      <TriangleAlert className="h-4 w-4 shrink-0 text-amber-600" aria-hidden />
      {total === 1
        ? "1 claim in the lessons isn't confirmed in official docs yet. See it."
        : `${total} claims in the lessons aren't confirmed in official docs yet. See the list.`}
      <ArrowRight className="ml-auto h-4 w-4 text-amber-600" aria-hidden />
    </Link>
  );
}

/**
 * Dashboard: the catalog entry point — continue card, learning paths, one card
 * per track with true per-track completion counts.
 */
export function DashboardPage() {
  const progress = useAllProgress();
  const next = nextUpLesson(progress);

  return (
    <div className="mx-auto max-w-3xl px-8 py-10">
      <h1 className="text-2xl font-semibold tracking-tight">StackLab</h1>
      <p className="mt-2 max-w-2xl leading-relaxed text-slate-600">
        Learn Go, TypeScript and PostgreSQL through short, runnable lessons. The app's own
        source code is part of the curriculum.
      </p>

      {next && <ContinueCard lessonId={next.id} title={next.title} />}

      <UnverifiedLink />

      <section aria-labelledby="paths-heading" className="mt-8">
        <h2 id="paths-heading" className="text-sm font-semibold uppercase tracking-wide text-slate-500">
          Learning paths
        </h2>
        <div className="mt-3 grid gap-3 sm:grid-cols-2">
          {curriculum.paths.map((path) => (
            <Link
              key={path.id}
              to={`/paths/${path.id}`}
              className="group rounded-xl border border-slate-200 bg-white p-4 transition-colors hover:border-teal-600 hover:bg-teal-50/50"
            >
              <div className="flex items-center justify-between">
                <h3 className="flex items-center gap-2 font-medium">
                  <Map className="h-4 w-4 text-teal-700" aria-hidden />
                  {path.title}
                </h3>
                <ArrowRight
                  className="h-4 w-4 text-slate-300 transition-colors group-hover:text-teal-700"
                  aria-hidden
                />
              </div>
              <p className="mt-1 text-xs uppercase tracking-wide text-slate-400">
                {path.sequence.length} modules · {countPathLessons(path)} lessons
              </p>
            </Link>
          ))}
        </div>
      </section>

      <section aria-labelledby="tracks-heading" className="mt-8">
        <h2 id="tracks-heading" className="text-sm font-semibold uppercase tracking-wide text-slate-500">
          Tracks
        </h2>
        <div className="mt-3 grid gap-4">
        {trackOrder().map((id) => {
          const track = curriculum.tracks[id];
          return (
            <Link
              key={id}
              to={`/tracks/${id}`}
              className="group rounded-xl border border-slate-200 bg-white p-5 transition-colors hover:border-teal-600 hover:bg-teal-50/50"
            >
              <div className="flex items-center justify-between">
                <h2 className="text-lg font-medium">{track.title}</h2>
                <ArrowRight
                  className="h-5 w-5 text-slate-300 transition-colors group-hover:text-teal-700"
                  aria-hidden
                />
              </div>
              <p className="mt-1 text-sm leading-relaxed text-slate-600">{track.description}</p>
              <ProgressCount track={track} />
            </Link>
          );
        })}
        </div>
      </section>
    </div>
  );
}

/** The first not-completed lesson on the default path — "continue here". */
function nextUpLesson(progress: Map<string, string>) {
  const sequence = curriculum.paths.find((p) => p.id === 'default')?.sequence ?? [];
  for (const mod of sequence) {
    const lesson = mod.lessons.find((l) => progress.get(l.id) !== 'completed');
    if (lesson) return lesson;
  }
  return sequence[0]?.lessons[0] ?? null;
}

function ContinueCard({ lessonId, title }: { lessonId: string; title: string }) {
  return (
    <Link
      to={`/lessons/${lessonId}`}
      className="group mt-6 flex items-center justify-between rounded-xl border border-teal-200 bg-teal-50/70 p-4 transition-colors hover:border-teal-600 hover:bg-teal-50"
    >
      <div>
        <p className="text-xs font-medium uppercase tracking-wide text-teal-800">Continue where you left off</p>
        <p className="mt-0.5 font-medium text-teal-950">{title}</p>
      </div>
      <span className="flex items-center gap-1.5 rounded-lg bg-teal-700 px-3 py-1.5 text-sm font-medium text-white transition-colors group-hover:bg-teal-800">
        <PlayCircle className="h-4 w-4" aria-hidden />
        Resume
      </span>
    </Link>
  );
}

function ProgressCount({ track }: {
  track: { modules: { lessons: { id: string }[] }[] };
}) {
  const progress = useAllProgress();
  const total = track.modules.reduce((n, m) => n + m.lessons.length, 0);
  const done = track.modules.reduce(
    (n, m) => n + m.lessons.filter((l) => progress.get(l.id) === 'completed').length,
    0
  );
  return (
    <p className="mt-3 text-xs font-medium uppercase tracking-wide text-slate-400">
      {track.modules.length} modules · {total} lessons
      {done > 0 && <span className="ml-2 text-emerald-700">{done} completed</span>}
    </p>
  );
}
