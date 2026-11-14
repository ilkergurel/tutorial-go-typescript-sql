// LEARN(ts.objects-classes.classes): parameter properties generate code, and field initializers run before the constructor body.

// `private readonly id` in the parameter list declares a field AND assigns it.
// That is TypeScript-only syntax with runtime effects; erasableSyntaxOnly says
// error TS1294: This syntax is not allowed when 'erasableSyntaxOnly' is enabled.
class Invoice {
  // @ts-expect-error TS2729: Property 'id' is used before its initialization.
  readonly label = `Fatura ${this.id}`;
  constructor(private readonly id: string) {}
}
// Fields are initialized before the constructor body runs, and the body is
// where the parameter property is assigned.
console.log(new Invoice("2026-001").label);

// The erasable version says everything it does, in the order it happens.
class InvoiceFixed {
  readonly #id: string;
  readonly label: string;
  constructor(id: string) {
    this.#id = id;
    this.label = `Fatura ${id}`;
  }
  get id(): string {
    return this.#id;
  }
}
const inv = new InvoiceFixed("2026-002");
console.log(inv.label, inv.id);

// readonly is a compile-time promise, checked outside the constructor:
// inv.label = "x";
// error TS2540: Cannot assign to 'label' because it is a read-only property.
// A getter without a setter is read-only too:
// inv.id = "x";
// error TS2540: Cannot assign to 'id' because it is a read-only property.

// At run time readonly is gone: only Object.freeze or #private stop a write.
Object.assign(inv, { label: "değişti" });
console.log(inv.label);
