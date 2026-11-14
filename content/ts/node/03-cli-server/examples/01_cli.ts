import { createInterface } from "node:readline";
import { Readable } from "node:stream";
import { text } from "node:stream/consumers";
import { parseArgs } from "node:util";

const LEVELS = ["DEBUG", "INFO", "WARN", "ERROR"];
type Print = (line: string) => void;

// The whole CLI as a function: arguments and streams in, exit status out.
async function main(argv: string[], stdin: Readable, out: Print, err: Print) {
  let opts;
  try {
    ({ values: opts } = parseArgs({
      args: argv,
      options: {
        min: { type: "string", default: "INFO" },
        json: { type: "boolean", short: "j", default: false },
      },
    }));
  } catch (e) {
    err(`levels: ${(e as Error).message}`);
    err("usage: levels [--min LEVEL] [-j] < app.log");
    return 2;
  }
  const min = LEVELS.indexOf(opts.min.toUpperCase()); // locale-independent
  if (min < 0) {
    err(`levels: unknown level "${opts.min}"`);
    return 2;
  }

  const counts = new Map<string, number>();
  const lines = createInterface({ input: stdin, crlfDelay: Infinity });
  for await (const line of lines) {
    const level = line.split(" ")[1] ?? "";
    if (LEVELS.indexOf(level) < min) continue;
    counts.set(level, (counts.get(level) ?? 0) + 1);
  }
  if (opts.json) out(JSON.stringify(Object.fromEntries(counts)));
  else for (const [level, n] of counts) out(`${level.padEnd(5)} ${n}`);
  return 0;
}

// In a real CLI the entry point is:
//   const argv = process.argv.slice(2);
//   process.exitCode = await main(argv, process.stdin, console.log, console.error);
// This demo reads stdin once and replays it for several command lines.
const input = await text(process.stdin);
const toStderr: Print = (s) => console.log(`(stderr) ${s}`);
for (const argv of [
  ["--min", "warn"],
  ["-j"],
  ["--min"],
  ["--level=ERROR"],
  ["--min", "trace"],
]) {
  console.log(`$ levels ${argv.join(" ")} < app.log`);
  const code = await main(argv, Readable.from([input]), console.log, toStderr);
  console.log(`exit status ${code}`);
}
