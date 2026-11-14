/**
 * Curriculum loading.
 *
 * Phase 1 loads the curriculum at build time: scripts/gen-curriculum.mjs compiles
 * content/curriculum.yaml into curriculum.generated.ts, which is imported here.
 * Rejected alternative: fetching the YAML from the backend at runtime — that needs
 * the backend up before the sidebar renders and adds loading states to every page
 * for data that only changes when the repo changes. Phase 2 adds the backend
 * lessons API; this module stays as the fallback and dev-time source of truth.
 */

import data from './curriculum.generated';
import type { Curriculum, LearningPath, Level, Lesson, Module, Track, TrackId } from './types';
import { trackOrder } from './types';

interface RawLesson {
  id: string;
  title: string;
  level: string;
  goal: string;
  runtime: string;
}

interface RawModule {
  id: string;
  title: string;
  order: number;
  lessons: RawLesson[];
}

interface RawTrack {
  title: string;
  description: string;
  modules: RawModule[];
}

function lesson(raw: RawLesson, moduleId: string): Lesson {
  // Defensive at the trust boundary: content is hand-edited YAML, so validate shape
  // here once instead of letting bad frontmatter crash a component render.
  const level = raw.level as Level;
  if (!['basic', 'intermediate', 'advanced'].includes(level)) {
    throw new Error(`lesson ${raw.id} in ${moduleId}: unknown level "${raw.level}"`);
  }
  if (!raw.id.startsWith(moduleId)) {
    throw new Error(`lesson ${raw.id} does not belong under module ${moduleId}`);
  }
  return { id: raw.id, title: raw.title, level, goal: raw.goal, runtime: raw.runtime };
}

function module(raw: RawModule): Module {
  return {
    id: raw.id,
    title: raw.title,
    order: raw.order,
    lessons: raw.lessons.map((l) => lesson(l, raw.id)),
  };
}

function track(id: TrackId, raw: RawTrack): Track {
  return { id, title: raw.title, description: raw.description, modules: raw.modules.map(module) };
}

interface RawPath {
  title: string;
  description?: string;
  sequence: string[];
}

function path(id: string, raw: RawPath, modulesById: Map<string, Module>): LearningPath {
  // Resolve module ids to modules at load time; an unknown id is a curriculum
  // typo and fails loudly here rather than rendering a broken path page.
  const sequence: Module[] = [];
  for (const moduleId of raw.sequence) {
    const mod = modulesById.get(moduleId);
    if (!mod) throw new Error(`learning path "${id}" references unknown module "${moduleId}"`);
    sequence.push(mod);
  }
  return {
    id,
    title: raw.title,
    description: raw.description ?? '',
    sequence,
  };
}

export function loadCurriculum(): Curriculum {
  const parsed = data as { tracks: Record<string, RawTrack>; learning_paths: Record<string, RawPath> };
  const tracks = {} as Record<TrackId, Track>;
  const modulesById = new Map<string, Module>();
  for (const id of trackOrder()) {
    const raw = parsed.tracks[id];
    if (!raw) throw new Error(`curriculum.yaml is missing track "${id}"`);
    tracks[id] = track(id, raw);
    for (const mod of tracks[id].modules) modulesById.set(mod.id, mod);
  }
  const paths = Object.entries(parsed.learning_paths ?? {}).map(([id, raw]) => path(id, raw, modulesById));
  return { tracks, paths };
}

export const curriculum: Curriculum = loadCurriculum();
