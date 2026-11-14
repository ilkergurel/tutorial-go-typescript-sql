import {
  QueryClient,
  QueryClientProvider,
  useMutation,
  useQuery,
} from "@tanstack/react-query";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

// --- A stand-in for the server: one requirement; PATCH writes what it gets.
const stored = { title: "Parola ile giriş", description: "", updatedAt: "v1" };
let version = 1; // updatedAt is an RFC 3339 time in the app
async function updateRequirement(patch: Partial<typeof stored>) {
  console.log(`  server: PATCH ${JSON.stringify(patch)}`);
  return { ...Object.assign(stored, patch, { updatedAt: `v${++version}` }) };
}
const keep = { gcTime: Infinity }; // no cleanup timer keeps the lesson running
const queryClient = new QueryClient({
  defaultOptions: { queries: keep, mutations: keep },
});

// --- src/features/requirements/RequirementPage.tsx
function RequirementPage({ id }: { id: string }) {
  const queryFn = async () => ({ ...stored }); // GET /api/requirements/{id}
  const query = useQuery({ queryKey: ["requirements", id], queryFn });
  const save = useMutation({
    mutationFn: updateRequirement,
    onSuccess: (r) => queryClient.setQueryData(["requirements", id], r),
  });
  if (!query.data) return <p>yükleniyor</p>;
  const r = query.data;
  return (
    <form
      key={`${id}@${r.updatedAt}`} // a new version remounts the inputs
      onSubmit={(e) => {
        e.preventDefault();
        const form = new FormData(e.currentTarget);
        const title = String(form.get("title")).trim(); // as the service does
        const description = String(form.get("description"));
        const patch: Partial<typeof stored> = {};
        if (title !== r.title) patch.title = title;
        if (description !== r.description) patch.description = description;
        if (Object.keys(patch).length > 0) save.mutate(patch);
      }}
    >
      <input name="title" defaultValue={r.title} />
      <textarea name="description" defaultValue={r.description} />
    </form>
  );
}

// --- Harness: set the title, submit, let TanStack settle, print the form.
const host = document.createElement("div");
flushSync(() =>
  createRoot(host).render(
    <QueryClientProvider client={queryClient}>
      <RequirementPage id="01937b6e-0000-7000-8000-000000000002" />
    </QueryClientProvider>,
  ),
);
const tick = () => new Promise((resolve) => setTimeout(resolve, 0));
async function settle(label: string) {
  await tick(); // the fake answers...
  await tick(); // ...and TanStack notifies components in a 0 ms timer batch
  const fields = Object.fromEntries(new FormData(host.querySelector("form")!));
  console.log(`${label}: ${JSON.stringify(fields)}`);
}
async function submit(title: string, label: string) {
  host.querySelector("input")!.value = title;
  // The lesson iframe may not submit forms: send the submit event ourselves.
  const event = new Event("submit", { bubbles: true, cancelable: true });
  host.querySelector("form")!.dispatchEvent(event);
  await settle(label);
}

await settle("loaded");
await updateRequirement({ description: "Başka sekmeden eklendi." });
await settle("another tab saved");
await submit("Parola ile oturum açma", "title saved");
await submit("Parola ile oturum açma  ", "unchanged once trimmed");
