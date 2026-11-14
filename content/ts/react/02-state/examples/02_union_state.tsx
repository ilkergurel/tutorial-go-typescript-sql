// LEARN(ts.react.state): one union-typed state instead of independent flags.
import { useState } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

// Not { sending: boolean; error?: string; orderNo?: string }: that type
// also admits "sending with an error" and "done without an order number".
type Submit =
  | { status: "editing" }
  | { status: "sending" }
  | { status: "failed"; error: string }
  | { status: "done"; orderNo: string };

function Checkout({ send }: { send: () => Promise<string> }) {
  const [state, setState] = useState<Submit>({ status: "editing" });
  async function submit() {
    if (state.status === "sending") return; // a double click sends once
    setState({ status: "sending" });
    try {
      setState({ status: "done", orderNo: await send() });
    } catch (e) {
      const error = e instanceof Error ? e.message : String(e);
      setState({ status: "failed", error });
    }
  }

  switch (state.status) {
    case "editing":
      return <button onClick={submit}>Siparişi gönder</button>;
    case "sending":
      return <button onClick={submit}>Gönderiliyor…</button>;
    case "failed": // only here does state.error exist
      return <button onClick={submit}>{`${state.error}, tekrar dene`}</button>;
    case "done":
      return <p>{`Sipariş alındı: ${state.orderNo}`}</p>;
  }
}

let requests = 0;
async function send(): Promise<string> {
  if (++requests === 1) throw new Error("Stok yetersiz");
  return "TR-1042";
}

const host = document.createElement("div");
const root = createRoot(host);
const click = () => flushSync(() => host.querySelector("button")!.click());
const tick = () => new Promise((r) => setTimeout(r, 0));
// Like a test's findByText: wait until the asynchronous answer is on screen.
async function show(text: string) {
  while (!host.textContent!.includes(text)) await tick();
  console.log(`${host.textContent} (requests: ${requests})`);
}

flushSync(() => root.render(<Checkout send={send} />));
await show("gönder");
click();
click(); // a double click: the second handler sees status "sending"
await show("Gönderiliyor");
await show("tekrar dene");
click();
await show("alındı");
root.unmount();
