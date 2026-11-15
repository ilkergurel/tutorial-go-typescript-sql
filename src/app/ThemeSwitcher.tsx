import { Monitor, Moon, Sun, type LucideIcon } from 'lucide-react';
import type { ThemePreference } from '../shared/theme';
import { useTheme } from '../shared/useTheme';

const OPTIONS: { value: ThemePreference; label: string; Icon: LucideIcon }[] = [
  { value: 'dark', label: 'Dark', Icon: Moon },
  { value: 'light', label: 'Light', Icon: Sun },
  { value: 'system', label: 'System', Icon: Monitor },
];

/**
 * Theme control: three toggle buttons, icons only, in the sidebar's footer.
 * Toggle buttons with aria-pressed rather than a radio group: each is one
 * click that applies at once, and a screen reader announces "Dark, toggle
 * button, pressed" without the arrow-key handling a radio group promises.
 */
export function ThemeSwitcher() {
  const { preference, resolved, setPreference } = useTheme();
  return (
    <div
      role="group"
      aria-label="Theme"
      className="flex items-center gap-0.5 rounded-lg border border-slate-200 bg-slate-100 p-0.5 dark:border-slate-700 dark:bg-slate-800"
    >
      {OPTIONS.map(({ value, label, Icon }) => {
        const pressed = preference === value;
        return (
          <button
            key={value}
            type="button"
            aria-pressed={pressed}
            aria-label={label}
            title={value === 'system' ? `System (now ${resolved})` : label}
            onClick={() => setPreference(value)}
            className={`flex h-7 w-7 items-center justify-center rounded-md transition-colors ${
              pressed
                ? 'bg-white text-teal-700 shadow-sm dark:bg-slate-600 dark:text-teal-300'
                : 'text-slate-500 hover:bg-white/70 hover:text-slate-900 dark:text-slate-400 dark:hover:bg-slate-700 dark:hover:text-slate-100'
            }`}
          >
            <Icon className="h-4 w-4" aria-hidden />
          </button>
        );
      })}
    </div>
  );
}
