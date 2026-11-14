// LEARN(ts.generics.keyof): keyof turns property names into a type, and T[K] looks up the property's type.

interface Product {
  id: string;
  name: string;
  price: number;
  tags: string[];
}
type ProductKey = keyof Product; // "id" | "name" | "price" | "tags"
type Price = Product["price"]; // number
type Tag = Product["tags"][number]; // string: [number] reads an array's element type

// K is the literal key the caller passed, so the result is that property's type.
function get<T, K extends keyof T>(obj: T, key: K): T[K] {
  return obj[key];
}
// Without the type parameter, every read is the union of all property types.
function getLoose(obj: Product, key: ProductKey): Product[ProductKey] {
  return obj[key];
}

const cay: Product = { id: "p1", name: "çay", price: 85, tags: ["içecek"] };
const price: Price = get(cay, "price"); // number
const loose = getLoose(cay, "price"); // string | number | string[]
console.log(price.toFixed(2), typeof loose);
// get(cay, "prize");
// error TS2345: Argument of type '"prize"' is not assignable to parameter of type 'keyof Product'.

// pluck: the element type of the result follows the key.
function pluck<T, K extends keyof T>(rows: readonly T[], key: K): T[K][] {
  return rows.map((row) => row[key]);
}
const rows = [cay, { id: "p2", name: "simit", price: 15, tags: [] }];
const names = pluck(rows, "name"); // string[]
const firstTag: Tag | undefined = pluck(rows, "tags").flat()[0];
console.log(names.map((n) => n.toLocaleUpperCase("tr")), firstTag);

// Object.keys returns string[], not (keyof Product)[]: a Product may carry
// more properties than its type lists, because typing is structural.
const fromApi = { id: "p3", name: "ılık su", price: 10, tags: [], stock: 40 };
const product: Product = fromApi; // extra properties are fine here
console.log(Object.keys(product).join(", "));
for (const key of Object.keys(product)) {
  // console.log(product[key]);
  // error TS7053: Element implicitly has an 'any' type because expression of type 'string' can't be used to index type 'Product'.
  if (key === "name" || key === "price") console.log(key, get(product, key));
}
