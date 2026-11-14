import {
  QueryClient,
  QueryClientProvider,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { expect, it } from "vitest";

// --- A stand-in for the server's trace links of one project.
const links = new Set(["AUTH-RESET × TC-3"]);
const server = { down: false, hold: Promise.resolve() }; // set by the test
async function getLinks() {
  if (server.down) throw new Error("503");
  return [...links];
}
async function putLink(link: string, on: boolean) {
  await server.hold; // PUT links, DELETE unlinks: …/test-cases/{testCaseId}
  if (server.down) throw new Error("503");
  if (on) links.add(link);
  else links.delete(link);
}

// --- src/features/matrix/useToggleLink.ts
const linksKey = ["projects", "p1", "trace-links"];
function useToggleLink() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (cell: { link: string; on: boolean }) =>
      putLink(cell.link, cell.on),
    onMutate: async ({ link, on }) => {
      await client.cancelQueries({ queryKey: linksKey }); // or it undoes this
      const before = client.getQueryData<string[]>(linksKey) ?? [];
      const rest = before.filter((l) => l !== link);
      client.setQueryData(linksKey, on ? [...rest, link] : rest);
      return { before };
    },
    onError: (_error, _cell, saved) =>
      client.setQueryData(linksKey, saved?.before),
    onSettled: () => client.invalidateQueries({ queryKey: ["projects", "p1"] }),
  });
}

// --- A cell of TraceMatrix.tsx: a toggle button named by row and column.
function Cell({ link }: { link: string }) {
  const linked = useQuery({ queryKey: linksKey, queryFn: getLinks });
  const toggle = useToggleLink();
  const on = linked.data?.includes(link) ?? false;
  const flip = () => toggle.mutate({ link, on: !on });
  return (
    <>
      <button aria-label={link} aria-pressed={on} onClick={flip}>
        {on ? "●" : "·"}
      </button>
      {toggle.isError && <p role="alert">kaydedilemedi</p>}
    </>
  );
}

it("flips a cell at once and puts a refused change back", async () => {
  const defaultOptions = { queries: { retry: false } };
  const name = "AUTH-RESET × TC-3";
  render(
    <QueryClientProvider client={new QueryClient({ defaultOptions })}>
      <Cell link={name} />
    </QueryClientProvider>,
  );
  const release = Promise.withResolvers<void>();
  server.hold = release.promise; // the server takes its time to answer
  const linked = await screen.findByRole("button", { name, pressed: true });
  await userEvent.click(linked);
  await screen.findByRole("button", { name, pressed: false }); // unanswered
  server.down = true; // the DELETE fails, and so does every refetch after it
  release.resolve();
  expect(await screen.findByRole("alert")).toBeTruthy();
  await screen.findByRole("button", { name, pressed: true }); // onError's doing
});
