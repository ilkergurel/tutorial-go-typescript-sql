// LEARN(design.patterns.constructors): a composition root wires once; React context is the injection channel.
import { createContext, useContext, useEffect, useState } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

// The port, declared by the code that uses it.
type Progress = { completed(): Promise<string[]> };

// Factory: a function that closes over its configuration and returns the object.
function httpProgress(opts: { baseUrl: string; fetch: typeof fetch }): Progress {
  return {
    async completed() {
      const res = await opts.fetch(`${opts.baseUrl}/api/progress`);
      if (!res.ok) throw new Error(`GET /api/progress: ${res.status}`);
      const rows: { lessonId: string; status: string }[] = await res.json();
      return rows.filter((r) => r.status === "completed").map((r) => r.lessonId);
    },
  };
}

const ProgressContext = createContext<Progress | null>(null);
function useProgress(): Progress {
  const p = useContext(ProgressContext);
  if (!p) throw new Error("no Progress provided: wire it in the composition root");
  return p;
}

function Completed() {
  const progress = useProgress(); // the component names what it needs, not where it comes from
  const [ids, setIds] = useState<string[] | null>(null);
  useEffect(() => {
    progress.completed().then(setIds);
  }, [progress]);
  return <p>{ids === null ? "loading" : `completed: ${ids.join(", ") || "none"}`}</p>;
}

// Render, then give the effect's promise a few turns to land.
async function mount(progress: Progress, label: string) {
  const el = document.createElement("div");
  const root = createRoot(el);
  flushSync(() =>
    root.render(
      <ProgressContext.Provider value={progress}>
        <Completed />
      </ProgressContext.Provider>,
    ),
  );
  for (let i = 0; i < 10 && el.textContent === "loading"; i++) {
    await new Promise((r) => setTimeout(r, 0));
  }
  console.log(label.padEnd(14), el.textContent);
  root.unmount();
}

const backend: typeof fetch = async () =>
  Response.json([
    { lessonId: "go.basics.defer", status: "completed" },
    { lessonId: "sql.basics.select", status: "viewed" },
  ]);
// Composition roots: the only lines that name concrete implementations.
await mount(httpProgress({ baseUrl: "http://127.0.0.1:8080", fetch: backend }), "production:");
await mount({ completed: async () => ["design.patterns.classic"] }, "test fake:");
