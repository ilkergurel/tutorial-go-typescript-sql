// LEARN(ts.react.state): state is a per-render snapshot; updater functions queue; replace, never mutate.
import { useState } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

function Tags() {
  const [count, setCount] = useState(0);
  const [tags, setTags] = useState<string[]>(["go"]);

  function addThreeNaive() {
    setCount(count + 1); // count is the same number in this whole handler
    setCount(count + 1);
    setCount(count + 1);
  }
  function addThree() {
    setCount((c) => c + 1); // each updater receives the pending value
    setCount((c) => c + 1);
    setCount((c) => c + 1);
  }
  function mutate() {
    tags.push("sql"); // changes the array React holds...
    setTags(tags); // ...but it is the same object: Object.is says "no change"
  }
  function replace() {
    setTags([...tags, "ts"]); // a new array: React re-renders
  }

  return (
    <div>
      <p>{`count=${count} tags=${tags.join(",")}`}</p>
      <button onClick={addThreeNaive}>naive</button>
      <button onClick={addThree}>updater</button>
      <button onClick={mutate}>mutate</button>
      <button onClick={replace}>replace</button>
    </div>
  );
}

const host = document.createElement("div");
const root = createRoot(host);
flushSync(() => root.render(<Tags />));
for (const button of host.querySelectorAll("button")) {
  flushSync(() => button.click());
  const shown = host.querySelector("p")!.textContent;
  console.log(`${button.textContent!.padEnd(8)}-> ${shown}`);
}
root.unmount();
