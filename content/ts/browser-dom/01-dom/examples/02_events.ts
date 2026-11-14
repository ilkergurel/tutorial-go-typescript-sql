// LEARN(ts.browser-dom.dom): the event map types each event; delegation narrows target; signals remove listeners.

document.body.innerHTML = `
  <ul id="cities">
    <li data-plate="06"><button>Ankara</button></li>
    <li data-plate="35"><button>İzmir</button></li>
  </ul>
  <input id="filter">`;

const list = document.querySelector("ul");
const filter = document.querySelector("input");
if (!list || !filter) throw new Error("markup changed");

// One listener on the parent handles every row, including rows added later.
list.addEventListener("click", (e) => {
  // e is PointerEvent; e.target is only EventTarget | null: narrow it.
  if (!(e.target instanceof Element)) return;
  const row = e.target.closest("li"); // HTMLLIElement | null
  if (row) console.log(`clicked plate ${row.dataset.plate}`);
});
list.querySelectorAll("button")[1].click();

// Setting value from code fires no event; real typing does.
filter.addEventListener("input", () => {
  console.log(`input event: ${filter.value}`);
});
filter.value = "iz";
filter.dispatchEvent(new InputEvent("input", { bubbles: true }));

// Custom events: declare the detail type once, in the event map.
declare global {
  interface HTMLElementEventMap {
    "city:select": CustomEvent<{ plate: string }>;
  }
}
list.addEventListener("city:select", (e) =>
  console.log(`plate ${e.detail.plate}`),
);
list.dispatchEvent(new CustomEvent("city:select", { detail: { plate: "34" } }));

// An AbortSignal removes listeners without keeping the function around.
const keys: string[] = [];
const listening = new AbortController();
document.addEventListener("keydown", (e) => keys.push(e.key), {
  signal: listening.signal, // e is KeyboardEvent: e.key is typed
});
document.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter" }));
listening.abort();
document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" }));
console.log(`keys seen: ${keys.join(", ")}`);

export {}; // module: needed for declare global
