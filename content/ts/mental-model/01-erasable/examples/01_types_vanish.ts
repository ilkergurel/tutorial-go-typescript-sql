// LEARN(ts.mental-model.erasable): types are erased; nothing checks them at run time.

interface User {
  name: string;
  age: number;
}

function describe(u: User): string {
  return `${u.name} (${u.age})`;
}

// A function's source text is what actually runs: the annotations are gone.
console.log(describe.toString());

// An interface leaves no value behind to inspect:
// console.log(typeof User);
// error TS2693: 'User' only refers to a type, but is being used as a value here.

// A type is a claim the checker trusts, not a conversion. JSON.parse returns
// any, so this compiles, and at run time age is still the string "41".
const parsed: User = JSON.parse('{"name":"Ayşe","age":"41"}');
console.log(typeof parsed.age, parsed.age + 1);

// @ts-expect-error -- tsc rejects this line (TS2322); esbuild runs it anyway.
const port: number = "8080";
console.log(typeof port, port + 1);
