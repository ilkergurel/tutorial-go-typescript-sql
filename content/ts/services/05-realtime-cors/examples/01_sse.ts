// LEARN(ts.services.realtime-cors): an SSE stream is lines of text; parse it from fetch's body when EventSource can't send your headers.

interface ServerEvent {
  type: string;
  data: string;
  id: string;
}

// The event-stream format, as EventSource interprets it.
async function* events(
  body: ReadableStream<BufferSource>,
): AsyncGenerator<ServerEvent> {
  const lines = body.pipeThrough(new TextDecoderStream()).getReader();
  let rest = "";
  let type = "";
  let data = "";
  let id = ""; // the last event ID carries over to later events
  for (;;) {
    const { done, value } = await lines.read();
    if (done) return; // an event without its blank line is dropped
    const parts = (rest + value).split(/\r?\n/);
    rest = parts.pop() ?? ""; // maybe half a line: wait for the next chunk
    for (const line of parts) {
      if (line.startsWith(":")) continue; // a comment, often a keep-alive
      if (line === "") {
        // A blank line ends the event; without data lines it dispatches nothing.
        const event = { type: type || "message", data: data.slice(0, -1), id };
        if (data) yield event;
        type = data = "";
        continue;
      }
      const colon = line.includes(":") ? line.indexOf(":") : line.length;
      const field = line.slice(0, colon);
      const val = line.slice(colon + 1).replace(/^ /, ""); // one space only
      if (field === "event") type = val;
      if (field === "data") data += val + "\n";
      if (field === "id") id = val;
      // "retry" sets EventSource's reconnection delay; other fields are ignored
    }
  }
}

const wire =
  ": ok\n\n" +
  'event: order\nid: 41\ndata: {"id":41,"city":"İzmir"}\n\n' +
  "data: first line\ndata: second line\n\n" +
  "retry: 5000\nid: 42\nevent: ping\ndata:\n\n" +
  "data: cut off before the blank line";

// The network delivers bytes in arbitrary pieces: 10 bytes here, which
// splits the two-byte İ between chunks.
const bytes = new TextEncoder().encode(wire);
const chunks: Uint8Array[] = [];
for (let i = 0; i < bytes.length; i += 10) {
  chunks.push(bytes.subarray(i, i + 10));
}
const perChunk = chunks.map((c) => new TextDecoder().decode(c)).join("");
console.log(`decoded chunk by chunk: ${/"city":"(.*?)"/.exec(perChunk)?.[1]}`);

const body = new ReadableStream<Uint8Array>({
  start(controller) {
    for (const chunk of chunks) controller.enqueue(chunk);
    controller.close();
  },
});
// In a page: const res = await fetch(url, { headers: { Authorization } });
const res = new Response(body, {
  headers: { "Content-Type": "text/event-stream" },
});
if (!res.body) throw new Error("no body");
for await (const e of events(res.body)) {
  console.log(`${e.type} #${e.id || "-"}: ${JSON.stringify(e.data)}`);
}
