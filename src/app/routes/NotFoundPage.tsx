import { Link } from 'react-router-dom';

export function NotFoundPage() {
  return (
    <div className="mx-auto max-w-3xl px-8 py-10">
      <h1 className="text-2xl font-semibold tracking-tight">Not found</h1>
      <p className="mt-2 text-slate-600 dark:text-slate-400">
        That track or lesson does not exist. <Link to="/" className="text-teal-700 underline underline-offset-2 dark:text-teal-400">Back to the dashboard</Link>.
      </p>
    </div>
  );
}
