// LEARN(ts.browser-dom.reading-files): size guard, await .text(), stream big files.

// In the lesson runner, the file comes from a simulated input; in a real page:
// input.addEventListener("change", () => handleFiles(input.files))

async function readUserFile(file: File): Promise<string> {
  // File extends Blob: bytes + name/type/size/lastModified. No path, ever.
  const MAX_WHOLE_FILE = 10 * 1024 * 1024; // 10 MB

  if (file.size > MAX_WHOLE_FILE) {
    // Too big for .text(): stream chunks at constant memory.
    const reader = file.stream().getReader();
    let bytes = 0;
    for (;;) {
      const { done, value } = await reader.read();
      if (done) break;
      bytes += value.byteLength;
    }
    return `streamed ${bytes} bytes`;
  }

  // Small enough: one awaited call. UTF-8 by default.
  const text = await file.text();
  const lines = text.trimEnd().split("\n");
  return `${file.name}: ${file.size} bytes, ${lines.length} lines, first: ${lines[0]}`;
}

const small = new File(
  ["2026-11-01 INFO started\n2026-11-01 ERROR failed\n"],
  "app.log",
  { type: "text/plain" }
);
console.log(await readUserFile(small));

// One File (name only, no filesystem access) and a fake big file:
console.log(await readUserFile(new File(["x".repeat(11 * 1024 * 1024)], "big.log")));

export {}; // module: top-level await allowed
