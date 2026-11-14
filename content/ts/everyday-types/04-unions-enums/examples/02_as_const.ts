// LEARN(ts.everyday-types.unions-enums): a const array or object plus a derived union replaces an enum, erasably.

// The values exist at run time; the type is derived from them, so they can't drift.
const LEVELS = ["debug", "info", "warn"] as const;
type Level = (typeof LEVELS)[number]; // "debug" | "info" | "warn"

// LEVELS.includes(s) with s: string fails: includes expects a Level.
// error TS2345: Argument of type 'string' is not assignable to parameter of type '"debug" | "info" | "warn"'.
function isLevel(s: string): s is Level {
  return (LEVELS as readonly string[]).includes(s);
}

// Identifiers are ASCII: lowercase them WITHOUT the Turkish locale.
const param = "INFO";
for (const key of [param.toLowerCase(), param.toLocaleLowerCase("tr")]) {
  console.log(isLevel(key) ? `level ${key}` : `unknown level ${key}`);
}

// When you also want named members, use an object with as const.
const Status = {
  Active: "active",
  Suspended: "suspended",
} as const;
type Status = (typeof Status)[keyof typeof Status]; // "active" | "suspended"

function label(s: Status): string {
  return s === Status.Active ? "Aktif" : "Askıda";
}
// Plain strings and named members are interchangeable: no cast needed.
console.log(label("active"), label(Status.Suspended));
console.log(Object.values(Status).join(","));
