// LEARN(ts.generics.keyof): a generic component infers T from its props, and keyof T ties every column to a real field.

import type { ReactNode } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

interface Column<T> {
  key: keyof T & string; // string keys only: they double as React keys
  header: string;
  render?: (row: T) => ReactNode; // receives the fully typed row
}
interface TableProps<T> {
  rows: readonly T[];
  columns: readonly Column<T>[];
  rowKey: (row: T) => string;
  sortBy?: keyof T;
}

const tr = new Intl.Collator("tr", { numeric: true });
function byKey<T>(key: keyof T): (a: T, b: T) => number {
  return (a, b) => tr.compare(String(a[key]), String(b[key]));
}

// A generic function component: T is inferred from the props at each use.
function Table<T>({ rows, columns, rowKey, sortBy }: TableProps<T>) {
  const sorted = sortBy === undefined ? rows : rows.toSorted(byKey(sortBy));
  return (
    <table>
      <tbody>
        <tr>{columns.map((c) => <th key={c.key}>{c.header}</th>)}</tr>
        {sorted.map((row) => (
          <tr key={rowKey(row)}>
            {columns.map((c) => (
              <td key={c.key}>{c.render ? c.render(row) : String(row[c.key])}</td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  );
}

interface Customer {
  id: string;
  name: string;
  joined: Date;
}
const customers: Customer[] = [
  { id: "c1", name: "Şule", joined: new Date(Date.UTC(2024, 0, 15)) },
  { id: "c2", name: "Çağrı", joined: new Date(Date.UTC(2023, 5, 1)) },
  { id: "c3", name: "Işıl", joined: new Date(Date.UTC(2025, 2, 9)) },
];
const date = new Intl.DateTimeFormat("tr-TR", { dateStyle: "medium", timeZone: "UTC" });

const el = document.createElement("div");
flushSync(() =>
  createRoot(el).render(
    <Table
      rows={customers}
      rowKey={(c) => c.id}
      sortBy="name"
      columns={[
        { key: "name", header: "Ad" },
        { key: "joined", header: "Katılım", render: (c) => date.format(c.joined) },
        // { key: "email", header: "E-posta" },
        // error TS2322: Type '"email"' is not assignable to type '"id" | "joined" | "name"'.
      ]}
    />,
  ),
);
for (const row of el.querySelectorAll("tr")) {
  console.log([...row.children].map((cell) => cell.textContent).join(" | "));
}
