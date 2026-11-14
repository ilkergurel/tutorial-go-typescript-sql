// LEARN(ts.react.forms-lists): a controlled input: React state is the source of truth; submit is a typed handler.
import { useState, type ChangeEvent, type FormEvent } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";

function CityForm({ onSave }: { onSave: (city: string) => void }) {
  const [city, setCity] = useState("");
  const [error, setError] = useState<string | null>(null);

  function change(e: ChangeEvent<HTMLInputElement>) {
    setCity(e.target.value); // exactly what was typed, synchronously
  }
  function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault(); // no page reload: the data stays in React
    const value = city.trim();
    setError(value === "" ? "Şehir zorunlu" : null);
    if (value !== "") onSave(value.toLocaleUpperCase("tr")); // normalize once
  }
  return (
    <form onSubmit={submit}>
      <label>
        Şehir <input name="city" value={city} onChange={change} />
      </label>
      {error && <p role="alert">{error}</p>}
      <button type="submit">Kaydet</button>
    </form>
  );
}

const host = document.createElement("div");
const root = createRoot(host);
const save = (city: string) => console.log(`saved ${city}`);
flushSync(() => root.render(<CityForm onSave={save} />));
const input = host.querySelector("input")!;

// Test plumbing, as Testing Library does it: a plain `input.value = ...`
// never reaches onChange, so use the native setter and fire "input".
function type(text: string) {
  const proto = HTMLInputElement.prototype;
  const { set } = Object.getOwnPropertyDescriptor(proto, "value")!;
  flushSync(() => {
    set!.call(input, text);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}
// The lesson iframe is sandboxed without allow-forms, so a click on the
// button submits nothing. Dispatch the submit event React listens to.
function submit() {
  const event = new Event("submit", { bubbles: true, cancelable: true });
  flushSync(() => host.querySelector("form")!.dispatchEvent(event));
  const alert = host.querySelector('[role="alert"]')?.textContent ?? "none";
  console.log(`prevented=${event.defaultPrevented}, error: ${alert}`);
}

submit();
type("  izmir ");
console.log(`input shows "${input.value}"`);
submit();
root.unmount();
