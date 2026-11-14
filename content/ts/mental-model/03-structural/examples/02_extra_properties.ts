// LEARN(ts.mental-model.structural): extra properties pass the type check and survive at run time.

interface PublicUser {
  name: string;
}

// A fresh object literal gets an excess-property check...
// const u: PublicUser = { name: "Ayşe", passwordHash: "x" };
// error TS2353: Object literal may only specify known properties, and 'passwordHash' does not exist in type 'PublicUser'.

interface DbUser {
  name: string;
  passwordHash: string;
}
const row: DbUser = { name: "Ayşe", passwordHash: "$2b$10$k1f..." };

// ...but a variable is checked by shape only, and a DbUser has every field a
// PublicUser needs. The annotation does not remove anything.
function toJson(u: PublicUser): string {
  return JSON.stringify(u);
}
console.log(toJson(row));

// Copy the fields you mean to expose: code removes properties, types don't.
function toPublic(u: DbUser): PublicUser {
  return { name: u.name };
}
console.log(toJson(toPublic(row)));
