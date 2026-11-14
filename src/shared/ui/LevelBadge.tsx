/**
 * The badge declares the levels it renders instead of importing the
 * curriculum's type: shared/ sits below the features and knows none of them.
 * The curriculum's identical union is assignable, and a fourth level there is
 * a compile error at every call site until it's added here too.
 */
type Level = 'basic' | 'intermediate' | 'advanced';

const STYLES: Record<Level, string> = {
  basic: 'bg-emerald-100 text-emerald-800',
  intermediate: 'bg-amber-100 text-amber-800',
  advanced: 'bg-rose-100 text-rose-800',
};

const LABELS: Record<Level, string> = {
  basic: 'basic',
  intermediate: 'interm.',
  advanced: 'adv.',
};

export function LevelBadge({ level }: { level: Level }) {
  return (
    <span
      className={`shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium leading-4 ${STYLES[level]}`}
    >
      {LABELS[level]}
    </span>
  );
}
