// LEARN(ts.objects-classes.spread): rest removes keys with a precise type; spread copies own data, not prototypes.

interface User {
  id: number;
  email: string;
  passwordHash: string;
}

// Rest collects the remaining properties, and its type drops the named ones.
function toPublic(user: User) {
  const { passwordHash, ...rest } = user;
  return rest; // { id: number; email: string }
}
const pub = toPublic({ id: 7, email: "ayse@example.com", passwordHash: "x1" });
console.log(pub);
// pub.passwordHash;
// error TS2339: Property 'passwordHash' does not exist on type '{ id: number; email: string; }'.

// Object spread copies own enumerable properties. Methods and getters live
// on the prototype, so a spread class instance is plain data.
class Price {
  amount: number;
  constructor(amount: number) {
    this.amount = amount;
  }
  get withVat(): number {
    return this.amount * 1.2;
  }
  format(): string {
    return `${this.amount} TL`;
  }
}
const copy = { ...new Price(100) };
console.log(Object.keys(copy), copy instanceof Price, "withVat" in copy);
// The spread type models this: no methods, no getters.
// copy.format();
// error TS2339: Property 'format' does not exist on type '{ amount: number; }'.

// Array spread takes any iterable: Set, Map entries, strings.
const tags = ["çay", "kahve", "çay"];
console.log([...new Set(tags)]);
const stock = new Map([["çay", 3]]);
const restocked = new Map([...stock, ["kahve", 5]]); // a new Map
console.log(stock.size, restocked.size);

// A string spreads into code points, while .length counts UTF-16 code units.
const word = "İz👍";
console.log(word.length, [...word].length);
const hex = (s: string) => [...s].map((ch) => ch.codePointAt(0)?.toString(16));
console.log(hex("İ".toLowerCase()), hex("İ".toLocaleLowerCase("tr")));
