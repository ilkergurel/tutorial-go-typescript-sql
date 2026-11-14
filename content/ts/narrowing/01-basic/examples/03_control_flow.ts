// LEARN(ts.narrowing.basic): narrowing follows control flow, and stops where flow is unknown.

// A check stored in a const still narrows (TS 4.4+), so a long test can get a name.
function label(v: string | number): string {
  const isText = typeof v === "string";
  return isText ? v.toLocaleUpperCase("tr-TR") : v.toFixed(1);
}

// Assignment narrows too: after this if, prefix is string on every path.
function greetAll(names: string[], prefix: string | null): string[] {
  if (prefix === null) prefix = "Sn.";

  // The callback may run later, so it only keeps the narrowing because nothing
  // assigns prefix after this point (TS 5.4+).
  return names.map((n) => `${prefix.trim()} ${n}`);
}

// filter infers the predicate c is string from a simple check (TS 5.5+)...
const raw = ["06100", null, "34000"];
const codes: string[] = raw.filter((c) => c !== null);

// ...but not from truthiness on numbers: !!s is false for 0 as well, so the
// type stays (number | undefined)[] and the zero score is silently dropped.
const scores = [0, 72, undefined];
const truthy = scores.filter((s) => !!s);
// const nums: number[] = scores.filter((s) => !!s);
//   error TS2322: Type '(number | undefined)[]' is not assignable to type 'number[]'.
const defined: number[] = scores.filter((s) => s !== undefined);

console.log(label("ilgaz"), label(3.14159));
console.log(greetAll(["Ayşe", "Işıl"], null));
console.log(codes, truthy, defined);
