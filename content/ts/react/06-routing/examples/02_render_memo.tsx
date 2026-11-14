// LEARN(ts.react.routing): a render re-renders the children; memo and composition skip work. Measure first.
import { memo, useCallback, useState, type ReactNode } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

const renders = new Map<string, number>();

// Stands in for a component that is expensive to render.
function Chart({ label }: { label: string; onSelect?: () => void }) {
  renders.set(label, (renders.get(label) ?? 0) + 1); // instrumentation only
  return <i>{label}</i>;
}
const MemoChart = memo(Chart); // skips rendering when props are shallow-equal

function Search() {
  const [text, setText] = useState("");
  const inline = () => {}; // a new function on every render
  const stable = useCallback(() => {}, []); // the same function every render
  return (
    <div>
      <button onClick={() => setText(text + "ş")}>type</button>
      <Chart label="plain" />
      <MemoChart label="memo" />
      <MemoChart label="memo + inline fn" onSelect={inline} />
      <MemoChart label="memo + useCallback" onSelect={stable} />
    </div>
  );
}

// Owns state; children arrive as elements its parent already created.
function Panel({ children }: { children: ReactNode }) {
  const [pinned, setPinned] = useState(false);
  return (
    <section data-pinned={pinned}>
      <button onClick={() => setPinned(!pinned)}>pin</button>
      <Chart label="created inside" />
      {children}
    </section>
  );
}

const host = document.createElement("div");
const root = createRoot(host);
flushSync(() =>
  root.render(
    <>
      <Search />
      <Panel>
        <Chart label="passed as children" />
      </Panel>
    </>,
  ),
);
const press = (name: string, times: number) => {
  const button = [...host.querySelectorAll("button")].find(
    (b) => b.textContent === name,
  )!;
  for (let i = 0; i < times; i++) flushSync(() => button.click());
};
press("type", 3);
press("pin", 3);
for (const [label, n] of renders) {
  console.log(`${label.padEnd(20)}${n} render(s)`);
}
root.unmount();
