// LEARN(ts.services.fetch): a body is a one-shot stream: read it once, clone() for a second reader, mind the charset.

const nameOf = (e: unknown) => (e instanceof Error ? e.name : String(e));

// 1. Reading consumes the stream; a second read rejects.
const res = Response.json({ city: "İzmir" });
console.log(await res.text(), `bodyUsed=${res.bodyUsed}`);
try {
  await res.json();
} catch (e) {
  console.log(`second read: ${nameOf(e)}`);
}

// 2. Two readers, e.g. a logger and the parser: clone before the first read.
const res2 = Response.json({ city: "İzmir" });
const forLog = res2.clone();
const data = (await res2.json()) as { city: string };
console.log(`parsed ${data.city}, logged ${await forLog.text()}`);

// 3. Empty and non-JSON bodies. Read text once, then parse, so the error can
// say what actually came back.
try {
  await new Response(null, { status: 204 }).json();
} catch (e) {
  console.log(`204 with json(): ${nameOf(e)}`);
}
async function readJson(r: Response): Promise<unknown> {
  const text = await r.text();
  if (text === "") return undefined; // 204 No Content is not a parse error
  try {
    return JSON.parse(text);
  } catch {
    const type = r.headers.get("Content-Type");
    throw new Error(`${r.status} ${type} is not JSON: ${text.slice(0, 20)}…`);
  }
}
console.log(await readJson(new Response(null, { status: 204 })));
const login = new Response("<!doctype html><title>Giriş</title>", {
  headers: { "Content-Type": "text/html; charset=utf-8" },
});
try {
  await readJson(login);
} catch (e) {
  console.log(e instanceof Error ? e.message : e);
}

// 4. text() always decodes UTF-8. A legacy Turkish export in windows-1254:
const csv = new Response(new Uint8Array([0xde, 0x75, 0x6c, 0x65]), {
  headers: { "Content-Type": "text/csv; charset=windows-1254" },
});
console.log(`text(): ${await csv.clone().text()}`); // 0xDE is Ş, not UTF-8
const type = csv.headers.get("Content-Type") ?? "";
const charset = /charset=([\w-]+)/i.exec(type)?.[1] ?? "utf-8";
const decoded = new TextDecoder(charset).decode(await csv.arrayBuffer());
console.log(`TextDecoder("${charset}"): ${decoded}`);

export {}; // module: top-level await allowed
