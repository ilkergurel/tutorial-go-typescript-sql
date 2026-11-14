// LEARN(ts.tooling.lint): values tsc accepts that print nonsense, and the rules that catch them.

type Status = "draft" | "published" | "archived";
interface Lesson {
  title: string;
  status: Status;
  retries?: number; // 0 means "don't retry"
}
const lesson: Lesson = { title: "Lint", status: "archived", retries: 0 };

// restrict-template-expressions, no-base-to-string: an object in a template.
console.log(`opened ${lesson}`);

// prefer-nullish-coalescing: || replaces every falsy value, 0 included.
console.log("retries:", lesson.retries || 3, "vs", lesson.retries ?? 3);

// switch-exhaustiveness-check: "archived" joined Status after this was written.
function badge(status: Status): string {
  let label = "";
  switch (status) {
    case "draft":
      label = "Taslak";
      break;
    case "published":
      label = "Yayında";
      break;
  }
  return label;
}
console.log(`badge: "${badge(lesson.status)}"`);
