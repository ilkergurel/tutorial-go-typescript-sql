// LEARN(ts.async.combinators): all fails fast, allSettled waits for all, race takes the first to settle, any the first success.

function call(name: string, ms: number, ok: boolean): Promise<string> {
  return new Promise((resolve, reject) =>
    setTimeout(() => {
      console.log(`  ${name} ${ok ? "succeeded" : "failed"} at ${ms} ms`);
      if (ok) resolve(name);
      else reject(new Error(`${name} failed`));
    }, ms),
  );
}

const message = (e: unknown) => (e instanceof Error ? e.message : String(e));

function watch(label: string, p: Promise<unknown>): Promise<void> {
  return p.then(
    (value) => console.log(`${label} fulfilled: ${JSON.stringify(value)}`),
    (reason) => console.log(`${label} rejected: ${message(reason)}`),
  );
}

// The three calls start once, here. Combinators start nothing: they watch.
const calls = [
  call("profile", 50, true),
  call("orders", 10, false),
  call("prefs", 30, true),
];

await Promise.all([
  watch("all       ", Promise.all(calls)),
  watch("race      ", Promise.race(calls)),
  watch("any       ", Promise.any(calls)),
  watch(
    "allSettled",
    Promise.allSettled(calls).then((results) =>
      results.map((r) =>
        r.status === "fulfilled" ? r.value : message(r.reason),
      ),
    ),
  ),
]);

// any rejects only when every input rejected, with all reasons in input order.
try {
  await Promise.any([call("mirror-1", 20, false), call("mirror-2", 10, false)]);
} catch (e) {
  if (e instanceof AggregateError) {
    console.log(`any rejected: ${e.errors.map(message).join(", ")}`);
  }
}

export {}; // module: top-level await allowed
