// LEARN(ts.services.zod): validate where data enters the app, and decide what one bad item means.
import { z } from "zod";

const Payment = z.discriminatedUnion("status", [
  z.object({ status: z.literal("paid"), id: z.string(), kurus: z.int() }),
  z.object({ status: z.literal("failed"), id: z.string(), reason: z.string() }),
]);
type Payment = z.infer<typeof Payment>;

// The server added a status this client has never heard of.
const fakeFetch: typeof fetch = async () =>
  Response.json([
    { status: "paid", id: "p1", kurus: 123450 },
    { status: "failed", id: "p2", reason: "kart reddedildi" },
    { status: "refunded", id: "p3", kurus: 123450 },
  ]);

// The only place that turns unknown JSON into typed data.
async function getJson<T>(url: string, schema: z.ZodType<T>): Promise<T> {
  const res = await fakeFetch(url);
  if (!res.ok) throw new Error(`${url}: HTTP ${res.status}`);
  const parsed = schema.safeParse(await res.json());
  if (!parsed.success) {
    throw new Error(
      `${url} broke the contract:\n${z.prettifyError(parsed.error)}`,
    );
  }
  return parsed.data;
}

const url = "https://api.example.com/payments";
// Policy 1: all or nothing. One unknown item fails the whole list.
try {
  await getJson(url, z.array(Payment));
} catch (e) {
  console.log(e instanceof Error ? e.message : e);
}

// Policy 2: keep what is valid, report the rest.
const items = await getJson(url, z.array(z.unknown()));
const payments: Payment[] = [];
for (const item of items) {
  const r = Payment.safeParse(item);
  if (r.success) payments.push(r.data);
  else console.log(`skipped: ${JSON.stringify(item)}`);
}

const tl = new Intl.NumberFormat("tr-TR", {
  style: "currency",
  currency: "TRY",
});
for (const p of payments) {
  // The check on status narrows the union: only "paid" has kurus.
  if (p.status === "paid") console.log(`${p.id}: ${tl.format(p.kurus / 100)}`);
  else console.log(`${p.id}: failed, ${p.reason}`);
}
