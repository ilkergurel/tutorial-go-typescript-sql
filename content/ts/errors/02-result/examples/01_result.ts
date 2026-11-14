// LEARN(ts.errors.result): a Result puts failure in the return type, so the value can't be used unchecked.

type Result<T, E> = { ok: true; value: T } | { ok: false; error: E };
const ok = <T>(value: T): Result<T, never> => ({ ok: true, value });
const err = <E>(error: E): Result<never, E> => ({ ok: false, error });

interface Link {
  requirementId: string;
  testId: string;
}
// Expected failures are data: a union the caller can switch over.
type LinkProblem =
  | { kind: "columns"; found: number }
  | { kind: "requirement"; raw: string }
  | { kind: "test"; raw: string };

function parseLink(line: string): Result<Link, LinkProblem> {
  const cols = line.split(";").map((c) => c.trim());
  if (cols.length !== 2) return err({ kind: "columns", found: cols.length });
  const [req = "", test = ""] = cols;
  if (!/^REQ-[A-Z0-9-]+$/.test(req)) return err({ kind: "requirement", raw: req });
  if (!/^TC-[0-9]+$/.test(test)) return err({ kind: "test", raw: test });
  return ok({ requirementId: req, testId: test });
}

// One place turns problems into messages; a new kind fails to compile here.
function message(p: LinkProblem): string {
  switch (p.kind) {
    case "columns":
      return `2 sütun bekleniyordu, ${p.found} bulundu`;
    case "requirement":
      return `geçersiz gereksinim "${p.raw}"`;
    case "test":
      return `geçersiz test "${p.raw}"`;
  }
}

for (const line of ["REQ-LOGIN;TC-10", "REQ-LOGIN", "giriş;TC-11", "REQ-RESET;TC-x"]) {
  const r = parseLink(line);
  // console.log(r.value.testId);
  // error TS2339: Property 'value' does not exist on type 'Result<Link, LinkProblem>'.
  //   Property 'value' does not exist on type '{ ok: false; error: LinkProblem; }'.
  if (r.ok) console.log(`ok:   ${r.value.requirementId} <- ${r.value.testId}`);
  else console.log(`hata: ${message(r.error)}`);
}
