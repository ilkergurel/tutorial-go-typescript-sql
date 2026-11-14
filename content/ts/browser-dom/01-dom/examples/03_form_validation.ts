// LEARN(ts.browser-dom.dom): built-in constraint validation, setCustomValidity for your rules, FormData for values.

document.body.innerHTML = `
  <form>
    <input name="email" type="email" required>
    <input name="qty" type="number" min="1" max="10" value="0">
    <input name="city" required>
  </form>`;

const form = document.querySelector("form");
if (!form) throw new Error("no form");
const [email, qty, city] = form.querySelectorAll("input"); // HTMLInputElement

// ValidityState has one flag per rule. validationMessage is in the browser's
// language, so map the flags to your own text.
function why(input: HTMLInputElement): string {
  const v = input.validity;
  if (v.valueMissing) return "missing";
  if (v.typeMismatch) return "not an email";
  if (v.rangeUnderflow) return "below min";
  if (v.customError) return input.validationMessage; // your own message
  return "invalid";
}

// Your own rule joins the built-in ones through setCustomValidity.
const CITIES = ["ANKARA", "İSTANBUL", "İZMİR"];
city.addEventListener("input", () => {
  const known = CITIES.includes(city.value.trim().toLocaleUpperCase("tr"));
  city.setCustomValidity(known ? "" : "unknown city"); // "" means valid
});

form.addEventListener("submit", (e) => {
  e.preventDefault(); // handle it in script instead of navigating
  if (!form.checkValidity()) {
    for (const input of [email, qty, city]) {
      if (!input.validity.valid) console.log(`  ${input.name}: ${why(input)}`);
    }
    return;
  }
  // FormData values are FormDataEntryValue: string | File. Narrow before use.
  const amount = new FormData(form).get("qty");
  if (typeof amount !== "string") return;
  console.log(`  ok: ${Number(amount)} x ${city.value}`);
});

// The lesson sandbox has no allow-forms, so requestSubmit() does nothing here;
// a synthetic submit event runs the same listener.
const attempt = (label: string): void => {
  console.log(label);
  city.dispatchEvent(new Event("input")); // code-set values fire no events
  form.dispatchEvent(new SubmitEvent("submit", { cancelable: true }));
};

email.value = "ayse@";
city.value = "izmit";
attempt("first try:");

email.value = "ayse@example.com";
qty.value = "2";
city.value = "izmir"; // tr upper-casing gives İZMİR; toUpperCase() gives IZMIR
attempt("second try:");
