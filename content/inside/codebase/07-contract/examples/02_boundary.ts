// LEARN(inside.codebase.contract): types.ts is a promise about the wire; only a run-time check at the boundary keeps it.
import { z } from "zod";

// Abridged from src/features/runner/types.ts, the hand-written mirror.
interface Diagnostic { file: string; line: number; col: number; message: string }
interface WrittenFile { path: string; size: number; preview: string }
interface RunResult {
  stdout: string;
  exitCode: number;
  diagnostics: Diagnostic[];
  files: WrittenFile[];
}

// A real body from POST /api/run before the fix: a Go hello world in local mode.
const wire =
  '{"stdout":"hi\\n","stderr":"","exitCode":0,"durationMs":776,"timedOut":false,' +
  '"truncated":false,"diagnostics":null,"files":null}';

// 1. What ServerRunner does: a cast. It compiles, and checks nothing.
const cast = JSON.parse(wire) as RunResult;
try {
  console.log(`cast:   ${cast.files.length} files`);
} catch (e) {
  console.log(`cast:   files.length threw ${(e as Error).name}`);
}

// 2. The hand-written guard runapi.go's comment once promised (never written).
function isRunResult(v: unknown): v is RunResult {
  if (typeof v !== "object" || v === null) return false;
  const r = v as Record<string, unknown>;
  return typeof r.stdout === "string" && typeof r.exitCode === "number" &&
    Array.isArray(r.diagnostics) && Array.isArray(r.files);
}
console.log(`guard:  ${isRunResult(JSON.parse(wire))}`);

// 3. A schema annotated with the mirror: tsc fails if the two disagree.
const Diag = z.object({ file: z.string(), line: z.number(), col: z.number(), message: z.string() });
const Written = z.object({ path: z.string(), size: z.number(), preview: z.string() });
const Strict: z.ZodType<RunResult> = z.object({
  stdout: z.string(),
  exitCode: z.number().int(),
  diagnostics: z.array(Diag),
  files: z.array(Written),
});
const checked = Strict.safeParse(JSON.parse(wire));
if (!checked.success) console.log(z.prettifyError(checked.error));

// 4. Tolerant at the edge: accept the null Go sends, hand the app [].
const Lenient: z.ZodType<RunResult, unknown> = z.object({
  stdout: z.string(),
  exitCode: z.number().int(),
  diagnostics: z.array(Diag).nullable().transform((v) => v ?? []),
  files: z.array(Written).nullable().transform((v) => v ?? []),
});
const r = Lenient.parse(JSON.parse(wire));
console.log(`lenient: ${r.files.length} files, exit ${r.exitCode}, stdout ${JSON.stringify(r.stdout)}`);
