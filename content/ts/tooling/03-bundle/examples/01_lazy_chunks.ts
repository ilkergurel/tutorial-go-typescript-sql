// LEARN(ts.tooling.bundle): a chunk is a module the browser loads and evaluates only when import() runs.

// A static import is loaded and evaluated before this module's first line runs,
// whether or not the code below ever uses it. (data: URLs stand in for chunk files.)
// @ts-expect-error -- tsc can't resolve a data: URL; in an app this is a file path
import { greeting } from "data:text/javascript,console.log('[static] evaluated before main');export const greeting='merhaba'";

console.log(`main starts: ${greeting}`);

// What `vite build` emitted for `await import("./editor")`: a separate file,
// fetched the first time the import() expression runs.
const editorChunk = `data:text/javascript,${encodeURIComponent(`
  console.log("[chunk] evaluating editor");
  const lines = Array.from({ length: 2000 }, (_, i) => "satır " + (i + 1));
  export function openEditor() { return lines.at(-1); }
`)}`;
type EditorModule = { openEditor: () => string };

async function onOpenClicked(): Promise<string> {
  const editor = (await import(editorChunk)) as EditorModule;
  return editor.openEditor();
}
console.log("first click:", await onOpenClicked());
console.log("second click:", await onOpenClicked()); // cached: no second evaluation

// Same URL, same module instance: the module map is keyed by URL.
const [a, b] = await Promise.all([import(editorChunk), import(editorChunk)]);
console.log("same instance:", a === b);

// After a deploy, an old page may ask for a chunk file that is gone. import()
// rejects (a missing file gives a TypeError; this stand-in chunk throws instead),
// so lazy UI needs an error state. Vite also fires a vite:preloadError event.
const missingChunk = "data:text/javascript,throw new Error('chunk failed to load')";
try {
  await import(missingChunk);
} catch (e) {
  console.log("lazy load failed:", e instanceof Error ? e.message : String(e));
}
