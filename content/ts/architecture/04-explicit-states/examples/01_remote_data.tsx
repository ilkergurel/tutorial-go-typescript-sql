// LEARN(ts.architecture.explicit-states): an async value is a union of states, each carrying only its own data.
import { renderToStaticMarkup } from "react-dom/server";

// One generic type for every async value in the app.
type Remote<T> =
  | { status: "idle" }
  | { status: "loading"; stale?: T } // stale: the last good value, kept while reloading
  | { status: "failed"; error: string; stale?: T }
  | { status: "ready"; data: T };

// One handler per status. The mapped type makes a missing handler a compile error.
type Handlers<T, R> = {
  [S in Remote<T>["status"]]: (state: Extract<Remote<T>, { status: S }>) => R;
};
function match<T, R>(state: Remote<T>, handlers: Handlers<T, R>): R {
  // tsc can't correlate handlers[state.status] with state's variant, so the
  // one cast lives here, behind a signature that is checked at every call.
  const handler = handlers[state.status] as (state: Remote<T>) => R;
  return handler(state);
}

function Completed({ state }: { state: Remote<number> }) {
  return match(state, {
    idle: () => <p>-</p>,
    loading: (s) => (
      <p aria-busy="true">{s.stale === undefined ? "yükleniyor" : `${s.stale} ders, yenileniyor`}</p>
    ),
    failed: (s) => (
      <p role="alert">{s.stale === undefined ? s.error : `${s.stale} ders (eski: ${s.error})`}</p>
    ),
    ready: (s) => <p>{`${s.data} ders`}</p>,
  });
}
// Leave a status out and the call does not compile:
// match(state, { idle: () => "", loading: () => "", ready: () => "" });
// error TS2741: Property 'failed' is missing in type '{ idle: () => string; loading: () => string; ready: () => string; }' but required in type 'Handlers<number, string>'.

// A first load, a background refresh, a failed refresh, a retry that succeeds.
const timeline: Remote<number>[] = [
  { status: "idle" },
  { status: "loading" },
  { status: "ready", data: 12 },
  { status: "loading", stale: 12 },
  { status: "failed", error: "ağ hatası", stale: 12 },
  { status: "loading", stale: 12 },
  { status: "ready", data: 13 },
];
for (const state of timeline) {
  console.log(state.status.padEnd(8), renderToStaticMarkup(<Completed state={state} />));
}
