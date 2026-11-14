import { writeFile } from "node:fs/promises";

// Each construct Node refuses, followed by its erasable spelling.
const modules: Record<string, string> = {
  "enum.ts": `export enum Level { Info, Warn }`,
  "as-const.ts": `export const Level = { Info: "info", Warn: "warn" } as const;
export type Level = (typeof Level)[keyof typeof Level];`,
  "param-prop.ts": `export class Point { constructor(public x: number) {} }`,
  "field.ts": `export class Point { x: number; constructor(x: number) { this.x = x; } }`,
  "namespace.ts": `export namespace Geo { export const R = 6371; }`,
  "plain.ts": `export const R = 6371;`,
  "types.ts": `export interface User { name: string }`,
  "value-import.ts": `import { User } from "./types.ts";`,
  "type-import.ts": `import type { User } from "./types.ts";`,
  "no-ext.ts": `import { R } from "./plain";`,
};

for (const [file, source] of Object.entries(modules)) {
  await writeFile(file, source);
}

for (const file of Object.keys(modules)) {
  try {
    const mod = await import(`./${file}`);
    const exports = Object.keys(mod).join(", ") || "(nothing)";
    console.log(`${file.padEnd(16)} ok, exports: ${exports}`);
  } catch (err) {
    const { code, name, message } = err as NodeJS.ErrnoException;
    console.log(`${file.padEnd(16)} ${code ?? name}`);
    if (name === "SyntaxError") {
      console.log(" ".repeat(17) + firstLine(message));
    }
  }
}

// Node 22 puts the message inside a code frame ("  x message"); 24 does not.
function firstLine(message: string): string {
  return message.trim().replace(/^x /, "").split("\n")[0];
}
