/**
 * Curriculum types and helpers.
 *
 * Why a hard-coded union for tracks: the six tracks are fixed by the build prompt,
 * and a string literal union keeps route params and badge colors exhaustive.
 * A generic `string` would silently accept typos like "golang".
 */

export type TrackId = 'go' | 'ts' | 'sql' | 'fullstack' | 'design' | 'inside';

export type Level = 'basic' | 'intermediate' | 'advanced';

export interface Lesson {
  id: string;
  title: string;
  level: Level;
  goal: string;
  runtime: string;
}

export interface Module {
  id: string;
  title: string;
  order: number;
  lessons: Lesson[];
}

export interface Track {
  id: TrackId;
  title: string;
  description: string;
  modules: Module[];
}

export interface Curriculum {
  tracks: Record<TrackId, Track>;
  paths: LearningPath[];
}

/** A learning path: an ordered module sequence across tracks (curriculum.yaml learning_paths). */
export interface LearningPath {
  id: string;
  title: string;
  description: string;
  /** Module ids in walking order; resolved to modules at load time. */
  sequence: Module[];
}

const TRACK_IDS: TrackId[] = ['go', 'ts', 'sql', 'fullstack', 'design', 'inside'];

export function isTrackId(value: string): value is TrackId {
  return (TRACK_IDS as string[]).includes(value);
}

export function trackOrder(): TrackId[] {
  return TRACK_IDS;
}

export function findLesson(curriculum: Curriculum, lessonId: string): { lesson: Lesson; module: Module; track: Track } | null {
  for (const trackId of TRACK_IDS) {
    const track = curriculum.tracks[trackId];
    if (!track) continue;
    for (const module of track.modules) {
      const lesson = module.lessons.find((l) => l.id === lessonId);
      if (lesson) return { lesson, module, track };
    }
  }
  return null;
}

export function countLessons(track: Track): number {
  return track.modules.reduce((n, m) => n + m.lessons.length, 0);
}

export function countPathLessons(path: LearningPath): number {
  return path.sequence.reduce((n, m) => n + m.lessons.length, 0);
}

const LEVEL_ORDER: Record<Level, number> = { basic: 0, intermediate: 1, advanced: 2 };

export function levelRank(level: Level): number {
  return LEVEL_ORDER[level];
}
