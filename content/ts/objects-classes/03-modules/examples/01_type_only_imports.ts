// LEARN(ts.objects-classes.modules): import type is erased before the code runs; imports without `type` are kept.

// This sandbox runs one file: ./billing.ts does not exist here. The program
// still runs, because a type-only import leaves no trace in the JavaScript.
import type { CURRENCIES, Invoice } from "./billing.ts";

// A value imported with `import type` still works in type positions...
type Currency = (typeof CURRENCIES)[number]; // "TRY" | "EUR" | "USD"
// ...but not at run time, where it doesn't exist:
// console.log(CURRENCIES.length);
// error TS1361: 'CURRENCIES' cannot be used as a value because it was imported using 'import type'.

const invoices: Invoice[] = [
  { id: "2026-001", total: 1250.5, currency: "TRY" },
  { id: "2026-002", total: 99, currency: "EUR" },
];

function format(amount: number, currency: Currency): string {
  return new Intl.NumberFormat("tr-TR", { style: "currency", currency }).format(amount);
}
for (const inv of invoices) console.log(inv.id, format(inv.total, inv.currency));

// Re-exports follow the same rule. `export { Invoice } from "./billing.ts"` gives
// error TS1205: Re-exporting a type when 'verbatimModuleSyntax' is enabled requires using 'export type'.
export type { Invoice } from "./billing.ts"; // erased as well

console.log("ran without loading ./billing.ts");
