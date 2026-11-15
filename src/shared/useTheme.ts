/**
 * useTheme: the app's one theme store, on the real browser, and the hook
 * that reads it. The app shell's control sets the preference; anything that
 * can't follow the `dark` class by itself (Monaco) reads `resolved`.
 *
 * It lives in shared/ because the theme belongs to no feature: the shell and
 * the runner both need it, and features don't import each other.
 */
import { useSyncExternalStore } from 'react';
import {
  createThemeStore,
  SYSTEM_DARK_QUERY,
  THEME_STORAGE_KEY,
  type ThemePreference,
  type ThemeState,
} from './theme';

const media = window.matchMedia(SYSTEM_DARK_QUERY);

const store = createThemeStore({
  // Inside the store's try/catch: even reading window.localStorage can throw.
  load: () => window.localStorage.getItem(THEME_STORAGE_KEY),
  save: (value) => window.localStorage.setItem(THEME_STORAGE_KEY, value),
  systemIsDark: () => media.matches,
  onSystemChange(callback) {
    media.addEventListener('change', callback);
    return () => media.removeEventListener('change', callback);
  },
  apply(theme) {
    const root = document.documentElement;
    root.classList.toggle('dark', theme === 'dark');
    root.style.colorScheme = theme;
  },
});

export function useTheme(): ThemeState & { setPreference: (preference: ThemePreference) => void } {
  const state = useSyncExternalStore(store.subscribe, store.get);
  return { ...state, setPreference: store.set };
}
