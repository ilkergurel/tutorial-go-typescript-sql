---
id: fullstack.tracelite.step-06-ui
title: "Step 6: The UI: requirement tree, detail form, trace matrix, coverage indicator"
track: fullstack
module: tracelite
order: 6
level: intermediate
minutes: 15
since: ts4.9
prerequisites: [fullstack.tracelite.step-05-client, ts.react.forms-lists, ts.react.query, ts.react.routing, ts.testing.vitest]
compare_with: [ts.architecture.state, ts.architecture.explicit-states, sql.subqueries-ctes.recursive, inside.codebase.testing-pyramid]
tags: [tracelite, react, recursion, tree, forms, optimistic-updates, accessibility, testing-library, coverage]
summary: >
  The API sends the requirement tree as a flat depth-first list; the client nests it once
  and renders it with a component that renders itself. A detail form PATCHes only what
  changed, and a trace matrix toggles links optimistically while coverage stays the
  server's.
examples:
  - id: tree
    file: examples/01_tree.ts
    title: "Flat list to nested tree in one pass, coverage rolled up per subtree"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "3"
        note: >
          The tree endpoint's items, abridged: each is a Requirement plus depth,
          ordered by step 2's SEARCH DEPTH FIRST BY code. Flat on the wire, so
          one JSON array and one zod parse, however deep the tree.
      - lines: "7-17"
        note: >
          Nesting in one pass: depth-first order puts every parent before its
          children, so the parent is always already in the map.
      - lines: "14"
        note: >
          Pushing keeps the server's sibling order, the byte order of COLLATE
          "C". Sorting again with localeCompare would make the tree disagree
          with the matrix and the flat list.
      - lines: "19"
        note: >
          The order a badge picks its color by. Ranking uncovered above
          incomplete is a choice: a requirement with no test is a gap in the
          plan, one with an unrun test is work in progress.
      - lines: "21-37"
        note: >
          A post-order walk: children are tallied before their parent, every
          node exactly once. Each node summing its own subtree on every render
          would cost depth times as much.
      - lines: "26-27"
        note: >
          A heading with no tests of its own adds nothing; a leaf without tests
          counts as uncovered. Step 8's report counts the same way.
      - lines: "49-53"
        note: >
          What GET /coverage says, abridged to the value: one item per
          requirement, computed by the server with step 2's rule. The client
          only counts and displays it.
      - lines: "69-72"
        note: >
          A refetch brings a new leaf without tests. AUTH-LOGIN's tally gains
          one uncovered requirement, which outranks incomplete, so its badge
          changes; AUTH stays failing.
  - id: tree-view
    file: examples/02_tree_view.tsx
    title: "A component that renders itself per child, with open/closed state above the recursion"
    runtime: ts-browser
    check: exact
    annotations:
      - lines: "8-25"
        note: >
          The recursion: TreeNode renders a TreeNode per child (line 19). The
          call depth is the tree's depth, a handful of levels for
          requirements, so stack depth is no concern.
      - lines: "13-15"
        note: >
          A real button with aria-expanded: keyboard and screen readers get a
          disclosure for free. role="tree" would promise the arrow-key
          navigation of the ARIA tree pattern, which this view doesn't
          implement.
      - lines: "16-22"
        note: >
          A closed node doesn't render its subtree at all, so a collapsed
          branch of 500 requirements costs one list item.
      - lines: "19"
        note: >
          key={child.id}, never the index: a move shifts siblings, and id keys
          let React keep each node's DOM and state with the node.
      - lines: "28-35"
        note: >
          Open and closed state lives above the recursion, keyed by id. Kept in
          each TreeNode, it would die when an ancestor closes. Recording closed
          nodes, not open ones, makes new nodes from a refetch arrive expanded.
      - lines: "74-78"
        note: >
          Closing AUTH unmounts AUTH-LOGIN's TreeNode. Reopening AUTH still
          shows AUTH-LOGIN closed, because the set lives in TreeView.
      - lines: "79-80"
        note: >
          A refetch delivers new objects. The closed set is keyed by id, so
          AUTH-LOGIN stays closed, and the new REPORT arrives expanded.
  - id: detail
    file: examples/03_detail.tsx
    title: "The detail form: keyed by version, PATCHing only what changed"
    runtime: ts-browser
    check: exact
    timeout_s: 15
    annotations:
      - lines: "11-16"
        note: >
          A stand-in for GET and PATCH /api/requirements/{id}, abridged to the
          members the form edits. PATCH writes the members it receives and
          answers with the whole updated requirement, as step 4's handler does.
      - lines: "26-29"
        note: >
          PATCH answers with the updated resource, so it goes straight into the
          cache instead of being refetched. In the app onSuccess also
          invalidates ["projects", projectId]: the tree and the matrix show
          titles and statuses too.
      - lines: "34"
        note: >
          The inputs are uncontrolled, and defaultValue is only their initial
          value: after the user has typed, a new defaultValue doesn't change
          what an input shows. A key built from the version remounts the form
          when the server's data changes, and keeps it when it doesn't, so
          typing survives a failed save. The app's key is
          `${id}@${updatedAt}`, with updatedAt an RFC 3339 time.
      - lines: "38-43"
        note: >
          The title is trimmed before comparing, as the service trims it (step
          4), so trailing spaces aren't a change. Only the edited members go
          into the PATCH, and an unchanged form sends nothing: an empty PATCH
          would break the spec's minProperties: 1.
      - lines: "70-72"
        note: >
          Harness: the lesson iframe has no allow-forms, and requestSubmit()
          doesn't fire the submit event then. Dispatching it is what Enter in
          the title, or a Kaydet button, does in the app.
      - lines: "77-79"
        note: >
          Another tab changes the description on the server. This form's PATCH
          carries only the title, the answer carries both changes, and the
          remounted form shows them. Sending the whole form would have written
          back the empty description it loaded.
      - lines: "80"
        note: >
          The same title with trailing spaces: nothing to send, so no PATCH
          line appears.
  - id: matrix
    file: examples/04_matrix.test.tsx
    title: "A trace matrix cell: an optimistic toggle with rollback, tested through its accessible name"
    runtime: ts-test
    check: contains
    timeout_s: 15
    annotations:
      - lines: "13-24"
        note: >
          A stand-in for the link endpoints. server.down makes reads fail as
          well as writes, which the test below needs.
      - lines: "31-32"
        note: >
          The request carries the wanted state (PUT links, DELETE unlinks), not
          "toggle". Both are idempotent (step 4), so a retried request can't
          flip a cell twice.
      - lines: "33-39"
        note: >
          The optimistic update from ts.react.query, per cell: cancel a refetch
          in flight, keep a snapshot, write the wanted state into the cache.
          The cell flips before the server has answered.
      - lines: "40-41"
        note: >
          A refused change goes back to the snapshot. The mutation's error also
          drives the alert.
      - lines: "42"
        note: >
          Settled either way, the project prefix is invalidated. Coverage is
          computed by the server and can't be guessed here, so the coverage
          column refetches rather than following the cell.
      - lines: "54"
        note: >
          Each cell is a toggle button with a name ("AUTH-RESET × TC-3") and
          aria-pressed. A screen reader announces both, and the test finds the
          cell the same way.
      - lines: "70-74"
        note: >
          The held answer: the cell is already unlinked while the DELETE is
          pending.
      - lines: "75-78"
        note: >
          The server goes down after the cell flipped: the DELETE fails and so
          does every refetch after it, so only onError's snapshot can put the
          cell back. Delete onError and this test fails; with a working server
          the refetch would have hidden the missing rollback.
use_cases:
  - "Any hierarchy in a UI: requirement trees, folders, org charts, bills of materials, nested comments"
  - "Roll-ups over a tree: coverage per subtree, totals per folder, worst status per branch"
  - "Edit forms over shared records where several people change different fields"
  - "Matrices of toggles (traceability, permissions, feature flags per tenant) that must feel instant and stay correct"
strengths:
  - "One request and one parse for the whole tree; nesting is a linear pass in the client, with no request per level as in a lazily expanded tree"
  - "A recursive component mirrors the data exactly, and keys by id keep node state correct across moves and refetches"
  - "PATCH with only the edited members never overwrites someone else's change to another field, unlike a PUT of the whole form"
  - "Testing Library queries by role and name double as an accessibility check: a cell without a name can't be found by the test either"
weaknesses:
  - "Whole-tree loading stops scaling at some thousands of requirements; then the tree endpoint needs a root parameter and lazy subtrees"
  - "The roll-up rule (headings don't count, uncovered outranks incomplete) is a client decision; a report generated in Go (step 8) must implement the same rule or show different numbers"
  - "Last writer wins on the same field: two people editing one title don't notice each other. Refusing stale saves needs an If-Match with the version, which TraceLite doesn't do"
  - "An optimistic toggle shows a state the server hasn't confirmed; a slow or failing server makes cells flicker back"
pitfalls:
  - title: "Fetching children per node"
    detail: "A TreeNode that runs its own useQuery for its children makes one request per expanded level, in sequence: a waterfall as deep as the tree. Fetch the tree once and nest it in memory."
    bad: "function TreeNode({ id }) { const children = useQuery({ queryKey: [\"children\", id], ... }); ... }"
    good: "const roots = useMemo(() => nest(tree.data?.items ?? []), [tree.data]);"
  - title: "Open/closed state inside the recursive component"
    detail: "useState in each TreeNode is lost when an ancestor collapses, because the subtree unmounts. Keep the set of closed ids above the recursion."
  - title: "Index keys in a tree"
    detail: "key={index} ties state to a position. After a move or an insert, a node inherits its neighbour's state and DOM. Use the requirement id."
    bad: "node.children.map((child, i) => <TreeNode key={i} ... />)"
    good: "node.children.map((child) => <TreeNode key={child.id} ... />)"
  - title: "A form that doesn't reset when the data changes"
    detail: "defaultValue is only the initial value. After a save, the inputs keep showing what they showed. Give the form a key that changes with the server's version (id and updatedAt)."
  - title: "Re-sorting what the server ordered"
    detail: "Codes are ordered by byte value (COLLATE \"C\") on the server. A client sort with localeCompare(\"tr\") orders them differently, and the tree, the list and the matrix stop agreeing. Render in the order received."
quiz:
  - q: "Why can nest() attach each item to its parent in a single pass?"
    options: ["Because ids are uuidv7 and sort by creation time", "Because depth-first order puts every parent before its children", "Because the client sorts by depth first"]
    answer: 1
    why: "SEARCH DEPTH FIRST emits a node, then its subtree. When a child arrives, its parent is already in the map."
  - q: "The user closes AUTH-LOGIN, then closes and reopens AUTH. Where must 'AUTH-LOGIN is closed' be stored for it to stay closed?"
    options: ["In AUTH-LOGIN's own useState", "Above the recursion, in a set of closed ids", "In the query cache"]
    answer: 1
    why: "Closing AUTH unmounts AUTH-LOGIN's TreeNode and its state with it. The set in TreeView survives, and it isn't server state, so it doesn't belong in the query cache."
  - q: "Another tab changed the description. The form sends a PATCH with only {\"title\": \"Parola ile oturum açma\"}. What does the requirement look like afterwards?"
    options: ["The other tab's description and the new title", "The description as the form loaded it", "The PATCH fails with 409"]
    answer: 0
    why: "Only the title is written. The answer carries the stored description, and the remounted form shows it."
  - q: "In the matrix test, why does the server go down after the cell flipped, instead of just failing the DELETE?"
    options: ["To test retries", "Because otherwise the refetch in onSettled would restore the cell even without onError", "Because Testing Library needs a failing fetch to find the alert"]
    answer: 1
    why: "With a working server, the invalidation refetches the real links and hides a missing rollback. A test should fail when the code it covers is removed."
further_reading:
  - title: "React: Preserving and resetting state (keys)"
    url: "https://react.dev/learn/preserving-and-resetting-state"
  - title: "React: Rendering lists (keys)"
    url: "https://react.dev/learn/rendering-lists"
  - title: "React: <input> (defaultValue, uncontrolled inputs)"
    url: "https://react.dev/reference/react-dom/components/input"
  - title: "MDN: aria-expanded"
    url: "https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Reference/Attributes/aria-expanded"
  - title: "MDN: aria-pressed"
    url: "https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Reference/Attributes/aria-pressed"
  - title: "MDN: ARIA tree role (the keyboard contract it implies)"
    url: "https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Reference/Roles/tree_role"
---

## Concept

TraceLite's main screen has three parts, each fed by the hooks of step 5:

- **The requirement tree.** `GET /requirement-tree` returns every requirement of the
  project as one flat list, depth first, with `depth`. The client nests it once (a map by
  id, one pass) and renders it with `TreeNode`, a component that renders a `TreeNode`
  for each child. Recursive data, rendered recursively.
- **The detail form.** `/projects/:projectId/requirements/:requirementId` shows one
  requirement. The form is uncontrolled, keyed by the server's version, and its PATCH
  carries only the fields the user changed.
- **The trace matrix.** Rows are the same depth-first list, indented; columns are the test
  cases; each cell is a toggle button for one trace link. Toggling is optimistic, and
  the coverage column waits for the server.

The **coverage indicator** uses step 2's vocabulary. The server computes it per
requirement (`GET /coverage`, new in this step); the tree rolls it up per subtree, so a
closed branch still shows its worst state and its counts.

The examples take one piece each: `tree.ts` (nesting and the roll-up), `TreeView` (the
recursive component), the detail form, and one matrix cell with its test.

## Coming from C/C++/Java/Python/JS

- **Java Swing/JavaFX**: a `TreeModel` hands the view children on demand. React has no
  model object: the nested data is the model, and the component tree mirrors it. Lazy
  loading is possible, but each level becomes a request.
- **C/C++**: building the tree is the classic "array of nodes with parent indices to
  pointer tree" pass; the `Map` plays the role of the index array, and ordering makes
  one pass enough.
- **Python/Django templates**: a recursive `{% include %}` renders the same shape on the
  server. Here the recursion runs in the browser, and the state that HTML can't keep
  (which nodes are closed) lives in a `Set`.
- **JS without a cache library**: a form that copies fetched data into `useState` and a
  `useEffect` to re-sync it is the usual source of stale forms. A `key` built from the
  version does the re-sync by remounting, with no effect at all.

## How it works

Routes and feature folders:

```text
/projects/:projectId                               TreeView + TraceMatrix
/projects/:projectId/requirements/:requirementId   RequirementPage (detail form)

web/src/features/requirements/   tree.ts (nest, tallies), TreeView.tsx, RequirementPage.tsx
web/src/features/matrix/         TraceMatrix.tsx, useToggleLink.ts
```

What each view reads and writes:

| View | Queries (keys) | Writes |
|---|---|---|
| Tree | `[…, "requirement-tree"]`, `[…, "coverage"]` | none |
| Detail | `["requirements", id]` | `PATCH /api/requirements/{id}` |
| Matrix | `[…, "requirement-tree"]`, `[…, "test-cases", "all"]`, `[…, "trace-links"]`, `[…, "coverage"]` | `PUT`/`DELETE …/test-cases/{testCaseId}` |

Every write invalidates `["projects", projectId]`, so all three views converge on the
server's state; step 9 triggers the same invalidation from server events.

The detail route reads its id from the URL. `useParams` returns `string | undefined`,
typed but unchecked, so the branded id comes only out of `safeParse`, and a malformed URL
shows "bulunamadı" without a request, like the server's 404 for a malformed path id:

```tsx
function RequirementPage() {
  const id = RequirementId.safeParse(useParams().requirementId);
  if (!id.success) return <p>bulunamadı</p>;
  return <RequirementDetail id={id.data} />; // example 3, with the real id
}
```

A failed save (422) doesn't change the version, so the keyed form isn't remounted: the
problem's errors appear under their fields, as in step 5's create form, next to what the
user typed.

The matrix doesn't nest: a table row can't contain rows. Each `<tr>` is one item of the
flat depth-first list, with the code in a `<th scope="row">` indented by `depth`, so its
order is the tree's. Each cell is example 4's toggle button, and the last cell shows the
server's coverage with `passingTests/linkedTests`, which keeps its old value until the
refetch that `onSettled` starts. A matrix with half its links is wrong, not partial, so
one query follows `nextCursor` to the end instead of a `useInfiniteQuery` that shows
pages as they arrive (at 200 per page, 2,000 links are ten sequential requests). The
labels are a `Record` over the union, the table form of the exhaustive switch:

```ts
async function fetchAll<T>(item: z.ZodType<T>, path: string): Promise<T[]> {
  const all: T[] = [];
  for (let cursor: string | null = null; ; ) {
    const query = new URLSearchParams(cursor ? { cursor } : {});
    const page = await request(pageOf(item), `${path}?${query}`);
    all.push(...page.items);
    if (page.nextCursor === null) return all;
    cursor = page.nextCursor;
  }
}

const label = {
  failing: "başarısız",
  uncovered: "testsiz",
  incomplete: "eksik",
  passing: "geçiyor",
} satisfies Record<Coverage, string>; // a value without a label fails tsc
```

The coverage endpoint, added to `openapi.yaml` next to the tree:

```yaml
  /api/projects/{projectId}/coverage:
    parameters: [{ $ref: "#/components/parameters/ProjectId" }]
    get:
      operationId: getCoverage
      responses:
        "200":
          description: One item per requirement of the project
          content: { application/json: { schema: { $ref: "#/components/schemas/CoverageList" } } }
        "404": { $ref: "#/components/responses/Problem" }
# components.schemas:
    RequirementCoverage:
      type: object
      required: [requirementId, linkedTests, passingTests, coverage]
      properties:
        requirementId: { type: string, format: uuid }
        linkedTests: { type: integer, minimum: 0 }
        passingTests: { type: integer, minimum: 0 }
        coverage: { type: string, enum: [uncovered, failing, passing, incomplete] }
    CoverageList:
      type: object
      required: [items]
      properties:
        items: { type: array, items: { $ref: "#/components/schemas/RequirementCoverage" } }
```

The store's query, `queries/coverage.sql`, is step 2's coverage query with the passing
count added and `count(*) FILTER` throughout:

```sql
-- name: ProjectCoverage :many
SELECT r.id AS requirement_id,
       count(l.test_case_id) AS linked_tests,
       count(*) FILTER (WHERE last.result = 'passed') AS passing_tests,
       CASE WHEN count(l.test_case_id) = 0 THEN 'uncovered'
            WHEN count(*) FILTER (WHERE last.result = 'failed') > 0 THEN 'failing'
            WHEN count(*) FILTER (WHERE last.result = 'passed') = count(*) THEN 'passing'
            ELSE 'incomplete' END AS coverage
FROM requirements r
LEFT JOIN trace_links l ON l.project_id = r.project_id AND l.requirement_id = r.id
LEFT JOIN LATERAL (SELECT result FROM test_runs WHERE test_case_id = l.test_case_id
                   ORDER BY finished_at DESC, id DESC LIMIT 1) AS last ON true
WHERE r.project_id = @project_id
GROUP BY r.id;
```

The roll-up rule the tree applies on top: a requirement with children and no tests of
its own is a heading and adds nothing; every other requirement adds its own coverage; a
parent shows the worst value in its subtree, in the order failing, uncovered,
incomplete, passing, and when closed, the counts. Step 8's report uses the same rule.

## Design notes

**Flat on the wire, nested in the client.** The server could return nested JSON, but the
recursive CTE produces rows, the matrix wants rows, and a flat list keeps the schema
non-recursive (zod and openapi-typescript both handle recursive schemas, with more
ceremony). Nesting is ten lines and runs once per fetch: `TreeView` computes `nest` and
`tallies` in a `useMemo` over the query data, and TanStack's structural sharing keeps the
old data object when a refetch returns equal JSON, so a no-op refetch doesn't rebuild the
tree. Lazy subtrees (`?root=`) are the upgrade path when a project outgrows one response.

**Server state and UI state stay apart.** The tree, coverage and requirement live in the
query cache; which nodes are closed lives in `useState` above the recursion; the
selected requirement lives in the URL (ts.react.routing), so it survives a reload and can
be shared. Nothing is copied from the cache into component state.

**Uncontrolled form, keyed by version.** Controlled inputs would need state initialized
from the query and an effect to re-sync it after saves and refetches. `defaultValue`
plus `key={id@updatedAt}` gets the same result with no effect: a new version remounts
the form, the same version leaves the user's typing alone. The cost is that a refetch
bringing someone else's change also discards unsaved typing in that form; TraceLite
accepts that for a single-user-at-a-time tool.

**Optimistic cells, authoritative coverage.** Whether a link exists is something the
client knows the moment the user clicks. Coverage depends on runs the client doesn't
have, so it is never guessed: the matrix shows the old value until the refetch. The
rejected alternative, recomputing coverage in the browser, would put step 2's rule in
two places, and the next change to the rule would make them disagree.
