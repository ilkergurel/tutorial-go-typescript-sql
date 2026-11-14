// LEARN(ts.services.advanced): download progress comes from reading the body stream; for uploads, send parts or use XMLHttpRequest.

// Every letter here is 2 bytes in UTF-8: progress is about bytes, not length.
const text = "ğüşıöç".repeat(5000);
const file = new File([text], "müşteriler.csv");
console.log(`${text.length} characters, ${file.size} bytes`);
const pct = (n: number, total: number) => `${Math.round((100 * n) / total)}%`;

// Download: the fake server streams the bytes in 16 KiB chunks.
const bytes = new Uint8Array(await file.arrayBuffer());
const server: typeof fetch = async () => {
  const body = new ReadableStream<Uint8Array>({
    start(controller) {
      for (let i = 0; i < bytes.length; i += 16_384) {
        controller.enqueue(bytes.subarray(i, i + 16_384));
      }
      controller.close();
    },
  });
  return new Response(body, {
    headers: { "Content-Length": String(bytes.length) },
  });
};

async function download(res: Response, onProgress: (n: number) => void) {
  const chunks: BlobPart[] = [];
  let loaded = 0;
  const reader = res.body?.getReader();
  for (;;) {
    const chunk = await reader?.read();
    if (!chunk || chunk.done) break;
    chunks.push(chunk.value);
    loaded += chunk.value.byteLength;
    onProgress(loaded);
  }
  return new Blob(chunks);
}

const res = await server("https://api.example.com/export.csv");
const total = Number(res.headers.get("Content-Length"));
const blob = await download(res, (n) =>
  console.log(`download ${pct(n, total)}`),
);
console.log(`downloaded ${blob.size} bytes`);

// Upload: fetch has no progress events. Send slices; each acknowledged part
// is real progress, and a failed part can be resent on its own.
const uploads: typeof fetch = async (_input, init) =>
  new Response(null, { status: init?.body instanceof Blob ? 204 : 400 });

async function upload(
  f: Blob,
  partSize: number,
  onProgress: (n: number) => void,
) {
  for (let offset = 0; offset < f.size; offset += partSize) {
    const part = f.slice(offset, offset + partSize); // bytes, not characters
    const url = `https://api.example.com/uploads/42?offset=${offset}`;
    const res = await uploads(url, { method: "PUT", body: part });
    if (!res.ok) throw new Error(`part at ${offset}: HTTP ${res.status}`);
    onProgress(offset + part.size);
  }
}
await upload(file, 25_000, (n) => console.log(`upload ${pct(n, file.size)}`));
