// LEARN(ts.services.advanced): offset pages shift when rows are inserted; cursor pages don't. Follow the server's next link.

interface Order {
  id: number;
  title: string;
}
let rows: Order[] = []; // the server's table, newest first

// The fake API. Offset: ORDER BY id DESC LIMIT 2 OFFSET n. Cursor: WHERE
// id < last LIMIT 2, the cursor opaque to the client (base64 of the last id).
const api: typeof fetch = async (input) => {
  const url = new URL(String(input));
  let page: Order[];
  let more: boolean;
  if (url.searchParams.has("offset")) {
    const offset = Number(url.searchParams.get("offset"));
    page = rows.slice(offset, offset + 2);
    more = offset + 2 < rows.length;
    url.searchParams.set("offset", String(offset + 2));
  } else {
    const cursor = url.searchParams.get("cursor");
    const last = cursor ? Number(atob(cursor)) : Infinity;
    page = rows.filter((r) => r.id < last).slice(0, 2);
    more = rows.some((r) => r.id < (page.at(-1)?.id ?? 0));
    url.searchParams.set("cursor", btoa(String(page.at(-1)?.id)));
  }
  return Response.json({ items: page, next: more ? url.href : null });
};

// The client never builds page URLs: it follows next until it is null.
async function* pages(first: string, signal?: AbortSignal) {
  let url: string | null = first;
  while (url) {
    const res = await api(url, { signal });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    const page = (await res.json()) as { items: Order[]; next: string | null };
    yield page.items;
    url = page.next;
  }
}

for (const start of ["?offset=0", "?cursor="]) {
  rows = [5, 4, 3, 2, 1].map((id) => ({ id, title: `Sipariş ${id}` }));
  const seen: string[] = [];
  for await (const items of pages(`https://api.example.com/orders${start}`)) {
    seen.push(items.map((o) => o.id).join(","));
    // A new order arrives while the user is reading page 1.
    if (seen.length === 1) rows.unshift({ id: 6, title: "Sipariş 6" });
  }
  console.log(`${start.padEnd(10)} ${seen.join(" | ")}`);
}
