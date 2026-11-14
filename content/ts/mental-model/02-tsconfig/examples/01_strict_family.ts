// LEARN(ts.mental-model.tsconfig): strict is a family of checks; unknown in catch is one of them.

// verbatimModuleSyntax: an import without `type` is kept in the JavaScript,
// so importing a type that way is rejected:
// import { Prefs } from "./prefs.ts";
// error TS1484: 'Prefs' is a type and must be imported using a type-only import when 'verbatimModuleSyntax' is enabled.
// import type { Prefs } from "./prefs.ts"; // fine: erased entirely

// noImplicitAny: a parameter without a type is an error, not a silent any.
// function total(items) { return items.length; }
// error TS7006: Parameter 'items' implicitly has an 'any' type.
function total(items: number[]): number {
  return items.reduce((sum, n) => sum + n, 0);
}
console.log(total([1, 2, 3]));

// strictNullChecks: find() can miss, and its return type says so.
const users = ["ayşe", "mehmet"];
const found = users.find((u) => u.startsWith("z"));
// console.log(found.length);
// error TS18048: 'found' is possibly 'undefined'.
console.log(found === undefined ? "no match" : found.length);

// useUnknownInCatchVariables: anything can be thrown, so e is unknown.
function save(): void {
  throw "disk full"; // a string, not an Error
}
try {
  save();
} catch (e) {
  // console.log(e.message);
  // error TS18046: 'e' is of type 'unknown'.
  console.log((e as Error).message); // a cast compiles, and lies
  console.log(e instanceof Error ? e.message : String(e));
}
