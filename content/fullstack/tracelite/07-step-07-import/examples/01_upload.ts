import { z } from "zod";

// The charset to declare. A strict decode of the first 64 KiB either works or
// throws; { stream: true } lets the slice end inside a character.
async function charsetOf(file: Blob): Promise<"utf-8" | "windows-1254"> {
  const head = await file.slice(0, 64 * 1024).arrayBuffer();
  try {
    new TextDecoder("utf-8", { fatal: true }).decode(head, { stream: true });
    return "utf-8";
  } catch {
    return "windows-1254"; // Excel's "CSV (Comma delimited)" on Turkish Windows
  }
}

// --- src/features/import: the file is the body, labelled with its charset.
const ImportResult = z.object({ imported: z.number().int() });
async function importFile(projectId: string, file: File) {
  const charset = await charsetOf(file);
  const res = await fetch(`/api/projects/${projectId}/requirement-import`, {
    method: "POST",
    headers: { "Content-Type": `text/csv; charset=${charset}` },
    body: file, // no FormData, and no copy of the file in JavaScript
  });
  if (!res.ok) throw new Error(`HTTP ${res.status}`); // ImportFailed in the app
  return ImportResult.parse(await res.json());
}

// --- No network in the lesson iframe: a fake server decodes by the label.
window.fetch = async (_path, init) => {
  const type = new Headers(init?.headers).get("Content-Type")!;
  const bytes = await (init?.body as Blob).arrayBuffer();
  const text = new TextDecoder(type.split("charset=")[1]).decode(bytes);
  const rows = text.split("\r\n").slice(1, -1); // without the header
  console.log(`  server: ${type}; line 3 is ${JSON.stringify(rows[1])}`);
  return Response.json({ imported: rows.length });
};

// --- One sheet, saved by Excel on Turkish Windows in its two CSV formats.
const sheet =
  "code;title;parent;status;description\r\n" +
  "AUTH-LOGIN;Parola ile giriş;AUTH;;\r\n" +
  "AUTH-LOGIN-2FA;İki adımlı doğrulama;AUTH-LOGIN;;\r\n";
const bom = new Uint8Array([0xef, 0xbb, 0xbf]);
// windows-1254 has one byte per Turkish letter; the rest of the sheet is ASCII.
const win1254: Record<string, number> = { ğ: 0xf0, ı: 0xfd, İ: 0xdd, ş: 0xfe };
const ansi = Uint8Array.from(sheet, (ch) => win1254[ch] ?? ch.charCodeAt(0));
const saved = {
  "CSV UTF-8": new File([bom, sheet], "gereksinimler.csv"),
  "CSV (Comma delimited)": new File([ansi], "gereksinimler.csv"),
};
const project = "01937b6e-0000-7000-8000-00000000000a";
for (const [format, file] of Object.entries(saved)) {
  const charset = await charsetOf(file);
  console.log(`"${format}": ${file.size} bytes, sent as ${charset}`);
  const { imported } = await importFile(project, file);
  console.log(`  ${imported} gereksinim içe aktarıldı`);
}

// ş's two UTF-8 bytes straddle the 64 KiB cut.
const cut = new Blob(["x".repeat(64 * 1024 - 1), "ş"]);
console.log(`ş cut at 64 KiB: charsetOf says ${await charsetOf(cut)}`);
try {
  const head = await cut.slice(0, 64 * 1024).arrayBuffer();
  new TextDecoder("utf-8", { fatal: true }).decode(head);
} catch (e) {
  console.log(`  without { stream: true }: ${(e as Error).name}`);
}
