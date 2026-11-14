// LEARN(ts.services.advanced): retry only what is safe to repeat, with exponential backoff, jitter and Retry-After.

interface Policy {
  attempts: number;
  baseMs: number;
  sleep: (ms: number) => Promise<void>; // injected: tests don't wait
  random: () => number; // Math.random in production
}
const RETRYABLE = new Set([408, 429, 500, 502, 503, 504]);
const IDEMPOTENT = new Set(["GET", "HEAD", "OPTIONS", "PUT", "DELETE"]);

async function send(doFetch: typeof fetch, request: Request, p: Policy) {
  const repeatable =
    IDEMPOTENT.has(request.method) || request.headers.has("Idempotency-Key");
  for (let attempt = 1; ; attempt++) {
    let res: Response | undefined;
    try {
      const signal = AbortSignal.timeout(5000); // per attempt
      res = await doFetch(request.clone(), { signal }); // a body is sent once
      if (!RETRYABLE.has(res.status)) return res;
    } catch (e) {
      const timeout = e instanceof DOMException && e.name === "TimeoutError";
      if (!(e instanceof TypeError) && !timeout) throw e; // bugs propagate
    }
    if (!repeatable || attempt === p.attempts) {
      if (res) return res; // the caller sees the last answer
      throw new Error("no response: did it happen or not?");
    }
    const after = res?.headers.get("Retry-After") ?? "";
    const backoff = Math.floor(p.random() * p.baseMs * 2 ** (attempt - 1));
    const wait = /^\d+$/.test(after) ? Number(after) * 1000 : backoff;
    console.log(
      `  #${attempt} ${res?.status ?? "no response"}: wait ${wait} ms`,
    );
    await p.sleep(wait);
  }
}

// A fake server that plays a script: "drop" loses the connection.
const script = (...answers: (number | "drop")[]): typeof fetch => {
  return async () => {
    const a = answers.shift() ?? 200;
    if (a === "drop") throw new TypeError("Failed to fetch");
    const headers = a === 503 ? { "Retry-After": "2" } : undefined;
    return new Response(null, { status: a, headers });
  };
};
const rolls = [0.9, 0.75, 0.5];
const policy: Policy = {
  attempts: 5,
  baseMs: 200,
  sleep: async () => {},
  random: () => rolls.shift() ?? 0.5,
};
const orders = "https://api.example.com/orders";
const payments = "https://api.example.com/payments";

console.log("GET");
const flaky = script("drop", 502, 503, 200);
console.log(`  -> ${(await send(flaky, new Request(orders), policy)).status}`);

const pay = { method: "POST", body: '{"orderId":7,"kurus":125000}' };
console.log("POST without a key");
try {
  await send(script("drop", 201), new Request(payments, pay), policy);
} catch (e) {
  console.log(`  ${e instanceof Error ? e.message : e}`);
}

console.log("POST with an Idempotency-Key");
const headers = { "Idempotency-Key": crypto.randomUUID() }; // one per intent
const keyed = new Request(payments, { ...pay, headers }); // reused by retries
console.log(`  -> ${(await send(script("drop", 201), keyed, policy)).status}`);
