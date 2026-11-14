// LEARN(ts.type-level.branded): state tracked in a type parameter pays off only where the state is known at compile time.

type Status = "draft" | "review" | "approved";
interface Req<S extends Status> {
  id: string;
  status: S;
}
function submit(r: Req<"draft">): Req<"review"> {
  return { ...r, status: "review" };
}
function approve(r: Req<"review">): Req<"approved"> {
  return { ...r, status: "approved" };
}

// Where the code creates the value, every transition is checked.
const created: Req<"draft"> = { id: "REQ-EXPORT", status: "draft" };
console.log(approve(submit(created)));
// approve(created);
// error TS2345: Argument of type 'Req<"draft">' is not assignable to parameter of type 'Req<"review">'.

// A row loaded from the database has a status known only at run time.
function loadRow(id: string): Req<Status> {
  return { id, status: "review" }; // stands in for a query
}
const row = loadRow("REQ-LOGIN");
// if (row.status === "review") approve(row);
// error TS2345: Argument of type 'Req<Status>' is not assignable to parameter of type 'Req<"review">'.
//   Type 'Status' is not assignable to type '"review"'.

// Checking row.status narrows the property, not the object, so it takes a guard...
function inStatus<S extends Status>(r: Req<Status>, s: S): r is Req<S> {
  return r.status === s;
}
if (inStatus(row, "review")) console.log(`${approve(row).id} onaylandı (guard)`);

// ...or the union of the states, where the plain check narrows on its own.
type AnyReq = Req<"draft"> | Req<"review"> | Req<"approved">;
const same: AnyReq = row; // no cast: each possible status maps to one member
if (same.status === "review") console.log(`${approve(same).id} onaylandı (union)`);
