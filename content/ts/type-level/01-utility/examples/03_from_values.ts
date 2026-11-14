// LEARN(ts.type-level.utility): derive types from values and functions that already exist.

// One list is the source; the union type is read off it.
const STATUSES = ["draft", "review", "approved"] as const;
type Status = (typeof STATUSES)[number]; // "draft" | "review" | "approved"

// Record<Status, string>: exactly one label per status, none missing, none extra.
const labels: Record<Status, string> = {
  draft: "Taslak",
  review: "İncelemede",
  approved: "Onaylandı",
};

// Stands in for a generated API client that exports no types for its results.
async function fetchRequirement(id: string) {
  return {
    id,
    title: "Şifre sıfırlama",
    status: "review" as Status,
    links: [
      { testId: "TC-10", passed: true },
      { testId: "TC-11", passed: false },
    ],
  };
}

type Loaded = Awaited<ReturnType<typeof fetchRequirement>>;
type TraceLink = Loaded["links"][number]; // { testId: string; passed: boolean }
type FetchArgs = Parameters<typeof fetchRequirement>; // [id: string]

function failing(links: readonly TraceLink[]): string[] {
  return links.filter((l) => !l.passed).map((l) => l.testId);
}

const args: FetchArgs = ["REQ-RESET"];
const req = await fetchRequirement(...args);
console.log(`${req.id}: ${labels[req.status]}, failing: ${failing(req.links).join(", ")}`);

// Add "rejected" to STATUSES and labels and counts both stop compiling until the
// new status is handled:
// error TS2741: Property 'rejected' is missing in type '{ draft: number; review: number; approved: number; }' but required in type 'Record<"approved" | "draft" | "rejected" | "review", number>'.
const counts: Record<Status, number> = { draft: 0, review: 0, approved: 0 };
for (const s of ["review", "draft", "review"] as const) counts[s] += 1;
console.log(STATUSES.map((s) => `${labels[s]}=${counts[s]}`).join(" "));
