// LEARN(ts.mental-model.structural): compatibility is decided by shape, not by declared names.

interface Point {
  x: number;
  y: number;
}

function norm(p: Point): number {
  return Math.hypot(p.x, p.y);
}

// No `implements Point`: having numeric x and y is enough. Extra members are fine.
class Pixel {
  x: number;
  y: number;
  color: string;
  constructor(x: number, y: number, color: string) {
    this.x = x;
    this.y = y;
    this.color = color;
  }
}
console.log(norm(new Pixel(3, 4, "red")), norm({ x: 6, y: 8 }));

// The flip side: same shape, different meaning, no error.
interface Meters {
  value: number;
}
interface Feet {
  value: number;
}
function climb(height: Meters): string {
  return `climbing ${height.value} m`;
}
const everest: Feet = { value: 29032 };
console.log(climb(everest)); // compiles; wrong by a factor of 3.28

// A #private field makes a class nominal: only instances of Altitude have it.
class Altitude {
  readonly #meters: number;
  constructor(meters: number) {
    this.#meters = meters;
  }
  get meters(): number {
    return this.#meters;
  }
}
// const fake: Altitude = { meters: 29032 };
// error TS2741: Property '#meters' is missing in type '{ meters: number; }' but required in type 'Altitude'.
const real = new Altitude(8849);
console.log(real.meters, real instanceof Altitude);
