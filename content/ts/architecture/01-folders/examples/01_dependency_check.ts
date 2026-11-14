// LEARN(ts.architecture.folders): check one direction of dependencies over this app's real imports.

// Every import in src/features and src/shared that leaves its own folder
// (this repo on 2026-11-13). app/ may import anything below it, so it's omitted.
const imports = `
features/learning/api          -> features/runner/types
features/learning/LessonNotes  -> features/lessons/types (type)
features/learning/LessonQuiz   -> features/lessons/types (type)
features/learning/LessonQuiz   -> shared/ui/Markdown
features/lessons/ExampleView   -> shared/ui/Markdown
features/lessons/LessonContent -> features/runner/RunPanel
features/lessons/LessonContent -> features/learning/LessonNotes
features/lessons/LessonContent -> features/learning/LessonQuiz
features/lessons/LessonContent -> shared/ui/LevelBadge
features/lessons/LessonContent -> shared/ui/Markdown
features/lessons/types         -> features/curriculum/types (type)
features/runner/judge          -> features/lessons/types (type)
features/runner/RunPanel       -> features/lessons/types
features/runner/RunPanel       -> features/learning/store
shared/ui/LevelBadge           -> features/curriculum/types (type)`;

const layers = ["shared", "features", "app"]; // lowest first

function problem(from: string, to: string): "upward" | "sibling" | null {
  const [fromLayer, fromFeature] = from.split("/");
  const [toLayer, toFeature] = to.split("/");
  if (layers.indexOf(toLayer) > layers.indexOf(fromLayer)) return "upward";
  if (fromLayer === "features" && toLayer === "features" && fromFeature !== toFeature) {
    return "sibling";
  }
  return null; // same feature, or a lower layer
}

const featureEdges = new Set<string>();
let typeOnly = 0;
for (const line of imports.trim().split("\n")) {
  const [, from, to, type] = /^(\S+)\s+-> (\S+)( \(type\))?$/.exec(line)!;
  const why = problem(from, to);
  if (why === null) continue;
  if (type) typeOnly++;
  console.log(`${why.padEnd(8)}${from} -> ${to}${type ?? ""}`);
  if (why === "sibling") featureEdges.add(`${from.split("/")[1]} ${to.split("/")[1]}`);
}
console.log(`${typeOnly} are import type: gone at run time, still coupled in the design`);

// ESLint judges one import string at a time; a cycle only shows in the graph.
for (const edge of [...featureEdges].sort()) {
  const [a, b] = edge.split(" ");
  if (a < b && featureEdges.has(`${b} ${a}`)) console.log(`cycle: ${a} <-> ${b}`);
}
