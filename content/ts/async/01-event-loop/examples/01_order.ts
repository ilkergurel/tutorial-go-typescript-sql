// LEARN(ts.async.event-loop): synchronous code first, then every microtask, then one task.

const log = (line: string) => console.log(line);

log("1 sync:  the module body starts");

setTimeout(() => log("8 task:  setTimeout(fn, 0) waits for its turn"), 0);

Promise.resolve().then(() => {
  log("4 micro: a then() callback");
  queueMicrotask(() => log("7 micro: queued by a microtask, still first"));
});
queueMicrotask(() => log("5 micro: queueMicrotask"));

async function load(): Promise<void> {
  log("2 sync:  an async function runs synchronously up to its first await");
  await null;
  log("6 micro: the code after await resumes as a microtask");
}
void load();

log("3 sync:  the module body ends; the microtask queue drains next");
