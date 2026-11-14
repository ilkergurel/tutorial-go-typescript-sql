// LEARN(design.patterns.classic): a decorator is a wrapper of the same type; @ is one way to apply it.

type Lookup = (word: string) => Promise<string>;

// The GoF decorator as a higher-order function: a Lookup in, a Lookup out.
const cached = (next: Lookup): Lookup => {
  const memo = new Map<string, Promise<string>>();
  return (word) => {
    if (!memo.has(word)) memo.set(word, next(word));
    return memo.get(word)!;
  };
};
const logged =
  (next: Lookup): Lookup =>
  async (word) => {
    console.log("  reached:", word);
    return next(word);
  };

const upper: Lookup = async (word) => word.toLocaleUpperCase("tr");

// Same two decorators, two orders, two behaviours: order is semantics.
const stacks: [string, Lookup][] = [
  ["logged(cached(upper))", logged(cached(upper))],
  ["cached(logged(upper))", cached(logged(upper))],
];
for (const [name, lookup] of stacks) {
  console.log(name);
  for (const word of ["istanbul", "ılık", "istanbul"]) await lookup(word);
}

// TS 5.0 standard decorators: @memo receives the method and returns its replacement.
function memo<This, R>(
  method: (this: This, word: string) => R,
  _context: ClassMethodDecoratorContext<This, (this: This, word: string) => R>,
) {
  const seen = new Map<string, R>(); // runs once per class definition
  return function (this: This, word: string): R {
    if (!seen.has(word)) seen.set(word, method.call(this, word));
    return seen.get(word)!;
  };
}

class Dictionary {
  computed = 0;
  @memo
  upper(word: string) {
    this.computed++;
    return word.toLocaleUpperCase("tr");
  }
}
const a = new Dictionary();
const b = new Dictionary();
console.log(a.upper("iğne"), b.upper("iğne"), "computed:", a.computed, b.computed);
