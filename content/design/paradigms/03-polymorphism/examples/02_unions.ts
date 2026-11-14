// LEARN(design.paradigms.polymorphism): a closed set of variants as plain tagged data; tsc checks every switch over it.
type Discount =
  | { kind: "percent"; percent: number }
  | { kind: "fixed"; amountKurus: number };

function apply(d: Discount, priceKurus: number): number {
  switch (d.kind) {
    case "percent":
      return priceKurus - Math.trunc((priceKurus * d.percent) / 100);
    case "fixed":
      return Math.max(priceKurus - d.amountKurus, 0);
  }
}

const offers: [code: string, d: Discount][] = [
  ["DEV250", { kind: "fixed", amountKurus: 25000 }],
  ["HOSGELDIN", { kind: "fixed", amountKurus: 5000 }],
  ["KASIM15", { kind: "percent", percent: 15 }],
];
for (const [code, d] of offers) {
  const price = String(apply(d, 20000)).padStart(5);
  console.log(`${code.padEnd(9)} ${price}  ${d.kind.padEnd(7)} ${JSON.stringify(d)}`);
}

// A value with the wrong fields for its kind doesn't compile:
// const bad: Discount = { kind: "percent", amountKurus: 5000 };
// error TS2353: Object literal may only specify known properties, and 'amountKurus' does not exist in type '{ kind: "percent"; percent: number; }'.

// Add | { kind: "chain"; steps: Discount[] } to Discount and apply stops compiling:
// error TS2366: Function lacks ending return statement and return type does not include 'undefined'.
