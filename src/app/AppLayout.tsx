import { NavLink, Outlet } from 'react-router-dom';
import { GraduationCap } from 'lucide-react';
import { Sidebar } from './Sidebar';
import { ThemeSwitcher } from './ThemeSwitcher';

/**
 * App shell: fixed sidebar + scrolling content.
 * The sidebar owns track switching and lesson buttons (build prompt §9);
 * the content area is a pure Outlet so routes never re-mount the sidebar.
 * The theme control sits under the sidebar: always there, never in the way.
 */
export function AppLayout() {
  return (
    <div className="flex h-screen bg-slate-50 text-slate-900 dark:bg-slate-950 dark:text-slate-100">
      <div className="flex h-full flex-col border-r border-slate-200 bg-white dark:border-slate-800 dark:bg-slate-900">
        <NavLink
          to="/"
          className="flex items-center gap-2.5 border-b border-slate-200 px-5 py-4 hover:bg-slate-50 dark:border-slate-800 dark:hover:bg-slate-800/60"
          aria-label="StackLab dashboard"
        >
          <GraduationCap className="h-7 w-7 text-teal-700 dark:text-teal-400" aria-hidden />
          <span className="text-lg font-semibold tracking-tight">StackLab</span>
        </NavLink>
        <Sidebar />
        <div className="flex items-center justify-between border-t border-slate-200 px-5 py-2.5 dark:border-slate-800">
          <span className="text-xs font-medium text-slate-500 dark:text-slate-400" aria-hidden>
            Theme
          </span>
          <ThemeSwitcher />
        </div>
      </div>
      <main className="h-full flex-1 overflow-y-auto">
        <Outlet />
      </main>
    </div>
  );
}
