import {
  QueryClient,
  QueryClientProvider,
  useMutation,
  useQueryClient,
} from "@tanstack/react-query";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

// --- A stand-in for example 2's client: a refusal is an ApiError.
class ApiError extends Error {
  readonly problem: { errors: { pointer: string; detail: string }[] };
  constructor(pointer: string, detail: string) {
    super(detail);
    this.problem = { errors: [{ pointer, detail }] };
  }
}
const codes = ["AUTH", "AUTH-LOGIN"];
async function createRequirement(body: { code: string; title: string }) {
  if (codes.includes(body.code)) throw new ApiError("#/code", "already used");
  codes.push(body.code);
}

// --- The form. Its mutation is useCreateRequirement in src/api/queries.ts.
function CreateForm({ projectId }: { projectId: string }) {
  const queryClient = useQueryClient();
  const create = useMutation({
    mutationFn: createRequirement,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: ["projects", projectId] }),
  });
  const problem =
    create.error instanceof ApiError ? create.error.problem : null;
  const errorOf = (field: string) =>
    problem?.errors.find((e) => e.pointer === `#/${field}`)?.detail;
  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        const form = new FormData(e.currentTarget);
        // toUpperCase(), not toLocaleUpperCase(): in Turkish, i becomes İ
        const code = String(form.get("code")).trim().toUpperCase();
        create.mutate({ code, title: String(form.get("title")) });
      }}
    >
      <input name="code" />
      {errorOf("code") && <small>code: {errorOf("code")}</small>}
      <input name="title" />
      <button>Ekle</button>
      {create.isSuccess && <small>{create.variables.code} eklendi</small>}
    </form>
  );
}

// --- Harness: fill the inputs, submit, let the mutation finish, print notes.
const client = new QueryClient({
  defaultOptions: { mutations: { gcTime: Infinity } },
});
const host = document.createElement("div");
flushSync(() =>
  createRoot(host).render(
    <QueryClientProvider client={client}>
      <CreateForm projectId="01937b6e-0000-7000-8000-00000000000a" />
    </QueryClientProvider>,
  ),
);
const tick = () => new Promise((resolve) => setTimeout(resolve, 0));
async function submit(code: string, title: string) {
  host.querySelector<HTMLInputElement>("[name=code]")!.value = code;
  host.querySelector<HTMLInputElement>("[name=title]")!.value = title;
  // The lesson iframe may not submit forms: send the event Ekle would.
  const event = new Event("submit", { bubbles: true, cancelable: true });
  host.querySelector("form")!.dispatchEvent(event);
  await tick(); // the mutation settles...
  await tick(); // ...and TanStack notifies components in a 0 ms timer batch
  const notes = [...host.querySelectorAll("small")].map((s) => s.textContent);
  console.log(`${JSON.stringify([code, title])} -> ${notes.join(" | ")}`);
}
await submit("auth-login", "Parola ile giriş");
await submit("auth-login-sso", "Kurumsal giriş");
