// LEARN(ts.react.state): useReducer: every transition in one pure, testable function.
import { useReducer } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

type Submit =
  | { status: "editing" }
  | { status: "sending"; attempt: number }
  | { status: "failed"; attempt: number; error: string }
  | { status: "done"; orderNo: string };

type Action =
  | { type: "send" }
  | { type: "fail"; error: string }
  | { type: "succeed"; orderNo: string };

function reducer(s: Submit, a: Action): Submit {
  switch (a.type) {
    case "send":
      if (s.status === "editing") return { status: "sending", attempt: 1 };
      if (s.status !== "failed") return s; // sending or done: ignore
      return { status: "sending", attempt: s.attempt + 1 };
    case "fail":
      if (s.status !== "sending") return s;
      return { status: "failed", attempt: s.attempt, error: a.error };
    case "succeed":
      if (s.status !== "sending") return s;
      // s.orderNo;
      // error TS2339: Property 'orderNo' does not exist on type '{ status: "sending"; attempt: number; }'.
      return { status: "done", orderNo: a.orderNo };
  }
}

// A reducer is a plain function: test the machine without rendering anything.
const script: Action[] = [
  { type: "succeed", orderNo: "TR-1" }, // nothing was sent: ignored
  { type: "send" },
  { type: "send" }, // double click: ignored
  { type: "fail", error: "Stok yetersiz" },
  { type: "send" },
  { type: "succeed", orderNo: "TR-1042" },
];
let state: Submit = { status: "editing" };
for (const action of script) {
  const next = reducer(state, action);
  const shown = next === state ? "(no change)" : JSON.stringify(next);
  console.log(`${action.type.padEnd(8)} ${shown}`);
  state = next;
}

// In a component, useReducer holds the state and hands out dispatch.
function Checkout() {
  const [s, dispatch] = useReducer(reducer, { status: "editing" });
  // dispatch({ type: "succeed" });
  // error TS2345: Argument of type '{ type: "succeed"; }' is not assignable to parameter of type 'Action'.
  return <button onClick={() => dispatch({ type: "send" })}>{s.status}</button>;
}
const host = document.createElement("div");
const root = createRoot(host);
flushSync(() => root.render(<Checkout />));
flushSync(() => host.querySelector("button")!.click());
flushSync(() => host.querySelector("button")!.click());
console.log(`after two clicks: ${host.textContent}`);
root.unmount();
