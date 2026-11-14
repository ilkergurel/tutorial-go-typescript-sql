// LEARN(ts.type-level.conditionals): a conditional type decides per union member, and infer names the part it matched.

type TraceEvent =
  | { type: "requirement.created"; id: string; title: string }
  | { type: "requirement.approved"; id: string; by: string }
  | { type: "test.failed"; testId: string; reason: string };

// Extract<T, U> is T extends U ? T : never, checked for each member of T on its own.
type EventOf<K extends TraceEvent["type"]> = Extract<TraceEvent, { type: K }>;
type RequirementEvent = Extract<TraceEvent, { type: `requirement.${string}` }>;

function approval(e: EventOf<"requirement.approved">): string {
  return `${e.id} onaylandı (${e.by})`;
}
// A runtime guard that mirrors the type: same prefix, checked on the value.
function isRequirementEvent(e: TraceEvent): e is RequirementEvent {
  return e.type.startsWith("requirement.");
}

// infer names a part of the matched type; recursion unwraps nested promises.
type Unwrapped<T> = T extends PromiseLike<infer V> ? Unwrapped<V> : T;
async function loadEvents() {
  const events: TraceEvent[] = [
    { type: "requirement.created", id: "REQ-LOGIN", title: "Giriş" },
    { type: "test.failed", testId: "TC-10", reason: "zaman aşımı" },
    { type: "requirement.approved", id: "REQ-LOGIN", by: "Işıl" },
    { type: "requirement.created", id: "REQ-RESET", title: "Şifre sıfırlama" },
  ];
  return events;
}
const events: Unwrapped<ReturnType<typeof loadEvents>> = await loadEvents(); // TraceEvent[]

for (const e of events) {
  if (e.type === "requirement.approved") console.log(approval(e));
  // approval(e);
  // error TS2345: Argument of type 'TraceEvent' is not assignable to parameter of type '{ type: "requirement.approved"; id: string; by: string; }'.
  //   Property 'by' is missing in type '{ type: "requirement.created"; id: string; title: string; }' but required in type '{ type: "requirement.approved"; id: string; by: string; }'.
}
const touched = events.filter(isRequirementEvent).map((e) => e.id); // e: RequirementEvent
console.log(`${touched.length} of ${events.length} events: ${[...new Set(touched)].join(", ")}`);

// Distribution happens only on a bare type parameter; brackets switch it off.
type IsText<T> = T extends string ? "text" : "other";
type PerMember = IsText<"REQ-1" | 1>; // "text" | "other"
type Whole = ["REQ-1" | 1] extends [string] ? "text" : "other"; // "other"
type ElementOf<T> = T extends readonly (infer E)[] ? E : never;
type Ids = ElementOf<["REQ-LOGIN", "REQ-RESET"]>; // "REQ-LOGIN" | "REQ-RESET"
