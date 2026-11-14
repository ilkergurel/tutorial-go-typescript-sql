import {
  QueryClient,
  QueryClientProvider,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";
interface Source {
  readonly readyState: number; // 0 connecting, 1 open, 2 closed for good
  addEventListener(type: string, listener: () => void): void;
  close(): void;
}
type Live = "connecting" | "live" | "reconnecting" | "closed";
const topics = ["requirements", "trace-links", "test-runs", "resync"];
function useProjectEvents(id: string, connect: (url: string) => Source) {
  const client = useQueryClient();
  const [state, setState] = useState<Live>("connecting");
  useEffect(() => {
    const source = connect(`/api/projects/${id}/events`);
    const refetch = () =>
      client.invalidateQueries({ queryKey: ["projects", id] });
    source.addEventListener("open", () => {
      setState("live");
      refetch(); // whatever changed while there was no stream
    });
    source.addEventListener("error", () =>
      setState(source.readyState === 2 ? "closed" : "reconnecting"),
    );
    for (const topic of topics) source.addEventListener(topic, refetch);
    return () => source.close();
  }, [id, client, connect]);
  return state;
}
// --- Harness: the iframe has no network, so the script plays the server.
const source = Object.assign(new EventTarget(), {
  readyState: 0,
  close: () => console.log("unmounted: the page closed the stream"),
});
function emit(type: string, readyState = source.readyState) {
  source.readyState = readyState;
  source.dispatchEvent(new Event(type));
}
const connect = () => source;
let requests = 0;
function Matrix({ id }: { id: string }) {
  const state = useProjectEvents(id, connect);
  const { data } = useQuery({
    queryKey: ["projects", id, "trace-links"],
    queryFn: async () => ++requests, // stands in for request()
  });
  return `${state}, requests: ${data}`;
}
const client = new QueryClient({
  defaultOptions: { queries: { gcTime: Infinity } },
});
const host = document.createElement("div");
const root = createRoot(host);
flushSync(() =>
  root.render(
    <QueryClientProvider client={client}>
      <Matrix id="01937b6e-0000-7000-8000-00000000000a" />
    </QueryClientProvider>,
  ),
);
async function step(what: string, action: () => void) {
  action();
  do await new Promise((r) => setTimeout(r, 0));
  while (client.isFetching() > 0);
  await new Promise((r) => setTimeout(r, 0)); // renders come in a 0 ms timer
  console.log(`${what.padEnd(32)} ${host.textContent}`);
}
await step("mounted", () => {});
await step("the stream opens", () => emit("open", 1));
await step("a trace-links event", () => emit("trace-links"));
await step("the network drops", () => emit("error", 0));
await step("EventSource reconnects by itself", () => emit("open", 1));
await step("the server refuses a reconnect", () => emit("error", 2));
root.unmount();
