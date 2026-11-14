// LEARN(ts.async.promises): await suspends the function; rejection throws in place.
// (Top-level await: fine in browser modules and the lesson runner's iframe/worker.)

function fakeFetch(label: string, ms: number, fail: boolean): Promise<string> {
  return new Promise((resolve, reject) => {
    setTimeout(() => {
      if (fail) reject(new Error(`${label} failed`));
      else resolve(`${label} done`);
    }, ms);
  });
}

async function loadSequential(): Promise<string> {
  try {
    // await suspends THIS function; the event loop keeps running.
    const first = await fakeFetch("first", 20, false);
    const second = await fakeFetch("second", 10, false);
    return `${first}; ${second}`;
  } catch (e) {
    // A rejection from either call lands here. e is unknown: narrow it.
    return e instanceof Error ? `caught: ${e.message}` : "caught: unknown";
  }
}

// The naive "parallel" version: both start, but errors and sequencing are murky.
// Promise.all — the actual tool for this — is the next lesson's topic.
async function loadNaive(): Promise<string[]> {
  const a = fakeFetch("a", 20, false);
  const b = fakeFetch("b", 10, false);
  return [await a, await b];
}

const seq = await loadSequential();
const naive = await loadNaive();
console.log(seq);
console.log(naive.join("; "));

export {}; // marks this file as a module: top-level await is allowed here

