// LEARN(ts.everyday-types.objects): readonly arrays and tuples state what a function must not change.

// sort() sorts in place, so this "pure-looking" helper reorders its input.
function topTwoNaive(scores: number[]): number[] {
  return scores.sort((a, b) => b - a).slice(0, 2);
}
const mine = [70, 95, 80];
console.log(`${topTwoNaive(mine)} | caller now has ${mine}`);

// readonly number[] has no mutating methods, so the bug can't compile:
// return scores.sort((a, b) => b - a).slice(0, 2);
// error TS2339: Property 'sort' does not exist on type 'readonly number[]'.
function topTwo(scores: readonly number[]): number[] {
  return scores.toSorted((a, b) => b - a).slice(0, 2); // ES2023: sorts a copy
}
const yours = [70, 95, 80];
console.log(`${topTwo(yours)} | caller still has ${yours}`);

// A tuple: fixed length, one type per position, optional labels for readers.
type LatLon = readonly [lat: number, lon: number];
const istanbul: LatLon = [41.0082, 28.9784];
// istanbul[0] = 0;
// error TS2540: Cannot assign to '0' because it is a read-only property.
// const altitude = istanbul[2];
// error TS2493: Tuple type 'LatLon' of length '2' has no element at index '2'.
const [lat, lon] = istanbul;
console.log(`lat ${lat}, lon ${lon}`);

// Tuples are how TS types multiple return values and key/value pairs.
const minutes = { go: 12, ts: 13 };
const pairs: [string, number][] = Object.entries(minutes);
console.log(pairs.map(([track, n]) => `${track}=${n}`).join(" "));
