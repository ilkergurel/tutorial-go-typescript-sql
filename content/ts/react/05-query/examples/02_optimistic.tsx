// LEARN(ts.react.query): a mutation updates the cache optimistically, rolls back on error, then invalidates.
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useMutation, useQuery } from "@tanstack/react-query";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

type Todo = { id: number; title: string; done: boolean };

// Fake server: a PATCH waits for the harness, so the optimistic state shows.
const db: Todo[] = [{ id: 1, title: "Kanalları oku", done: false }];
let respond: ((ok: boolean) => void) | null = null;
async function getTodos(): Promise<Todo[]> {
  console.log("  server: GET /todos");
  return db.map((t) => ({ ...t }));
}
async function toggleTodo(id: number): Promise<void> {
  console.log(`  server: PATCH /todos/${id}`);
  const ok = await new Promise<boolean>((r) => (respond = r));
  if (!ok) throw new Error("503");
  db.forEach((t) => t.id === id && (t.done = !t.done));
}

const todosKey = ["todos"] as const;

function Todos() {
  const todos = useQuery({ queryKey: todosKey, queryFn: getTodos });
  const toggle = useMutation({
    mutationFn: toggleTodo,
    onMutate: async (id, { client }) => {
      await client.cancelQueries({ queryKey: todosKey }); // no stale overwrite
      const previous = client.getQueryData<Todo[]>(todosKey);
      client.setQueryData<Todo[]>(todosKey, (old) =>
        old?.map((t) => (t.id === id ? { ...t, done: !t.done } : t)),
      );
      return { previous }; // arrives in onError as onMutateResult
    },
    onError: (_err, _id, onMutateResult, { client }) =>
      client.setQueryData(todosKey, onMutateResult?.previous),
    onSettled: (_data, _err, _id, _result, { client }) =>
      client.invalidateQueries({ queryKey: todosKey }),
  });
  if (todos.status !== "success") return <p>{todos.status}</p>;
  const failed = toggle.isError ? " (kaydedilemedi)" : "";
  return todos.data.map((t) => (
    <button key={t.id} onClick={() => toggle.mutate(t.id)}>
      {`${t.done ? "[x]" : "[ ]"} ${t.title}${failed}`}
    </button>
  ));
}

const client = new QueryClient({
  defaultOptions: {
    queries: { retry: false, gcTime: Infinity }, // no timers outlive the run
    mutations: { gcTime: Infinity },
  },
});
const host = document.createElement("div");
const root = createRoot(host);
const tick = () => new Promise((r) => setTimeout(r, 0));
async function settle(done: () => boolean, step: string) {
  while (!done()) await tick();
  await tick(); // TanStack notifies components in a 0 ms timer batch
  console.log(`${step.padEnd(12)}${host.textContent}`);
}
const idle = () => client.isFetching() + client.isMutating() === 0;
const app = (
  <QueryClientProvider client={client}>
    <Todos />
  </QueryClientProvider>
);
flushSync(() => root.render(app));
await settle(idle, "loaded");
for (const ok of [true, false]) {
  flushSync(() => host.querySelector("button")!.click());
  await settle(() => respond !== null, "optimistic");
  respond!(ok);
  respond = null;
  await settle(idle, ok ? "confirmed" : "rolled back");
}
root.unmount();
