// LEARN(ts.functions.higher-order): overloads tie return types to argument types; prefer unions.

// Two overload signatures, then one implementation. Callers see only the
// overloads: a string gives a number, an array gives number[].
function parseAmount(text: string): number;
function parseAmount(texts: readonly string[]): number[];
function parseAmount(input: string | readonly string[]): number | number[] {
  const one = (t: string) => Number(t.replaceAll(".", "").replace(",", "."));
  return typeof input === "string" ? one(input) : input.map(one);
}

const single = parseAmount("1.234,50"); // number
const many = parseAmount(["12,5", "3"]); // number[]
console.log(single + 1, many.length);

// The cost: the implementation signature is invisible, so a value that is
// "string or array" matches no overload and the caller has to split it.
function fromField(value: string | string[]): number | number[] {
  // return parseAmount(value);
  //   error TS2769: No overload matches this call.
  return typeof value === "string" ? parseAmount(value) : parseAmount(value);
}
console.log(fromField("7,5"), fromField(["1", "2"]));

// Same return type for every argument shape? A union parameter is simpler:
// no implementation signature, and the mixed case just works.
function letterCount(input: string | readonly string[]): number {
  return typeof input === "string" ? input.length : input.join("").length;
}
console.log(letterCount("ılık"), letterCount(["su", "çay"]));

// You already use overloads: the DOM typings pick the element type from the
// tag name, so canvas is an HTMLCanvasElement, not a plain HTMLElement.
const canvas = document.createElement("canvas");
console.log(canvas.tagName, canvas.width);
