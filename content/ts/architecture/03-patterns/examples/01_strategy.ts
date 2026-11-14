// LEARN(ts.architecture.patterns): a strategy is a function value; a family of them is a typed record.

type Lesson = { title: string; level: "basic" | "advanced"; minutes: number };
type Compare = (a: Lesson, b: Lesson) => number;

// Java shape, for contrast: interface SortStrategy { int compare(Lesson a, Lesson b); }
// plus one class per strategy and a factory to pick one. Here the function type is
// the interface, and every function with that shape implements it.

const turkish = new Intl.Collator("tr");
const sorters = {
  title: (a, b) => turkish.compare(a.title, b.title),
  level: (a, b) => Number(a.level === "advanced") - Number(b.level === "advanced"),
  minutes: (a, b) => a.minutes - b.minutes,
} satisfies Record<string, Compare>;
type SortKey = keyof typeof sorters; // a union of the three keys

// Composing strategies is a higher-order function, not a decorator class.
const thenBy =
  (...rules: Compare[]): Compare =>
  (a, b) => {
    for (const rule of rules) {
      const order = rule(a, b);
      if (order !== 0) return order;
    }
    return 0;
  };
const reversed =
  (rule: Compare): Compare =>
  (a, b) => rule(b, a);

const lessons: Lesson[] = [
  { title: "Şablonlar", level: "basic", minutes: 12 },
  { title: "İndeksler", level: "advanced", minutes: 14 },
  { title: "Zaman", level: "basic", minutes: 12 },
  { title: "Çeviri", level: "advanced", minutes: 9 },
];
const show = (label: string, list: Lesson[]) =>
  console.log(label.padEnd(13) + list.map((l) => l.title).join(", "));
const sortBy = (key: SortKey) => lessons.toSorted(sorters[key]);

// < on strings (like sort() without a comparator) compares UTF-16 code units:
// Ç, İ and Ş land after Z.
show("code units", lessons.toSorted((a, b) => (a.title < b.title ? -1 : 1)));
show("title", sortBy("title"));
show("minutes", sortBy("minutes"));
show("level, -min", lessons.toSorted(thenBy(sorters.level, reversed(sorters.minutes))));
// sortBy("popularity");
// error TS2345: Argument of type '"popularity"' is not assignable to parameter of type '"level" | "minutes" | "title"'.
