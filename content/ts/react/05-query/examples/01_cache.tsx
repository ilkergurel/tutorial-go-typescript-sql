// LEARN(ts.react.query): server state lives in a cache keyed by queryKey; components subscribe to it.
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useQuery } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

const requests: string[] = []; // the fake server's access log
async function fetchTitles(track: string): Promise<string[]> {
  requests.push(track);
  return track === "go" ? ["defer", "Kanallar"] : ["Tipler"];
}

type Props = { track: string; staleTime?: number };

function Titles({ track, staleTime = 0 }: Props) {
  const query = useQuery({
    queryKey: ["titles", track],
    queryFn: () => fetchTitles(track),
    staleTime,
  });
  // query.data.join(", ");
  // error TS18048: 'query.data' is possibly 'undefined'.
  if (query.status === "pending") return <p>{`${track}: yükleniyor`}</p>;
  if (query.status === "error") return <p>{`${track}: hata`}</p>;
  const refreshing = query.isFetching ? " (yenileniyor)" : "";
  return <p>{`${track}: ${query.data.join(", ")}${refreshing}`}</p>;
}

// gcTime: Infinity only so no garbage-collection timer outlives this run.
const client = new QueryClient({
  defaultOptions: { queries: { retry: false, gcTime: Infinity } },
});
const host = document.createElement("div");
const root = createRoot(host);
const render = (ui: ReactNode) =>
  flushSync(() =>
    root.render(
      <QueryClientProvider client={client}>{ui}</QueryClientProvider>,
    ),
  );
function print(step: string) {
  const shown = [...host.querySelectorAll("p")].map((p) => p.textContent);
  console.log(`${step.padEnd(9)} ${shown.join(" | ")}  [server: ${requests}]`);
}
async function settle() {
  const tick = () => new Promise((r) => setTimeout(r, 0));
  do await tick();
  while (client.isFetching() > 0);
  await tick(); // TanStack notifies components in a 0 ms timer batch
}

render(
  <>
    <Titles track="go" />
    <Titles track="go" />
    <Titles track="ts" />
  </>,
);
print("mount");
await settle();
print("loaded");
render(null); // unmounted: the cache keeps the data
render(<Titles track="go" />);
print("remount");
await settle();
print("refetched");
render(<Titles track="ts" staleTime={60_000} />);
print("fresh");
root.unmount();
client.clear();
