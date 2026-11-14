// LEARN(ts.react.forms-lists): keys give list items identity; an index is not an identity.
import { useState } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

type City = { plate: number; name: string }; // plate codes: stable, unique ids

const initial: City[] = [
  { plate: 34, name: "İstanbul" },
  { plate: 6, name: "Ankara" },
  { plate: 35, name: "İzmir" },
];

// Each row owns local state, like any input the user has typed into.
function Row({ city }: { city: City }) {
  const [note, setNote] = useState("");
  return (
    <li>
      {city.name}
      <input value={note} onChange={(e) => setNote(e.target.value)} />
    </li>
  );
}

function Cities({ keyBy }: { keyBy: "index" | "plate" }) {
  const [cities, setCities] = useState(initial);
  return (
    <>
      <button onClick={() => setCities((cs) => cs.slice(1))}>İlkini sil</button>
      <ul>
        {cities.map((city, i) => (
          <Row key={keyBy === "index" ? i : city.plate} city={city} />
        ))}
      </ul>
    </>
  );
}

const proto = HTMLInputElement.prototype;
const { set } = Object.getOwnPropertyDescriptor(proto, "value")!; // types like a user
for (const keyBy of ["index", "plate"] as const) {
  const host = document.createElement("div");
  const root = createRoot(host);
  flushSync(() => root.render(<Cities keyBy={keyBy} />));
  const ankara = host.querySelectorAll("input")[1]; // type a note next to Ankara
  flushSync(() => {
    set!.call(ankara, "başkent");
    ankara.dispatchEvent(new Event("input", { bubbles: true }));
  });
  flushSync(() => host.querySelector("button")!.click()); // remove İstanbul
  const rows = [...host.querySelectorAll("li")].map(
    (li) => `${li.textContent}=${li.querySelector("input")!.value || "-"}`,
  );
  console.log(`key=${keyBy.padEnd(5)} ${rows.join(", ")}`);
  root.unmount();
}
