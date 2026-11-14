// LEARN(ts.errors.try-catch): try/catch only sees a rejection it awaits; finally runs on every path.

const log: string[] = [];
async function save(id: string): Promise<string> {
  await Promise.resolve(); // stands in for a network call that fails
  log.push(`save ${id} failed`);
  throw new Error(`${id} kaydedilemedi`);
}

async function saveWithoutAwait(id: string): Promise<string> {
  try {
    return save(id); // the promise leaves the try block before it rejects
  } catch {
    return `${id}: handled`; // never runs
  } finally {
    log.push(`finally ${id}`);
  }
}
async function saveWithAwait(id: string): Promise<string> {
  try {
    return await save(id); // the rejection is thrown here, inside the try
  } catch {
    return `${id}: handled`;
  } finally {
    log.push(`finally ${id}`);
  }
}

console.log(await saveWithAwait("REQ-EXPORT"));
console.log(log.splice(0).join(" -> "));

const escaped = await saveWithoutAwait("REQ-RESET").catch((e: unknown) =>
  e instanceof Error ? `escaped to the caller: ${e.message}` : "escaped",
);
console.log(escaped);
console.log(log.splice(0).join(" -> "));
