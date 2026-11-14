import { Link, Navigate, useParams } from 'react-router-dom';
import { CheckCircle2, Circle, PlayCircle } from 'lucide-react';
import { curriculum } from '../../features/curriculum/load';
import { isTrackId } from '../../features/curriculum/types';
import { useAllProgress } from '../../features/learning/store';
import { LevelBadge } from '../../shared/ui/LevelBadge';

/** Track page: the full module/lesson outline for one track, with progress. */
export function TrackPage() {
  const { trackId } = useParams();
  if (!trackId || !isTrackId(trackId)) return <Navigate to="/" replace />;
  const track = curriculum.tracks[trackId];

  return (
    <div className="mx-auto max-w-3xl px-8 py-10">
      <h1 className="text-2xl font-semibold tracking-tight">{track.title}</h1>
      <p className="mt-2 leading-relaxed text-slate-600">{track.description}</p>
      <TrackCounts track={track} />

      <div className="mt-8 space-y-8">
        {track.modules.map((module) => (
          <section key={module.id} aria-label={module.title}>
            <ModuleHeading module={module} />
            <ul className="mt-3 divide-y divide-slate-100 rounded-xl border border-slate-200 bg-white">
              {module.lessons.map((lesson) => (
                <li key={lesson.id}>
                  <LessonLink lessonId={lesson.id} title={lesson.title} goal={lesson.goal} level={lesson.level} />
                </li>
              ))}
            </ul>
          </section>
        ))}
      </div>
    </div>
  );
}

function TrackCounts({ track }: { track: { modules: { lessons: { id: string }[] }[]; title: string } }) {
  const progress = useAllProgress();
  const total = track.modules.reduce((n, m) => n + m.lessons.length, 0);
  const done = track.modules.reduce(
    (n, m) => n + m.lessons.filter((l) => progress.get(l.id) === 'completed').length,
    0
  );
  return (
    <p className="mt-1 text-xs font-medium uppercase tracking-wide text-slate-400">
      {track.modules.length} modules · {total} lessons
      {done > 0 && <span className="ml-2 text-emerald-700">{done} completed</span>}
    </p>
  );
}

function ModuleHeading({ module }: { module: { id: string; title: string; lessons: { id: string }[] } }) {
  const progress = useAllProgress();
  const done = module.lessons.filter((l) => progress.get(l.id) === 'completed').length;
  return (
    <h2 className="flex items-baseline justify-between text-sm font-semibold uppercase tracking-wide text-slate-500">
      {module.title}
      <span className="font-normal normal-case text-slate-400">{done}/{module.lessons.length}</span>
    </h2>
  );
}

function LessonLink({ lessonId, title, goal, level }: {
  lessonId: string;
  title: string;
  goal: string;
  level: 'basic' | 'intermediate' | 'advanced';
}) {
  const status = useAllProgress().get(lessonId) ?? null;
  const Icon = status === 'completed' ? CheckCircle2 : status === 'ran' ? PlayCircle : Circle;
  const color =
    status === 'completed' ? 'text-emerald-600' : status === 'ran' ? 'text-teal-700' : 'text-slate-300';
  return (
    <Link
      to={`/lessons/${lessonId}`}
      className="block px-5 py-3.5 transition-colors hover:bg-slate-50"
    >
      <span className="flex items-center justify-between gap-3">
        <span className="flex min-w-0 items-center gap-2">
          <Icon className={`h-4 w-4 shrink-0 ${color}`} aria-hidden />
          <span className="truncate font-medium">{title}</span>
        </span>
        <LevelBadge level={level} />
      </span>
      <span className="mt-0.5 block pl-6 text-sm text-slate-600">{goal}</span>
    </Link>
  );
}
