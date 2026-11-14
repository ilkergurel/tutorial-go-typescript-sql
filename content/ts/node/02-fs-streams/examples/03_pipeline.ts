import { createReadStream, createWriteStream } from "node:fs";
import { open, readFile, stat, writeFile } from "node:fs/promises";
import { pipeline } from "node:stream/promises";

// Chunks are bytes. Decoding each 4-byte chunk alone splits letters apart.
await writeFile("iller.txt", "İstanbul Şanlıurfa Iğdır\n");
let broken = "";
for await (const chunk of createReadStream("iller.txt", { highWaterMark: 4 })) {
  broken += (chunk as Buffer).toString("utf8");
}
// With an encoding, the stream decodes across chunk boundaries.
let fixed = "";
const utf8 = { highWaterMark: 4, encoding: "utf8" } as const;
for await (const chunk of createReadStream("iller.txt", utf8)) {
  fixed += chunk;
}
console.log("chunk by chunk:", broken.trimEnd());
console.log("stream decoded:", fixed.trimEnd());

// A transform is an async generator: lines in, lines out.
async function* errorsOnly(lines: AsyncIterable<string>) {
  for await (const line of lines) {
    if (line.includes(" ERROR ")) yield line + "\n";
  }
}
const log = await open("app.log");
await pipeline(log.readLines(), errorsOnly, createWriteStream("errors.log"));
console.log((await readFile("errors.log", "utf8")).trimEnd());

// A failing stage rejects the pipeline and destroys every other stage.
const out = createWriteStream("partial.log");
try {
  await pipeline(createReadStream("missing.log"), out);
} catch (err) {
  const { code } = err as NodeJS.ErrnoException;
  const { size } = await stat("partial.log");
  console.log(`${code}, destination destroyed: ${out.destroyed}, size ${size}`);
}
