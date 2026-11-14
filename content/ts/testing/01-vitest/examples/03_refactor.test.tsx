import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useReducer, useRef, useState, type FormEvent } from "react";
import { afterEach, describe, expect, it } from "vitest";

const Items = ({ items }: { items: string[] }) => (
  <ul>
    {items.map((t) => (
      <li key={t}>{t}</li>
    ))}
  </ul>
);

// Version 1: useState and a controlled input.
function ListV1() {
  const [items, setItems] = useState<string[]>([]);
  const [draft, setDraft] = useState("");
  function add(e: FormEvent) {
    e.preventDefault();
    const text = draft.trim();
    if (text && !items.includes(text)) setItems([...items, text]);
    setDraft("");
  }
  return (
    <form onSubmit={add}>
      <label>
        Item <input value={draft} onChange={(e) => setDraft(e.target.value)} />
      </label>
      <button>Add</button>
      <Items items={items} />
    </form>
  );
}

// Version 2: useReducer and an uncontrolled input behind a ref.
const addOnce = (xs: string[], x: string) =>
  x && !xs.includes(x) ? [...xs, x] : xs;
function ListV2() {
  const [items, add] = useReducer(addOnce, []);
  const input = useRef<HTMLInputElement>(null);
  function submit(e: FormEvent) {
    e.preventDefault();
    add(input.current!.value.trim());
    input.current!.value = "";
  }
  return (
    <form onSubmit={submit}>
      <label>
        Item <input ref={input} />
      </label>
      <button>Add</button>
      <Items items={items} />
    </form>
  );
}

afterEach(cleanup);

// One suite, both versions: it only uses what a user sees and does.
describe.each([
  ["useState", ListV1],
  ["useReducer + ref", ListV2],
])("shopping list, %s version", (_, List) => {
  it("adds trimmed items once each and clears the field", async () => {
    const user = userEvent.setup();
    render(<List />);
    const field = screen.getByRole("textbox", { name: "Item" });
    await user.type(field, " çay {Enter}{Enter}simit{Enter}çay");
    await user.click(screen.getByRole("button", { name: "Add" }));
    const items = screen.getAllByRole("listitem").map((li) => li.textContent);
    expect(items).toEqual(["çay", "simit"]);
    expect(field).toHaveProperty("value", "");
  });
});
