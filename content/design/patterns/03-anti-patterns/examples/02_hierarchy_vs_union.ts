// LEARN(design.patterns.anti-patterns): a deep class hierarchy vs a discriminated union with an exhaustive switch.

// Java-shaped: each level overrides a little, and defaults live far away.
class Item {
  constructor(readonly title: string) {}
  minutes(): number {
    return 15; // a "sensible default" for every subclass, forever
  }
  label() {
    return `${this.title}: ${this.minutes()} min`;
  }
}
class Lesson extends Item {
  constructor(title: string, readonly examples: number) {
    super(title);
  }
  minutes() {
    return 4 + 3 * this.examples;
  }
}
class SqlLesson extends Lesson {
  label() {
    return `[sql] ${super.label()}`;
  }
}
// Added months later: "an exam is a kind of lesson", so it extends Lesson.
class Exam extends Lesson {
  constructor(title: string, readonly questions: number) {
    super(title, 0);
  }
}

for (const item of [new Lesson("Defer", 2), new SqlLesson("Joins", 3), new Exam("Final", 30)]) {
  console.log("class:", item.label());
}

// TS-shaped: the variants are data; behaviour is a function over the union.
type Content =
  | { kind: "lesson"; title: string; examples: number; sql: boolean }
  | { kind: "exam"; title: string; questions: number };

function minutes(c: Content): number {
  switch (c.kind) {
    case "lesson":
      return 4 + 3 * c.examples;
    case "exam":
      return 2 * c.questions;
  }
}
const label = (c: Content) =>
  `${c.kind === "lesson" && c.sql ? "[sql] " : ""}${c.title}: ${minutes(c)} min`;

const items: Content[] = [
  { kind: "lesson", title: "Defer", examples: 2, sql: false },
  { kind: "lesson", title: "Joins", examples: 3, sql: true },
  { kind: "exam", title: "Final", questions: 30 },
];
for (const c of items) console.log("union:", label(c));

// Add | { kind: "project"; title: string; days: number } to Content and tsc stops at minutes:
// error TS2366: Function lacks ending return statement and return type does not include 'undefined'.
