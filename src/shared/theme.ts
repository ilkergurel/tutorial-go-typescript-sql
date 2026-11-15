/**
 * Theme preference: dark, light, or whatever the operating system says.
 *
 * The page is themed by one `dark` class on <html> (Tailwind's class
 * strategy) plus `color-scheme`, which makes scrollbars and form controls
 * match. This module holds the rules and a store built on an injected
 * browser environment, so the tests can drive it without a DOM;
 * useTheme.ts wires it to the real localStorage, matchMedia and <html>.
 *
 * Why a class and not only the `prefers-color-scheme` media query: the owner
 * asked for dark by default whatever the system says, and a manual choice
 * needs something the page controls. The rejected alternative was a React
 * context: the class lives outside React anyway, and the one value it holds
 * changes rarely, so a small store read with useSyncExternalStore is enough.
 *
 * index.html's inline script repeats parseThemePreference and resolveTheme
 * in plain JavaScript, to set the class before the first paint. Change the
 * key, the values or the default here and there together.
 */

export type ThemePreference = 'dark' | 'light' | 'system';
export type ResolvedTheme = 'dark' | 'light';

export const THEME_STORAGE_KEY = 'stacklab-theme';
/** The owner asked for dark; "system" is opt-in. */
export const DEFAULT_THEME_PREFERENCE: ThemePreference = 'dark';
export const SYSTEM_DARK_QUERY = '(prefers-color-scheme: dark)';

/** A stored value back into a preference; anything unknown is the default. */
export function parseThemePreference(raw: string | null): ThemePreference {
  switch (raw) {
    case 'dark':
    case 'light':
    case 'system':
      return raw;
    default:
      return DEFAULT_THEME_PREFERENCE;
  }
}

export function resolveTheme(preference: ThemePreference, systemIsDark: boolean): ResolvedTheme {
  switch (preference) {
    case 'dark':
    case 'light':
      return preference;
    case 'system':
      return systemIsDark ? 'dark' : 'light';
    default: {
      const unknown: never = preference;
      throw new Error(`unknown theme preference ${JSON.stringify(unknown)}`);
    }
  }
}

/** What the store needs from the browser. load and save may throw. */
export interface ThemeEnvironment {
  load(): string | null;
  save(value: ThemePreference): void;
  systemIsDark(): boolean;
  /** Calls back when the system setting flips; returns the unsubscribe. */
  onSystemChange(callback: () => void): () => void;
  apply(theme: ResolvedTheme): void;
}

export interface ThemeState {
  preference: ThemePreference;
  resolved: ResolvedTheme;
}

export interface ThemeStore {
  get(): ThemeState;
  set(preference: ThemePreference): void;
  subscribe(listener: () => void): () => void;
}

export function createThemeStore(env: ThemeEnvironment): ThemeStore {
  const stateFor = (preference: ThemePreference): ThemeState => ({
    preference,
    resolved: resolveTheme(preference, env.systemIsDark()),
  });

  let stored: string | null = null;
  try {
    stored = env.load();
  } catch {
    // Storage can be off (blocked site data, some private windows): the
    // default still applies, and a choice made now lasts until reload.
  }
  let state = stateFor(parseThemePreference(stored));
  // index.html's script has normally done this already; applying again keeps
  // the page and the store in step if that script ever didn't run.
  env.apply(state.resolved);

  const listeners = new Set<() => void>();
  let stopSystem: (() => void) | null = null;

  // A new object only when something changed: useSyncExternalStore
  // re-renders whenever the snapshot's identity changes.
  const update = (next: ThemeState) => {
    if (next.preference === state.preference && next.resolved === state.resolved) return;
    state = next;
    env.apply(state.resolved);
    for (const listener of listeners) listener();
  };

  return {
    get: () => state,
    set(preference) {
      try {
        env.save(preference);
      } catch {
        // Not saved; still applied for this page (see the load above).
      }
      update(stateFor(preference));
    },
    // The system setting is watched while anything subscribes, the way
    // react.dev's useSyncExternalStore examples watch online/offline; the
    // theme control in the app shell always does. The first subscriber
    // re-reads it, in case it flipped while nobody was watching.
    subscribe(listener) {
      listeners.add(listener);
      if (stopSystem === null) {
        stopSystem = env.onSystemChange(() => update(stateFor(state.preference)));
        update(stateFor(state.preference));
      }
      return () => {
        listeners.delete(listener);
        if (listeners.size === 0 && stopSystem !== null) {
          stopSystem();
          stopSystem = null;
        }
      };
    },
  };
}
