// LEARN(ts.narrowing.basic): instanceof checks the prototype chain; in checks for a key.

class HttpError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.name = "HttpError";
    this.status = status;
  }
}

// catch hands you unknown; instanceof narrows it to a class. Subclass first:
// an HttpError is also an Error, so the order of the checks matters.
function describe(e: unknown): string {
  if (e instanceof HttpError) return `HTTP ${e.status}: ${e.message}`;
  if (e instanceof Error) return `${e.name}: ${e.message}`;
  return `non-Error thrown: ${String(e)}`;
}
console.log(describe(new HttpError(404, "kayıt yok")));
console.log(describe(new RangeError("sayfa < 1")));
console.log(describe("oops"));

type BankTransfer = { iban: string };
type Card = { last4: string };

// Plain object types vanish at runtime, so instanceof can't test them. For
// shapes without a tag field, in narrows on the presence of a key instead.
function payment(p: BankTransfer | Card): string {
  // if (p instanceof BankTransfer) return "transfer";
  //   error TS2693: 'BankTransfer' only refers to a type, but is being used as a value here.
  return "iban" in p ? `IBAN ${p.iban.slice(0, 4)}…` : `card •••• ${p.last4}`;
}
console.log(payment({ iban: "TR330006100519786457841326" }));
console.log(payment({ last4: "4242" }));

// unknown JSON: each check narrows a little further. Since TS 4.9, "id" in v
// adds id: unknown to the type, and typeof then narrows the property itself.
function readId(json: string): number | undefined {
  const v: unknown = JSON.parse(json);
  if (typeof v === "object" && v !== null && "id" in v) {
    return typeof v.id === "number" ? v.id : undefined;
  }
  return undefined;
}
console.log(readId('{"id": 7}'), readId('{"id": "7"}'), readId("null"));
