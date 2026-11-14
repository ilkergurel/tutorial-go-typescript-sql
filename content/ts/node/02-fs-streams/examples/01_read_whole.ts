import { readFile, writeFile } from "node:fs/promises";

const data = await readFile("app.log"); // whole file -> Buffer (bytes)
const lines = data.toString("utf8").trimEnd().split("\n");
console.log(`read ${data.length} bytes, ${lines.length} lines`);
console.log("first line:", lines[0]);

// A failure is a rejected promise; err.code carries the POSIX name.
try {
  await readFile("missing.log", "utf8");
} catch (err) {
  const { code } = err as NodeJS.ErrnoException;
  console.log("missing.log:", code === "ENOENT" ? "does not exist" : code);
}

// Bytes vs characters: a Buffer counts bytes, a string counts UTF-16 units.
await writeFile("iller.txt", "Iğdır,Şanlıurfa,Çanakkale\n");
const raw = await readFile("iller.txt");
const text = await readFile("iller.txt", "utf8");
console.log(`${raw.length} bytes, ${text.length} characters`);
console.log(raw.subarray(0, 3)); // "I" is one byte, "ğ" is two
