// LEARN(ts.type-level.conditionals): template literal types build string types, and infer inside them takes strings apart.

// Building: each union in a template multiplies out.
type Entity = "requirement" | "test";
type Change = "created" | "deleted";
type Topic = `${Entity}.${Change}`; // 4 members
const topics: Topic[] = ["requirement.created", "test.deleted"];
// topics.push("requirement.approved");
// error TS2345: Argument of type '"requirement.approved"' is not assignable to parameter of type '"requirement.created" | "requirement.deleted" | "test.created" | "test.deleted"'.

// Taking apart: infer inside a template, recursing on the rest of the string.
type ParamNames<P extends string> = P extends `${string}:${infer Name}/${infer Rest}`
  ? Name | ParamNames<Rest>
  : P extends `${string}:${infer Name}`
    ? Name
    : never;
type Params<P extends string> = { [K in ParamNames<P>]: string };

function match<P extends string>(pattern: P, path: string): Params<P> | undefined {
  const want = pattern.split("/");
  const got = path.split("/");
  if (want.length !== got.length) return undefined;
  const params: Record<string, string> = {};
  for (const [i, part] of want.entries()) {
    if (part.startsWith(":")) params[part.slice(1)] = decodeURIComponent(got[i]);
    else if (part !== got[i]) return undefined;
  }
  return params as Params<P>; // the runtime loop and the type must agree; nothing checks it
}

const route = "/requirements/:reqId/tests/:testId";
const m = match(route, "/requirements/REQ-LOGIN/tests/TC-10");
console.log(m ? `${m.reqId} ${m.testId}` : "no match");
// console.log(m?.reqID);
// error TS2551: Property 'reqID' does not exist on type 'Params<"/requirements/:reqId/tests/:testId">'. Did you mean 'reqId'?
console.log(match(route, "/requirements/REQ-LOGIN") ?? "no match");
console.log(topics.join(" "));

// Uppercase<S> applies JavaScript's toUpperCase(), which uses no locale.
const city = "izmir";
const shouted: Uppercase<typeof city> = "IZMIR";
// const turkish: Uppercase<typeof city> = "İZMİR";
// error TS2322: Type '"İZMİR"' is not assignable to type '"IZMIR"'.
console.log(shouted, city.toLocaleUpperCase("tr"));
