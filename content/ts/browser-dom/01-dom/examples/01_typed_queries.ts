// LEARN(ts.browser-dom.dom): tag selectors are typed; anything else is Element and needs instanceof.

document.body.innerHTML = `
  <h1 id="title">Sipariş</h1>
  <form><input id="qty" name="qty" type="number" value="3"></form>
  <ul><li>Ankara</li><li>İzmir</li><li>Iğdır</li></ul>`;

// A bare tag name picks a typed overload: HTMLFormElement | null.
const form = document.querySelector("form");
console.log(`form has ${form?.elements.length ?? 0} control(s)`);

// Any other selector is just a string to tsc: the result is Element | null.
const qty = document.querySelector("#qty");
// qty.valueAsNumber;
// error TS18047: 'qty' is possibly 'null'.
// after a null check: error TS2339: Property 'valueAsNumber' does not exist on type 'Element'.
if (qty instanceof HTMLInputElement) {
  console.log(`qty + 1 = ${qty.valueAsNumber + 1}`); // checked at run time
}

// The type argument is an unchecked cast: it compiles, and it lies.
const lie = document.querySelector<HTMLInputElement>("#title");
console.log(`lie.value is ${typeof lie?.value}: #title is an <h1>`);

// Check once, fail loudly, return the precise type.
function must<T extends Element>(selector: string, type: new () => T): T {
  const el = document.querySelector(selector);
  if (el instanceof type) return el;
  throw new Error(`${selector}: expected ${type.name}, found ${el?.tagName}`);
}
try {
  must("#title", HTMLInputElement);
} catch (e) {
  console.log(e instanceof Error ? e.message : String(e));
}

// querySelectorAll("li") is NodeListOf<HTMLLIElement>: iterable, not an array.
const cities = [...document.querySelectorAll("li")].map((li) =>
  (li.textContent ?? "").toLocaleUpperCase("tr"),
);
console.log(cities.join(", "));
