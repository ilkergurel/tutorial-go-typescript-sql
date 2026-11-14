// LEARN(ts.objects-classes.classes): `private` is checked by tsc and erased; `#private` is enforced by the engine.

class Account {
  private pin: string; // TypeScript modifier: erased, an ordinary property at run time
  #balance: number; // JavaScript private field (ES2022): invisible outside the class

  constructor(pin: string, balance: number) {
    this.pin = pin;
    this.#balance = balance;
  }
  get balance(): number {
    return this.#balance;
  }
  // Brand check (ES2022): only objects built by this class have #balance.
  static isAccount(x: unknown): x is Account {
    return typeof x === "object" && x !== null && #balance in x;
  }
}

const acc = new Account("1234", 500);
// acc.pin;
// error TS2341: Property 'pin' is private and only accessible within class 'Account'.
console.log(acc["pin"]); // bracket access: tsc's deliberate escape hatch
console.log(JSON.stringify(acc), Object.keys(acc));
// acc.#balance;
// error TS18013: Property '#balance' is not accessible outside class 'Account' because it has a private identifier.
console.log(acc.balance);

// A private member makes the class nominal: look-alikes are rejected...
const fake = { pin: "0000", balance: 1_000_000 };
// const stolen: Account = fake;
// error TS2741: Property '#balance' is missing in type '{ pin: string; balance: number; }' but required in type 'Account'.
console.log(Account.isAccount(acc), Account.isAccount(fake));

// ...while a class with only public members is just a shape.
class Point {
  x: number;
  y: number;
  constructor(x: number, y: number) {
    this.x = x;
    this.y = y;
  }
}
const p: Point = { x: 1, y: 2 }; // compiles: same shape
console.log(p instanceof Point);
