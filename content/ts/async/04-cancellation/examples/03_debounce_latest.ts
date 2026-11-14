// LEARN(ts.async.cancellation): each new query aborts the previous request; debounce starts fewer of them.

const CITIES = ["Ankara", "Antalya", "Iğdır", "İstanbul", "İzmir"];

function fakeSearch(query: string, signal: AbortSignal): Promise<string[]> {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      const q = query.toLocaleLowerCase("tr"); // İ -> i, I -> ı
      resolve(CITIES.filter((c) => c.toLocaleLowerCase("tr").startsWith(q)));
    }, 0); // stands in for the network
    signal.addEventListener(
      "abort",
      () => {
        clearTimeout(timer);
        reject(signal.reason);
      },
      { once: true },
    );
  });
}

let current: AbortController | undefined;
async function search(query: string): Promise<void> {
  current?.abort(); // the previous query no longer matters
  const controller = new AbortController();
  current = controller;
  console.log(`search "${query}"`);
  try {
    const hits = await fakeSearch(query, controller.signal);
    console.log(`  "${query}" -> ${hits.join(", ")}`);
  } catch (e) {
    if (!controller.signal.aborted) throw e; // a real failure: let it surface
    console.log(`  "${query}" aborted`);
  }
}

function debounce<A extends unknown[]>(fn: (...args: A) => void, ms: number) {
  let timer: ReturnType<typeof setTimeout> | undefined;
  return (...args: A): void => {
    clearTimeout(timer); // every call restarts the wait
    timer = setTimeout(() => fn(...args), ms);
  };
}

// 1. One request per keystroke: only the last one may render.
await Promise.all(["a", "an", "ant"].map(search));

// 2. Debounced: three fast keystrokes, one request once typing pauses.
const onInput = debounce((query: string) => void search(query), 50);
for (const query of ["i", "iz", "izm"]) onInput(query);

export {}; // module: top-level await allowed
