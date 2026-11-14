// LEARN(ts.mental-model.tsconfig): exactOptionalPropertyTypes separates "missing" from "undefined".

const defaults = { theme: "light", pageSize: 20 };

// What `theme?: ...` means WITHOUT the flag: absent, or present and undefined.
interface LoosePrefs {
  theme?: "dark" | "light" | undefined;
}

// A cleared form field copied as undefined: spread copies it over the default.
const loose: LoosePrefs = { theme: undefined };
const mergedLoose = { ...defaults, ...loose };
console.log("theme" in loose, mergedLoose.theme);

// WITH the flag, `theme?: ...` means the key is either absent or a real value:
interface Prefs {
  theme?: "dark" | "light";
}
// const cleared: Prefs = { theme: undefined };
// error TS2375: Type '{ theme: undefined; }' is not assignable to type 'Prefs' with 'exactOptionalPropertyTypes: true'. Consider adding 'undefined' to the types of the target's properties.

// So code that builds a Prefs has to leave the key out instead.
function fromForm(theme: string): Prefs {
  return theme === "dark" || theme === "light" ? { theme } : {};
}
const prefs = fromForm("");
const merged = { ...defaults, ...prefs };
console.log("theme" in prefs, merged.theme);

// JSON hides the difference, which is why the bug survives code review.
console.log(JSON.stringify(loose), JSON.stringify(prefs));
