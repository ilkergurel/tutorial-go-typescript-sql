// LEARN(ts.async.event-loop): no data races on one thread, but every await lets other code in.

// Stands in for any I/O: a fetch, an IndexedDB read, an audit call.
const io = () => new Promise<void>((resolve) => setTimeout(resolve, 0));

// 1000 concurrent tasks, each counting after its I/O. No update is lost:
// the read-modify-write between two awaits runs without interruption.
let hits = 0;
await Promise.all(
  Array.from({ length: 1000 }, async () => {
    await io();
    hits = hits + 1;
  }),
);
console.log(`hits: ${hits}`);

// A read, an await, then a write based on the stale read: a lost update.
let balance = 100;
async function withdrawRacy(amount: number): Promise<void> {
  const current = balance;
  await io(); // the other withdrawal runs here
  balance = current - amount;
}
await Promise.all([withdrawRacy(30), withdrawRacy(50)]);
console.log(`racy balance: ${balance}`);

// When the await must sit between read and write, chain the calls:
// each one starts after the previous one settled, like holding a mutex.
balance = 100;
let tail: Promise<void> = Promise.resolve();
function serialized(fn: () => Promise<void>): Promise<void> {
  const run = tail.then(fn);
  tail = run.catch(() => {}); // a failure must not block the queue
  return run;
}
await Promise.all([
  serialized(() => withdrawRacy(30)),
  serialized(() => withdrawRacy(50)),
]);
console.log(`serialized balance: ${balance}`);

export {}; // module: top-level await allowed
