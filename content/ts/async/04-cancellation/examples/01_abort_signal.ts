// LEARN(ts.async.cancellation): pass one AbortSignal down; abort() and AbortSignal.timeout() reject with different names.

// Stands in for fetch(url, { signal }) and honours the signal the same way.
function fakeFetch(
  url: string,
  ms: number,
  signal: AbortSignal,
): Promise<string> {
  return new Promise((resolve, reject) => {
    signal.throwIfAborted(); // already aborted: reject at once, start nothing
    const onAbort = () => {
      clearTimeout(timer); // stop the work itself, not just the waiting
      reject(signal.reason);
    };
    const timer = setTimeout(() => {
      signal.removeEventListener("abort", onAbort); // done: don't pile up
      resolve(`200 ${url}`);
    }, ms);
    signal.addEventListener("abort", onAbort, { once: true });
  });
}

async function report(label: string, request: Promise<string>): Promise<void> {
  try {
    console.log(`${label}: ${await request}`);
  } catch (e) {
    // A DOMException's name says why the signal fired; a custom reason is
    // whatever value was passed to abort().
    if (e instanceof DOMException) console.log(`${label}: ${e.name}`);
    else console.log(`${label}: ${e instanceof Error ? e.message : String(e)}`);
  }
}

// 1. The caller gives up, e.g. the user left the page.
const user = new AbortController();
const pending = fakeFetch("/report", 5000, user.signal);
user.abort();
await report("abort()", pending);

// 2. A deadline. Its reason is a TimeoutError, not an AbortError.
await report("timeout", fakeFetch("/slow", 5000, AbortSignal.timeout(50)));

// 3. Either one: whichever signal fires first decides the reason.
const page = new AbortController();
const signal = AbortSignal.any([page.signal, AbortSignal.timeout(5000)]);
await report("any, in time", fakeFetch("/fast", 10, signal));
page.abort(new Error("page closed")); // any value can be the reason
await report("any, aborted", fakeFetch("/next", 10, signal));

export {}; // module: top-level await allowed
