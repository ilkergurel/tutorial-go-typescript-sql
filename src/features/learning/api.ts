/**
 * Learning API client: thin fetch wrappers over the backend's learning
 * endpoints. No query library: ADR-0005 chose TanStack Query for server state,
 * but the app never adopted it (see the ADR's 2026-11-14 update). A tiny cache
 * with explicit invalidation (store.ts) keeps this debuggable; TanStack Query
 * replaces it if the app outgrows it.
 */

import { API_BASE } from '../../shared/api';

export type ProgressStatus = 'viewed' | 'ran' | 'completed';

export interface ProgressRow {
  lessonId: string;
  status: ProgressStatus;
  completedAt?: string;
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, init);
  if (!res.ok) throw new Error(`${path}: ${res.statusText}`);
  return (await res.json()) as T;
}

function send(path: string, method: string, body?: unknown): Promise<void> {
  return fetch(`${API_BASE}${path}`, {
    method,
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  }).then((res) => {
    if (!res.ok) throw new Error(`${path}: ${res.statusText}`);
  });
}

export const learningApi = {
  progress: () => req<ProgressRow[]>('/api/progress'),
  markViewed: (lessonId: string) => send(`/api/progress/${lessonId}/viewed`, 'POST'),
  markRan: (lessonId: string) => send(`/api/progress/${lessonId}/ran`, 'POST'),
  markCompleted: (lessonId: string) => send(`/api/progress/${lessonId}/completed`, 'POST'),
  note: (lessonId: string) => req<{ body: string }>(`/api/notes/${lessonId}`),
  saveNote: (lessonId: string, body: string) => send(`/api/notes/${lessonId}`, 'PUT', { body }),
  bookmarks: () => req<string[]>('/api/bookmarks'),
  addBookmark: (lessonId: string) => send(`/api/bookmarks/${lessonId}`, 'POST'),
  removeBookmark: (lessonId: string) => send(`/api/bookmarks/${lessonId}`, 'DELETE'),
  quizAnswers: (lessonId: string) => req<Record<string, number>>(`/api/quiz/${lessonId}`),
  saveQuizAnswers: (lessonId: string, answers: Record<string, number>) =>
    send(`/api/quiz/${lessonId}`, 'PUT', answers),
};

/**
 * Offline fallback: when the backend is unreachable (e.g. frontend started
 * standalone), progress degrades to localStorage instead of breaking the UI.
 * Same shape, same keys — switching storage is invisible to components.
 */
export const localLearningApi = {
  progress: async (): Promise<ProgressRow[]> => {
    const raw = localStorage.getItem('stacklab.progress');
    return raw ? (JSON.parse(raw) as ProgressRow[]) : [];
  },
  markViewed: (lessonId: string) => bumpLocal(lessonId, 'viewed'),
  markRan: (lessonId: string) => bumpLocal(lessonId, 'ran'),
  markCompleted: (lessonId: string) => bumpLocal(lessonId, 'completed'),
  note: async (lessonId: string) => ({ body: localStorage.getItem(`stacklab.note.${lessonId}`) ?? '' }),
  saveNote: async (lessonId: string, body: string) => {
    localStorage.setItem(`stacklab.note.${lessonId}`, body);
  },
  bookmarks: async (): Promise<string[]> => {
    const raw = localStorage.getItem('stacklab.bookmarks');
    return raw ? (JSON.parse(raw) as string[]) : [];
  },
  addBookmark: async (lessonId: string) => toggleBookmark(lessonId, true),
  removeBookmark: async (lessonId: string) => toggleBookmark(lessonId, false),
  quizAnswers: async (lessonId: string) => {
    const raw = localStorage.getItem(`stacklab.quiz.${lessonId}`);
    return raw ? (JSON.parse(raw) as Record<string, number>) : {};
  },
  saveQuizAnswers: async (lessonId: string, answers: Record<string, number>) => {
    localStorage.setItem(`stacklab.quiz.${lessonId}`, JSON.stringify(answers));
  },
};

const RANK: Record<ProgressStatus, number> = { viewed: 0, ran: 1, completed: 2 };

async function bumpLocal(lessonId: string, to: ProgressStatus): Promise<void> {
  const rows = await localLearningApi.progress();
  const cur = rows.find((r) => r.lessonId === lessonId);
  if (cur && RANK[cur.status] >= RANK[to]) return;
  const next = rows.filter((r) => r.lessonId !== lessonId);
  next.push({ lessonId, status: to, completedAt: to === 'completed' ? new Date().toISOString() : cur?.completedAt });
  localStorage.setItem('stacklab.progress', JSON.stringify(next));
}

async function toggleBookmark(lessonId: string, add: boolean): Promise<void> {
  const ids = await localLearningApi.bookmarks();
  const next = add
    ? ids.includes(lessonId)
      ? ids
      : [lessonId, ...ids]
    : ids.filter((id) => id !== lessonId);
  localStorage.setItem('stacklab.bookmarks', JSON.stringify(next));
}

/**
 * Whether the backend answers /healthz. A non-OK status counts as "no backend"
 * too: with the backend down, Vite's dev proxy still answers (500), so
 * fetch resolves and only the status tells.
 */
export async function backendReachable(fetchImpl: typeof fetch = fetch): Promise<boolean> {
  try {
    const res = await fetchImpl(`${API_BASE}/healthz`, { signal: AbortSignal.timeout(1500) });
    return res.ok;
  } catch {
    return false;
  }
}

/** Backend first, localStorage fallback; components never know which won. */
let chosen: typeof learningApi | null = null;
export async function pickApi(): Promise<typeof learningApi> {
  if (chosen) return chosen;
  chosen = (await backendReachable()) ? learningApi : (localLearningApi as unknown as typeof learningApi);
  return chosen;
}
