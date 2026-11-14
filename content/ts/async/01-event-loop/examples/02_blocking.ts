// LEARN(ts.async.event-loop): a long synchronous loop blocks every timer and event.

function burn(ms: number): void {
  const end = performance.now() + ms;
  while (performance.now() < end) {
    // CPU-bound work: nothing else on this thread runs until the loop ends.
  }
}

type Pause = () => Promise<void>;
const microtask: Pause = () => Promise.resolve();
const task: Pause = () => new Promise((resolve) => setTimeout(resolve, 0));

async function work(label: string, pause: Pause | null): Promise<void> {
  let chunks = 0;
  // Stands in for a click or a network reply that arrives during the work.
  const waiting = new Promise<number>((resolve) =>
    setTimeout(() => resolve(chunks), 0),
  );
  for (let i = 0; i < 3; i++) {
    burn(20);
    chunks++;
    if (pause) await pause();
  }
  console.log(`${label} the timer ran after ${await waiting} of 3 chunks`);
}

await work("no pause:             ", null);
await work("await Promise.resolve:", microtask);
await work("await setTimeout 0:   ", task);

export {}; // module: top-level await allowed
