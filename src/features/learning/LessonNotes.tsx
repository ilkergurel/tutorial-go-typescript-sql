/**
 * LessonNotes: a private per-lesson notes editor. Saves on a debounce so typing
 * feels live; "Saved" appears briefly after each write settles.
 */
import { useEffect, useRef, useState } from 'react';
import { NotebookPen } from 'lucide-react';
import { pickApi } from './api';

/**
 * Render with key={lessonId}: the note loads into local state, and the
 * debounced save must never write one lesson's text under another's id.
 */
export function LessonNotes({ lessonId, lessonTitle }: { lessonId: string; lessonTitle: string }) {
  const [body, setBody] = useState('');
  const [loaded, setLoaded] = useState(false);
  const [saved, setSaved] = useState<'no' | 'saving' | 'yes'>('no');
  const timer = useRef<number | undefined>(undefined);

  useEffect(() => {
    let live = true;
    void (async () => {
      const api = await pickApi();
      const { body } = await api.note(lessonId).catch(() => ({ body: '' }));
      if (live) {
        setBody(body);
        setLoaded(true);
      }
    })();
    return () => {
      live = false;
      window.clearTimeout(timer.current);
    };
  }, [lessonId]);

  const onChange = (v: string) => {
    setBody(v);
    setSaved('saving');
    window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => {
      void pickApi()
        .then((api) => api.saveNote(lessonId, v))
        .then(() => setSaved('yes'))
        .catch(() => setSaved('no'));
    }, 600);
  };

  return (
    <div>
      <div className="flex items-center gap-2 text-xs text-slate-400">
        <NotebookPen className="h-3.5 w-3.5" aria-hidden />
        Private notes, saved automatically
        {saved === 'saving' && <span>saving…</span>}
        {saved === 'yes' && <span className="text-emerald-600">saved</span>}
      </div>
      <textarea
        value={body}
        onChange={(e) => onChange(e.target.value)}
        disabled={!loaded}
        rows={6}
        placeholder="What clicked? What didn't?"
        aria-label={`Notes for ${lessonTitle}`}
        className="mt-2 block w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm leading-relaxed focus:border-teal-600 focus:outline-none"
      />
    </div>
  );
}
