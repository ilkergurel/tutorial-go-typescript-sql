// LEARN(ts.functions.signatures): annotate parameters; infer returns, except at boundaries.

// Parameters have nothing to infer from. Under strict, a missing type is an
// error, not a silent any:
// function shout(text) {}
//   error TS7006: Parameter 'text' implicitly has an 'any' type.

// The return type is inferred as string: hover it in the editor, don't repeat it.
function initials(first: string, last: string) {
  return (first.charAt(0) + last.charAt(0)).toLocaleUpperCase("tr-TR");
}

// Unlike plain JS, the number of arguments is part of the contract:
// initials("ilker");
//   error TS2554: Expected 2 arguments, but got 1.

// Annotate the return type when it is a promise to callers: a wrong return is
// then reported here, not wherever some caller trips over it.
function parsePrice(text: string): number | undefined {
  // Turkish prices use "." for thousands and "," for decimals: "1.234,50".
  const n = Number(text.replaceAll(".", "").replace(",", "."));
  // if (Number.isNaN(n)) return null;
  //   error TS2322: Type 'null' is not assignable to type 'number | undefined'.
  return Number.isNaN(n) ? undefined : n;
}

// Recursive functions need it too: inference would need the answer first.
// Without ": number" you get:
//   error TS7023: 'digitSum' implicitly has return type 'any' because it does
//   not have a return type annotation and is referenced directly or indirectly
//   in one of its return expressions.
function digitSum(n: number): number {
  return n < 10 ? n : (n % 10) + digitSum(Math.floor(n / 10));
}

console.log(initials("ilker", "ışık")); // toUpperCase() would give "II"
console.log(parsePrice("1.234,50"), parsePrice("12,5 TL"));
console.log(digitSum(2026));
