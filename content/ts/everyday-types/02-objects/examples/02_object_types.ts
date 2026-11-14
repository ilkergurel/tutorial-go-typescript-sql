// LEARN(ts.everyday-types.objects): readonly is shallow and compile-time only; Object.freeze is the runtime version.

interface Lesson {
  readonly id: string;
  title: string;
  minutes?: number; // may be absent: reads give number | undefined
  readonly tags: string[];
}

const lesson: Lesson = { id: "ts.objects", title: "Objects", tags: ["ts"] };
console.log(lesson.minutes ?? "no estimate");

// readonly blocks assignment through this type:
// lesson.id = "x";
// error TS2540: Cannot assign to 'id' because it is a read-only property.

// It is shallow: the array behind a readonly property is still mutable.
lesson.tags.push("objects");
console.log(lesson.tags.join(","));

// And it is not checked in assignments: a writable view of the same object
// compiles, and writes through it.
const writable: { id: string } = lesson;
writable.id = "changed";
console.log(lesson.id);

// Object.freeze enforces at run time. Modules run in strict mode, so a write
// to a frozen property throws instead of being ignored.
const frozen = Object.freeze({ id: "a1", tags: ["x"] });
const sneaky: { id: string } = frozen;
try {
  sneaky.id = "b2";
} catch (e) {
  console.log(e instanceof TypeError ? "TypeError: object is frozen" : e);
}
frozen.tags.push("y"); // freeze is shallow too
console.log(`${frozen.id} ${frozen.tags}`);
