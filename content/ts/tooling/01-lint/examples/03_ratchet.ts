// LEARN(ts.tooling.lint): ratchet a new check in: fail on new errors, not on the ones you already have.
import { readFileSync } from "node:fs";

// Real output of: tsc --noEmit -p tsconfig.app.json --noUncheckedIndexedAccess
const output = readFileSync("unchecked-index.txt", "utf8");
// Errors per file, recorded when the flag was first measured (invented for this demo).
const baseline: Record<string, number> = JSON.parse(readFileSync("baseline.json", "utf8"));

const current = new Map<string, number>();
for (const line of output.split("\n")) {
  const match = /^(\S+)\(\d+,\d+\): error TS\d+:/.exec(line);
  if (match) current.set(match[1], (current.get(match[1]) ?? 0) + 1);
}

let failed = false;
const files = new Set([...current.keys(), ...Object.keys(baseline)]);
for (const file of [...files].sort()) {
  const before = baseline[file] ?? 0;
  const now = current.get(file) ?? 0;
  const verdict = now > before ? "FAIL: new errors" : now < before ? "ok, lower the baseline" : "ok";
  if (now > before) failed = true;
  console.log(`${file.padEnd(40)}${String(before).padStart(3)} ->${String(now).padStart(3)}  ${verdict}`);
}
const total = [...current.values()].reduce((a, b) => a + b, 0);
console.log(`${total} errors in ${current.size} files; ratchet ${failed ? "FAILED" : "passed"}`);
// In CI, end with: process.exitCode = failed ? 1 : 0;
