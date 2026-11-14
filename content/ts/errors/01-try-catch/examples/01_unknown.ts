// LEARN(ts.errors.try-catch): a catch variable is unknown because JavaScript can throw any value.

// Some libraries throw strings or plain objects instead of Errors.
function legacyLib(): never {
  throw "zaman aşımı";
}
function fetchWrapper(): never {
  throw { status: 404 };
}
const attempts: [string, () => unknown][] = [
  ["JSON.parse", () => JSON.parse("{bozuk")],
  ["new URL", () => new URL("izmir")],
  ["legacy lib", legacyLib],
  ["fetch wrapper", fetchWrapper],
];

// Narrow before use. Engine messages differ between browsers, so print the name.
function describe(e: unknown): string {
  if (e instanceof Error) return `an Error: ${e.name}`;
  if (typeof e === "string") return `a string: "${e}"`;
  return `a non-Error ${typeof e}: ${JSON.stringify(e)}`;
}

for (const [label, run] of attempts) {
  try {
    run();
  } catch (e) {
    // console.log(e.message);
    // error TS18046: 'e' is of type 'unknown'.
    console.log(`${label} threw ${describe(e)}`);
  }
}
// try { run() } catch (e: Error) { ... }
// error TS1196: Catch clause variable type annotation must be 'any' or 'unknown' if specified.

// Promise callbacks are typed by the lib, not by the strict flag: the reason is any.
await Promise.reject("zaman aşımı").catch((e) => console.log(`e.message: ${e.message}`));
await Promise.reject("zaman aşımı").catch((e: unknown) => console.log(`reason: ${describe(e)}`));
