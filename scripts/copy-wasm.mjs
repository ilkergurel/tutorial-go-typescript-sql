// Copies the esbuild wasm binary into public/ so the worker can fetch it from
// a stable URL regardless of how the bundler hashes asset names.
import { copyFileSync, mkdirSync, existsSync } from "node:fs";
import { join } from "node:path";

const src = join("node_modules", "esbuild-wasm", "esbuild.wasm");
const outDir = "public";
if (!existsSync(src)) {
  console.error("esbuild-wasm not installed — run npm install first");
  process.exit(1);
}
mkdirSync(outDir, { recursive: true });
copyFileSync(src, join(outDir, "esbuild.wasm"));
console.log("copied esbuild.wasm -> public/");
