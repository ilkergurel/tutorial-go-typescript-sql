interface Circle { kind: "circle"; radius: number }
interface Rect { kind: "rect"; w: number; h: number }
type Shape = Circle | Rect;

// Type predicate: narrows `shape` to Circle wherever it returns true.
function isCircle(shape: Shape): shape is Circle {
  return shape.kind === "circle";
}

// Assertion function: throws instead of branching; code after the call can
// assume a Circle with no else branch.
function assertCircle(shape: Shape): asserts shape is Circle {
  if (shape.kind !== "circle") throw new Error("expected a circle");
}

function area(shape: Shape): number {
  switch (shape.kind) {
    case "circle": return Math.PI * shape.radius ** 2;
    case "rect":   return shape.w * shape.h;
    default: {
      // Exhaustiveness: if a variant is added without a case, `rest` is not
      // never and this line stops compiling.
      const rest: never = shape;
      return rest;
    }
  }
}

// satisfies: validate that every Shape variant is covered WITHOUT widening
// the values, so each function keeps its specific parameter type.
const renderers = {
  circle: (c: Circle) => `circle r=${c.radius}`,
  rect:   (r: Rect) => `rect ${r.w}x${r.h}`,
} satisfies { [k in Shape["kind"]]: (s: Extract<Shape, { kind: k }>) => string };

const shapes: Shape[] = [
  { kind: "circle", radius: 2 },
  { kind: "rect", w: 2, h: 5 },
];

for (const s of shapes) {
  const line = isCircle(s) ? renderers.circle(s) : renderers.rect(s);
  console.log(line, "area=" + area(s).toFixed(1));
}

const c: Shape = { kind: "circle", radius: 3 };
assertCircle(c);
console.log("asserted radius", c.radius);
