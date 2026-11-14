// LEARN(ts.functions.signatures): rest parameters are arrays; many optionals become one object.

// A rest parameter arrives as a real, typed array. The legacy `arguments`
// object is neither an array nor typed.
function total(...amounts: number[]): number {
  return amounts.reduce((sum, a) => sum + a, 0);
}

function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max);
}

const prices = [120, 80, 45];
console.log(total(), total(10, 20), total(...prices));

// Spreading into fixed parameters needs a known length; number[] has none:
// clamp(150, ...prices);
//   error TS2556: A spread argument must either have a tuple type or be passed to a rest parameter.
const bounds = [0, 100] as const; // type: readonly [0, 100]
console.log(clamp(150, ...bounds));

// Naive: three positional optionals. What does (undefined, undefined, 1) mean?
function delaysPositional(retries = 3, baseMs = 100, factor = 2): number[] {
  return Array.from({ length: retries }, (_, i) => baseMs * factor ** i);
}

// Idiomatic: one options object, destructured with a default per field.
// "= {}" lets callers omit the whole object.
type RetryOptions = { retries?: number; baseMs?: number; factor?: number };

function delays({
  retries = 3,
  baseMs = 100,
  factor = 2,
}: RetryOptions = {}): number[] {
  return Array.from({ length: retries }, (_, i) => baseMs * factor ** i);
}

console.log(delaysPositional(undefined, undefined, 1));
console.log(delays({ factor: 1 }), delays(), delays({ retries: 5 }));
