// LEARN(ts.tooling.bundle): the cheapest dependency is the one the platform already ships.

// Each line replaces a library people commonly install for it; the browser's
// Intl data and the standard library cost the bundle zero bytes.
const when = new Date("2026-11-13T13:45:00Z");
const tr = "tr-TR";

// Date formatting with time zones (moment-timezone, date-fns-tz):
const date = new Intl.DateTimeFormat(tr, {
  dateStyle: "full",
  timeStyle: "short",
  timeZone: "Europe/Istanbul",
});
console.log("date:     ", date.format(when));

// Relative time ("3 days ago"):
console.log("relative: ", new Intl.RelativeTimeFormat(tr, { numeric: "auto" }).format(-3, "day"));

// Money and number formatting (numeral.js, accounting.js):
const lira = new Intl.NumberFormat(tr, { style: "currency", currency: "TRY" });
console.log("money:    ", lira.format(1234.5), new Intl.NumberFormat(tr).format(0.075));

// Joining a list with the right conjunction:
const list = new Intl.ListFormat(tr, { type: "conjunction" });
console.log("list:     ", list.format(["Go", "TypeScript", "PostgreSQL"]));

// Locale-aware sorting and case mapping (the İ/ı problem):
const names = ["Zeynep", "İlker", "Çağla", "Ilgın", "Şule"];
console.log("sorted:   ", names.toSorted(new Intl.Collator(tr).compare).join(", "));
console.log("upper:    ", "istanbul ılık".toLocaleUpperCase(tr), "vs", "istanbul ılık".toUpperCase());

// Grouping and deep copies (lodash groupBy, cloneDeep):
const lessons = [
  { id: "folders", module: "architecture" },
  { id: "lint", module: "tooling" },
  { id: "state", module: "architecture" },
];
const byModule = Object.groupBy(lessons, (l) => l.module);
console.log("groupBy:  ", Object.entries(byModule).map(([m, ls]) => `${m}=${ls?.length}`).join(" "));
const copy = structuredClone({ when, tags: new Set(["ts"]) });
console.log("clone:    ", copy.when instanceof Date, copy.tags.has("ts"), copy.when !== when);
