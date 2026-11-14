// LEARN(ts.async.combinators): a concurrency limit is N workers pulling from one shared index.

async function mapLimit<T, R>(
  items: readonly T[],
  limit: number,
  fn: (item: T) => Promise<R>,
): Promise<R[]> {
  const results: R[] = new Array(items.length);
  let next = 0;
  async function worker(): Promise<void> {
    while (next < items.length) {
      const i = next++; // claimed synchronously: no other worker can take i
      results[i] = await fn(items[i]);
    }
  }
  const workers = Array.from({ length: Math.min(limit, items.length) }, worker);
  await Promise.all(workers); // the first failure rejects; see the lesson text
  return results;
}

let inFlight = 0;
let peak = 0;
async function upload(name: string): Promise<string> {
  inFlight++;
  peak = Math.max(peak, inFlight);
  console.log(`start ${name} (${inFlight} in flight)`);
  await new Promise((resolve) => setTimeout(resolve, 0)); // the network
  inFlight--;
  return `${name}:ok`;
}

const files = ["a.csv", "b.csv", "c.csv", "d.csv", "e.csv"];

const limited = await mapLimit(files, 2, upload);
console.log(`limit 2 -> peak ${peak}: ${limited.join(" ")}`);

peak = 0;
const unbounded = await Promise.all(files.map(upload));
console.log(`Promise.all -> peak ${peak}: ${unbounded.length} results`);

export {}; // module: top-level await allowed
