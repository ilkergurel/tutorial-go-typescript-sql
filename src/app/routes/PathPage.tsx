import { Link, useParams } from 'react-router-dom';
import { CheckCircle2, Circle, CircleDot } from 'lucide-react';
import { curriculum } from '../../features/curriculum/load';
import type { Lesson, Module } from '../../features/curriculum/types';
import type { ProgressStatus } from '../../features/learning/api';
import { useAllProgress, useProgressStatus } from '../../features/learning/store';
import { LevelBadge } from '../../shared/ui/LevelBadge';

/**
 * PathPage: one learning path as a vertical timeline. Modules in walking
 * order, each lesson with its status icon and level badge — the same visual
 * language as the sidebar dots so the states stay consistent.
 */

interface StepStat {
  done: number;
  total: number;
}

function moduleStat(mod: Module, progress: Map<string, ProgressStatus>): StepStat {
  const total = mod.lessons.length;
  const done = mod.lessons.filter((l) => progress.get(l.id) === 'completed').length;
  return { done, total };
}

export function PathPage() {
  const { pathId } = useParams<{ pathId: string }>();
  const progress = useAllProgress();

  const path = curriculum.paths.find((p) => p.id === pathId);
  if (!path) {
    return (
      <div className="mx-auto max-w-3xl px-8 py-10">
        <h1 className="text-2xl font-semibold">Unknown path</h1>
        <p className="mt-2 text-slate-600">
          No learning path with id &ldquo;{pathId}&rdquo;.{' '}
          <Link to="/" className="text-teal-700 hover:underline">
            Back to dashboard
          </Link>
        </p>
      </div>
    );
  }

  return (
    <article className="mx-auto max-w-3xl px-8 py-10">
      <header className="mb-8">
        <p className="text-sm font-semibold uppercase tracking-wide text-teal-700">
          Learning path
        </p>
        <h1 className="mt-1 text-2xl font-semibold tracking-tight">{path.title}</h1>
        {path.description && (
          <p className="mt-2 max-w-2xl leading-relaxed text-slate-600">{path.description}</p>
        )}
      </header>

      <ol className="relative border-l border-slate-200 pl-6">
        {path.sequence.map((mod) => {
          const stat = moduleStat(mod, progress);
          const complete = stat.total > 0 && stat.done === stat.total;
          return (
            <li key={mod.id} className="relative mb-10">
              <span
                aria-label={complete ? 'module complete' : 'module in progress'}
                className={`absolute -left-[31px] flex h-5 w-5 items-center justify-center rounded-full border bg-white ${
                  complete ? 'border-teal-600 text-teal-700' : 'border-slate-300 text-slate-400'
                }`}
              >
                {complete ? <CheckCircle2 className="h-3.5 w-3.5" /> : <Circle className="h-3 w-3" />}
              </span>

              <div className="flex items-baseline justify-between gap-4">
                <div>
                  <p className="text-xs font-medium uppercase tracking-wide text-slate-400">
                    {mod.id.split('.')[0]}
                  </p>
                  <h2 className="text-lg font-medium">
                    <Link to={`/tracks/${mod.id.split('.')[0]}`} className="hover:text-teal-700">
                      {mod.title}
                    </Link>
                  </h2>
                </div>
                <span className="whitespace-nowrap text-sm text-slate-500">
                  {stat.done}/{stat.total} done
                </span>
              </div>

              <ul className="mt-3 space-y-1">
                {mod.lessons.map((lesson) => (
                  <LessonRow key={lesson.id} lesson={lesson} />
                ))}
              </ul>
            </li>
          );
        })}
      </ol>
    </article>
  );
}

function LessonRow({ lesson }: { lesson: Lesson }) {
  const status = useProgressStatus(lesson.id);
  return (
    <li>
      <Link
        to={`/lessons/${lesson.id}`}
        className="group flex items-center gap-3 rounded-lg px-2 py-1.5 transition-colors hover:bg-slate-50"
      >
        <StatusIcon status={status} />
        <span className="flex-1 text-sm group-hover:text-teal-800">{lesson.title}</span>
        <LevelBadge level={lesson.level} />
      </Link>
    </li>
  );
}

function StatusIcon({ status }: { status: ProgressStatus | null }) {
  if (status === 'completed') return <CheckCircle2 className="h-4 w-4 shrink-0 text-teal-700" />;
  if (status === 'ran') return <CircleDot className="h-4 w-4 shrink-0 text-sky-600" />;
  if (status === 'viewed') return <CircleDot className="h-4 w-4 shrink-0 text-slate-400" />;
  return <Circle className="h-4 w-4 shrink-0 text-slate-300" />;
}
