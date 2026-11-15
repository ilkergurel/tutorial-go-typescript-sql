import { useMemo, useState } from 'react';
import { ChevronRight } from 'lucide-react';
import type { Example, Lesson } from './types';
import { exampleSource } from './types';
import { Inline } from '../../shared/ui/Markdown';

/**
 * Example display with walkthrough mode: stepping through annotations
 * highlights the referenced lines in the code (Phase 3 replaces the read-only
 * pre with a Monaco editor; the annotation overlay stays).
 */
export function ExampleView({ lesson, example, walkthrough = false }: {
  lesson: Lesson;
  example: Example;
  walkthrough?: boolean;
}) {
  const source = exampleSource(lesson, example);
  const lines = source.replace(/\n$/, '').split('\n');
  const [step, setStep] = useState(0);
  const annotations = example.annotations ?? [];
  const current = walkthrough && annotations.length > 0 ? annotations[Math.min(step, annotations.length - 1)] : null;

  const highlighted = useMemo(() => {
    const set = new Set<number>();
    if (!current) return set;
    const [a, b = a] = current.lines.split('-').map(Number);
    for (let i = a; i <= b; i++) set.add(i);
    return set;
  }, [current]);

  return (
    <figure className="mt-3 overflow-hidden rounded-lg border border-slate-200 bg-slate-950 dark:border-slate-700 dark:bg-slate-900">
      <pre className="overflow-x-auto p-4 text-[13px] leading-6 text-slate-100">
        <code>
          {lines.map((line, i) => {
            const n = i + 1;
            const hit = highlighted.has(n);
            return (
              <span
                key={n}
                className={`-mx-4 block px-4 ${hit ? 'bg-amber-400/25 shadow-[inset_2px_0_0_0_#fbbf24]' : ''}`}
              >
                <span className="mr-4 inline-block w-6 select-none text-right text-slate-500">{n}</span>
                {line}
              </span>
            );
          })}
        </code>
      </pre>
      {walkthrough && annotations.length > 0 && (
        <figcaption className="border-t border-slate-800 bg-slate-900 p-4 text-sm text-slate-200 dark:border-slate-700 dark:bg-slate-800/60">
          <p className="text-xs font-medium uppercase tracking-wide text-slate-400">
            Annotation {step + 1} of {annotations.length} (lines {current?.lines})
          </p>
          <p className="mt-1.5 leading-relaxed">
            <Inline text={current?.note ?? ''} />
          </p>
          <div className="mt-3 flex items-center gap-2">
            <button
              type="button"
              disabled={step === 0}
              onClick={() => setStep((s) => Math.max(0, s - 1))}
              className="rounded-md bg-slate-800 px-3 py-1.5 text-slate-100 disabled:opacity-40 dark:bg-slate-700"
            >
              Previous
            </button>
            <button
              type="button"
              disabled={step >= annotations.length - 1}
              onClick={() => setStep((s) => Math.min(annotations.length - 1, s + 1))}
              className="flex items-center gap-1 rounded-md bg-teal-700 px-3 py-1.5 text-white disabled:opacity-40"
            >
              Next <ChevronRight className="h-4 w-4" aria-hidden />
            </button>
          </div>
        </figcaption>
      )}
    </figure>
  );
}
