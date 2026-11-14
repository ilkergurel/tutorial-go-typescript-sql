// LEARN(ts.type-level.branded): model only the states that can exist, so impossible ones don't type-check.

// Loose: optional fields allow every combination, including nonsense.
interface LooseRequirement {
  id: string;
  status: "draft" | "approved" | "rejected";
  approvedBy?: string;
  reason?: string;
}
function describeLoose(r: LooseRequirement): string {
  return r.approvedBy ? `${r.id}: ${r.approvedBy} onayladı` : `${r.id}: ${r.status}`;
}
const nonsense: LooseRequirement = { id: "REQ-LOGIN", status: "draft", approvedBy: "Işıl" };
console.log(describeLoose(nonsense)); // a draft that reads as approved

// Tight: each state carries exactly its own data. Approval needs evidence,
// and a tuple type says "at least one test", which string[] can't.
type NonEmpty<T> = readonly [T, ...T[]];
type Requirement =
  | { status: "draft"; id: string }
  | { status: "approved"; id: string; approvedBy: string; evidence: NonEmpty<string> }
  | { status: "rejected"; id: string; reason: string };
type Draft = Extract<Requirement, { status: "draft" }>;
type Approved = Extract<Requirement, { status: "approved" }>;
// const bad: Requirement = { id: "REQ-LOGIN", status: "draft", approvedBy: "Işıl" };
// error TS2353: Object literal may only specify known properties, and 'approvedBy' does not exist in type '{ status: "draft"; id: string; }'.

function approve(r: Draft, by: string, evidence: NonEmpty<string>): Approved {
  return { status: "approved", id: r.id, approvedBy: by, evidence };
}
// Data from outside is a plain array; a guard turns it into the stronger type.
function isNonEmpty<T>(xs: readonly T[]): xs is NonEmpty<T> {
  return xs.length > 0;
}

function describe(r: Requirement): string {
  switch (r.status) {
    case "draft":
      return `${r.id}: taslak`;
    case "approved":
      return `${r.id}: ${r.approvedBy} onayladı, kanıt ${r.evidence.join(", ")}`;
    case "rejected":
      return `${r.id}: reddedildi (${r.reason})`;
  }
}

const draft: Draft = { status: "draft", id: "REQ-RESET" };
// approve(draft, "Işıl", []);
// error TS2345: Argument of type '[]' is not assignable to parameter of type 'NonEmpty<string>'.
for (const linked of [[], ["TC-11", "TC-12"]]) {
  console.log(isNonEmpty(linked) ? describe(approve(draft, "Işıl", linked)) : describe(draft));
}
