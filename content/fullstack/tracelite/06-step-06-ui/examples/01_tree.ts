// LEARN(fullstack.tracelite.step-06-ui): the API sends the tree flat and depth first; the client nests it once and rolls coverage up per subtree.
type Coverage = "failing" | "uncovered" | "incomplete" | "passing";
type Row = { id: string; parentId: string | null; code: string; depth: number };

// --- src/features/requirements/tree.ts: nest once, tally once.
type Node = Row & { children: Node[] };
function nest(rows: readonly Row[]): Node[] {
  const byId = new Map<string, Node>();
  const roots: Node[] = [];
  for (const row of rows) {
    const node: Node = { ...row, children: [] };
    byId.set(node.id, node);
    const parent = row.parentId === null ? undefined : byId.get(row.parentId);
    (parent?.children ?? roots).push(node); // keeps the server's sibling order
  }
  return roots;
}

const worstFirst = ["failing", "uncovered", "incomplete", "passing"] as const;
type Tally = Record<Coverage, number>;
function tallies(roots: readonly Node[], own: ReadonlyMap<string, Coverage>) {
  const out = new Map<string, Tally>();
  const walk = (node: Node): Tally => {
    const t: Tally = { failing: 0, uncovered: 0, incomplete: 0, passing: 0 };
    const mine = own.get(node.id) ?? "uncovered";
    // A heading with no tests of its own is covered through its children.
    if (node.children.length === 0 || mine !== "uncovered") t[mine]++;
    for (const child of node.children) {
      const c = walk(child);
      for (const k of worstFirst) t[k] += c[k];
    }
    out.set(node.id, t);
    return t;
  };
  roots.forEach(walk);
  return out;
}

// --- GET /requirement-tree (abridged; ids are uuids) and GET /coverage.
const rows: Row[] = [
  { id: "r1", parentId: null, code: "AUTH", depth: 0 },
  { id: "r2", parentId: "r1", code: "AUTH-LOGIN", depth: 1 },
  { id: "r3", parentId: "r2", code: "AUTH-LOGIN-2FA", depth: 2 },
  { id: "r4", parentId: "r2", code: "AUTH-LOGIN-LOCKOUT", depth: 2 },
  { id: "r5", parentId: "r1", code: "AUTH-RESET", depth: 1 },
  { id: "r6", parentId: null, code: "EXPORT", depth: 0 },
  { id: "r7", parentId: "r6", code: "EXPORT-CSV", depth: 1 },
];
const own = new Map<string, Coverage>([
  ["r3", "passing"],
  ["r4", "incomplete"],
  ["r5", "failing"],
]); // every other requirement: uncovered

// Print the flat list as received, each requirement with its roll-up.
function show(title: string) {
  const sums = tallies(nest(rows), own);
  console.log(title);
  for (const row of rows) {
    const sum = sums.get(row.id)!;
    const counts = worstFirst.filter((k) => sum[k] > 0);
    const detail = counts.map((k) => `${sum[k]} ${k}`).join(", ");
    const indent = "  ".repeat(row.depth + 1);
    console.log(`${indent}${row.code}: ${counts[0]} (${detail})`);
  }
}

show("as loaded");
// A refetch brings AUTH-LOGIN-SSO, with no tests yet, in depth-first order.
const sso = { id: "r8", parentId: "r2", code: "AUTH-LOGIN-SSO", depth: 2 };
rows.splice(4, 0, sso);
show("after AUTH-LOGIN-SSO is added");
