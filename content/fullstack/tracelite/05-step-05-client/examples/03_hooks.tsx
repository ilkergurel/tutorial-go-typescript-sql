import {
  QueryClient,
  QueryClientProvider,
  useInfiniteQuery,
} from "@tanstack/react-query";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

// --- A stand-in for example 2's client: one project in memory, pages of two.
const codes = ["AUTH", "AUTH-LOGIN", "EXPORT"];
async function listRequirements(projectId: string, cursor: string | null) {
  // The real cursor is opaque base64url; this fake's is the last code sent.
  const rest = codes.filter((code) => code > (cursor ?? ""));
  const items = rest.slice(0, 2);
  return { items, nextCursor: rest.length > 2 ? items[1] : null };
}

// --- src/api/queries.ts: every key starts with the path of what it caches.
const keys = {
  project: (id: string) => ["projects", id] as const,
  requirements: (id: string) => ["projects", id, "requirements"] as const,
};

function useRequirements(projectId: string) {
  return useInfiniteQuery({
    queryKey: keys.requirements(projectId),
    queryFn: ({ pageParam }) => listRequirements(projectId, pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) => lastPage.nextCursor, // null: no next page
  });
}

// --- A feature component: it renders the cache and never copies it.
function Requirements({ projectId }: { projectId: string }) {
  const list = useRequirements(projectId);
  if (list.status === "pending") return <p>yükleniyor</p>;
  if (list.status === "error") return <p>hata: {list.error.message}</p>;
  return (
    <>
      <p>{list.data.pages.flatMap((page) => page.items).join(", ")}</p>
      {list.hasNextPage && (
        <button onClick={() => list.fetchNextPage()}>daha fazla</button>
      )}
    </>
  );
}

// --- Harness: render, act, wait until the cache is idle, print the screen.
const project = "01937b6e-0000-7000-8000-00000000000a";
const client = new QueryClient({
  defaultOptions: { queries: { retry: false, gcTime: Infinity } },
});
const host = document.createElement("div");
flushSync(() =>
  createRoot(host).render(
    <QueryClientProvider client={client}>
      <Requirements projectId={project} />
    </QueryClientProvider>,
  ),
);
const tick = () => new Promise((resolve) => setTimeout(resolve, 0));
async function show(label: string) {
  do await tick();
  while (client.isFetching() > 0);
  await tick(); // TanStack notifies components in a 0 ms timer batch
  const screen = [...host.children].map((element) => element.textContent);
  console.log(`${label.padEnd(8)} ${screen.join(" | ")}`);
}
console.log(`mount    ${host.textContent}`);
await show("loaded");
host.querySelector("button")!.click();
await show("more");
// AUTH-2FA is created; example 4's mutation then invalidates the project.
codes.push("AUTH-2FA");
codes.sort(); // the server's order
await client.invalidateQueries({ queryKey: keys.project(project) });
await show("created");
