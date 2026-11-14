// LEARN(ts.architecture.explicit-states): the shell performs effects and feeds results back as events.

// The core from the previous example, condensed into if-chains. Same rules.
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
type Effect = { type: "fetch"; attempt: number } | { type: "sleep"; ms: number };
type Next = [State, Effect[]];
const load = (attempt: number): Next => [{ status: "loading", attempt }, [{ type: "fetch", attempt }]];
function step(s: State, e: Event): Next {
  if (e.type === "open") return s.status === "idle" || s.status === "failed" ? load(1) : [s, []];
  if (e.type === "timer") return s.status === "backoff" ? load(s.attempt + 1) : [s, []];
  if (s.status !== "loading" || s.attempt !== e.attempt) return [s, []];
  if (e.type === "loaded") return [{ status: "ready", titles: e.titles }, []];
  if (s.attempt === 3) return [{ status: "failed", error: e.error }, []];
  const backoff: State = { status: "backoff", attempt: s.attempt, error: e.error };
  return [backoff, [{ type: "sleep", ms: 100 * 2 ** (s.attempt - 1) }]];
}

// The view is part of the core as well: state in, text out (JSX in a real app).
const view = (s: State): string =>
  s.status === "idle" ? "-"
  : s.status === "loading" ? `yükleniyor (deneme ${s.attempt})`
  : s.status === "backoff" ? `${s.error}, tekrar denenecek`
  : s.status === "failed" ? `başarısız: ${s.error}`
  : s.titles.join(", ");

// The shell: the only code that touches the network, timers and the screen.
// It makes no decisions; every one of them came from step().
function start(fetchTitles: (attempt: number) => Promise<string[]>): (e: Event) => void {
  let state: State = { status: "idle" };
  const dispatch = (event: Event) => {
    const [next, effects] = step(state, event);
    if (next !== state) console.log(`${next.status.padEnd(8)}${view(next)}`);
    state = next;
    for (const fx of effects) {
      if (fx.type === "sleep") setTimeout(() => dispatch({ type: "timer" }), fx.ms);
      else
        fetchTitles(fx.attempt).then(
          (titles) => dispatch({ type: "loaded", attempt: fx.attempt, titles }),
          (err: Error) => dispatch({ type: "rejected", attempt: fx.attempt, error: err.message }),
        );
    }
  };
  return dispatch;
}

// A server that fails the first request, like a cold start.
const fakeFetch = async (attempt: number) => {
  if (attempt === 1) throw new Error("503");
  return ["Klasörler", "Durum", "Kalıplar"];
};
start(fakeFetch)({ type: "open" });
