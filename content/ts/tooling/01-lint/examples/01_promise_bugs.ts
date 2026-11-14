// LEARN(ts.tooling.lint): three promise bugs that tsc accepts and type-aware lint rejects.

async function isAdmin(login: string): Promise<boolean> {
  await Promise.resolve(); // stands in for a request to the server
  return login === "ayşe";
}
const logins = ["ayşe", "mehmet", "ışıl"];

// no-misused-promises: an async predicate returns a Promise, which is truthy.
const admins = logins.filter(async (login) => await isAdmin(login));
console.log("filter(async):  ", admins.join(", "));

// no-misused-promises: forEach ignores the promises its callback returns.
function countAdmins(): number {
  let count = 0;
  logins.forEach(async (login) => {
    if (await isAdmin(login)) count++;
  });
  return count; // runs before any callback has resumed
}
console.log("forEach(async): ", countAdmins());

// no-floating-promises: the caller moves on before the work is done.
let savedAt = "never";
async function saveDraft(): Promise<void> {
  await Promise.resolve();
  savedAt = "12:00";
}
function submit(): string {
  saveDraft(); // nobody awaits it, and nobody would see it fail
  return savedAt;
}
console.log("floating:       ", submit());

// The fixes: await, and collect parallel work with Promise.all.
const flags = await Promise.all(logins.map(isAdmin));
console.log("Promise.all:    ", logins.filter((_, i) => flags[i]).join(", "));
console.log("savedAt later:  ", savedAt);
