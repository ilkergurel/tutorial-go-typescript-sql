// LEARN(ts.functions.higher-order): closures capture variables; arrows capture `this`.

// Each call creates a new count variable; the returned arrow keeps it alive.
function makeCounter(): () => number {
  let count = 0;
  return () => ++count;
}
const a = makeCounter();
const b = makeCounter();
console.log(a(), a(), b());

class Basket {
  items: string[] = [];

  // A method: `this` is whatever the call site provides.
  add(item: string): void {
    this.items.push(item);
  }

  // An arrow function in a field: `this` is fixed to the instance.
  addBound = (item: string): void => {
    this.items.push(item);
  };
}

const basket = new Basket();
try {
  // Compiles: tsc does not check `this` when a method is passed as a value.
  ["çay"].forEach(basket.add);
} catch (e) {
  console.log("forEach(basket.add) threw", e instanceof Error ? e.name : e);
}
["simit"].forEach(basket.addBound);
["peynir"].forEach((item) => basket.add(item)); // the usual fix
["zeytin"].forEach(basket.add.bind(basket));
console.log(basket.items.join(", "));

// A `this` parameter is erased at runtime but lets tsc check direct calls.
function summary(this: Basket): string {
  return `${this.items.length} items`;
}
console.log(summary.call(basket));
// summary();
//   error TS2684: The 'this' context of type 'void' is not assignable to method's 'this' of type 'Basket'.
