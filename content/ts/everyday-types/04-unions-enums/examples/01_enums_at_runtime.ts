// LEARN(ts.everyday-types.unions-enums): enums emit objects, accept any number, and reject plain strings.

enum Level {
  Debug,
  Info,
  Warn,
}

// A numeric enum is an object with a reverse mapping, so it has twice the keys.
console.log(Object.keys(Level).join(","));
console.log(`${Level.Info} ${Level[1]}`);

// Literals are checked (TS 5.0+), but any number value is accepted:
// const bad: Level = 7;
// error TS2322: Type '7' is not assignable to type 'Level'.
const fromDb: number = 7;
const level: Level = fromDb; // compiles
console.log(`${level} -> ${Level[level]}`);

// String enums have no reverse mapping, and they are nominal: the plain
// string "active" is not a Status, even though that is its runtime value.
enum Status {
  Active = "active",
  Suspended = "suspended",
}
function label(s: Status): string {
  return s === Status.Active ? "Aktif" : "Askıda";
}
// label("active");
// error TS2345: Argument of type '"active"' is not assignable to parameter of type 'Status'.
const fromApi: string = "active";
console.log(label(Status.Active), label(fromApi as Status));
console.log(JSON.stringify(Status));
