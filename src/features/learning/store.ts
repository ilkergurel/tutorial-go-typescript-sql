/**
 * learningStore: a tiny shared cache of learning state with subscriber
 * notifications. Components read via hooks; mutations update the cache and
 * notify. One store instance per app keeps sidebar badges, the dashboard, and
 * the lesson view consistent without a global state library (ADR-0005
 * preference for boring solutions at this scale).
 */

import { useCallback, useEffect, useState } from 'react';
import type { ProgressRow, ProgressStatus } from './api';
import { pickApi } from './api';

interface LearningState {
  progress: Map<string, ProgressStatus>;
  bookmarks: Set<string>;
  loaded: boolean;
}

let state: LearningState = { progress: new Map(), bookmarks: new Set(), loaded: false };
const listeners = new Set<() => void>();

function notify() {
  for (const l of listeners) l();
}

function subscribe(l: () => void): () => void {
  listeners.add(l);
  return () => listeners.delete(l);
}

/** Loads progress + bookmarks once per session; safe to call from many hooks. */
export function ensureLoaded(): void {
  if (state.loaded) return;
  state.loaded = true;
  void (async () => {
    const api = await pickApi();
    const [progress, bookmarks] = await Promise.all([
      api.progress().catch(() => [] as ProgressRow[]),
      api.bookmarks().catch(() => [] as string[]),
    ]);
    state = {
      loaded: true,
      progress: new Map(progress.map((p) => [p.lessonId, p.status])),
      bookmarks: new Set(bookmarks),
    };
    notify();
  })();
}

export async function markLesson(lessonId: string, to: ProgressStatus): Promise<void> {
  const api = await pickApi();
  const rank: Record<ProgressStatus, number> = { viewed: 0, ran: 1, completed: 2 };
  const cur = state.progress.get(lessonId);
  if (cur && rank[cur] >= rank[to]) return;
  state.progress.set(lessonId, to);
  notify();
  await (to === 'viewed'
    ? api.markViewed(lessonId)
    : to === 'ran'
      ? api.markRan(lessonId)
      : api.markCompleted(lessonId)).catch(() => {
    state.progress.set(lessonId, cur ?? to);
    notify();
  });
}

export async function toggleBookmark(lessonId: string): Promise<void> {
  const api = await pickApi();
  const has = state.bookmarks.has(lessonId);
  if (has) state.bookmarks.delete(lessonId);
  else state.bookmarks.add(lessonId);
  notify();
  const call = has ? api.removeBookmark(lessonId) : api.addBookmark(lessonId);
  await call.catch(() => {
    if (has) state.bookmarks.add(lessonId);
    else state.bookmarks.delete(lessonId);
    notify();
  });
}

export function useProgressStatus(lessonId: string): ProgressStatus | null {
  ensureLoaded();
  const [, force] = useState(0);
  useEffect(() => subscribe(() => force((n) => n + 1)), []);
  return state.progress.get(lessonId) ?? null;
}

export function useAllProgress(): Map<string, ProgressStatus> {
  ensureLoaded();
  const [, force] = useState(0);
  useEffect(() => subscribe(() => force((n) => n + 1)), []);
  return state.progress;
}

export function useIsBookmarked(lessonId: string): { bookmarked: boolean; toggle: () => void } {
  ensureLoaded();
  const [, force] = useState(0);
  useEffect(() => subscribe(() => force((n) => n + 1)), []);
  const toggle = useCallback(() => {
    void toggleBookmark(lessonId);
  }, [lessonId]);
  return { bookmarked: state.bookmarks.has(lessonId), toggle };
}
