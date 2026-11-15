import { describe, expect, it } from 'vitest';
import {
  createThemeStore,
  DEFAULT_THEME_PREFERENCE,
  parseThemePreference,
  resolveTheme,
  type ResolvedTheme,
  type ThemeEnvironment,
  type ThemePreference,
} from './theme';

describe('parseThemePreference', () => {
  it('keeps the three known values', () => {
    expect(parseThemePreference('dark')).toBe('dark');
    expect(parseThemePreference('light')).toBe('light');
    expect(parseThemePreference('system')).toBe('system');
  });

  it('falls back to dark for nothing stored or anything else', () => {
    expect(DEFAULT_THEME_PREFERENCE).toBe('dark');
    for (const raw of [null, '', 'Dark', ' dark', '"dark"', 'auto', 'blue']) {
      expect(parseThemePreference(raw), String(raw)).toBe('dark');
    }
  });
});

describe('resolveTheme', () => {
  it('ignores the system setting for an explicit choice', () => {
    for (const systemIsDark of [true, false]) {
      expect(resolveTheme('dark', systemIsDark)).toBe('dark');
      expect(resolveTheme('light', systemIsDark)).toBe('light');
    }
  });

  it('follows the system setting for "system"', () => {
    expect(resolveTheme('system', true)).toBe('dark');
    expect(resolveTheme('system', false)).toBe('light');
  });
});

/** A browser stand-in: a storage slot, a switchable system setting, the applied theme. */
function fakeEnvironment(init: { stored?: string | null; systemIsDark?: boolean; storageThrows?: boolean } = {}) {
  const fake = {
    stored: init.stored ?? null,
    systemIsDark: init.systemIsDark ?? false,
    applied: [] as ResolvedTheme[],
    systemListeners: new Set<() => void>(),
    flipSystem(dark: boolean) {
      fake.systemIsDark = dark;
      for (const callback of fake.systemListeners) callback();
    },
  };
  const env: ThemeEnvironment = {
    load() {
      if (init.storageThrows) throw new DOMException('denied', 'SecurityError');
      return fake.stored;
    },
    save(value: ThemePreference) {
      if (init.storageThrows) throw new DOMException('denied', 'SecurityError');
      fake.stored = value;
    },
    systemIsDark: () => fake.systemIsDark,
    onSystemChange(callback) {
      fake.systemListeners.add(callback);
      return () => fake.systemListeners.delete(callback);
    },
    apply: (theme) => void fake.applied.push(theme),
  };
  return { fake, env };
}

describe('createThemeStore', () => {
  it('starts dark and applies it when nothing valid is stored', () => {
    const { fake, env } = fakeEnvironment({ stored: 'purple', systemIsDark: false });
    const store = createThemeStore(env);
    expect(store.get()).toEqual({ preference: 'dark', resolved: 'dark' });
    expect(fake.applied).toEqual(['dark']);
  });

  it('starts dark when storage throws, and still switches for this page', () => {
    const { fake, env } = fakeEnvironment({ storageThrows: true });
    const store = createThemeStore(env);
    expect(store.get().preference).toBe('dark');
    store.set('light');
    expect(store.get()).toEqual({ preference: 'light', resolved: 'light' });
    expect(fake.applied).toEqual(['dark', 'light']);
  });

  it('saves a choice and tells subscribers', () => {
    const { fake, env } = fakeEnvironment({ stored: 'light' });
    const store = createThemeStore(env);
    let calls = 0;
    store.subscribe(() => calls++);
    store.set('dark');
    expect(fake.stored).toBe('dark');
    expect(store.get()).toEqual({ preference: 'dark', resolved: 'dark' });
    expect(calls).toBe(1);
    store.set('dark');
    expect(calls).toBe(1);
  });

  it('follows the system setting live in "system" mode, and only while subscribed', () => {
    const { fake, env } = fakeEnvironment({ stored: 'system', systemIsDark: true });
    const store = createThemeStore(env);
    expect(store.get().resolved).toBe('dark');

    const unsubscribe = store.subscribe(() => {});
    fake.flipSystem(false);
    expect(store.get()).toEqual({ preference: 'system', resolved: 'light' });
    expect(fake.applied[fake.applied.length - 1]).toBe('light');

    unsubscribe();
    expect(fake.systemListeners.size).toBe(0);
  });

  it('re-reads the system setting when the first subscriber arrives', () => {
    const { fake, env } = fakeEnvironment({ stored: 'system', systemIsDark: false });
    const store = createThemeStore(env);
    fake.systemIsDark = true; // flipped with nobody listening
    store.subscribe(() => {});
    expect(store.get().resolved).toBe('dark');
  });

  it('keeps an explicit choice when the system setting flips', () => {
    const { fake, env } = fakeEnvironment({ stored: 'light', systemIsDark: false });
    const store = createThemeStore(env);
    store.subscribe(() => {});
    fake.flipSystem(true);
    expect(store.get()).toEqual({ preference: 'light', resolved: 'light' });
    expect(fake.applied).toEqual(['light']);
  });
});
