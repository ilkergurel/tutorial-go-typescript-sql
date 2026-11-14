// LEARN(ts.browser-dom.storage): a Worker is a second thread; messages are structured clones or transfers.

// In a Vite app: new Worker(new URL("./totals.worker.ts", import.meta.url),
// { type: "module" }). A lesson is one file, so the worker's code is a string
// (not type-checked) loaded from a blob: URL.
const source = String.raw`
  self.onmessage = (e) => {
    const lines = e.data.csv.split(/\r?\n/).slice(1).filter(Boolean);
    let total = 0;
    for (const line of lines) total += Number(line.split(";")[1].replace(",", "."));
    self.postMessage({ rows: lines.length, total, dom: typeof document });
  };`;

interface TotalsRequest {
  csv: string;
}
interface TotalsReply {
  rows: number;
  total: number;
  dom: string;
}

const url = URL.createObjectURL(
  new Blob([source], { type: "text/javascript" }),
);
const worker = new Worker(url);
URL.revokeObjectURL(url); // the worker has its URL; the Blob can go

// One request at a time. With several in flight, tag each with an id and
// keep a Map of pending resolvers.
function ask(request: TotalsRequest): Promise<TotalsReply> {
  return new Promise((resolve, reject) => {
    worker.onmessage = (e: MessageEvent<TotalsReply>) => resolve(e.data);
    worker.onerror = (e) => reject(new Error(e.message));
    worker.postMessage(request);
  });
}

const csv = "Müşteri;Tutar\r\nŞule;1234,5\r\nZeynep;99,9\r\n";
const reply = await ask({ csv });
const total = new Intl.NumberFormat("tr-TR").format(reply.total);
console.log(`worker: ${reply.rows} rows, ${total} ₺, document: ${reply.dom}`);
worker.terminate(); // workers live until terminated or the page goes away

// postMessage copies with structuredClone: Map and Date survive, classes don't.
class Money {
  constructor(readonly kurus: number) {}
}
const copy = structuredClone({ prices: new Map([["çay", new Money(1500)]]) });
const tea = copy.prices.get("çay");
console.log(
  `Map kept: ${copy.prices instanceof Map}, Money kept: ${tea instanceof Money}`,
);
try {
  structuredClone({ format: (n: number) => n.toFixed(2) });
} catch (e) {
  console.log(`functions: ${e instanceof DOMException ? e.name : e}`);
}

// Transferring moves an ArrayBuffer instead of copying it: the sender's is emptied.
const buffer = new ArrayBuffer(1024);
const moved = structuredClone(buffer, { transfer: [buffer] });
console.log(
  `after transfer: sender ${buffer.byteLength} bytes, receiver ${moved.byteLength}`,
);

export {}; // module: top-level await allowed
