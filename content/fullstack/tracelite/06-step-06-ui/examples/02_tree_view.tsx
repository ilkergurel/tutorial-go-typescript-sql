import { useState } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

// --- src/features/requirements/TreeView.tsx; Node is example 1's, abridged.
type Node = { id: string; code: string; children: Node[] };
type View = { closed: ReadonlySet<string>; toggle: (id: string) => void };
function TreeNode({ node, view }: { node: Node; view: View }) {
  if (node.children.length === 0) return <li>{`· ${node.code}`}</li>;
  const open = !view.closed.has(node.id);
  return (
    <li>
      <button aria-expanded={open} onClick={() => view.toggle(node.id)}>
        {`${open ? "▾" : "▸"} ${node.code}`}
      </button>
      {open && (
        <ul>
          {node.children.map((child) => (
            <TreeNode key={child.id} node={child} view={view} />
          ))}
        </ul>
      )}
    </li>
  );
}

function TreeView({ roots }: { roots: Node[] }) {
  // UI state above the recursion: it outlives collapsed subtrees and refetches.
  const [closed, setClosed] = useState<ReadonlySet<string>>(new Set());
  const toggle = (id: string) =>
    setClosed((prev) => {
      const next = new Set(prev);
      if (!next.delete(id)) next.add(id);
      return next;
    });
  return (
    <ul>
      {roots.map((node) => (
        <TreeNode key={node.id} node={node} view={{ closed, toggle }} />
      ))}
    </ul>
  );
}

// --- Harness: render, click, print each visible item indented by its depth.
function node(code: string, children: Node[] = []): Node {
  return { id: `id-${code}`, code, children }; // a uuid in the app
}
const tree = () => [
  node("AUTH", [
    node("AUTH-LOGIN", [node("AUTH-LOGIN-2FA"), node("AUTH-LOGIN-LOCKOUT")]),
    node("AUTH-RESET"),
  ]),
  node("EXPORT", [node("EXPORT-CSV")]),
];
const host = document.createElement("div");
const root = createRoot(host);
const render = (roots: Node[]) =>
  flushSync(() => root.render(<TreeView roots={roots} />));
function click(code: string) {
  const buttons = [...host.querySelectorAll("button")];
  flushSync(() => buttons.find((b) => b.textContent!.endsWith(code))!.click());
}
function print(title: string) {
  console.log(title);
  const items = [...host.querySelectorAll("li")];
  for (const li of items) {
    const depth = items.filter((a) => a !== li && a.contains(li)).length;
    console.log("  ".repeat(depth + 1) + li.firstChild!.textContent);
  }
}

render(tree());
click("AUTH-LOGIN");
click("AUTH");
print("AUTH-LOGIN closed, then AUTH");
click("AUTH");
print("AUTH open again");
render([...tree(), node("REPORT", [node("REPORT-PDF")])]); // a refetch
print("refetched, with a new REPORT");
