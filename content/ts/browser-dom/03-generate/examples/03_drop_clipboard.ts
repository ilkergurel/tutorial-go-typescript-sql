// LEARN(ts.browser-dom.generate): a drop target cancels dragover; copy/paste handlers own clipboardData.

document.body.innerHTML = `<div id="zone">Drop a CSV here</div>`;
const zone = document.querySelector("#zone");
if (!(zone instanceof HTMLDivElement)) throw new Error("no drop zone");

// Cancelling dragover is what makes an element a drop target.
zone.addEventListener("dragover", (e) => e.preventDefault());

let reading: Promise<void> = Promise.resolve();
zone.addEventListener("drop", (e) => {
  e.preventDefault(); // or the browser may open the file itself
  const files = [...(e.dataTransfer?.files ?? [])];
  const csvs = files.filter((f) => f.name.toLowerCase().endsWith(".csv"));
  // dispatchEvent can't wait for an async listener: keep the promise.
  reading = Promise.all(
    csvs.map(async (f) => {
      const [header] = (await f.text()).split(/\r?\n/);
      console.log(`dropped ${f.name}, header: ${header}`);
    }),
  ).then(() =>
    console.log(`skipped ${files.length - csvs.length} other file(s)`),
  );
});

// Real drags come from the user; synthetic events run the same listeners.
const drag = new DataTransfer();
drag.items.add(new File(["Müşteri;Tutar\r\nŞule;12,5\r\n"], "siparis.csv"));
drag.items.add(new File(["not a csv"], "logo.png", { type: "image/png" }));
const allowed = !zone.dispatchEvent(
  new DragEvent("dragover", { dataTransfer: drag, cancelable: true }),
);
console.log(`drop allowed: ${allowed}`);
zone.dispatchEvent(
  new DragEvent("drop", { dataTransfer: drag, cancelable: true }),
);
await reading;

// Copy: put tab-separated rows on the clipboard; spreadsheets paste them as cells.
const rows = [
  ["Müşteri", "Tutar"],
  ["Şule", "12,5"],
];
document.addEventListener("copy", (e) => {
  e.preventDefault(); // replace what the browser would copy
  e.clipboardData?.setData(
    "text/plain",
    rows.map((r) => r.join("\t")).join("\n"),
  );
});
const clip = new DataTransfer();
document.dispatchEvent(new ClipboardEvent("copy", { clipboardData: clip }));
console.log(`copied: ${JSON.stringify(clip.getData("text/plain"))}`);

// navigator.clipboard needs a user gesture or a permission; this sandboxed,
// unfocused iframe has neither, so the promise rejects.
try {
  await navigator.clipboard.writeText("Şule\t12,5");
} catch (e) {
  console.log(`clipboard.writeText: ${e instanceof DOMException ? e.name : e}`);
}

export {}; // module: top-level await allowed
