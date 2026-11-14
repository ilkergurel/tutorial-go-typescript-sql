import { open } from "node:fs/promises";

const file = await open("app.log"); // like os.Open: a missing file fails here

const counts = new Map<string, number>();
for await (const line of file.readLines()) {
  // one line per iteration, "\n" (and "\r\n") stripped
  const fields = line.trim().split(/\s+/); // like strings.Fields
  if (fields.length < 2) continue; // skip malformed lines
  const level = fields[1]; // INFO, WARN, ERROR
  counts.set(level, (counts.get(level) ?? 0) + 1);
}
// The loop closed the file when it ended; a throw or break closes it too.
console.log("closed:", file.fd === -1);

// A Map iterates in insertion order (INFO, WARN, ERROR): stable, but sort
// anyway to print what the Go version prints.
for (const level of [...counts.keys()].sort()) {
  console.log(`${level.padEnd(5)} ${counts.get(level)}`);
}
