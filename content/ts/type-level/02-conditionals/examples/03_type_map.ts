// LEARN(ts.type-level.conditionals): a conditional return type can't be checked inside the function; a type map can.

// Version 1: the return type is computed by a conditional type.
type Kind = "number" | "date" | "text";
type Parsed<K extends Kind> = K extends "number" ? number : K extends "date" ? Date : string;

function parseCell<K extends Kind>(kind: K, raw: string): Parsed<K> {
  // if (kind === "number") return Number(raw);
  // error TS2322: Type 'number' is not assignable to type 'Parsed<K>'.
  if (kind === "number") return Number(raw) as Parsed<K>;
  if (kind === "date") return raw as Parsed<K>; // wrong, and the cast hides it
  return raw.trim() as Parsed<K>;
}

// Version 2: a type map, a mapped type over it, and an indexed access.
interface CellTypes {
  number: number;
  date: Date;
  text: string;
}
const parsers: { [K in keyof CellTypes]: (raw: string) => CellTypes[K] } = {
  number: (raw) => Number(raw),
  date: (raw) => new Date(`${raw}T00:00:00Z`),
  // date: (raw) => raw,
  // error TS2322: Type 'string' is not assignable to type 'Date'.
  text: (raw) => raw.trim(),
};
function parseCell2<K extends keyof CellTypes>(kind: K, raw: string): CellTypes[K] {
  return parsers[kind](raw); // no cast: the call returns CellTypes[K]
}

const due1 = parseCell("date", "2026-12-01"); // typed Date, actually a string
const due2 = parseCell2("date", "2026-12-01");
console.log(typeof due1, due2 instanceof Date, due2.toISOString().slice(0, 10));
console.log(parseCell2("number", "3") + 1, parseCell2("text", "  Şifre sıfırlama "));
// due1.getTime(); // compiles, and throws a TypeError at run time
