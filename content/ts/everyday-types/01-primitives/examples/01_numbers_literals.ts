// LEARN(ts.everyday-types.primitives): one number type, a separate bigint, and literal types.

// number is an IEEE-754 double for integers and fractions alike.
console.log(`${0.1 + 0.2} ${Number.MAX_SAFE_INTEGER}`);

// A Postgres bigint id above 2^53 - 1 silently changes in JSON.parse.
const body = '{"id": 9007199254740993}';
const parsed: { id: number } = JSON.parse(body);
console.log(`${parsed.id} ${Number.isSafeInteger(parsed.id)}`);

// bigint is exact but a separate type; the two never mix implicitly.
const id = 9007199254740993n;
// const bumped = id + 1;
// error TS2365: Operator '+' cannot be applied to types '9007199254740993n' and '1'.
console.log(`${id + 1n} ${typeof id}`);

// Literal types: a const keeps the exact value as its type, a let widens.
type Method = "GET" | "POST";
function request(method: Method, path: string): string {
  return `${method} ${path}`;
}
const get = "GET"; // type "GET"
let verb = "GET"; // type string: it could be reassigned to anything
console.log(request(get, "/api/lessons"));
// console.log(request(verb, "/api/lessons"));
// error TS2345: Argument of type 'string' is not assignable to parameter of type 'Method'.

// Annotate the variable with the union when it must stay assignable.
let next: Method = "GET";
next = "POST";
console.log(request(next, "/api/runs"), verb);
