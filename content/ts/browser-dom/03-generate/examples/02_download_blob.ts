// LEARN(ts.browser-dom.generate): Blob + object URL + a clicked <a download> saves a file; then revoke the URL.

function download(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob); // blob:<origin>/<uuid>, valid until revoked
  const a = document.createElement("a");
  a.href = url;
  a.download = filename; // a suggestion: the browser may adjust it
  document.body.append(a);
  a.click();
  a.remove();
  // The URL pins the Blob in memory. Deferring the revoke by one task leaves
  // the browser time to start the download: a common defensive choice.
  setTimeout(() => URL.revokeObjectURL(url), 0);
}

// The lesson sandbox has no allow-downloads, so this listener stands in for
// the browser's save dialog. On a real page, leave it out.
document.addEventListener("click", (e) => {
  if (!(e.target instanceof HTMLAnchorElement) || !e.target.download) return;
  e.preventDefault();
  const scheme = new URL(e.target.href).protocol;
  console.log(`browser would save "${e.target.download}" from a ${scheme} URL`);
});

const BOM = "\uFEFF"; // byte order mark: how Excel recognizes UTF-8
const csv = "Müşteri;Tutar (₺)\r\nŞule Kaya;1234,5\r\n";
const blob = new Blob([BOM, csv], { type: "text/csv;charset=utf-8" });

// Blob stores strings as UTF-8: ü, ş, Ş take 2 bytes, ₺ 3, the BOM 3.
const bytes = new Uint8Array(await blob.arrayBuffer());
const hex = [...bytes.slice(0, 4)].map((b) => b.toString(16).padStart(2, "0"));
console.log(`${csv.length} code units -> ${blob.size} bytes: ${hex.join(" ")}`);
const text = await blob.text();
console.log(`text() drops the BOM: ${!text.startsWith(BOM)}`);

download(blob, "siparişler.csv");

// JSON is the same three steps with another type and no BOM.
const json = JSON.stringify({ exportedAt: "2026-11-13", count: 1 }, null, 2);
download(new Blob([json], { type: "application/json" }), "export.json");

export {}; // module: top-level await allowed
