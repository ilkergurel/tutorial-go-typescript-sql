// LEARN(ts.functions.higher-order): function types, contextual typing, void means "ignored".

// A function type. Parameter names are documentation; only types and order count.
type Compare = (a: string, b: string) => number;

// Contextual typing: a and b get string from Compare, no annotations needed.
const byLength: Compare = (a, b) => a.length - b.length;
// const broken: Compare = (a, b) => a < b;
//   error TS2322: Type 'boolean' is not assignable to type 'number'.

function sorted(words: readonly string[], compare?: Compare): string[] {
  return [...words].sort(compare); // copy first: sort() mutates
}

const words = ["şeker", "su", "çay", "cam", "ılık", "iğne", "zil"];
const tr = new Intl.Collator("tr");

// No comparator: UTF-16 code unit order puts ç, ı and ş after z.
console.log(sorted(words).join(" "));
// Collator's compare is bound to its collator, so it can be passed as is.
console.log(sorted(words, tr.compare).join(" "));
// An inline callback is contextually typed too: a and b are strings.
const shortFirst = sorted(words, (a, b) => byLength(a, b) || tr.compare(a, b));
console.log(shortFirst.join(" "));

// In a function TYPE, void means "the caller ignores the result".
type Listener = (event: string) => void;
const events: string[] = [];
const record: Listener = (e) => events.push(e); // push returns a number: allowed
const result = record("saved");
console.log(events, typeof result); // the number is still there at runtime
// if (result) {}
//   error TS1345: An expression of type 'void' cannot be tested for truthiness.

// A function DECLARED void, or a function type returning undefined, is strict:
// function save(): void { return events.length; }
//   error TS2322: Type 'number' is not assignable to type 'void'.
// const done: () => undefined = () => events.length;
//   error TS2322: Type 'number' is not assignable to type 'undefined'.
