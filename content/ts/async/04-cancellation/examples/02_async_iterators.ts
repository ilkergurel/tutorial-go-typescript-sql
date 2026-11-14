// LEARN(ts.async.cancellation): break out of for await and the generator's finally runs; a signal stops it from outside.

interface Page {
  items: string[];
  next: number | null;
}
const PAGES: Page[] = [
  { items: ["A", "B", "C"], next: 1 },
  { items: ["D", "E", "F"], next: 2 },
  { items: ["G", "H"], next: null },
];
// Stands in for fetch(`/api/items?cursor=${cursor}`).
const fetchPage = async (cursor: number): Promise<Page> => PAGES[cursor];

async function* allItems(signal?: AbortSignal): AsyncGenerator<string> {
  let cursor: number | null = 0;
  try {
    while (cursor !== null) {
      signal?.throwIfAborted(); // check between pages, before more work
      const page = await fetchPage(cursor);
      console.log(`  fetched page ${cursor}`);
      yield* page.items;
      cursor = page.next;
    }
  } finally {
    console.log("  cleanup in finally");
  }
}

// 1. The consumer has what it needs: break ends the generator.
for await (const item of allItems()) {
  if (item === "E") {
    console.log(`found ${item}`);
    break;
  }
}

// 2. Someone else cancels: the generator throws at its next check.
const controller = new AbortController();
const seen: string[] = [];
try {
  for await (const item of allItems(controller.signal)) {
    seen.push(item);
    if (seen.length === 2) controller.abort(); // e.g. the dialog was closed
  }
} catch (e) {
  console.log(
    `stopped after ${seen.join("")}: ${e instanceof Error ? e.name : e}`,
  );
}

export {}; // module: top-level await allowed
