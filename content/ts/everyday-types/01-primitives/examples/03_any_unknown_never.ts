// LEARN(ts.everyday-types.primitives): any turns checking off; unknown forces a check; never has no values.

const payload = '{"count": "3"}';

// any: every operation compiles, the result is any again, and errors surface late.
const loose: any = JSON.parse(payload);
const total = loose.count + 1; // any: string concatenation, not addition
console.log(total);
try {
  console.log(loose.items.length);
} catch (e) {
  console.log(e instanceof TypeError ? "TypeError at run time" : e);
}

// unknown: anything can be assigned to it, nothing can be done with it unchecked.
const safe: unknown = JSON.parse(payload);
// console.log(safe.count);
// error TS18046: 'safe' is of type 'unknown'.
function readCount(v: unknown): number {
  if (typeof v === "object" && v !== null && "count" in v) {
    if (typeof v.count === "number") return v.count;
    if (typeof v.count === "string") return Number.parseInt(v.count, 10);
  }
  return fail("payload has no count");
}
console.log(readCount(safe) + 1);

// never: the type with no values. A function that always throws returns never,
// so tsc knows the code after a call to it is unreachable.
function fail(message: string): never {
  throw new Error(message);
}
try {
  readCount({ total: 3 });
} catch (e) {
  console.log(e instanceof Error ? e.message : e);
}
// const impossible: string & number = 1;
// error TS2322: Type '1' is not assignable to type 'never'.
