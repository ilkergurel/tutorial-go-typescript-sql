// LEARN(ts.architecture.state): a copy of server state is a second source of truth, and it drifts.
import { useState, useSyncExternalStore } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

// Stands in for the server-state cache (TanStack Query in lesson ts.react.query).
let done: ReadonlySet<string> = new Set(["ts.react.state"]);
const listeners = new Set<() => void>();
const progress = {
  subscribe(listener: () => void) {
    listeners.add(listener);
    return () => listeners.delete(listener);
  },
  snapshot: () => done,
  markDone(id: string) {
    done = new Set([...done, id]); // a new object, so React sees the change
    for (const listener of listeners) listener();
  },
};
const useDone = () => useSyncExternalStore(progress.subscribe, progress.snapshot);

// Wrong home: the count is copied into component state.
function DashboardCopy() {
  const ids = useDone(); // re-renders on every change...
  const [count] = useState(ids.size); // ...but this argument is read once
  return <p>{`dashboard ${count}`}</p>;
}

// Right home: read the one source, derive the count during render.
function Sidebar() {
  return <p>{`sidebar ${useDone().size}`}</p>;
}

const host = document.createElement("div");
const root = createRoot(host);
const show = (label: string) =>
  console.log(label.padEnd(7), [...host.querySelectorAll("p")].map((p) => p.textContent).join(" | "));

flushSync(() =>
  root.render(
    <>
      <DashboardCopy />
      <Sidebar />
    </>,
  ),
);
show("start");
flushSync(() => progress.markDone("ts.architecture.state"));
show("marked");
root.unmount();
