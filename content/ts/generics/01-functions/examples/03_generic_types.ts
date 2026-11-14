// LEARN(ts.generics.functions): generic types with defaults, NoInfer, and the return-only type parameter trap.

// A generic type alias; E defaults to Error when the caller omits it.
type Result<T, E = Error> = { ok: true; value: T } | { ok: false; error: E };

function parsePrice(text: string): Result<number> {
  const n = Number(text.replace(",", "."));
  if (Number.isNaN(n)) return { ok: false, error: new Error(`not a price: ${text}`) };
  return { ok: true, value: n };
}
function parsePort(text: string): Result<number, "empty" | "range"> {
  if (text === "") return { ok: false, error: "empty" };
  const n = Number(text);
  return n > 0 && n < 65536 ? { ok: true, value: n } : { ok: false, error: "range" };
}
for (const r of [parsePrice("12,5"), parsePrice("on iki"), parsePort("70000")]) {
  console.log(r.ok ? r.value : r.error); // r.error: Error | "empty" | "range"
}

// NoInfer (TS 5.4): T is inferred from options only; fallback is checked against it.
function pick<T extends string>(options: readonly T[], wanted: string, fallback: NoInfer<T>): T {
  return options.find((o) => o === wanted) ?? fallback;
}
console.log(pick(["tr", "en"], "de", "tr"));
// pick(["tr", "en"], "de", "fr");
// error TS2345: Argument of type '"fr"' is not assignable to parameter of type '"en" | "tr"'.
// Without NoInfer, T would widen to "tr" | "en" | "fr" and accept it.

// A type parameter that appears only in the return type is a cast in disguise.
function parseJson<T>(text: string): T {
  return JSON.parse(text); // any, silently assignable to T
}
const port = parseJson<number>('"8080"');
console.log(typeof port, port + 1); // a string, so + concatenates
// Honest version: return unknown and make the caller check.
function parseJsonSafe(text: string): unknown {
  return JSON.parse(text);
}
const raw = parseJsonSafe('"8080"');
console.log(typeof raw === "number" ? raw + 1 : `not a number: ${JSON.stringify(raw)}`);
