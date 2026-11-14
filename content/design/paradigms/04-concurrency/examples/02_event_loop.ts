// LEARN(design.paradigms.concurrency): one thread and no data races, but every await lets other tasks run between a check and a write.
// Stands in for the payment call: any I/O that is awaited.
const pay = () => new Promise<void>((resolve) => setTimeout(resolve, 0));

let left = 10;

// Racy: the await sits between the check and the write.
async function reserveRacy(): Promise<boolean> {
  if (left === 0) return false;
  await pay();
  left--;
  return true;
}

// Fixed: check and write in one synchronous step, then await.
async function reserve(): Promise<boolean> {
  if (left === 0) return false;
  left--;
  await pay(); // if payment fails, put the item back: left++
  return true;
}

async function twentyBuyers(buy: () => Promise<boolean>): Promise<string> {
  left = 10;
  const results = await Promise.all(Array.from({ length: 20 }, () => buy()));
  const sold = results.filter((ok) => ok).length;
  return `sold ${sold}, refused ${20 - sold}, left ${left}`;
}

console.log(`racy:  ${await twentyBuyers(reserveRacy)}`);
console.log(`fixed: ${await twentyBuyers(reserve)}`);

export {}; // a module, so top-level await is allowed
