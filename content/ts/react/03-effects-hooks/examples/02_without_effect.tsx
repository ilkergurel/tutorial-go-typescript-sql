// LEARN(ts.react.effects-hooks): derive values during render; an effect that only sets state is a smell.
import { useEffect, useState } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

type Name = { first: string; last: string };
const initials = (first: string, last: string) =>
  (first[0] + last[0]).toLocaleUpperCase("tr");

// Anti-pattern: a copy of props in state, kept in sync by an effect.
function EffectBadge({ first, last }: Name) {
  const [label, setLabel] = useState("");
  useEffect(() => setLabel(initials(first, last)), [first, last]);
  console.log(`  EffectBadge renders "${label}"`);
  return <b>{label}</b>;
}

// Derived during render: one render, never out of date.
function DerivedBadge({ first, last }: Name) {
  const label = initials(first, last);
  console.log(`  DerivedBadge renders "${label}"`);
  return <b>{label}</b>;
}

const host = document.createElement("div");
const root = createRoot(host);
const tick = () => new Promise((r) => setTimeout(r, 0));

for (const Badge of [EffectBadge, DerivedBadge]) {
  console.log(Badge.name);
  flushSync(() => root.render(<Badge first="ilkay" last="ışık" />));
  console.log(`  committed: "${host.textContent}"`);
  await tick(); // effects and the updates they schedule get their turn
  console.log(`  a moment later: "${host.textContent}"`);
}
root.unmount();
