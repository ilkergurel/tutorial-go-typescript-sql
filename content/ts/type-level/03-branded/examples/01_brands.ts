// LEARN(ts.type-level.branded): a brand makes structurally identical types incompatible, and only a parser hands one out.

// Plain aliases are only names: both parameters are string, so swapping them compiles.
function linkLoose(requirement: string, test: string): string {
  return `${requirement} -> ${test}`;
}
console.log(linkLoose("TC-10", "REQ-LOGIN")); // backwards, no error

// A brand adds a property that exists only in the type. A unique symbol key can't
// be written by accident, and like every type it is erased.
declare const brand: unique symbol;
type Brand<T, B extends string> = T & { readonly [brand]: B };
type RequirementId = Brand<string, "RequirementId">;
type TestCaseId = Brand<string, "TestCaseId">;

// The only places that say "as RequirementId": the check and the cast together.
function parseRequirementId(raw: string): RequirementId | undefined {
  const s = raw.trim().toUpperCase();
  return /^REQ-[A-Z0-9]+(-[A-Z0-9]+)*$/.test(s) ? (s as RequirementId) : undefined;
}
function parseTestCaseId(raw: string): TestCaseId | undefined {
  const s = raw.trim().toUpperCase();
  return /^TC-[0-9]+$/.test(s) ? (s as TestCaseId) : undefined;
}

function link(requirement: RequirementId, test: TestCaseId): string {
  return `${requirement} -> ${test}`;
}

const req = parseRequirementId(" req-login ");
const tc = parseTestCaseId("tc-10");
if (req && tc) {
  console.log(link(req, tc));
  // link(tc, req);
  // error TS2345: Argument of type 'TestCaseId' is not assignable to parameter of type 'RequirementId'.
  // link("REQ-LOGIN", tc);
  // error TS2345: Argument of type 'string' is not assignable to parameter of type 'RequirementId'.
  console.log(typeof req, JSON.stringify({ req }), req.toLowerCase()); // plain strings at run time
}

// IDs are machine text: upper-case them without a locale. Turkish rules turn i into İ.
for (const raw of ["req-login", "req-login".toLocaleUpperCase("tr")]) {
  console.log(`${raw} -> ${parseRequirementId(raw) ?? "rejected"}`);
}
