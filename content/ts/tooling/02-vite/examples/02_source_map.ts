// LEARN(ts.tooling.vite): a source map is a table from generated positions back to the TypeScript.

// Real esbuild 0.28 output for a 13-line average.ts, built with --minify --sourcemap.
const generated =
  'function t(e){return e.reduce((n,r)=>n+r.minutes,0)}export function average(e){if(e.length===0)throw new Error("no lessons");return t(e)/e.length}';
const map = {
  sources: ["average.ts"],
  sourcesContent: [
    'interface Lesson {\n  id: string;\n  minutes: number;\n}\n\nfunction totalMinutes(lessons: Lesson[]): number {\n  return lessons.reduce((sum, l) => sum + l.minutes, 0);\n}\n\nexport function average(lessons: Lesson[]): number {\n  if (lessons.length === 0) throw new Error("no lessons");\n  return totalMinutes(lessons) / lessons.length;\n}\n',
  ],
  names: ["totalMinutes", "lessons", "sum", "l"],
  mappings:
    "AAKA,SAASA,EAAaC,EAA2B,CAC/C,OAAOA,EAAQ,OAAO,CAACC,EAAKC,IAAMD,EAAMC,EAAE,QAAS,CAAC,CACtD,CAEO,gBAAS,QAAQF,EAA2B,CACjD,GAAIA,EAAQ,SAAW,EAAG,MAAM,IAAI,MAAM,YAAY,EACtD,OAAOD,EAAaC,CAAO,EAAIA,EAAQ,MACzC",
};

// Base64 VLQ: 5 data bits per character, the 6th bit means "more follows",
// and the lowest bit of the decoded number is its sign.
const BASE64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
function vlq(segment: string): number[] {
  const numbers: number[] = [];
  let value = 0;
  let shift = 0;
  for (const ch of segment) {
    const digit = BASE64.indexOf(ch);
    value += (digit & 31) << shift;
    shift += 5;
    if (digit & 32) continue;
    numbers.push(value & 1 ? -(value >> 1) : value >> 1);
    value = shift = 0;
  }
  return numbers;
}

// A segment is [generated column, source, line, column, name?], each a delta from
// the previous segment. ";" would start the next generated line; minified code has one.
type Segment = { genCol: number; line: number; col: number; name?: string };
const segments: Segment[] = [];
const at = { genCol: 0, line: 0, col: 0, name: 0 }; // running totals
for (const text of map.mappings.split(",")) {
  const [dGenCol, , dLine, dCol, dName] = vlq(text);
  at.genCol += dGenCol;
  at.line += dLine;
  at.col += dCol;
  if (dName !== undefined) at.name += dName;
  const name = dName === undefined ? undefined : map.names[at.name];
  segments.push({ genCol: at.genCol, line: at.line, col: at.col, name });
}

// What DevTools does with a stack frame such as "average.js:1:96".
const source = map.sourcesContent[0].split("\n");
for (const needle of ["n+r.minutes", "throw", "t(e)/"]) {
  const column = generated.indexOf(needle);
  const s = segments.findLast((seg) => seg.genCol <= column)!;
  const from = `1:${column + 1}`.padEnd(6) + needle.padEnd(12);
  const to = `${map.sources[0]}:${s.line + 1}:${s.col + 1}${s.name ? ` [${s.name}]` : ""}`;
  console.log(`${from}-> ${to.padEnd(34)}${source[s.line].trim()}`);
}
