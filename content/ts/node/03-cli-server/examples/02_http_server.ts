import { once } from "node:events";
import { createServer, STATUS_CODES } from "node:http";
import type { AddressInfo } from "node:net";
import { json } from "node:stream/consumers";

const TEXT = { "Content-Type": "text/plain; charset=utf-8" };

// One callback receives every request; routing is ordinary code.
const server = createServer(async (req, res) => {
  try {
    const { pathname } = new URL(req.url ?? "/", "http://localhost");
    const hello = pathname.match(/^\/hello\/([^/]+)$/);
    if (req.method === "GET" && hello) {
      res.end(`Merhaba, ${decodeURIComponent(hello[1])}!\n`); // no Content-Type
    } else if (req.method === "POST" && pathname === "/users") {
      const { name } = (await json(req)) as { name: string };
      res.writeHead(201, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ id: 7, name }));
    } else {
      res.writeHead(404, TEXT).end("not found\n");
    }
  } catch (err) {
    // A throw that escapes an async handler crashes the whole process.
    const bad = err instanceof SyntaxError || err instanceof URIError;
    const status = bad ? 400 : 500;
    res.writeHead(status, TEXT).end(`${STATUS_CODES[status]}\n`);
  }
});

server.listen(0, "127.0.0.1"); // port 0: the OS picks a free port
await once(server, "listening");
const { port } = server.address() as AddressInfo;

async function show(path: string, init?: RequestInit) {
  const res = await fetch(`http://127.0.0.1:${port}${path}`, init);
  const type = res.headers.get("content-type") ?? "(none)";
  console.log(res.status, type.padEnd(25), (await res.text()).trim());
}
await show("/hello/Ayşe"); // fetch percent-encodes the ş
await show("/hello/%C4%B0zmir");
await show("/hello/%ZZ");
const post = (body: string) => ({ method: "POST", body });
await show("/users", post(JSON.stringify({ name: "Çağla" })));
await show("/users", post("{oops"));
await show("/users/7");
server.close(); // stop accepting; idle keep-alive connections close too
