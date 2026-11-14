// LEARN(inside.codebase.testing-pyramid): judge.ts is the cheapest next frontend test.
import { describe, expect, it } from "vitest";

// A miniature of src/features/runner/judge.ts: same rules, same order, regex left out.
type Check = "exact" | "contains" | "sorted-lines" | "none";
interface RunResult { stdout: string; exitCode: number }

function normalize(s: string, check: Check): string {
  const text = s.replace(/\r\n/g, "\n").trimEnd();
  return check === "sorted-lines" ? text.split("\n").sort().join("\n") : text;
}

function judge(r: RunResult, check: Check, expected: string | null): { pass: boolean } | null {
  if (check === "none" || expected === null) return null;
  if (r.exitCode !== 0) return { pass: false };
  const actual = normalize(r.stdout, check);
  const want = normalize(expected, check);
  if (check === "contains") return { pass: want.split("\n").every((l) => actual.includes(l.trim())) };
  return { pass: actual === want };
}

describe("judge", () => {
  it.each([
    ["exact ignores CRLF and whitespace at the very end", "a\r\nb  \r\n", "exact", "a\nb\n", true],
    ["exact still sees spaces ending an inner line", "a  \nb\n", "exact", "a\nb\n", false],
    ["exact sees a changed line", "a\nc\n", "exact", "a\nb\n", false],
    ["sorted-lines ignores order", "b\na\n", "sorted-lines", "a\nb\n", true],
    ["contains matches trimmed lines", "--- PASS: TestX (0.00s)\nok\n", "contains", "  --- PASS: TestX", true],
  ] as const)("%s", (_name, stdout, check, expected, pass) => {
    expect(judge({ stdout, exitCode: 0 }, check, expected)).toEqual({ pass });
  });

  it("fails any check when the program failed, whatever it printed", () => {
    expect(judge({ stdout: "a\n", exitCode: 1 }, "exact", "a\n")).toEqual({ pass: false });
  });

  it("has no verdict without an expected file or with check none", () => {
    expect(judge({ stdout: "x", exitCode: 0 }, "exact", null)).toBeNull();
    expect(judge({ stdout: "x", exitCode: 0 }, "none", "x")).toBeNull();
  });
});
