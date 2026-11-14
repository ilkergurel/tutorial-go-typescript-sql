import { NavLink, Outlet } from 'react-router-dom';
import { GraduationCap } from 'lucide-react';
import { Sidebar } from './Sidebar';

/**
 * App shell: fixed sidebar + scrolling content.
 * The sidebar owns track switching and lesson buttons (build prompt §9);
 * the content area is a pure Outlet so routes never re-mount the sidebar.
 */
export function AppLayout() {
  return (
    <div className="flex h-screen bg-slate-50 text-slate-900">
      <div className="flex h-full flex-col border-r border-slate-200 bg-white">
        <NavLink
          to="/"
          className="flex items-center gap-2.5 border-b border-slate-200 px-5 py-4 hover:bg-slate-50"
          aria-label="StackLab dashboard"
        >
          <GraduationCap className="h-7 w-7 text-teal-700" aria-hidden />
          <span className="text-lg font-semibold tracking-tight">StackLab</span>
        </NavLink>
        <Sidebar />
      </div>
      <main className="h-full flex-1 overflow-y-auto">
        <Outlet />
      </main>
    </div>
  );
}
