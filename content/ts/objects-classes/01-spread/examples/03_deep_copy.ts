// LEARN(ts.objects-classes.spread): structuredClone deep-copies data; JSON round trips lose types; neither copies behaviour.

const order = {
  id: "o-1",
  createdAt: new Date(Date.UTC(2026, 10, 13, 9, 30)),
  lines: [{ sku: "çay-500g", qty: 2 }],
  discounts: new Map([["kampanya", 10]]),
  note: undefined as string | undefined,
};

const shallow = { ...order }; // new top level, nested values shared
const viaJson = JSON.parse(JSON.stringify(order)); // deep, typed any
const deep = structuredClone(order); // deep, same type

for (const line of order.lines) line.qty = 99; // mutate the original
console.log(`qty: shallow ${shallow.lines[0]?.qty}, deep ${deep.lines[0]?.qty}`);

// JSON keeps only what JSON can say: Date -> string, Map -> {}, undefined -> gone.
console.log(typeof viaJson.createdAt, viaJson.discounts, "note" in viaJson);
// structuredClone keeps Date, Map, Set, typed arrays, undefined and even cycles.
console.log(deep.createdAt instanceof Date, deep.discounts.get("kampanya"), "note" in deep);

// viaJson is any, so this compiles and fails at run time:
try {
  viaJson.createdAt.getTime();
} catch (e) {
  console.log("JSON copy:", e instanceof TypeError ? "TypeError" : e);
}

// Classes: the clone is a plain object, but its static type is still Money.
class Money {
  amount: number;
  constructor(amount: number) {
    this.amount = amount;
  }
  format(): string {
    return `${this.amount.toFixed(2)} TL`;
  }
}
const cloned: Money = structuredClone(new Money(42));
console.log(cloned instanceof Money, cloned);
try {
  cloned.format(); // compiles: structuredClone<T>(value: T): T
} catch (e) {
  console.log("cloned.format():", e instanceof TypeError ? "TypeError" : e);
}

// Functions can't be cloned at all.
try {
  structuredClone({ onSave: () => "saved" });
} catch (e) {
  console.log(e instanceof DOMException ? e.name : e);
}
