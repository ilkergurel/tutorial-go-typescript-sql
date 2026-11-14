// LEARN(design.architecture.reliability): degrade per dependency, each with its own deadline, and say so.

// A stand-in backend: the lesson answers, recommendations hang, progress is down
// (503), and /healthz still says ok because it checks nothing.
function backend(path: string, signal?: AbortSignal): Promise<Response> {
  switch (path) {
    case "/api/lessons/go.basics.defer":
      return Promise.resolve(Response.json({ title: "Defer" }));
    case "/api/progress":
      return Promise.resolve(new Response("db down", { status: 503 }));
    case "/healthz":
      return Promise.resolve(new Response("ok"));
  }
  return new Promise((_, reject) => {
    const timer = setTimeout(() => reject(new Error("never answers")), 60_000);
    signal?.addEventListener("abort", () => {
      clearTimeout(timer);
      reject(signal.reason);
    });
  });
}

type Loaded<T> =
  | { source: "live"; value: T }
  | { source: "cached" | "default"; value: T; reason: string };

async function load<T>(path: string, ms: number, cached: T | undefined, dflt: T): Promise<Loaded<T>> {
  try {
    const res = await backend(path, AbortSignal.timeout(ms));
    if (!res.ok) throw new Error(`HTTP ${res.status}`); // fetch resolves on 503
    return { source: "live", value: (await res.json()) as T };
  } catch (e) {
    const timedOut = e instanceof DOMException && e.name === "TimeoutError";
    const reason = timedOut ? `no answer in ${ms} ms` : (e as Error).message;
    return cached !== undefined
      ? { source: "cached", value: cached, reason }
      : { source: "default", value: dflt, reason };
  }
}

// The lesson is critical: no fallback, a failure is an error page.
const lesson = (await (await backend("/api/lessons/go.basics.defer")).json()) as { title: string };
// The rest is optional: each call has its own budget, and they run in parallel.
const [recs, progress] = await Promise.all([
  load<string[]>("/api/recommendations", 100, undefined, []),
  load<string[]>("/api/progress", 500, ["go.basics.defer"], []),
]);

console.log("lesson:", lesson.title);
for (const [name, r] of [["recommendations", recs], ["progress", progress]] as const) {
  const why = r.source === "live" ? "" : ` (${r.reason})`;
  console.log(`${name.padEnd(15)} ${r.source.padEnd(7)} ${JSON.stringify(r.value)}${why}`);
}
const degraded = [recs, progress].filter((r) => r.source !== "live").length;
console.log(`banner: ${degraded} of 2 panels degraded`);

// A one-time start-up probe would have said "backend is fine" and never looked again.
const health = await backend("/healthz");
const api = await backend("/api/progress");
console.log("probe /healthz:", health.status, "| /api/progress:", api.status);
