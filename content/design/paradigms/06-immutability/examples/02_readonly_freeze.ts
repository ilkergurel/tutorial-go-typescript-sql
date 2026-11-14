// LEARN(design.paradigms.immutability): readonly is a compile-time promise, Object.freeze a shallow run-time one; a change is a new object.
type PriceList = Readonly<Record<string, number>>; // kuruş

const october: PriceList = Object.freeze({ ayran: 1000, simit: 1250 });
const order = { id: "S-1", prices: october };

// Copy-on-write: spread builds a new object and october is untouched.
const november: PriceList = Object.freeze({ ...october, simit: 1500 });
console.log(`order S-1 simit: ${order.prices.simit}`);
console.log(`current simit:   ${november.simit}`);

// november.ayran = 0;
// error TS2542: Index signature in type 'Readonly<Record<string, number>>' only permits reading.

// A cast removes readonly; only the freeze stops the write at run time
// (modules run in strict mode, so it throws instead of failing silently).
const writable = november as Record<string, number>;
try {
  writable.ayran = 0;
} catch (e) {
  console.log(`writing to a frozen list: ${(e as Error).name}`);
}

// Both freeze and Readonly<T> are shallow: the nested object is still writable.
const draft = Object.freeze({ id: "S-2", prices: { ...november } });
draft.prices.ayran = 0;
console.log(`frozen draft ayran after a write: ${draft.prices.ayran}`);
