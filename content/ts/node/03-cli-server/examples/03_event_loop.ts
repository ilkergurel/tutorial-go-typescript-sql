import { once } from "node:events";
import { createServer } from "node:http";
import type { AddressInfo } from "node:net";
import { setTimeout as sleep } from "node:timers/promises";
import { Worker } from "node:worker_threads";

function spin(ms: number) {
  const end = Date.now() + ms;
  while (Date.now() < end); // pure CPU work: never yields to the event loop
}
const SPIN_300 = "const end = Date.now() + 300; while (Date.now() < end);";

// Three ways to spend 300 ms inside the /report handler.
const strategies: Record<string, () => unknown> = {
  "CPU work on the event loop": () => spin(300),
  "waiting on I/O (a timer here)": () => sleep(300),
  "CPU work in a worker thread": () =>
    once(new Worker(SPIN_300, { eval: true }), "exit"),
};

for (const [name, work] of Object.entries(strategies)) {
  const log: string[] = [];
  const reportStarted = Promise.withResolvers<void>();
  const server = createServer(async (req, res) => {
    log.push(`${req.url} start`);
    if (req.url === "/report") {
      reportStarted.resolve();
      await work();
    }
    log.push(`${req.url} end`);
    res.end();
  });
  server.listen(0, "127.0.0.1");
  await once(server, "listening");
  const base = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;

  const report = fetch(`${base}/report`);
  await reportStarted.promise; // the server is now inside the /report handler
  await fetch(`${base}/health`); // a cheap request arrives meanwhile
  await report;
  server.close();
  console.log(`${name}:\n  ${log.join(" -> ")}`);
}
