// LEARN(ts.narrowing.discriminated-unions): tag with literals; switch exhaustively; never as guard.

interface OkResult {
  status: "ok";
  rows: string[];
  durationMs: number;
}

interface FailedResult {
  status: "failed";
  error: string;
}

type RunResult = OkResult | FailedResult;

function render(r: RunResult): string {
  // Narrowing works on the tag alone — no type assertions anywhere.
  switch (r.status) {
    case "ok":
      // Here tsc knows: rows exists, error doesn't.
      return `${r.rows.length} rows in ${r.durationMs}ms`;
    case "failed":
      // Here tsc knows: error exists, rows doesn't.
      return `error: ${r.error}`;
    default: {
      // Exhaustiveness guard: if a variant is added without a case here,
      // this assignment stops compiling, because nothing assigns to never.
      const exhaustive: never = r;
      return exhaustive;
    }
  }
}

const ok: RunResult = { status: "ok", rows: ["a", "b"], durationMs: 42 };
const failed: RunResult = { status: "failed", error: "timeout" };

console.log(render(ok));
console.log(render(failed));
