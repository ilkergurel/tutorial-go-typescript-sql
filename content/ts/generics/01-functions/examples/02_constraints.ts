// LEARN(ts.generics.functions): `T extends X` states what T must have, and T keeps everything else.

interface HasId {
  id: string;
}

// Plain parameter type: accepts any HasId, but returns only HasId.
function indexLossy(items: readonly HasId[]): Map<string, HasId> {
  return new Map(items.map((item) => [item.id, item]));
}
// Constrained type parameter: same requirement, precise result.
function indexById<T extends HasId>(items: readonly T[]): Map<string, T> {
  return new Map(items.map((item) => [item.id, item]));
}

const products = [
  { id: "p1", name: "Şeker", price: 45 },
  { id: "p2", name: "Çay", price: 85 },
  { id: "p3", name: "İrmik", price: 30 },
  { id: "p4", name: "Ilık su", price: 10 },
];
// indexLossy(products).get("p2")?.price;
// error TS2339: Property 'price' does not exist on type 'HasId'.
const tea = indexById(products).get("p2");
console.log(tea?.name, tea?.price);
// indexById([{ name: "no id" }]);
// error TS2353: Object literal may only specify known properties, and 'name' does not exist in type 'HasId'.

// You can't build a T yourself: T may be any subtype of the constraint.
// function withDefault<T extends HasId>(): T { return { id: "new" }; }
// error TS2322: Type '{ id: string; }' is not assignable to type 'T'.
//   '{ id: string; }' is assignable to the constraint of type 'T', but 'T' could be instantiated with a different subtype of constraint 'HasId'.

// T inferred from items flows into the callback: p is the full product type.
const tr = new Intl.Collator("tr");
function sortBy<T>(items: readonly T[], key: (item: T) => string): T[] {
  return items.toSorted((a, b) => tr.compare(key(a), key(b)));
}
const sorted = sortBy(products, (p) => p.name);
console.log(sorted.map((p) => `${p.name} ${p.price}`).join(", "));
// The default sort compares UTF-16 code units: I < Ç < İ < Ş.
console.log(products.map((p) => p.name).toSorted().join(", "));
