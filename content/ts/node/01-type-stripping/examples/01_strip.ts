import { stripTypeScriptTypes } from "node:module";

interface City {
  name: string;
  plate: number;
}

function label(city: City, upper = false): string {
  const name = upper ? city.name.toLocaleUpperCase("tr") : city.name;
  return `${String(city.plate).padStart(2, "0")} ${name}`;
}

console.log(label({ name: "İzmir", plate: 35 }));
console.log(label({ name: "diyarbakır", plate: 21 }, true));

// What Node does before V8 sees the code: type syntax becomes spaces (·).
for (const src of [
  "function label(city: City, upper = false): string {}",
  "const n = cities.at(-1)!.plate as number; interface X { a: 1 }",
]) {
  console.log(src);
  console.log(stripTypeScriptTypes(src).replaceAll(" ", "·"));
}

// Same lines and columns, so a stack trace points into this file as written.
const frame = new Error("where am I").stack?.split("\n")[1] ?? "";
console.log("Error created on line", frame.match(/:(\d+):\d+\)?$/)?.[1]);

// Nothing is type-checked: tsc rejects the call below, Node runs it.
// @ts-expect-error -- plate is missing
console.log(label({ name: "Ankara" }));
