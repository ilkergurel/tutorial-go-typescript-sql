import { useState } from 'react';
import { NavLink, useParams } from 'react-router-dom';
import { ChevronRight } from 'lucide-react';
import { curriculum } from '../features/curriculum/load';
import type { TrackId } from '../features/curriculum/types';
import { countLessons, isTrackId } from '../features/curriculum/types';
import { LevelBadge } from '../shared/ui/LevelBadge';
import { useAllProgress } from '../features/learning/store';

/**
 * Sidebar: track switcher + collapsible modules, one button per lesson.
 * Only the active track's modules render, keeping ~230 lesson buttons from all
 * being in the DOM at once. Progress badges arrive in Phase 4.
 */
export function Sidebar() {
  const { trackId } = useParams();
  const activeTrackId: TrackId = trackId && isTrackId(trackId) ? trackId : 'go';
  const activeTrack = curriculum.tracks[activeTrackId];
  const progress = useAllProgress();
  const [openModules, setOpenModules] = useState<Set<string>>(
    () => new Set(activeTrack.modules.slice(0, 1).map((m) => m.id))
  );

  const toggle = (id: string) => {
    setOpenModules((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  return (
    <nav className="flex w-72 flex-1 flex-col overflow-y-auto" aria-label="Curriculum">
      <div className="flex flex-wrap gap-1 border-b border-slate-200 px-3 py-3 dark:border-slate-800" role="tablist" aria-label="Tracks">
        {Object.values(curriculum.tracks).map((track) => (
          <NavLink
            key={track.id}
            to={`/tracks/${track.id}`}
            role="tab"
            aria-selected={track.id === activeTrackId}
            className={({ isActive }) =>
              `rounded-md px-2.5 py-1.5 text-sm font-medium transition-colors ${
                isActive
                  ? 'bg-teal-700 text-white'
                  : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900 dark:text-slate-400 dark:hover:bg-slate-800 dark:hover:text-slate-100'
              }`
            }
          >
            {track.title}
          </NavLink>
        ))}
      </div>

      <div className="px-3 py-4 text-xs font-medium uppercase tracking-wide text-slate-400">
        {countLessons(activeTrack)} lessons
      </div>

      <ul className="space-y-1 px-3 pb-6">
        {activeTrack.modules.map((module) => {
          const open = openModules.has(module.id);
          return (
            <li key={module.id}>
              <button
                type="button"
                onClick={() => toggle(module.id)}
                aria-expanded={open}
                className="flex w-full items-center gap-1 rounded-md px-2 py-1.5 text-left text-sm font-medium text-slate-700 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800"
              >
                <ChevronRight
                  className={`h-4 w-4 shrink-0 text-slate-400 transition-transform dark:text-slate-500 ${open ? 'rotate-90' : ''}`}
                  aria-hidden
                />
                {module.title}
              </button>
              {open && (
                <ul className="ml-4 border-l border-slate-200 pl-2 dark:border-slate-700">
                  {module.lessons.map((lesson) => (
                    <li key={lesson.id}>
                      <NavLink
                        to={`/lessons/${lesson.id}`}
                        className={({ isActive }) =>
                          `block rounded-md px-2 py-1.5 text-sm transition-colors ${
                            isActive
                              ? 'bg-teal-50 font-medium text-teal-900 dark:bg-teal-900/40 dark:text-teal-200'
                              : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900 dark:text-slate-400 dark:hover:bg-slate-800 dark:hover:text-slate-100'
                          }`
                        }
                      >
                        <span className="flex items-center justify-between gap-2">
                          <span className="flex min-w-0 items-center gap-1.5">
                            <StatusDot status={progress.get(lesson.id)} />
                            <span className="truncate">{lesson.title}</span>
                          </span>
                          <LevelBadge level={lesson.level} />
                        </span>
                      </NavLink>
                    </li>
                  ))}
                </ul>
              )}
            </li>
          );
        })}
      </ul>
    </nav>
  );
}

function StatusDot({ status }: { status: string | undefined }) {
  if (!status) return <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-slate-200 dark:bg-slate-700" aria-hidden />;
  const color = status === 'completed' ? 'bg-emerald-500' : status === 'ran' ? 'bg-sky-500' : 'bg-slate-300 dark:bg-slate-500';
  return <span className={`h-1.5 w-1.5 shrink-0 rounded-full ${color}`} aria-hidden />;
}
