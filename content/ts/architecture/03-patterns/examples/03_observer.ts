// LEARN(ts.architecture.patterns): an observer is a typed event map plus subscribe returning unsubscribe.

type RunEvents = {
  started: { exampleId: string };
  finished: { exampleId: string; ok: boolean };
};

// One generic function instead of Subject/Observer interfaces and their classes.
function createEmitter<E extends Record<string, object>>() {
  const listeners: { [K in keyof E]?: Set<(payload: E[K]) => void> } = {};
  return {
    on<K extends keyof E>(type: K, listener: (payload: E[K]) => void): () => void {
      (listeners[type] ??= new Set()).add(listener);
      return () => listeners[type]?.delete(listener);
    },
    emit<K extends keyof E>(type: K, payload: E[K]): void {
      for (const listener of listeners[type] ?? []) listener(payload);
    },
  };
}

const runs = createEmitter<RunEvents>();
const stopLog = runs.on("started", (e) => console.log(`log     started ${e.exampleId}`));
runs.on("finished", (e) => console.log(`badge   ${e.exampleId}: ${e.ok ? "geçti" : "kaldı"}`));
// runs.emit("finished", { exampleId: "strategy" });
// error TS2741: Property 'ok' is missing in type '{ exampleId: string; }' but required in type '{ exampleId: string; ok: boolean; }'.

runs.emit("started", { exampleId: "strategy" });
runs.emit("finished", { exampleId: "strategy", ok: true });
stopLog(); // unsubscribing is calling the function subscribe returned
runs.emit("started", { exampleId: "observer" });
runs.emit("finished", { exampleId: "observer", ok: false });

// The platform's own observer: EventTarget. Loosely typed, but an AbortSignal
// removes every listener registered with it in one call.
const target = new EventTarget();
const scope = new AbortController();
target.addEventListener(
  "finished",
  (e) => console.log(`target  ${(e as CustomEvent<{ exampleId: string }>).detail.exampleId}`),
  { signal: scope.signal },
);
target.dispatchEvent(new CustomEvent("finished", { detail: { exampleId: "adapter" } }));
scope.abort(); // like a cleanup function for a whole component's listeners
target.dispatchEvent(new CustomEvent("finished", { detail: { exampleId: "unheard" } }));
console.log("done");
