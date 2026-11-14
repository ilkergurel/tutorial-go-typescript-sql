// LEARN(ts.errors.result): Results compose with early returns, collect for batch input, and meet exceptions at the boundary.

type Result<T, E> = { ok: true; value: T } | { ok: false; error: E };
const ok = <T>(value: T): Result<T, never> => ({ ok: true, value });
const err = <E>(error: E): Result<never, E> => ({ ok: false, error });

// The boundary with throwing code: convert the failure you expect, rethrow the rest.
function parseJson(text: string): Result<unknown, "invalid-json"> {
  try {
    return ok(JSON.parse(text));
  } catch (e) {
    if (e instanceof SyntaxError) return err("invalid-json");
    throw e; // anything else is a bug and stays an exception
  }
}

// Early return, the shape of Go's `if err != nil { return nil, err }`.
// The error type is the union of everything that can go wrong on the way.
function readIds(text: string): Result<string[], "invalid-json" | "not-an-array"> {
  const parsed = parseJson(text);
  if (!parsed.ok) return parsed;
  if (!Array.isArray(parsed.value)) return err("not-an-array");
  return ok(parsed.value.map(String));
}
const readMessages: Record<"invalid-json" | "not-an-array", string> = {
  "invalid-json": "bozuk JSON",
  "not-an-array": "dizi bekleniyordu",
};

// Batch input: check every item and report all problems at once, like errors.Join.
function checkIds(ids: readonly string[]): Result<string[], string[]> {
  const bad = ids.flatMap((id, i) => (/^REQ-[A-Z0-9-]+$/.test(id) ? [] : [`#${i + 1} "${id}"`]));
  return bad.length === 0 ? ok([...ids]) : err(bad);
}

const inputs = ['["REQ-LOGIN","REQ-RESET"]', '["REQ-LOGIN","giriş",""]', "{bozuk", '{"id":1}'];
for (const input of inputs) {
  const ids = readIds(input);
  if (!ids.ok) {
    console.log(`okunamadı: ${readMessages[ids.error]}`);
    continue;
  }
  const checked = checkIds(ids.value);
  if (checked.ok) console.log(`${checked.value.length} kimlik geçerli`);
  else console.log(`geçersiz: ${checked.error.join(", ")}`);
}
