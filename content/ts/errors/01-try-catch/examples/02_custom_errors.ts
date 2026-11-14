// LEARN(ts.errors.try-catch): subclass Error for failures callers branch on, and chain causes instead of losing them.

class NotFoundError extends Error {
  override name = "NotFoundError";
  readonly id: string;
  constructor(id: string) {
    super(`${id} bulunamadı`);
    this.id = id;
  }
}
class ImportError extends Error {
  override name = "ImportError";
  readonly line: number;
  constructor(line: number, options: ErrorOptions) {
    super(`satır ${line} içe aktarılamadı`, options);
    this.line = line;
  }
}

const known = new Set(["REQ-LOGIN", "REQ-RESET"]);
function findRequirement(id: string): string {
  if (!known.has(id)) throw new NotFoundError(id);
  return id;
}
// The import layer adds what it knows (the line) and keeps the original as cause.
function importLine(line: number, text: string): string {
  try {
    const [reqId = "", testId = ""] = text.split(";");
    return `${findRequirement(reqId)} <- ${testId}`;
  } catch (e) {
    throw new ImportError(line, { cause: e });
  }
}

// Go's errors.As by hand: walk the cause chain looking for one class.
type Class<T> = new (...args: never[]) => T;
function findCause<T extends Error>(err: unknown, type: Class<T>): T | undefined {
  for (let e = err; e instanceof Error; e = e.cause) {
    if (e instanceof type) return e;
  }
  return undefined;
}

for (const [i, text] of ["REQ-LOGIN;TC-10", "REQ-EXPORT;TC-12"].entries()) {
  try {
    console.log(importLine(i + 1, text));
  } catch (e) {
    const missing = findCause(e, NotFoundError);
    if (!(e instanceof ImportError) || missing === undefined) throw e; // not ours: rethrow
    console.log(`${missing.id} yok, satır ${e.line} atlandı`);
    console.log(`${e}\n  cause: ${String(e.cause)}`);
  }
}
