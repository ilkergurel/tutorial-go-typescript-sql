// LEARN(design.paradigms.errors): throw for what nobody nearby can handle; return a Result union for expected failures.
class InvalidError extends Error {
  override name = "InvalidError";
}
class EmailTakenError extends Error {
  override name = "EmailTakenError";
}

// Exceptions: the signature says void, and tsc knows nothing about the failures.
function registerOrThrow(emails: Set<string>, email: string): void {
  if (!email.includes("@")) throw new InvalidError("email: must contain @");
  if (emails.has(email)) throw new EmailTakenError(`register ${email}: email already registered`);
  emails.add(email);
}

function statusOf(emails: Set<string>, email: string): number {
  try {
    registerOrThrow(emails, email);
    return 201;
  } catch (e) {
    if (e instanceof EmailTakenError) return 409;
    if (e instanceof InvalidError) return 422;
    throw e; // not a failure we expected: let it reach the top-level handler
  }
}

// Result: the expected failures are part of the return type.
type Problem = "invalid" | "taken";
type Result = { ok: true } | { ok: false; problem: Problem };

function register(emails: Set<string>, email: string): Result {
  if (!email.includes("@")) return { ok: false, problem: "invalid" };
  if (emails.has(email)) return { ok: false, problem: "taken" };
  emails.add(email);
  return { ok: true };
}

// A new Problem without a status here fails to compile.
const statusFor = { invalid: 422, taken: 409 } satisfies Record<Problem, number>;

const thrown = new Set(["ayse@example.com"]);
const returned = new Set(["ayse@example.com"]);
for (const email of ["mehmet@example.com", "ayse@example.com", "can.example.com"]) {
  const r = register(returned, email);
  const viaResult = r.ok ? 201 : statusFor[r.problem];
  console.log(`${email.padEnd(19)} throw: ${statusOf(thrown, email)}  result: ${viaResult}`);
}
