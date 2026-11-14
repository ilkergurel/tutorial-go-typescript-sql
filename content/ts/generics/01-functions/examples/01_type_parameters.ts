// LEARN(ts.generics.functions): a type parameter links input and output types, so each call keeps its precise type.

// any: every call compiles, and the result is any again.
function firstAny(items: any[]): any {
  return items[0];
}
// unknown: safe, but every caller has to narrow the result before use.
function firstUnknown(items: readonly unknown[]): unknown {
  return items[0];
}
// Generic: T is inferred at each call and flows into the return type.
function first<T>(items: readonly T[]): T | undefined {
  return items[0];
}

const cities = ["izmir", "istanbul"];
try {
  console.log(firstAny(cities).toFixed(2)); // compiles: any accepts anything
} catch (e) {
  console.log("any:", e instanceof TypeError ? "TypeError at run time" : e);
}
// firstUnknown(cities).toUpperCase();
// error TS2571: Object is of type 'unknown'.
const city = first(cities); // string | undefined
console.log(city?.toLocaleUpperCase("tr"), city?.toUpperCase());
// const n: number | undefined = first(cities);
// error TS2322: Type 'string | undefined' is not assignable to type 'number | undefined'.

// Several parameters, inferred together; explicit arguments when inference can't know.
function pair<A, B>(a: A, b: B): [A, B] {
  return [a, b];
}
const p = pair("çay", 85); // [string, number]
const empty = first<number>([]); // number | undefined, not unknown
console.log(p, empty ?? "none");

// const type parameter (TS 5.0): infer literals as if the caller wrote `as const`.
function defineLocales<const T extends readonly string[]>(locales: T): T {
  return locales;
}
const locales = defineLocales(["tr-TR", "en-US"]); // readonly ["tr-TR", "en-US"]
type Locale = (typeof locales)[number]; // "tr-TR" | "en-US"
const fallback: Locale = "tr-TR";
// const wrong: Locale = "de-DE";
// error TS2322: Type '"de-DE"' is not assignable to type '"en-US" | "tr-TR"'.
console.log(locales.join(", "), fallback);
