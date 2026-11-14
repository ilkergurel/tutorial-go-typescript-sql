/**
 * LessonQuiz: the quiz tab with persisted answers. Choosing an option saves
 * immediately (same debounce-free one-shot as the original) and the reveal
 * logic matches Phase 2's inline quiz, now state survives reloads.
 */
import { useEffect, useState } from 'react';
import { pickApi } from './api';
import { Inline } from '../../shared/ui/Markdown';
import { markLesson } from './store';

/** One question as the lesson frontmatter states it; the lesson's own type fits. */
export interface QuizQuestion {
  q: string;
  options: string[];
  /** Index of the right option. */
  answer: number;
  why: string;
}

/**
 * Render with key={lessonId}: answers load into local state, and without a
 * fresh instance per lesson the previous lesson's answers could be saved
 * under the next lesson's id.
 */
export function LessonQuiz({ lessonId, questions }: { lessonId: string; questions: QuizQuestion[] }) {
  const [answers, setAnswers] = useState<Record<string, number>>({});
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    let live = true;
    void (async () => {
      const api = await pickApi();
      const a = await api.quizAnswers(lessonId).catch(() => ({}));
      if (live) {
        setAnswers(a);
        setLoaded(true);
      }
    })();
    return () => {
      live = false;
    };
  }, [lessonId]);

  const choose = (qi: number, oi: number) => {
    if (answers[String(qi)] !== undefined) return;
    const next = { ...answers, [String(qi)]: oi };
    setAnswers(next);
    void pickApi().then((api) => api.saveQuizAnswers(lessonId, next)).catch(() => {});
    // Answering every question correctly completes the lesson automatically.
    const allRight = questions.every((item, i) => next[String(i)] === item.answer);
    if (allRight) void markLesson(lessonId, 'completed');
  };

  const score = questions.filter((item, i) => answers[String(i)] === item.answer).length;

  return (
    <div>
      {loaded && Object.keys(answers).length > 0 && (
        <p className="mb-4 rounded-lg bg-slate-100 px-3 py-2 text-sm text-slate-700">
          Score so far: {score} / {questions.length}
        </p>
      )}
      <ol className="space-y-5">
        {questions.map((item, qi) => {
          const chosen = answers[String(qi)];
          return (
            <li key={item.q}>
              <p className="font-medium">{item.q}</p>
              <div className="mt-2 space-y-1">
                {item.options.map((opt, oi) => {
                  const isChosen = chosen === oi;
                  const isRight = oi === item.answer;
                  const revealed = chosen !== undefined;
                  return (
                    <button
                      key={opt}
                      type="button"
                      onClick={() => choose(qi, oi)}
                      disabled={revealed}
                      className={`block w-full rounded-md border px-3 py-1.5 text-left text-sm transition-colors ${
                        revealed && isRight
                          ? 'border-emerald-500 bg-emerald-50 text-emerald-900'
                          : isChosen
                            ? 'border-rose-400 bg-rose-50 text-rose-900'
                            : 'border-slate-200 bg-white hover:bg-slate-50'
                      }`}
                    >
                      {opt}
                    </button>
                  );
                })}
              </div>
              {chosen !== undefined && (
                <p className="mt-2 text-sm text-slate-600">
                  <Inline text={item.why} />
                </p>
              )}
            </li>
          );
        })}
      </ol>
    </div>
  );
}
