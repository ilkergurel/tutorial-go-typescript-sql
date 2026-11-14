// LEARN(ts.mental-model.erasable): enums and parameter properties are not erasable; they emit code.

// Erasable: a union type leaves nothing behind in the output.
type Level = "debug" | "info";
const level: Level = "info";
console.log(level);

// Not erasable: an enum compiles to a real object, with a reverse mapping.
// Under erasableSyntaxOnly (TS 5.8+) tsc reports here:
// error TS1294: This syntax is not allowed when 'erasableSyntaxOnly' is enabled.
enum Severity {
  Low,
  High,
}
console.log(Severity);

// Not erasable: a parameter property makes the transpiler write an assignment.
// erasableSyntaxOnly reports the same TS1294 on `private readonly id`.
class Account {
  constructor(private readonly id: string) {}
}
console.log(Account.toString());

// The erasable spelling: declare the field and assign it yourself.
class Invoice {
  private readonly id: string;
  constructor(id: string) {
    this.id = id;
  }
}
console.log(Invoice.toString());
