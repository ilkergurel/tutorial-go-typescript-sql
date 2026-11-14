/**
 * Frontend mirror of server/internal/runapi (the wire format). One typed shape
 * for every runtime; the browser TS runner produces the same Result locally.
 */

export type Runtime =
  | 'go'
  | 'go-test'
  | 'go-server'
  | 'sql'
  | 'sql-nontx'
  | 'sql-multisession'
  | 'ts-browser'
  | 'ts-node'
  | 'ts-test';

export interface Diagnostic {
  file: string;
  line: number;
  col: number;
  message: string;
}

export interface WrittenFile {
  path: string;
  size: number;
  preview: string;
}

/** The server's error fields for one failed statement. */
export interface SqlError {
  message: string;
  code: string;
  detail?: string;
  hint?: string;
  /** 1-based character position the server blamed; 0 when it didn't say. */
  pos: number;
}

/**
 * One statement's outcome, in script order. Cells are Postgres' text form;
 * null is SQL NULL, which must never render like an empty string.
 */
export interface StatementResult {
  /** Command tag, e.g. "INSERT 0 3". */
  command: string;
  columns: { name: string; type: string }[];
  rows: (string | null)[][];
  rowsAffected: number;
  /** e.g. "NOTICE: ..." from RAISE NOTICE. */
  notices: string[];
  error?: SqlError;
  durationMs: number;
}

/** One tagged statement from a multisession (A:/B:) script. */
export interface StepResult extends StatementResult {
  /** 1-based position in the script. */
  step: number;
  session: 'A' | 'B';
  sql: string;
  /** Still running when the next step was sent (usually blocked on a lock). */
  waited: boolean;
  /** When its result arrived, counting every step's result from 1. */
  finishedOrder: number;
}

export interface RunResult {
  stdout: string;
  stderr: string;
  exitCode: number;
  durationMs: number;
  timedOut: boolean;
  truncated: boolean;
  diagnostics: Diagnostic[];
  files: WrittenFile[];
  /**
   * SQL runtimes: statements in script order (for sql-multisession, the setup
   * statements). stdout holds the same results as text, which is what the
   * expected output compares against.
   */
  sql?: {
    schema: string;
    statements: StatementResult[];
  };
  /** sql-multisession: every step in script order, tagged with its session. */
  steps?: StepResult[];
}

export interface RunRequest {
  runtime: Runtime;
  example: string;
  files: Record<string, string>;
  fixtures?: Record<string, string>;
  stdin?: string;
  timeoutS?: number;
  vet?: boolean;
  setup?: string;
  lessonId?: string;
}

/** How a run's stdout is compared with the expected output (lesson.schema.json's check). */
export type Check = 'exact' | 'contains' | 'regex' | 'sorted-lines' | 'none';

/**
 * Everything RunPanel needs to show, edit, run and judge one example, in the
 * runner's own terms. The app builds it from a lesson's example
 * (app/routes/LessonPage.tsx), so this feature doesn't depend on the lesson
 * content types.
 */
export interface RunnableExample {
  lessonId: string;
  exampleId: string;
  runtime: Runtime;
  /** The file name the runner gets, e.g. "main.go" or "01_props.tsx". */
  fileName: string;
  source: string;
  // "| undefined" because the lesson page copies optional frontmatter fields
  // as they are; with exactOptionalPropertyTypes that must be allowed.
  stdin?: string | undefined;
  /** SQL the runtime executes before the example (sql runtimes only). */
  setup?: string | undefined;
  fixtures?: Record<string, string> | undefined;
  /** null when the example has no expected file. */
  expected: string | null;
  check: Check;
  timeoutS?: number | undefined;
}

/** Discriminated union for UI state machine: idle -> running -> done/error. */
export type RunState =
  | { kind: 'idle' }
  | { kind: 'running' }
  | { kind: 'done'; result: RunResult }
  | { kind: 'failed'; message: string };
