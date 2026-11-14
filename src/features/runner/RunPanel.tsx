/**
 * RunPanel: code editor + Run button + output panel with expected-output
 * diff. The lesson page renders one per selected example, keyed by example,
 * so switching examples mounts a fresh panel (no effect resets its state).
 *
 * UX: Ctrl/Cmd+Enter runs; expected output renders above actual with a
 * pass/fail verdict per the example's check mode. Run history is recorded
 * server-side by the backend (browser TS runs are browser-local by design and
 * skip history — noted in the UI).
 *
 * The props are the runner's own RunnableExample, and recording progress is
 * the onRan callback: the panel knows nothing of lessons or the learning
 * feature, and app/routes/LessonPage.tsx connects them.
 */
import { lazy, Suspense, useCallback, useMemo, useState } from 'react';
import { Play, Loader2, Check, X } from 'lucide-react';
import { runnerFor } from './CodeRunner';
import { judge } from './judge';
import type { RunnableExample, RunResult, RunState, Runtime, StatementResult, StepResult } from './types';

// Monaco is a few MB: loaded with the first panel, not with the app.
const CodeEditor = lazy(() => import('./CodeEditor'));

export function RunPanel({ example, onRan }: { example: RunnableExample; onRan: () => void }) {
  const [code, setCode] = useState(example.source);
  const [stdin, setStdin] = useState(example.stdin ?? '');
  const [state, setState] = useState<RunState>({ kind: 'idle' });

  const run = useCallback(async () => {
    if (state.kind === 'running') return;
    setState({ kind: 'running' });
    try {
      const result = await runnerFor(example.runtime).run({
        runtime: example.runtime,
        example: example.exampleId,
        files: { [example.fileName]: code },
        fixtures: example.fixtures,
        setup: example.setup,
        stdin,
        timeoutS: example.timeoutS,
        lessonId: example.lessonId,
      });
      setState({ kind: 'done', result });
      if (result.exitCode === 0) onRan();
    } catch (err) {
      setState({ kind: 'failed', message: err instanceof Error ? err.message : String(err) });
    }
  }, [code, stdin, example, onRan, state.kind]);

  const lang = example.runtime.startsWith('sql') ? 'sql' : example.runtime.startsWith('ts') ? 'typescript' : 'go';
  const { expected, check } = example;
  const verdict = useMemo(
    () => (state.kind === 'done' ? judge(state.result, check, expected) : null),
    [state, check, expected]
  );

  return (
    <div className="mt-3 overflow-hidden rounded-lg border border-slate-200 bg-white">
      <div className="flex items-center justify-between gap-2 border-b border-slate-200 bg-slate-50 px-3 py-2">
        <span className="text-xs font-medium uppercase tracking-wide text-slate-500">{example.runtime}</span>
        <div className="flex items-center gap-2">
          {verdict && (
            <span
              className={`flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ${
                verdict.pass ? 'bg-emerald-100 text-emerald-800' : 'bg-rose-100 text-rose-800'
              }`}
            >
              {verdict.pass ? <Check className="h-3.5 w-3.5" aria-hidden /> : <X className="h-3.5 w-3.5" aria-hidden />}
              {verdict.pass ? 'matches expected' : 'differs from expected'}
            </span>
          )}
          <button
            type="button"
            onClick={run}
            disabled={state.kind === 'running'}
            className="flex items-center gap-1.5 rounded-md bg-teal-700 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-teal-800 disabled:opacity-50"
          >
            {state.kind === 'running' ? (
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
            ) : (
              <Play className="h-4 w-4" aria-hidden />
            )}
            Run
            <kbd className="ml-1 rounded bg-white/20 px-1 text-[10px]">Ctrl+↵</kbd>
          </button>
        </div>
      </div>

      {example.setup && <SetupSql setup={example.setup} runtime={example.runtime} />}

      <Suspense
        fallback={
          <p className="flex h-[320px] items-center justify-center gap-2 bg-[#1e1e1e] text-sm text-slate-400">
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> Loading editor…
          </p>
        }
      >
        <CodeEditor
          value={code}
          onChange={setCode}
          language={lang}
          path={`file:///${example.lessonId}/${example.exampleId}/${example.fileName}`}
          onRunShortcut={() => void run()}
        />
      </Suspense>

      {example.runtime === 'go' && (
        <textarea
          value={stdin}
          onChange={(e) => setStdin(e.target.value)}
          placeholder="stdin (optional)"
          rows={2}
          className="block w-full border-t border-slate-200 px-3 py-2 font-mono text-xs"
          aria-label="Standard input"
        />
      )}

      <OutputPanel state={state} expected={expected} sqlMode={example.runtime.startsWith('sql')} />
    </div>
  );
}

/**
 * The example's `setup` (frontmatter SQL the runtime executes before the
 * script). Read-only and collapsed: it seeds data the learner needn't read,
 * but its statements show up in the output, so it must be findable.
 */
function SetupSql({ setup, runtime }: { setup: string; runtime: Runtime }) {
  const when =
    runtime === 'sql'
      ? 'in the same transaction, rolled back with it; its results are listed first'
      : runtime === 'sql-nontx'
        ? 'in autocommit mode; its results are listed first'
        : runtime === 'sql-multisession'
          ? 'and commits, so both sessions see it'
          : 'before the code below';
  return (
    <details data-testid="example-setup" className="border-b border-slate-200 bg-slate-50 px-3 py-2 text-xs">
      <summary className="cursor-pointer text-slate-600">
        <span className="font-medium text-slate-700">Setup SQL</span> runs first, {when} (read-only)
      </summary>
      <pre className="mt-2 max-h-56 overflow-auto rounded bg-slate-900 p-2 text-[12px] leading-5 text-slate-100">
        {setup.trimEnd()}
      </pre>
    </details>
  );
}

function OutputPanel({ state, expected, sqlMode }: {
  state: RunState;
  expected: string | null;
  sqlMode: boolean;
}) {
  if (state.kind === 'idle') {
    return <p className="border-t border-slate-200 px-3 py-3 text-sm text-slate-500">Press Run to execute.</p>;
  }
  if (state.kind === 'failed') {
    return (
      <p className="border-t border-slate-200 bg-rose-50 px-3 py-3 text-sm text-rose-800" role="alert">
        {state.message}
      </p>
    );
  }
  if (state.kind === 'running') {
    return (
      <p className="flex items-center gap-2 border-t border-slate-200 px-3 py-3 text-sm text-slate-500">
        <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> Running…
      </p>
    );
  }
  const r = state.result;
  return (
    <div className="border-t border-slate-200">
      {r.timedOut && <p className="bg-amber-50 px-3 py-2 text-sm text-amber-800">Run timed out and was killed.</p>}
      {r.truncated && <p className="bg-amber-50 px-3 py-2 text-sm text-amber-800">Output truncated at 64 KB.</p>}
      {r.steps ? (
        <MultiSessionOutput steps={r.steps} setup={r.sql?.statements ?? []} />
      ) : sqlMode && r.sql ? (
        <SqlOutput statements={r.sql.statements} />
      ) : (
        <TextOutput r={r} expected={expected} />
      )}
      {sqlMode && <SqlTextCompare actual={r.stdout} expected={expected} />}
      <p className="px-3 pb-2 text-xs text-slate-400">finished in {r.durationMs} ms</p>
    </div>
  );
}

function TextOutput({ r, expected }: { r: RunResult; expected: string | null }) {
  const actual = (r.stdout + (r.stderr ? `\n${r.stderr}` : '')).trimEnd();
  return (
    <div className="grid gap-0 md:grid-cols-2">
      {expected !== null && (
        <div className="border-b border-slate-200 p-3 md:border-b-0 md:border-r">
          <p className="text-xs font-medium uppercase tracking-wide text-slate-400">Expected</p>
          <pre className="mt-1 max-h-56 overflow-auto whitespace-pre-wrap text-[13px] leading-6 text-slate-700">{expected}</pre>
        </div>
      )}
      <div className="p-3">
        <p className="text-xs font-medium uppercase tracking-wide text-slate-400">Output (exit {r.exitCode})</p>
        <pre className="mt-1 max-h-56 overflow-auto whitespace-pre-wrap text-[13px] leading-6 text-slate-800">{actual || '(no output)'}</pre>
      </div>
    </div>
  );
}

/**
 * SQL results as text, next to the expected text: the "matches expected"
 * badge compares these two, so a red badge must be checkable line by line.
 */
function SqlTextCompare({ actual, expected }: { actual: string; expected: string | null }) {
  if (expected === null) return null;
  return (
    <details className="border-t border-slate-100 px-3 py-2">
      <summary className="cursor-pointer text-xs text-slate-500">Compare as text with the expected output</summary>
      <div className="mt-2 grid gap-3 md:grid-cols-2">
        {([
          ['Expected', expected],
          ['This run', actual],
        ] as const).map(([label, text]) => (
          <div key={label}>
            <p className="text-xs font-medium uppercase tracking-wide text-slate-400">{label}</p>
            <pre className="mt-1 max-h-72 overflow-auto whitespace-pre text-xs leading-5 text-slate-700">{text.trimEnd()}</pre>
          </div>
        ))}
      </div>
    </details>
  );
}

function SqlOutput({ statements }: { statements: StatementResult[] }) {
  return (
    <div className="space-y-3 p-3">
      {statements.map((s, i) => (
        <div key={i}>
          <p className="font-mono text-xs text-slate-500">
            {i + 1}. {s.error ? 'error' : s.command} · {s.durationMs} ms
          </p>
          <StatementBody s={s} />
        </div>
      ))}
    </div>
  );
}

/** Notices, then the error, the result table, or the affected-row count. */
function StatementBody({ s }: { s: StatementResult }) {
  return (
    <>
      {s.notices.map((n, i) => (
        <p key={i} className="mt-1 font-mono text-xs text-sky-800">{n}</p>
      ))}
      {s.error ? (
        <div className="mt-1 rounded bg-rose-50 px-2 py-1 text-sm text-rose-800">
          <p>
            {s.error.message}
            {/* Refusals and connection errors come from the runner, not the server: no SQLSTATE. */}
            {s.error.code && <span className="text-rose-500"> (SQLSTATE {s.error.code})</span>}
          </p>
          {s.error.detail && <p className="text-xs">DETAIL: {s.error.detail}</p>}
          {s.error.hint && <p className="text-xs">HINT: {s.error.hint}</p>}
        </div>
      ) : s.columns.length > 0 ? (
        <ResultTable s={s} />
      ) : (
        <p className="mt-1 text-xs text-slate-500">{s.command}</p>
      )}
    </>
  );
}

function ResultTable({ s }: { s: StatementResult }) {
  return (
    <div className="mt-1 overflow-x-auto">
      <table className="text-left text-[13px]">
        <thead>
          <tr>
            {s.columns.map((c, ci) => (
              <th key={ci} className="border-b border-slate-200 px-2 py-1 font-medium text-slate-600">
                {c.name} <span className="font-normal text-slate-400">{c.type}</span>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {s.rows.map((row, ri) => (
            <tr key={ri}>
              {row.map((cell, ci) => (
                <td key={ci} className="border-b border-slate-100 px-2 py-1 font-mono text-slate-800">
                  {cell === null ? (
                    <span className="rounded bg-slate-100 px-1 text-xs text-slate-500">NULL</span>
                  ) : cell === '' ? (
                    <span className="text-xs text-slate-300">(empty string)</span>
                  ) : (
                    cell
                  )}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

/**
 * sql-multisession: one row per step in script order, placed in its
 * session's column, so the interleaving reads top to bottom like two
 * terminals. A step that had to wait says when its result arrived.
 */
function MultiSessionOutput({ steps, setup }: { steps: StepResult[]; setup: StatementResult[] }) {
  const failedSetup = setup.find((s) => s.error);
  return (
    <div className="p-3">
      {failedSetup && (
        <div className="mb-3">
          <p className="text-xs font-medium uppercase tracking-wide text-rose-700">Setup failed; no step ran</p>
          <StatementBody s={failedSetup} />
        </div>
      )}
      <div className="grid grid-cols-2 gap-x-3 gap-y-2">
        <p className="text-xs font-medium uppercase tracking-wide text-slate-400">Session A</p>
        <p className="text-xs font-medium uppercase tracking-wide text-slate-400">Session B</p>
        {steps.map((st) => (
          <div
            key={st.step}
            className={`rounded border border-slate-100 bg-slate-50 px-2 py-1.5 ${st.session === 'B' ? 'col-start-2' : 'col-start-1'}`}
          >
            <p className="flex items-center gap-2 text-[11px] text-slate-400">
              step {st.step}
              {st.waited && (
                <span className="rounded bg-amber-100 px-1 text-amber-800">
                  waited · result #{st.finishedOrder}
                </span>
              )}
            </p>
            <pre className="overflow-x-auto whitespace-pre-wrap font-mono text-xs text-slate-700">{st.sql}</pre>
            {failedSetup ? null : <StatementBody s={st} />}
          </div>
        ))}
      </div>
    </div>
  );
}
