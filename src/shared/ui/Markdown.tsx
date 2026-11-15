/**
 * Markdown: renders the small Markdown subset lesson prose uses into React
 * elements: paragraphs, ### / #### headings, - and 1. lists, fenced code, pipe
 * tables, and inline `code`, **bold**, *italic* and [links](https://...). HTML comments are
 * hidden, except `<!-- VERIFY: ... -->`, which renders as a warning (see
 * markdownBlocks.ts).
 *
 * Why hand-written: the subset is fixed by the content style guide, and a
 * parser that only ever creates React elements can't inject HTML from lesson
 * text. Rejected alternative: react-markdown, a dependency tree for a need
 * this small. Underscore emphasis is deliberately unsupported: lesson prose is
 * full of names like search_path and pg_trgm.
 */
import type { ReactNode } from 'react';
import { TriangleAlert } from 'lucide-react';
import { parseBlocks, verifyText } from './markdownBlocks';

// Comments come first: a VERIFY note may itself contain `code` or *stars*.
const INLINE =
  /(<!--[\s\S]*?-->)|`([^`]+)`|\*\*([^*]+)\*\*|\*([^*\s][^*]*)\*|\[([^\]]+)\]\((https?:\/\/[^)\s]+)\)/g;

const UNVERIFIED_LABEL = 'Not confirmed in official docs';

/** Inline Markdown in one string: code, bold, italic, links, VERIFY notes. */
export function Inline({ text }: { text: string }): ReactNode {
  const out: ReactNode[] = [];
  let last = 0;
  for (const m of text.matchAll(INLINE)) {
    const at = m.index ?? 0;
    if (at > last) out.push(text.slice(last, at));
    const key = out.length;
    if (m[1] !== undefined) {
      // A marker inside a sentence or list item: flag it in place.
      const claim = verifyText(m[1]);
      if (claim !== null) {
        out.push(
          <span
            key={key}
            className="mx-0.5 rounded border border-amber-300 bg-amber-50 px-1 py-0.5 text-amber-950 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-100"
            title={UNVERIFIED_LABEL}
          >
            <TriangleAlert className="mr-1 inline h-3.5 w-3.5 align-[-2px] text-amber-600 dark:text-amber-400" aria-label={UNVERIFIED_LABEL} />
            <Inline text={claim} />
          </span>
        );
      }
    } else if (m[2] !== undefined) {
      out.push(
        <code key={key} className="rounded bg-slate-100 px-1 py-0.5 font-mono text-[0.9em] text-slate-800 dark:bg-slate-800 dark:text-slate-200">
          {m[2]}
        </code>
      );
    } else if (m[3] !== undefined) {
      out.push(<strong key={key}>{m[3]}</strong>);
    } else if (m[4] !== undefined) {
      out.push(<em key={key}>{m[4]}</em>);
    } else {
      out.push(
        <a key={key} href={m[6]} target="_blank" rel="noreferrer" className="text-teal-700 underline underline-offset-2 dark:text-teal-400">
          {m[5]}
        </a>
      );
    }
    last = at + m[0].length;
  }
  if (last < text.length) out.push(text.slice(last));
  return <>{out}</>;
}

export function Markdown({ source }: { source: string }) {
  return (
    <div className="space-y-3">
      {parseBlocks(source).map((b, i) => {
        switch (b.kind) {
          case 'heading':
            return b.level === 3 ? (
              <h3 key={i} className="pt-2 font-semibold text-slate-900 dark:text-slate-100">
                <Inline text={b.text} />
              </h3>
            ) : (
              <h4 key={i} className="pt-1 font-medium text-slate-900 dark:text-slate-100">
                <Inline text={b.text} />
              </h4>
            );
          case 'paragraph':
            return (
              <p key={i}>
                <Inline text={b.text} />
              </p>
            );
          case 'list': {
            const items = b.items.map((it, j) => (
              <li key={j}>
                <Inline text={it} />
              </li>
            ));
            return b.ordered ? (
              <ol key={i} className="list-decimal space-y-1.5 pl-5">
                {items}
              </ol>
            ) : (
              <ul key={i} className="list-disc space-y-1.5 pl-5">
                {items}
              </ul>
            );
          }
          case 'code':
            return (
              <pre key={i} className="overflow-x-auto rounded-md bg-slate-900 p-3 text-[13px] leading-6 text-slate-100 dark:ring-1 dark:ring-inset dark:ring-slate-700/70">
                {b.lang === 'mermaid' && <span className="mb-1 block text-xs text-slate-400">diagram source (mermaid)</span>}
                <code>{b.code}</code>
              </pre>
            );
          case 'table':
            return (
              <div key={i} className="overflow-x-auto">
                <table className="w-full border-collapse text-sm">
                  <thead>
                    <tr>
                      {b.header.map((cell, j) => (
                        <th key={j} className="border-b border-slate-300 px-2 py-1.5 text-left align-bottom font-semibold dark:border-slate-600">
                          <Inline text={cell} />
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {b.rows.map((row, r) => (
                      <tr key={r} className="border-b border-slate-100 dark:border-slate-800">
                        {row.map((cell, j) => (
                          <td key={j} className="px-2 py-1.5 align-top">
                            <Inline text={cell} />
                          </td>
                        ))}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            );
          case 'unverified':
            return <UnverifiedNote key={i} text={b.text} />;
          default: {
            const unknown: never = b;
            return unknown;
          }
        }
      })}
    </div>
  );
}

/** A claim the author couldn't confirm in official docs, shown for checking. */
export function UnverifiedNote({ text }: { text: string }) {
  return (
    <aside role="note" className="rounded-lg border border-amber-300 bg-amber-50 px-4 py-3 text-sm leading-relaxed text-amber-950 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-100">
      <p className="flex items-center gap-1.5 font-medium">
        <TriangleAlert className="h-4 w-4 shrink-0 text-amber-600 dark:text-amber-400" aria-hidden />
        {UNVERIFIED_LABEL}: check before relying on this
      </p>
      <p className="mt-1">
        <Inline text={text} />
      </p>
    </aside>
  );
}
