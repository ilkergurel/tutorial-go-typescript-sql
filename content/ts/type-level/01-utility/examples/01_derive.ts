// LEARN(ts.type-level.utility): derive related types from one source type instead of restating them.

interface Requirement {
  readonly id: string;
  title: string;
  status: "draft" | "review" | "approved";
  priority: 1 | 2 | 3;
  owner?: string;
  createdAt: Date;
}

// What a client sends to create one: everything the server doesn't assign.
type RequirementDraft = Omit<Requirement, "id" | "createdAt">;
// What an edit form may change, every field optional.
type RequirementPatch = Partial<Pick<Requirement, "title" | "status" | "priority" | "owner">>;

let seq = 0;
function create(draft: RequirementDraft): Readonly<Requirement> {
  seq += 1;
  return { ...draft, id: `REQ-${seq}`, createdAt: new Date(Date.UTC(2026, 10, 13)) };
}
function update(r: Requirement, patch: RequirementPatch): Requirement {
  return { ...r, ...patch };
}

const r1 = create({ title: "Giriş ekranı", status: "draft", priority: 2 });
const r2 = update(r1, { status: "review", owner: "Işıl" });
console.log(`${r2.id} ${r2.status} ${r2.owner}`);
// update(r1, { id: "REQ-9" });
// error TS2353: Object literal may only specify known properties, and 'id' does not exist in type 'Partial<Pick<Requirement, "owner" | "priority" | "status" | "title">>'.
// r1.title = "Çıkış";
// error TS2540: Cannot assign to 'title' because it is a read-only property.

// Pick checks its keys against keyof Requirement; Omit accepts any string.
// type Checked = Pick<Requirement, "createdOn">;
// error TS2344: Type '"createdOn"' does not satisfy the constraint 'keyof Requirement'.
type Oops = Omit<Requirement, "createdOn">; // no error, and nothing removed
const oops: Oops = r2;
console.log(oops.createdAt.toISOString().slice(0, 10)); // compiles: createdAt is still there

// Partial means "may be missing", and by default it also accepts an explicit undefined.
const r3 = update(r2, { title: undefined });
console.log(`title is ${typeof r3.title}, typed string`);
