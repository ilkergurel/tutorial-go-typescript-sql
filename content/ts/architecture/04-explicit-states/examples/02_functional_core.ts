// LEARN(ts.architecture.explicit-states): the core decides (state, effects as data); it never performs I/O.

type State =
  | { status: "idle" }
  | { status: "loading"; attempt: number }
  | { status: "backoff"; attempt: number; error: string }
  | { status: "failed"; error: string }
  | { status: "ready"; titles: string[] };

type Event =
  | { type: "open" }
  | { type: "loaded"; attempt: number; titles: string[] }
  | { type: "rejected"; attempt: number; error: string }
  | { type: "timer" };

// Effects are descriptions of I/O, for the shell to carry out.
type Effect = { type: "fetch"; attempt: number } | { type: "sleep"; ms: number };

const MAX_ATTEMPTS = 3;
const load = (attempt: number): [State, Effect[]] => [
  { status: "loading", attempt },
  [{ type: "fetch", attempt }],
];

function step(s: State, e: Event): [State, Effect[]] {
  switch (e.type) {
    case "open":
      return s.status === "idle" || s.status === "failed" ? load(1) : [s, []];
    case "loaded":
      if (s.status !== "loading" || s.attempt !== e.attempt) return [s, []]; // stale reply
      return [{ status: "ready", titles: e.titles }, []];
    case "rejected":
      if (s.status !== "loading" || s.attempt !== e.attempt) return [s, []];
      if (s.attempt === MAX_ATTEMPTS) return [{ status: "failed", error: e.error }, []];
      return [
        { status: "backoff", attempt: s.attempt, error: e.error },
        [{ type: "sleep", ms: 100 * 2 ** (s.attempt - 1) }],
      ];
    case "timer":
      return s.status === "backoff" ? load(s.attempt + 1) : [s, []];
  }
}

// Testing the core is replaying events: no timers, no fetch mocks, no React.
const script: Event[] = [
  { type: "open" },
  { type: "rejected", attempt: 1, error: "503" },
  { type: "timer" },
  { type: "rejected", attempt: 1, error: "late reply" }, // from a request we gave up on
  { type: "loaded", attempt: 2, titles: ["Klasörler", "Durum"] },
  { type: "open" },
];
let state: State = { status: "idle" };
for (const event of script) {
  const [next, effects] = step(state, event);
  const shown = next === state ? "(ignored)" : JSON.stringify(next);
  console.log(event.type.padEnd(9) + [shown, ...effects.map((fx) => JSON.stringify(fx))].join(" "));
  state = next;
}
