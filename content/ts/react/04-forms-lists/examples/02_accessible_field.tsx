// LEARN(ts.react.forms-lists): accessible fields: useId wires label, input and error message together.
import { useId, type ComponentPropsWithoutRef } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

type FieldProps = ComponentPropsWithoutRef<"input"> & {
  label: string;
  error?: string;
};

// One component owns the wiring, so no form can forget it.
function Field({ label, error, ...input }: FieldProps) {
  const id = useId(); // unique per instance, stable across renders
  const errorId = `${id}-error`;
  return (
    <div>
      <label htmlFor={id}>{label}</label>
      <input
        id={id}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? errorId : undefined}
        {...input}
      />
      {error && <p id={errorId}>{error}</p>}
    </div>
  );
}

const host = document.body.appendChild(document.createElement("div"));
const root = createRoot(host);
flushSync(() =>
  root.render(
    <form noValidate>
      <Field label="Ad" name="name" defaultValue="Ayşe" />
      <Field label="E-posta" name="email" error="Geçerli bir e-posta girin" />
    </form>,
  ),
);

// Roughly what a screen reader announces when each input gets focus.
for (const input of host.querySelectorAll("input")) {
  const label = input.labels?.[0]?.textContent;
  const describedBy = input.getAttribute("aria-describedby");
  const hint = describedBy && document.getElementById(describedBy)?.textContent;
  const invalid = input.getAttribute("aria-invalid") === "true";
  const state = invalid ? `invalid, described by "${hint}"` : "valid";
  console.log(`${label} [id ${input.id}]: ${state}`);
}
root.unmount();
