// LEARN(ts.services.realtime-cors): a WebSocket client: validate every message, resubscribe after a reconnect, back off.
import { z } from "zod";

// The protocol is yours to design: a discriminated union, checked on arrival.
const ServerMsg = z.discriminatedUnion("type", [
  z.object({ type: z.literal("order"), id: z.int(), status: z.string() }),
  z.object({ type: z.literal("error"), message: z.string() }),
]);

const parseJson = (text: string): unknown => {
  try {
    return JSON.parse(text);
  } catch {
    return undefined; // a malformed frame must not kill the listener
  }
};

// The part of WebSocket the client uses: the real one fits, and so does a fake.
type Socket = Pick<WebSocket, "send" | "addEventListener">;

function subscribe(
  connect: () => Socket,
  topic: string,
  later: (fn: () => void, ms: number) => void, // setTimeout in production
) {
  let failures = 0;
  const open = () => {
    const ws = connect();
    ws.addEventListener("open", () => {
      failures = 0;
      // Subscriptions live in the connection: a new one starts empty.
      ws.send(JSON.stringify({ type: "subscribe", topic }));
    });
    ws.addEventListener("message", (e) => {
      const msg = ServerMsg.safeParse(parseJson(String(e.data)));
      if (!msg.success) console.log("  dropped a malformed message");
      else if (msg.data.type === "order") {
        console.log(`  order ${msg.data.id}: ${msg.data.status}`);
      } else console.log(`  server says: ${msg.data.message}`);
    });
    ws.addEventListener("close", (e) => {
      if (e.wasClean) return; // we, or the server, closed on purpose
      const wait = Math.min(30_000, 1000 * 2 ** failures++); // add jitter too
      console.log(`  closed (${e.code}), reconnecting in ${wait} ms`);
      later(open, wait);
    });
  };
  open();
}

// A stand-in for new WebSocket(url): same events, driven by the example.
class FakeSocket extends EventTarget {
  send(data: string) {
    console.log(`  sent ${data}`);
  }
  emit(event: Event) {
    this.dispatchEvent(event);
  }
}
const sockets: FakeSocket[] = [];
const connect = () => sockets[sockets.push(new FakeSocket()) - 1];
subscribe(connect, "orders", (fn) => fn()); // reconnect without waiting

const msg = (data: string) => new MessageEvent("message", { data });
const [first] = sockets;
first.emit(new Event("open"));
first.emit(msg('{"type":"order","id":41,"status":"paid"}'));
first.emit(msg('{"type":"order","id":"41"}'));
first.emit(msg("<html>"));
first.emit(new CloseEvent("close", { code: 1006, wasClean: false }));
sockets[1].emit(new Event("open"));
sockets[1].emit(msg('{"type":"order","id":41,"status":"shipped"}'));
