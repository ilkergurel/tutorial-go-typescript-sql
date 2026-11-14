// LEARN(ts.generics.keyof): a generic key keeps a key and its value type together; a union of keys does not.

interface Settings {
  locale: "tr-TR" | "en-US";
  pageSize: number;
  darkMode: boolean;
}

// Loose: key and value are two independent unions, so any pairing compiles.
function setLoose(s: Settings, key: keyof Settings, value: Settings[keyof Settings]): Settings {
  return { ...s, [key]: value };
}
// Generic: value must be Settings[K] for the K that was actually passed.
function set<K extends keyof Settings>(s: Settings, key: K, value: Settings[K]): Settings {
  return { ...s, [key]: value };
}

const base: Settings = { locale: "tr-TR", pageSize: 20, darkMode: false };
const broken = setLoose(base, "pageSize", "tr-TR"); // compiles
console.log(`setLoose: pageSize is a ${typeof broken.pageSize}`);
// set(base, "pageSize", "tr-TR");
// error TS2345: Argument of type 'string' is not assignable to parameter of type 'number'.
console.log(set(base, "darkMode", true));

// The same idea types a form's change handler for every field at once.
type Handlers<T> = { [K in keyof T]: (value: T[K]) => void };
const log: string[] = [];
const onChange: Handlers<Settings> = {
  locale: (v) => log.push(`locale=${v}`),
  pageSize: (v) => log.push(`pageSize=${Math.min(v, 100)}`), // v: number
  darkMode: (v) => log.push(`darkMode=${v ? "açık" : "kapalı"}`), // v: boolean
};
onChange.pageSize(25);
onChange.darkMode(true);
console.log(log.join("; "));
