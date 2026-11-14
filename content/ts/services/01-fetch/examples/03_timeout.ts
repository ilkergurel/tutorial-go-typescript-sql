// LEARN(ts.services.fetch): fetch has no timeout option; pass AbortSignal.timeout() and classify every outcome in one place.

// Answers after `ms` and honours init.signal the way fetch does.
function slowServer(ms: number, status = 200): typeof fetch {
  return (_input, init) =>
    new Promise((resolve, reject) => {
      const signal = init?.signal;
      signal?.throwIfAborted();
      const reply = () => resolve(Response.json({ status }, { status }));
      const timer = setTimeout(reply, ms);
      signal?.addEventListener("abort", () => {
        clearTimeout(timer);
        reject(signal.reason);
      });
    });
}

// Every way a request can end, as data the caller must switch over.
type Outcome =
  | { kind: "ok"; body: unknown }
  | { kind: "http"; status: number }
  | { kind: "timeout" }
  | { kind: "cancelled" }
  | { kind: "network" };

async function request(
  doFetch: typeof fetch,
  url: string,
  opts: { timeoutMs?: number; signal?: AbortSignal } = {},
): Promise<Outcome> {
  const deadline = AbortSignal.timeout(opts.timeoutMs ?? 5000);
  const signal = opts.signal
    ? AbortSignal.any([opts.signal, deadline])
    : deadline;
  try {
    const res = await doFetch(url, { signal });
    if (!res.ok) return { kind: "http", status: res.status };
    return { kind: "ok", body: await res.json() }; // the deadline covers this too
  } catch (e) {
    if (e instanceof DOMException && e.name === "TimeoutError") {
      return { kind: "timeout" };
    }
    if (e instanceof DOMException && e.name === "AbortError") {
      return { kind: "cancelled" };
    }
    if (e instanceof TypeError) return { kind: "network" };
    throw e; // a SyntaxError from a 200 that isn't JSON is a bug, not an outcome
  }
}

const url = "https://api.example.com/report";
const show = async (label: string, outcome: Promise<Outcome>) =>
  console.log(`${label.padEnd(10)} ${JSON.stringify(await outcome)}`);

await show("fast", request(slowServer(0), url));
await show("503", request(slowServer(0, 503), url));
await show("too slow", request(slowServer(500), url, { timeoutMs: 20 }));

const dialog = new AbortController();
const pending = request(slowServer(500), url, { signal: dialog.signal });
dialog.abort(); // the user closed the dialog
await show("closed", pending);

await show("real fetch", request(fetch, url)); // the sandbox has no network
