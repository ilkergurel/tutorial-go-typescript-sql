// LEARN(design.paradigms.generics-absence): in TS absence is part of the type, T | undefined, and must be narrowed away.
function minOf<T>(xs: readonly T[], compare: (a: T, b: T) => number): T | undefined {
  if (xs.length === 0) return undefined; // reduce without a seed throws on []
  return xs.reduce((best, x) => (compare(x, best) < 0 ? x : best));
}

// Prices in kuruş; "ayran" is not listed at all.
const offers = new Map<string, number[]>([
  ["kahve", []],
  ["simit", [1500, 1250]],
  ["su", [500, 0]], // a free promotion: 0 is a real price
]);
const byPrice = (a: number, b: number) => a - b;

for (const p of ["ayran", "kahve", "simit", "su"]) {
  const prices = offers.get(p); // number[] | undefined
  if (prices === undefined) {
    console.log(`${p.padEnd(6)} not listed`);
    continue;
  }
  const cheapest = minOf(prices, byPrice); // number | undefined
  console.log(`${p.padEnd(6)} ${cheapest === undefined ? "no offers" : `cheapest ${cheapest}`}`);
}

// Shortcuts that turn absence into a wrong answer:
console.log(`Math.min() of no prices: ${Math.min(...[])}`);
console.log(`and in JSON: ${JSON.stringify({ cheapest: Math.min() })}`);
console.log(`free price with ||: ${minOf([0], byPrice) || "no offers"}`);
console.log(`free price with ??: ${minOf([0], byPrice) ?? "no offers"}`);
