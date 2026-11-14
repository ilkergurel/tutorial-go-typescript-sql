// LEARN(design.paradigms.side-by-side): TS types describe shapes; the checker enforces them, the runtime never sees them.
type Status = "passed" | "failed";

interface Run {
  readonly req: string;
  readonly test: string;
  readonly status: Status;
}

interface Tally {
  req: string;
  passed: number;
  failed: number;
}

function report(runs: readonly Run[]): Tally[] {
  const count = (rs: Run[], s: Status) => rs.filter((r) => r.status === s).length;
  return [...Map.groupBy(runs, (r) => r.req)]
    .map(([req, rs]) => ({ req, passed: count(rs, "passed"), failed: count(rs, "failed") }))
    .sort((a, b) => b.failed - a.failed || (a.req < b.req ? -1 : 1));
}

const run = (req: string, test: string, status: Status): Run => ({ req, test, status });
const tallies = report([
  run("REQ-LOGIN", "TC-1", "passed"),
  run("REQ-LOGIN", "TC-2", "failed"),
  run("REQ-LOGIN", "TC-3", "passed"),
  run("REQ-EXPORT", "TC-4", "passed"),
  run("REQ-SEARCH", "TC-5", "failed"),
  run("REQ-SEARCH", "TC-6", "failed"),
]);
for (const t of tallies) {
  console.log(`${t.req.padEnd(10)} passed=${t.passed} failed=${t.failed}`);
}

// report([run("REQ-LOGIN", "TC-7", "pased")]);
// error TS2345: Argument of type '"pased"' is not assignable to parameter of type 'Status'.

// Any object with the right shape fits, extra fields included. The cast is a
// promise the checker believes; nothing checks this data at run time.
const fromApi = JSON.parse('[{"req":"REQ-LOGIN","test":"TC-7","status":"pased","ms":12}]') as Run[];
console.log(report(fromApi));
