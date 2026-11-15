import { useMemo, useState, type ReactNode } from 'react';
import type { Example, Lesson } from './types';
import { LevelBadge } from '../../shared/ui/LevelBadge';
import { ExampleView } from './ExampleView';
import { Inline, Markdown } from '../../shared/ui/Markdown';

type Tab =
  | 'concept'
  | 'coming-from'
  | 'walkthrough'
  | 'use-cases'
  | 'swot'
  | 'pitfalls'
  | 'design'
  | 'quiz'
  | 'notes'
  | 'reading';

const TABS: { id: Tab; label: string }[] = [
  { id: 'concept', label: 'Concept' },
  { id: 'coming-from', label: 'Coming from…' },
  { id: 'walkthrough', label: 'Walkthrough' },
  { id: 'use-cases', label: 'Use cases' },
  { id: 'swot', label: 'Strengths & weaknesses' },
  { id: 'pitfalls', label: 'Pitfalls' },
  { id: 'design', label: 'Design notes' },
  { id: 'quiz', label: 'Quiz' },
  { id: 'notes', label: 'Notes' },
  { id: 'reading', label: 'Further reading' },
];

/**
 * Pull one H2 section's prose out of the lesson body. Splitting on "## " lines
 * instead of one regex: JavaScript has no \Z, and the regex this replaced
 * treated it as a literal "Z", cutting sections off at the first capital Z.
 */
function section(body: string, heading: string): string {
  const found = body.split(/^## /m).find((part) => part.split('\n', 1)[0].trim() === heading);
  return found ? found.slice(found.indexOf('\n') + 1).trim() : '';
}

/**
 * The tabbed lesson view. The parts that belong to other features (the run
 * panel, the quiz, the notes) come in as slots filled by the lesson page, so
 * this feature imports none of them: features meet in app/, never in each
 * other (lesson ts.architecture.folders).
 */
export function LessonContent({ lesson, runPanel, quiz, notes }: {
  lesson: Lesson;
  /** The editor and Run button for the selected example. */
  runPanel: (example: Example) => ReactNode;
  quiz: ReactNode;
  notes: ReactNode;
}) {
  const [tab, setTab] = useState<Tab>('concept');
  const [exampleId, setExampleId] = useState(lesson.examples[0]?.id ?? '');
  const example = useMemo(
    () => lesson.examples.find((e) => e.id === exampleId) ?? lesson.examples[0],
    [lesson, exampleId]
  );

  return (
    <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
      {/* Left: tabbed prose */}
      <section aria-label="Explanation">
        <div className="flex flex-wrap gap-1 border-b border-slate-200 pb-2 dark:border-slate-800" role="tablist">
          {TABS.map((t) => (
            <button
              key={t.id}
              role="tab"
              aria-selected={tab === t.id}
              onClick={() => setTab(t.id)}
              className={`rounded-md px-2.5 py-1.5 text-sm font-medium transition-colors ${
                tab === t.id ? 'bg-teal-700 text-white' : 'text-slate-600 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800 dark:hover:text-slate-100'
              }`}
            >
              {t.label}
            </button>
          ))}
        </div>
        <div className="prose-sm mt-4 max-w-none leading-relaxed text-slate-800 dark:text-slate-200">
          {tab === 'concept' && (
            <>
              <Prose text={section(lesson.body, 'Concept')} />
              {section(lesson.body, 'How it works') && (
                <>
                  <h3 className="mt-6 font-semibold text-slate-900 dark:text-slate-100">How it works</h3>
                  <div className="mt-2">
                    <Prose text={section(lesson.body, 'How it works')} />
                  </div>
                </>
              )}
            </>
          )}
          {tab === 'coming-from' && <Prose text={section(lesson.body, 'Coming from C/C++/Java/Python/JS')} />}
          {tab === 'walkthrough' && example && (
            <ExampleView lesson={lesson} example={example} walkthrough />
          )}
          {tab === 'use-cases' && (
            <ul className="list-disc space-y-1.5 pl-5">
              {lesson.use_cases.map((u) => (
                <li key={u}>
                  <Inline text={u} />
                </li>
              ))}
            </ul>
          )}
          {tab === 'swot' && (
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <h3 className="font-semibold text-emerald-800 dark:text-emerald-400">Strengths</h3>
                <ul className="mt-2 list-disc space-y-1.5 pl-5">
                  {lesson.strengths.map((s) => (
                    <li key={s}>
                      <Inline text={s} />
                    </li>
                  ))}
                </ul>
              </div>
              <div>
                <h3 className="font-semibold text-rose-800 dark:text-rose-400">Weaknesses</h3>
                <ul className="mt-2 list-disc space-y-1.5 pl-5">
                  {lesson.weaknesses.map((w) => (
                    <li key={w}>
                      <Inline text={w} />
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          )}
          {tab === 'pitfalls' && (
            <ul className="space-y-4">
              {lesson.pitfalls.map((p) => (
                <li key={p.title} className="rounded-lg border border-slate-200 bg-white p-4 dark:border-slate-800 dark:bg-slate-900">
                  <p className="font-medium">
                    <Inline text={p.title} />
                  </p>
                  <p className="mt-1 text-slate-700 dark:text-slate-300">
                    <Inline text={p.detail} />
                  </p>
                  {(p.bad || p.good) && (
                    <div className="mt-3 grid gap-2 sm:grid-cols-2">
                      {p.bad && <CodeSample label="Instead of" tone="bad" code={p.bad} />}
                      {p.good && <CodeSample label="Write" tone="good" code={p.good} />}
                    </div>
                  )}
                </li>
              ))}
            </ul>
          )}
          {tab === 'design' && <Prose text={section(lesson.body, 'Design notes')} />}
          {tab === 'quiz' && quiz}
          {tab === 'notes' && notes}
          {tab === 'reading' && (
            <ul className="list-disc space-y-1.5 pl-5">
              {lesson.further_reading.map((r) => (
                <li key={r.url}>
                  <a href={r.url} target="_blank" rel="noreferrer" className="text-teal-700 underline underline-offset-2 dark:text-teal-400">
                    {r.title}
                  </a>
                </li>
              ))}
            </ul>
          )}
        </div>
      </section>

      {/* Right: example code (also shown from the walkthrough tab) */}
      <section aria-label="Example" className={tab === 'walkthrough' ? 'hidden lg:block' : ''}>
        <div className="flex items-center justify-between gap-2">
          <select
            value={exampleId}
            onChange={(e) => setExampleId(e.target.value)}
            aria-label="Example"
            className="rounded-md border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-slate-500 dark:bg-slate-900"
          >
            {lesson.examples.map((e) => (
              <option key={e.id} value={e.id}>
                {e.title}
              </option>
            ))}
          </select>
          <span className="flex items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
            <code className="rounded bg-slate-100 px-1.5 py-0.5 dark:bg-slate-800">{example?.runtime}</code>
            <code className="rounded bg-slate-100 px-1.5 py-0.5 dark:bg-slate-800">since {lesson.since}</code>
            <LevelBadge level={lesson.level} />
          </span>
        </div>
        {example && tab !== 'walkthrough' && runPanel(example)}
      </section>
    </div>
  );
}

function Prose({ text }: { text: string }) {
  if (!text) return <p className="text-slate-500 dark:text-slate-400">This section is empty.</p>;
  return <Markdown source={text} />;
}

function CodeSample({ label, tone, code }: { label: string; tone: 'bad' | 'good'; code: string }) {
  return (
    <div>
      <p className={`text-xs font-medium uppercase tracking-wide ${tone === 'bad' ? 'text-rose-700 dark:text-rose-400' : 'text-emerald-700 dark:text-emerald-400'}`}>
        {label}
      </p>
      <pre className="mt-1 overflow-x-auto rounded-md bg-slate-900 p-2 text-xs leading-5 text-slate-100 dark:bg-slate-950">
        <code>{code}</code>
      </pre>
    </div>
  );
}

